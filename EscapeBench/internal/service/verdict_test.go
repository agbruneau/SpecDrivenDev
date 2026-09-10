package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

type verdictFixture struct {
	service    *VerdictService
	store      *memoryStore
	hypotheses *fakeHypotheses
	useCases   *fakeUseCases
	code       *fakeCodeIndex
	tests      *fakeTests
	dashboard  *fakeDashboard
	clock      *fakeClock
	matrix     models.Matrix
	campaign   models.Campaign
}

func newVerdictFixture(t *testing.T) *verdictFixture {
	t.Helper()
	f := &verdictFixture{
		store:      newStore(),
		hypotheses: &fakeHypotheses{hypotheses: catalogue()},
		useCases: &fakeUseCases{useCases: []ports.UseCaseStatus{
			{ID: "UC-001", Title: "Générer la matrice", Status: "Implemented", LinkedFR: []string{"FR-001"}},
			{ID: "UC-003", Title: "Exécuter une campagne", Status: "Approved", LinkedFR: []string{"FR-003"}},
			{ID: "UC-005", Title: "Produire les verdicts", Status: "Reviewed", LinkedFR: []string{"FR-005"}},
		}},
		code: &fakeCodeIndex{
			code:        map[string]bool{"UC-001": true, "UC-003": true},
			integration: map[string]bool{"UC-001": true},
		},
		tests: &fakeTests{
			perUseCase: map[string]ports.TestOutcome{
				"UC-001": {Selected: 3, Passed: true},
				"UC-003": {Selected: 0, Passed: true},
			},
			all: ports.TestOutcome{Passed: true},
		},
		dashboard: &fakeDashboard{},
		clock:     newClock(),
	}
	params := models.MatrixParameters{
		Sizes:                []int{8, 16, 24, 32},
		PointerFieldVariants: []bool{false},
		Profiles:             []models.LifetimeProfile{models.ProfileLocal, models.ProfileReturned},
		PassingModes:         models.PassingModes(),
		Probes: []models.ProbeSpec{
			{Kind: models.ProbeSequentialScan, Parameter: L2Threshold},
			{Kind: models.ProbeScatteredScan, Parameter: L2Threshold},
			{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize},
			{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize},
		},
	}
	matrix, err := models.NewMatrix(params, "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.matrix = matrix
	f.store.matrices[matrix.ID] = matrix

	digest, err := digestOf(catalogue(), idsOf(catalogue()))
	if err != nil {
		t.Fatalf("digestOf : %v", err)
	}
	f.campaign = models.Campaign{
		ID: "C-1", MatrixID: matrix.ID, HarnessDigest: "harness-v1", HypothesesDigest: digest,
		HypothesisIDs: idsOf(catalogue()), Count: models.MinCount, Status: models.CampaignCompleted,
		Provenance: newProvenance().provenance, StartedAt: f.clock.Now(),
	}
	if err := f.store.CreateCampaign(context.Background(), f.campaign); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	f.service = NewVerdictService(f.store, f.store, f.store, f.store, f.hypotheses, f.useCases,
		f.code, f.tests, f.dashboard, digestOf, f.clock)
	return f
}

func idsOf(hypotheses []models.Hypothesis) []string {
	var ids []string
	for _, h := range hypotheses {
		ids = append(ids, h.ID)
	}
	return ids
}

// measure écrit une mesure complète pour un sujet de la campagne.
func (f *verdictFixture) measure(t *testing.T, subjectID string, ns float64, bytes, allocs int64) {
	t.Helper()
	m := completeMeasurementOf(subjectID, models.MinCount, ns, bytes, allocs)
	m.CampaignID = f.campaign.ID
	if err := f.store.WriteMeasurement(context.Background(), m); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
}

// withComparisons enregistre un fichier de comparaison pour la campagne.
func (f *verdictFixture) withComparisons(t *testing.T, comparisons []models.Comparison, tipping map[models.TippingKey]int) {
	t.Helper()
	set := models.ComparisonSet{CampaignID: f.campaign.ID, MatrixID: f.matrix.ID,
		ComputedAt: f.clock.Now(), Comparisons: comparisons, TippingPoints: tipping}
	if _, err := f.store.WriteComparisonSet(context.Background(), set); err != nil {
		t.Fatalf("WriteComparisonSet : %v", err)
	}
}

// withEscapes enregistre un fichier de verdicts d'échappement pour la toolchain de la campagne.
func (f *verdictFixture) withEscapes(t *testing.T, verdicts []models.EscapeVerdict) {
	t.Helper()
	report := models.EscapeReport{MatrixID: f.matrix.ID, Provenance: f.campaign.Provenance, Verdicts: verdicts}
	if _, err := f.store.WriteEscapeReport(context.Background(), report, f.clock.Now()); err != nil {
		t.Fatalf("WriteEscapeReport : %v", err)
	}
}

func verdictsByID(report models.VerdictReport) map[string]models.Verdict {
	out := map[string]models.Verdict{}
	for _, verdict := range report.Verdicts {
		out[verdict.HypothesisID] = verdict
	}
	return out
}

func TestUC005_MainFlow(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	f.withComparisons(t,
		[]models.Comparison{{SizeBytes: 8, Profile: models.ProfileLocal, DeltaNsPerOp: 2, CILow: 1, CIHigh: 3, Significant: true}},
		map[models.TippingKey]int{{Profile: models.ProfileLocal}: 32})
	f.withEscapes(t, []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryReturnPointer, Status: models.EscapeStatusOK},
	})
	for _, pair := range f.matrix.ValuePointerPairs() {
		allocs := int64(0)
		if pair[1].Profile == models.ProfileReturned {
			allocs = 1
		}
		f.measure(t, pair[0].ID(), 10, 8, 0)
		f.measure(t, pair[1].ID(), 10, 8, allocs)
	}
	f.measure(t, models.Probe{Kind: models.ProbeSequentialScan, Parameter: L2Threshold}.ID(), 100, 8, 0)
	f.measure(t, models.Probe{Kind: models.ProbeScatteredScan, Parameter: L2Threshold}.ID(), 5000, 8, 0)
	f.measure(t, models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID(), 100, 100, 1)
	f.measure(t, models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID(), 1000, 1000, 20)

	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	if len(summary.Report.Verdicts) != 6 {
		t.Fatalf("%d verdicts, 6 attendus", len(summary.Report.Verdicts))
	}
	byID := verdictsByID(summary.Report)
	expected := map[string]models.Outcome{
		"H-001": models.OutcomeConfirmed,
		"H-002": models.OutcomeConfirmed,
		"H-003": models.OutcomeConfirmed,
		"H-004": models.OutcomeConfirmed,
		"H-005": models.OutcomeConfirmed,
		"H-006": models.OutcomeConfirmed,
	}
	for id, want := range expected {
		if byID[id].Outcome != want {
			t.Fatalf("%s : verdict %s, %s attendu — %s", id, byID[id].Outcome, want, byID[id].Rationale)
		}
	}
	// BR-005-2 : chaque verdict cite ses fichiers de résultats.
	for _, verdict := range summary.Report.Verdicts {
		if len(verdict.ResultFiles) == 0 || verdict.Rationale == "" {
			t.Fatalf("verdict non traçable : %+v", verdict)
		}
	}
	// Étape 7 : le tableau de bord est régénéré.
	if summary.DashboardPath != "docs/dashboard.md" {
		t.Fatalf("tableau de bord = %q", summary.DashboardPath)
	}
	if len(f.dashboard.data.UseCases) != 3 || len(f.dashboard.data.Hypotheses) != 6 {
		t.Fatalf("données du tableau de bord = %+v", f.dashboard.data)
	}
	if len(summary.Inconclusive) != 0 {
		t.Fatalf("hypothèses non concluantes : %+v", summary.Inconclusive)
	}
}

