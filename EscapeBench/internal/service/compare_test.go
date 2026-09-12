package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

type compareFixture struct {
	comparator *Comparator
	store      *memoryStore
	digester   *fakeDigester
	clock      *fakeClock
	matrix     models.Matrix
	campaign   models.Campaign
}

func newCompareFixture(t *testing.T) *compareFixture {
	t.Helper()
	f := &compareFixture{store: newStore(), digester: &fakeDigester{digest: "harness-v1"}, clock: newClock()}
	params := models.MatrixParameters{
		Sizes:                []int{8, 16, 24, 32},
		PointerFieldVariants: []bool{false},
		Profiles:             []models.LifetimeProfile{models.ProfileLocal},
		PassingModes:         models.PassingModes(),
	}
	matrix, err := models.NewMatrix(params, "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.matrix = matrix
	f.store.matrices[matrix.ID] = matrix
	f.campaign = models.Campaign{
		ID: "C-1", MatrixID: matrix.ID, HarnessDigest: "harness-v1", HypothesesDigest: "d",
		HypothesisIDs: []string{"H-001"}, Count: models.MinCount, Status: models.CampaignCompleted,
		Provenance: newProvenance().provenance, StartedAt: f.clock.Now(), FinishedAt: f.clock.Now(),
	}
	if err := f.store.CreateCampaign(context.Background(), f.campaign); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	f.comparator = NewComparator(f.store, f.store, f.digester, f.clock)
	return f
}

// measure écrit une mesure constante pour un sujet.
func (f *compareFixture) measure(t *testing.T, subjectID string, ns float64, allocs int64) {
	t.Helper()
	m := completeMeasurementOf(subjectID, models.MinCount, ns, 8, allocs)
	m.CampaignID = f.campaign.ID
	if err := f.store.WriteMeasurement(context.Background(), m); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
}

// measureAll écrit, pour chaque taille, le temps du mode valeur puis du mode pointeur.
func (f *compareFixture) measureAll(t *testing.T, valueNs, pointerNs map[int]float64) {
	t.Helper()
	for _, cell := range f.matrix.Cells {
		ns := valueNs[cell.TypeSpec.SizeBytes]
		if cell.PassingMode == models.PassingPointer {
			ns = pointerNs[cell.TypeSpec.SizeBytes]
		}
		f.measure(t, cell.ID(), ns, 1)
	}
}

func TestUC004_MainFlow(t *testing.T) {
	t.Parallel()
	f := newCompareFixture(t)
	// La valeur est plus rapide jusqu'à 16 octets, le pointeur l'emporte à partir de 24.
	f.measureAll(t,
		map[int]float64{8: 10, 16: 10, 24: 20, 32: 30},
		map[int]float64{8: 12, 16: 12, 24: 12, 32: 12})

	report, err := f.comparator.Compare(context.Background(), f.campaign.ID)
	if err != nil {
		t.Fatalf("Compare : %v", err)
	}
	if len(report.Set.Comparisons) != 4 {
		t.Fatalf("%d comparaisons, 4 attendues", len(report.Set.Comparisons))
	}
	if report.Path == "" {
		t.Fatal("un fichier de comparaison doit être écrit")
	}
	if report.Set.Method == "" {
		t.Fatal("la méthode d'estimation doit être consignée")
	}
	// Étape 4 : delta des médianes et intervalle de confiance.
	bySize := map[int]models.Comparison{}
	for _, comparison := range report.Set.Comparisons {
		bySize[comparison.SizeBytes] = comparison
	}
	if got := bySize[8].DeltaNsPerOp; math.Abs(got-2) > 1e-9 {
		t.Fatalf("delta à 8 octets = %v, attendu 2", got)
	}
	if got := bySize[32].DeltaNsPerOp; math.Abs(got+18) > 1e-9 {
		t.Fatalf("delta à 32 octets = %v, attendu -18", got)
	}
	if bySize[8].MedianValueNs != 10 || bySize[8].MedianPointerNs != 12 {
		t.Fatalf("médianes = %+v", bySize[8])
	}
	// Étape 5 : le point de bascule est la plus petite taille au-delà de laquelle le pointeur
	// l'emporte partout. Mutation : rendre la plus petite taille favorable ⇒ échec attendu.
	tipping := report.Set.TippingPoints[models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill}]
	if tipping != 24 {
		t.Fatalf("point de bascule = %d, attendu 24", tipping)
	}
	if len(report.NotObserved) != 0 {
		t.Fatalf("aucune série sans bascule attendue : %v", report.NotObserved)
	}
	if report.ExcludedCount != 0 {
		t.Fatalf("%d paires exclues", report.ExcludedCount)
	}
}