func TestUC005_A1_CritereModifie(t *testing.T) {
	t.Parallel()
	// A1 : le système refuse, nomme les hypothèses concernées et n'écrit aucun fichier.
	// Mutation : ne pas comparer l'empreinte des critères ⇒ échec attendu.
	f := newVerdictFixture(t)
	modified := catalogue()
	modified[0].RefutationCriterion = "critère H-001 réécrit après la campagne"
	f.hypotheses.hypotheses = modified

	_, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if !errors.Is(err, ErrCriteriaChanged) {
		t.Fatalf("erreur = %v, ErrCriteriaChanged attendue", err)
	}
	if !strings.Contains(err.Error(), "nouvelle H-###") {
		t.Fatalf("le message doit rappeler la règle : %v", err)
	}
	if len(f.store.verdicts) != 0 {
		t.Fatal("A1 : aucun fichier n'est écrit")
	}
}

func TestUC005_A2_DonneesInsuffisantes(t *testing.T) {
	t.Parallel()
	// A2 : chaque hypothèse inévaluable reçoit INCONCLUSIVE avec sa raison ; les autres sont
	// évaluées quand même.
	f := newVerdictFixture(t)
	f.withEscapes(t, []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryOther, CompilerReason: "raison inconnue", Status: models.EscapeStatusOK},
	})
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	byID := verdictsByID(summary.Report)
	for _, id := range []string{"H-001", "H-002", "H-003", "H-004", "H-005"} {
		if byID[id].Outcome != models.OutcomeInconclusive {
			t.Fatalf("%s : verdict %s, INCONCLUSIVE attendu", id, byID[id].Outcome)
		}
		if byID[id].Rationale == "" {
			t.Fatalf("%s : la raison doit être donnée", id)
		}
	}
	// H-006 reste évaluable : le scénario principal se poursuit pour les autres hypothèses.
	if byID["H-006"].Outcome != models.OutcomeRefuted {
		t.Fatalf("H-006 : verdict %s, REFUTED attendu — %s", byID["H-006"].Outcome, byID["H-006"].Rationale)
	}
	if len(summary.Inconclusive) != 5 {
		t.Fatalf("%d hypothèses non concluantes, 5 attendues", len(summary.Inconclusive))
	}
}

func TestUC005_H001_Infirmee(t *testing.T) {
	t.Parallel()
	// H-001 est infirmée dès que deux tailles ≤ 24 octets ont significant vrai et ciHigh < 0.
	// Mutation : abaisser le seuil à une seule taille ⇒ échec attendu.
	f := newVerdictFixture(t)
	f.withComparisons(t, []models.Comparison{
		{SizeBytes: 8, Profile: models.ProfileLocal, DeltaNsPerOp: -2, CILow: -3, CIHigh: -1, Significant: true},
		{SizeBytes: 16, Profile: models.ProfileLocal, DeltaNsPerOp: -2, CILow: -3, CIHigh: -1, Significant: true},
		{SizeBytes: 32, Profile: models.ProfileLocal, DeltaNsPerOp: -5, CILow: -6, CIHigh: -4, Significant: true},
	}, map[models.TippingKey]int{{Profile: models.ProfileLocal}: 32})
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	verdict := verdictsByID(summary.Report)["H-001"]
	if verdict.Outcome != models.OutcomeRefuted {
		t.Fatalf("verdict = %s — %s", verdict.Outcome, verdict.Rationale)
	}
	if !strings.Contains(verdict.Rationale, "2 paires") {
		t.Fatalf("le rationale doit citer le décompte : %q", verdict.Rationale)
	}
}

func TestUC005_H001_UneSeuleTailleNeSuffitPas(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	f.withComparisons(t, []models.Comparison{
		{SizeBytes: 8, Profile: models.ProfileLocal, DeltaNsPerOp: -2, CILow: -3, CIHigh: -1, Significant: true},
		{SizeBytes: 16, Profile: models.ProfileLocal, DeltaNsPerOp: 1, CILow: 0.5, CIHigh: 2, Significant: true},
	}, map[models.TippingKey]int{{Profile: models.ProfileLocal}: models.TippingNotObserved})
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	if got := verdictsByID(summary.Report)["H-001"].Outcome; got != models.OutcomeConfirmed {
		t.Fatalf("verdict = %s, CONFIRMED attendu", got)
	}
}