func TestUC004_BR2_SignificationParLIntervalleSeulement(t *testing.T) {
	t.Parallel()
	// BR-004-2 : significant est vrai si et seulement si l'intervalle exclut zéro.
	// Mutation : appliquer un seuil sur le delta ⇒ échec attendu.
	f := newCompareFixture(t)
	// Des mesures identiques donnent un intervalle centré sur zéro.
	f.measureAll(t,
		map[int]float64{8: 10, 16: 10, 24: 10, 32: 10},
		map[int]float64{8: 10, 16: 10, 24: 10, 32: 10})
	report, err := f.comparator.Compare(context.Background(), f.campaign.ID)
	if err != nil {
		t.Fatalf("Compare : %v", err)
	}
	for _, comparison := range report.Set.Comparisons {
		if comparison.Significant {
			t.Fatalf("des mesures identiques ne sont pas significatives : %+v", comparison)
		}
		if comparison.CILow > 0 || comparison.CIHigh < 0 {
			t.Fatalf("l'intervalle doit contenir zéro : %+v", comparison)
		}
	}
	// A3 : aucune bascule observée.
	if got := report.Set.TippingPoints[models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill}]; got != models.TippingNotObserved {
		t.Fatalf("point de bascule = %d, « non observé » attendu", got)
	}
	if len(report.NotObserved) != 1 {
		t.Fatalf("la série sans bascule doit être signalée : %v", report.NotObserved)
	}
}

func TestUC004_A1_CampagneNonCompletee(t *testing.T) {
	t.Parallel()
	f := newCompareFixture(t)
	ctx := context.Background()
	for _, status := range []models.CampaignStatus{models.CampaignRunning, models.CampaignAborted} {
		if err := f.store.SetCampaignStatus(ctx, f.campaign.ID, status, f.clock.Now(), ""); err != nil {
			t.Fatalf("SetCampaignStatus : %v", err)
		}
		_, err := f.comparator.Compare(ctx, f.campaign.ID)
		if !errors.Is(err, ErrPrecondition) {
			t.Fatalf("statut %s : erreur = %v, ErrPrecondition attendue", status, err)
		}
		if len(f.store.comparisons) != 0 {
			t.Fatal("A1 : aucun fichier n'est écrit")
		}
	}
}

func TestUC004_A2_PaireIncomplete(t *testing.T) {
	t.Parallel()
	// A2 : la paire est exclue avec sa raison, la comparaison se poursuit (BR-004-1).
	f := newCompareFixture(t)
	for _, cell := range f.matrix.Cells {
		switch {
		case cell.TypeSpec.SizeBytes == 8 && cell.PassingMode == models.PassingPointer:
			// Mesure absente : la paire est incomplète.
		case cell.TypeSpec.SizeBytes == 16 && cell.PassingMode == models.PassingValue:
			m := models.Measurement{CampaignID: f.campaign.ID, SubjectID: cell.ID(),
				Status: models.MeasurementFailed, FailureReason: "panic"}
			if err := f.store.WriteMeasurement(context.Background(), m); err != nil {
				t.Fatalf("WriteMeasurement : %v", err)
			}
		default:
			f.measure(t, cell.ID(), 10, 1)
		}
	}
	report, err := f.comparator.Compare(context.Background(), f.campaign.ID)
	if err != nil {
		t.Fatalf("Compare : %v", err)
	}
	if report.ExcludedCount != 2 {
		t.Fatalf("%d paires exclues, 2 attendues", report.ExcludedCount)
	}
	for _, excluded := range report.Set.ExcludedPairs {
		if excluded.Reason == "" {
			t.Fatalf("BR-004-1 : aucune exclusion silencieuse : %+v", excluded)
		}
	}
	if len(report.Set.Comparisons) != 2 {
		t.Fatalf("%d comparaisons, 2 attendues", len(report.Set.Comparisons))
	}
}