func TestUC005_H001_HorsPerimetre(t *testing.T) {
	t.Parallel()
	// Seules les paires LOCAL sans champ pointeur de taille ≤ 24 octets comptent.
	f := newVerdictFixture(t)
	f.withComparisons(t, []models.Comparison{
		{SizeBytes: 8, Profile: models.ProfileReturned, DeltaNsPerOp: -2, CIHigh: -1, Significant: true},
		{SizeBytes: 8, Profile: models.ProfileLocal, HasPointerField: true, DeltaNsPerOp: -2, CIHigh: -1, Significant: true},
		{SizeBytes: 32, Profile: models.ProfileLocal, DeltaNsPerOp: -2, CIHigh: -1, Significant: true},
	}, map[models.TippingKey]int{{Profile: models.ProfileLocal}: 32})
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	verdict := verdictsByID(summary.Report)["H-001"]
	if verdict.Outcome != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s — %s", verdict.Outcome, verdict.Rationale)
	}
}

func TestUC005_H002(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		tipping map[models.TippingKey]int
		want    models.Outcome
	}{
		{"bascule au-delà de 24 octets", map[models.TippingKey]int{{Profile: models.ProfileLocal}: 32}, models.OutcomeConfirmed},
		{"bascule à 24 octets", map[models.TippingKey]int{{Profile: models.ProfileLocal}: 24}, models.OutcomeRefuted},
		{"bascule sous 24 octets", map[models.TippingKey]int{{Profile: models.ProfileLocal}: 8}, models.OutcomeRefuted},
		{"non observé", map[models.TippingKey]int{{Profile: models.ProfileLocal}: models.TippingNotObserved}, models.OutcomeConfirmed},
		{"série avec champ pointeur seule", map[models.TippingKey]int{{Profile: models.ProfileLocal, HasPointerField: true}: 16}, models.OutcomeRefuted},
		{"aucune série LOCAL", map[models.TippingKey]int{{Profile: models.ProfileReturned}: 8}, models.OutcomeInconclusive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			f.withComparisons(t, []models.Comparison{{SizeBytes: 8, Profile: models.ProfileLocal}}, tc.tipping)
			summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
			if err != nil {
				t.Fatalf("Produce : %v", err)
			}
			verdict := verdictsByID(summary.Report)["H-002"]
			if verdict.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", verdict.Outcome, tc.want, verdict.Rationale)
			}
		})
	}
}

func TestUC005_H003(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		valueAllocs   int64
		pointerAllocs int64
		want          models.Outcome
	}{
		{"zéro vers une allocation", 0, 1, models.OutcomeConfirmed},
		{"facteur deux", 2, 4, models.OutcomeConfirmed},
		{"facteur insuffisant", 2, 3, models.OutcomeRefuted},
		{"aucune allocation supplémentaire", 0, 0, models.OutcomeRefuted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			for _, pair := range f.matrix.ValuePointerPairs() {
				if pair[0].Profile != models.ProfileReturned {
					continue
				}
				f.measure(t, pair[0].ID(), 10, 8, tc.valueAllocs)
				f.measure(t, pair[1].ID(), 10, 8, tc.pointerAllocs)
			}
			summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
			if err != nil {
				t.Fatalf("Produce : %v", err)
			}
			verdict := verdictsByID(summary.Report)["H-003"]
			if verdict.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", verdict.Outcome, tc.want, verdict.Rationale)
			}
		})
	}
}

func TestUC005_H004(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		sequentialNs float64
		scatteredNs  float64
		want         models.Outcome
	}{
		{"rapport dans l'intervalle", 100, 5000, models.OutcomeConfirmed},
		{"borne basse atteinte", 100, 1000, models.OutcomeConfirmed},
		{"borne haute atteinte", 100, 20000, models.OutcomeConfirmed},
		{"rapport trop faible", 100, 500, models.OutcomeRefuted},
		{"rapport trop élevé", 100, 30000, models.OutcomeRefuted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			f.measure(t, models.Probe{Kind: models.ProbeSequentialScan, Parameter: L2Threshold}.ID(), tc.sequentialNs, 8, 0)
			f.measure(t, models.Probe{Kind: models.ProbeScatteredScan, Parameter: L2Threshold}.ID(), tc.scatteredNs, 8, 0)
			summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
			if err != nil {
				t.Fatalf("Produce : %v", err)
			}
			verdict := verdictsByID(summary.Report)["H-004"]
			if verdict.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", verdict.Outcome, tc.want, verdict.Rationale)
			}
		})
	}
}

func TestUC005_H004_IgnoreLesPetitsJeuxDeTravail(t *testing.T) {
	t.Parallel()
	// Le critère ne porte que sur les jeux de travail ≥ 32 MiB : une paire plus petite ne suffit
	// pas à l'évaluer. Mutation : retirer le filtre de taille ⇒ échec attendu.
	f := newVerdictFixture(t)
	f.measure(t, models.Probe{Kind: models.ProbeSequentialScan, Parameter: 262144}.ID(), 100, 8, 0)
	f.measure(t, models.Probe{Kind: models.ProbeScatteredScan, Parameter: 262144}.ID(), 100, 8, 0)
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	if got := verdictsByID(summary.Report)["H-004"].Outcome; got != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu", got)
	}
}

func TestUC005_H005(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                  string
		preallocNs, preallocB float64
		growNs, growB         float64
		want                  models.Outcome
	}{
		{"les deux rapports sous la moitié", 100, 100, 1000, 1000, models.OutcomeConfirmed},
		{"temps trop proche", 600, 100, 1000, 1000, models.OutcomeRefuted},
		{"mémoire trop proche", 100, 600, 1000, 1000, models.OutcomeRefuted},
		{"exactement la moitié", 500, 500, 1000, 1000, models.OutcomeConfirmed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			f.measure(t, models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID(), tc.preallocNs, int64(tc.preallocB), 1)
			f.measure(t, models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID(), tc.growNs, int64(tc.growB), 20)
			summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
			if err != nil {
				t.Fatalf("Produce : %v", err)
			}
			verdict := verdictsByID(summary.Report)["H-005"]
			if verdict.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", verdict.Outcome, tc.want, verdict.Rationale)
			}
		})
	}
}

func TestUC005_H005_MedianesNulles(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	f.measure(t, models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID(), 100, 100, 1)
	f.measure(t, models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID(), 0, 0, 0)
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	if got := verdictsByID(summary.Report)["H-005"].Outcome; got != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu : le rapport est indéfini", got)
	}
}

func TestUC005_H006(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		verdicts []models.EscapeVerdict
		want     models.Outcome
	}{
		{
			name: "toutes les causes sont dans le livre",
			verdicts: []models.EscapeVerdict{
				{CellID: "a", Escapes: true, Category: models.CategoryReturnPointer, Status: models.EscapeStatusOK},
				{CellID: "b", Escapes: true, Category: models.CategoryChannelSend, Status: models.EscapeStatusOK},
				{CellID: "c", Category: models.CategoryNone, Status: models.EscapeStatusOK},
			},
			want: models.OutcomeConfirmed,
		},
		{
			name: "une cause hors du livre",
			verdicts: []models.EscapeVerdict{
				{CellID: "a", Escapes: true, Category: models.CategoryReturnPointer, Status: models.EscapeStatusOK},
				{CellID: "b", Escapes: true, Category: models.CategoryOther, CompilerReason: "taille de pile", Status: models.EscapeStatusOK},
			},
			want: models.OutcomeRefuted,
		},
		{
			name: "aucune cellule n'échappe",
			verdicts: []models.EscapeVerdict{
				{CellID: "a", Category: models.CategoryNone, Status: models.EscapeStatusOK},
			},
			want: models.OutcomeInconclusive,
		},
		{
			name: "les erreurs de compilation ne comptent pas",
			verdicts: []models.EscapeVerdict{
				{CellID: "a", Status: models.EscapeStatusCompileError, CompilerError: "boom"},
			},
			want: models.OutcomeInconclusive,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			f.withEscapes(t, tc.verdicts)
			summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
			if err != nil {
				t.Fatalf("Produce : %v", err)
			}
			verdict := verdictsByID(summary.Report)["H-006"]
			if verdict.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", verdict.Outcome, tc.want, verdict.Rationale)
			}
		})
	}
}