func TestUC004_AucunePaireComplete(t *testing.T) {
	t.Parallel()
	f := newCompareFixture(t)
	_, err := f.comparator.Compare(context.Background(), f.campaign.ID)
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}

func TestUC004_MatriceSansPaire(t *testing.T) {
	t.Parallel()
	f := newCompareFixture(t)
	sansPaire := f.matrix
	sansPaire.Cells = []models.Cell{f.matrix.Cells[0]}
	f.store.matrices[f.matrix.ID] = sansPaire
	if _, err := f.comparator.Compare(context.Background(), f.campaign.ID); !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}

func TestUC004_HarnaisIncoherent(t *testing.T) {
	t.Parallel()
	f := newCompareFixture(t)
	autre := f.matrix
	autre.HarnessDigest = "harness-v2"
	f.store.matrices[f.matrix.ID] = autre
	if _, err := f.comparator.Compare(context.Background(), f.campaign.ID); !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
}

func TestUC004_ErreursDesPorts(t *testing.T) {
	t.Parallel()
	t.Run("campagne absente", func(t *testing.T) {
		t.Parallel()
		f := newCompareFixture(t)
		if _, err := f.comparator.Compare(context.Background(), "C-inconnue"); err == nil {
			t.Fatal("Compare aurait dû échouer")
		}
	})
	t.Run("matrice absente", func(t *testing.T) {
		t.Parallel()
		f := newCompareFixture(t)
		delete(f.store.matrices, f.matrix.ID)
		if _, err := f.comparator.Compare(context.Background(), f.campaign.ID); err == nil {
			t.Fatal("Compare aurait dû échouer")
		}
	})
	t.Run("écriture impossible", func(t *testing.T) {
		t.Parallel()
		f := newCompareFixture(t)
		f.measureAll(t, map[int]float64{8: 10, 16: 10, 24: 10, 32: 10}, map[int]float64{8: 10, 16: 10, 24: 10, 32: 10})
		f.store.writeErr = errors.New("boom")
		if _, err := f.comparator.Compare(context.Background(), f.campaign.ID); err == nil {
			t.Fatal("Compare aurait dû échouer")
		}
	})
}

func TestBootstrapDeterministe(t *testing.T) {
	t.Parallel()
	// Deux exécutions sur la même campagne produisent le même intervalle : la graine est dérivée
	// des identifiants de la paire, jamais du temps ni d'une source d'aléa.
	// Mutation : semer le générateur avec l'heure ⇒ échec attendu.
	value := []float64{10, 11, 9, 10, 12}
	pointer := []float64{8, 7, 9, 8, 8}
	firstLow, firstHigh := bootstrapDeltaCI(value, pointer, seedFor("v", "p"))
	secondLow, secondHigh := bootstrapDeltaCI(value, pointer, seedFor("v", "p"))
	if firstLow != secondLow || firstHigh != secondHigh {
		t.Fatalf("intervalle instable : [%v, %v] vs [%v, %v]", firstLow, firstHigh, secondLow, secondHigh)
	}
	if firstLow > firstHigh {
		t.Fatalf("intervalle inversé : [%v, %v]", firstLow, firstHigh)
	}
	// Un écart net doit produire un intervalle qui exclut zéro.
	if firstHigh >= 0 {
		t.Fatalf("le pointeur est nettement plus rapide : intervalle [%v, %v]", firstLow, firstHigh)
	}
	if low, high := bootstrapDeltaCI(nil, pointer, 1); low != 0 || high != 0 {
		t.Fatalf("un échantillon vide donne un intervalle nul, obtenu [%v, %v]", low, high)
	}
	if seedFor("a", "b") == seedFor("b", "a") {
		t.Fatal("la graine doit dépendre de l'ordre des identifiants")
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()
	sorted := []float64{1, 2, 3, 4, 5}
	if got := percentile(sorted, 0); got != 1 {
		t.Fatalf("percentile(0) = %v", got)
	}
	if got := percentile(sorted, 1); got != 5 {
		t.Fatalf("percentile(1) = %v", got)
	}
	if got := percentile(sorted, 2); got != 5 {
		t.Fatalf("un quantile hors bornes est borné : %v", got)
	}
	if got := percentile(nil, 0.5); got != 0 {
		t.Fatalf("percentile(nil) = %v", got)
	}
}

func TestTippingPoints(t *testing.T) {
	t.Parallel()
	local := models.ProfileLocal
	cases := []struct {
		name        string
		comparisons []models.Comparison
		want        int
	}{
		{
			name: "bascule au milieu",
			comparisons: []models.Comparison{
				{SizeBytes: 8, Profile: local, DeltaNsPerOp: 2, Significant: true},
				{SizeBytes: 16, Profile: local, DeltaNsPerOp: -2, Significant: true},
				{SizeBytes: 32, Profile: local, DeltaNsPerOp: -3, Significant: true},
			},
			want: 16,
		},
		{
			name: "bascule non tenue à la taille supérieure",
			comparisons: []models.Comparison{
				{SizeBytes: 8, Profile: local, DeltaNsPerOp: -2, Significant: true},
				{SizeBytes: 16, Profile: local, DeltaNsPerOp: 1, Significant: true},
				{SizeBytes: 32, Profile: local, DeltaNsPerOp: -3, Significant: true},
			},
			want: 32,
		},
		{
			name: "écart non significatif",
			comparisons: []models.Comparison{
				{SizeBytes: 8, Profile: local, DeltaNsPerOp: -2},
				{SizeBytes: 16, Profile: local, DeltaNsPerOp: -2},
			},
			want: models.TippingNotObserved,
		},
		{
			name: "toutes les tailles favorables",
			comparisons: []models.Comparison{
				{SizeBytes: 8, Profile: local, DeltaNsPerOp: -2, Significant: true},
				{SizeBytes: 16, Profile: local, DeltaNsPerOp: -2, Significant: true},
			},
			want: 8,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := TippingPoints(tc.comparisons)[models.TippingKey{Profile: local, Layout: models.LayoutArrayFill}]
			if got != tc.want {
				t.Fatalf("point de bascule = %d, attendu %d", got, tc.want)
			}
		})
	}
	// Les deux séries d'un même profil sont indépendantes.
	mixed := TippingPoints([]models.Comparison{
		{SizeBytes: 8, Profile: local, DeltaNsPerOp: -1, Significant: true},
		{SizeBytes: 8, Profile: local, HasPointerField: true, DeltaNsPerOp: 1, Significant: true},
	})
	if mixed[models.TippingKey{Profile: local, Layout: models.LayoutArrayFill}] != 8 {
		t.Fatalf("série sans champ pointeur = %d", mixed[models.TippingKey{Profile: local, Layout: models.LayoutArrayFill}])
	}
	if mixed[models.TippingKey{Profile: local, Layout: models.LayoutArrayFill, HasPointerField: true}] != models.TippingNotObserved {
		t.Fatalf("série avec champ pointeur = %d", mixed[models.TippingKey{Profile: local, Layout: models.LayoutArrayFill, HasPointerField: true}])
	}
}