func TestUC005_H006_FichierDesigneExplicitement(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	autre := f.campaign.Provenance
	autre.GOARCH = "arm64"
	report := models.EscapeReport{MatrixID: f.matrix.ID, Provenance: autre, Verdicts: []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryOther, CompilerReason: "x", Status: models.EscapeStatusOK},
	}}
	path, err := f.store.WriteEscapeReport(context.Background(), report, f.clock.Now())
	if err != nil {
		t.Fatalf("WriteEscapeReport : %v", err)
	}
	// Sans désignation, un fichier d'une autre toolchain est ignoré.
	summary, err := f.service.Produce(context.Background(), f.campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	if got := verdictsByID(summary.Report)["H-006"].Outcome; got != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu", got)
	}
	// Désigné explicitement, il est utilisé.
	summary, err = f.service.Produce(context.Background(), f.campaign.ID, path)
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	verdict := verdictsByID(summary.Report)["H-006"]
	if verdict.Outcome != models.OutcomeRefuted {
		t.Fatalf("verdict = %s, REFUTED attendu", verdict.Outcome)
	}
	if verdict.ResultFiles[0] != path {
		t.Fatalf("le fichier cité doit être celui désigné : %v", verdict.ResultFiles)
	}
}

func TestUC005_HypotheseSansEvaluateur(t *testing.T) {
	t.Parallel()
	// Un critère sans évaluation mécanique est signalé INCONCLUSIVE, jamais deviné.
	f := newVerdictFixture(t)
	extended := append(catalogue(), models.Hypothesis{ID: "H-099", SourcePages: "p. 9",
		Statement: "énoncé libre", RefutationCriterion: "critère non mécanisable", UseCases: []string{"UC-003"}})
	f.hypotheses.hypotheses = extended
	digest, err := digestOf(extended, idsOf(extended))
	if err != nil {
		t.Fatalf("digestOf : %v", err)
	}
	campaign := f.campaign
	campaign.HypothesisIDs = idsOf(extended)
	campaign.HypothesesDigest = digest
	f.store.campaigns[campaign.ID] = campaign

	summary, err := f.service.Produce(context.Background(), campaign.ID, "")
	if err != nil {
		t.Fatalf("Produce : %v", err)
	}
	verdict := verdictsByID(summary.Report)["H-099"]
	if verdict.Outcome != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s", verdict.Outcome)
	}
	if !strings.Contains(verdict.Rationale, "aucun évaluateur") {
		t.Fatalf("rationale = %q", verdict.Rationale)
	}
	if len(verdict.ResultFiles) == 0 {
		t.Fatal("BR-005-2 : même un verdict non concluant cite un fichier")
	}
}

func TestUC005_CampagneNonCompletee(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	if err := f.store.SetCampaignStatus(context.Background(), f.campaign.ID, models.CampaignAborted, f.clock.Now(), "x"); err != nil {
		t.Fatalf("SetCampaignStatus : %v", err)
	}
	if _, err := f.service.Produce(context.Background(), f.campaign.ID, ""); !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}

func TestUC005_ErreursDesPorts(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := map[string]func(*verdictFixture){
		"campagne absente":             func(f *verdictFixture) { delete(f.store.campaigns, f.campaign.ID) },
		"hypothèses illisibles":        func(f *verdictFixture) { f.hypotheses.err = boom },
		"matrice absente":              func(f *verdictFixture) { delete(f.store.matrices, f.matrix.ID) },
		"écriture impossible":          func(f *verdictFixture) { f.store.writeErr = boom },
		"cas d'utilisation illisibles": func(f *verdictFixture) { f.useCases.err = boom },
		"index de code cassé":          func(f *verdictFixture) { f.code.err = boom },
		"tests injoignables":           func(f *verdictFixture) { f.tests.err = boom },
		"tableau de bord en panne":     func(f *verdictFixture) { f.dashboard.err = boom },
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newVerdictFixture(t)
			breakIt(f)
			if _, err := f.service.Produce(context.Background(), f.campaign.ID, ""); err == nil {
				t.Fatal("Produce aurait dû échouer")
			}
		})
	}
}

func TestUC005_FichierDEchappementIntrouvable(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	if _, err := f.service.Produce(context.Background(), f.campaign.ID, "results/escape/absent.json"); err == nil {
		t.Fatal("Produce aurait dû échouer")
	}
}

func TestUC005_DashboardSeul(t *testing.T) {
	t.Parallel()
	// Étape 7 : la régénération seule est disponible sans nouveau verdict.
	f := newVerdictFixture(t)
	path, err := f.service.Dashboard(context.Background())
	if err != nil {
		t.Fatalf("Dashboard : %v", err)
	}
	if path != "docs/dashboard.md" {
		t.Fatalf("chemin = %q", path)
	}
	if len(f.store.verdicts) != 0 {
		t.Fatal("aucun fichier de verdicts ne doit être écrit")
	}
	rows := map[string]ports.DashboardUseCase{}
	for _, row := range f.dashboard.data.UseCases {
		rows[row.ID] = row
	}
	// UC-001 : code et tests présents et verts, suite complète verte ⇒ Strong.
	if rows["UC-001"].Integrity != "Strong" || !rows["UC-001"].Unit || rows["UC-001"].Integration != "✔" {
		t.Fatalf("UC-001 = %+v", rows["UC-001"])
	}
	// UC-003 : du code, aucun test sélectionné ⇒ Partial.
	if rows["UC-003"].Integrity != "Partial" || rows["UC-003"].Unit {
		t.Fatalf("UC-003 = %+v", rows["UC-003"])
	}
	// UC-005 : aucun code ⇒ Weak.
	if rows["UC-005"].Integrity != "Weak" || rows["UC-005"].Code {
		t.Fatalf("UC-005 = %+v", rows["UC-005"])
	}
	// Le gel des critères suit le statut des cas d'utilisation porteurs.
	for _, row := range f.dashboard.data.Hypotheses {
		if !row.Frozen {
			t.Fatalf("UC-003 est Approved : les hypothèses qu'il porte sont gelées — %+v", row)
		}
		if row.Outcome != "" {
			t.Fatalf("aucun verdict n'existe encore : %+v", row)
		}
	}
}

func TestUC005_DashboardRefleteLeDernierVerdict(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	f.withEscapes(t, []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryOther, CompilerReason: "x", Status: models.EscapeStatusOK},
	})
	if _, err := f.service.Produce(context.Background(), f.campaign.ID, ""); err != nil {
		t.Fatalf("Produce : %v", err)
	}
	byID := map[string]ports.DashboardHypothesis{}
	for _, row := range f.dashboard.data.Hypotheses {
		byID[row.ID] = row
	}
	if byID["H-006"].Outcome != string(models.OutcomeRefuted) || byID["H-006"].CampaignID != f.campaign.ID {
		t.Fatalf("H-006 = %+v", byID["H-006"])
	}
}

func TestUC005_HypothesesNonGelees(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	for i := range f.useCases.useCases {
		f.useCases.useCases[i].Status = "Reviewed"
	}
	if _, err := f.service.Dashboard(context.Background()); err != nil {
		t.Fatalf("Dashboard : %v", err)
	}
	for _, row := range f.dashboard.data.Hypotheses {
		if row.Frozen {
			t.Fatalf("aucun UC n'est Approved : rien n'est gelé — %+v", row)
		}
	}
}
