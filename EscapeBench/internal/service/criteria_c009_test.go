package service

import (
	"context"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// replicat construit l'un des cinq réplicats d'une cellule de H-012. Les réplicats d'une même
// cellule partagent taille, série, disposition et profil, et se distinguent par valueCellId.
func replicat(size int, hasPointerField bool, index int, delta, ciHigh, medianValue float64, significant bool) models.Comparison {
	return models.Comparison{
		CampaignID:  "C-9",
		ValueCellID: cellName(size, hasPointerField, index, "VALUE"),
		// Deux réplicats se distinguent par leur identifiant, jamais par un champ de Comparison :
		// C-009 retire délibérément le réplicat de Comparison et de TippingKey.
		PointerCellID: cellName(size, hasPointerField, index, "POINTER"),
		SizeBytes:     size, HasPointerField: hasPointerField,
		Layout: models.LayoutNamedFields, Profile: models.ProfileLocal,
		DeltaNsPerOp: delta, CILow: ciHigh - 0.05, CIHigh: ciHigh,
		Significant: significant, MedianValueNs: medianValue, MedianPointerNs: medianValue + delta,
	}
}

func cellName(size int, hasPointerField bool, index int, mode string) string {
	suffix := "Plain"
	if hasPointerField {
		suffix = "Ptr"
	}
	name := "Size" + strings.Repeat("0", 4-len(itoa(size))) + itoa(size) + "Fields" + suffix
	segment := "LOCAL"
	if index > 1 {
		segment += "_X" + itoa(index)
	}
	return name + "/" + segment + "/" + mode
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

// h012Evidence assemble une campagne complète pour H-012 : les six couples taille-série en cinq
// réplicats, plus un témoin de sensibilité par série.
func h012Evidence(t *testing.T, mutate func(set *models.ComparisonSet)) Evidence {
	t.Helper()
	set := models.ComparisonSet{CampaignID: "C-9"}
	for _, hasPointerField := range []bool{false, true} {
		for _, size := range []int{8, 16, 24} {
			for index := 1; index <= models.ReplicateCount; index++ {
				// Par défaut la valeur tient : écart positif, non significatif, plancher étroit.
				set.Comparisons = append(set.Comparisons,
					replicat(size, hasPointerField, index, 0.02+float64(index)*0.001, 0.05, 1.0, false))
			}
		}
		// Témoin de sensibilité : au-delà des registres d'argument, le pointeur l'emporte nettement.
		for index := 1; index <= models.ReplicateCount; index++ {
			set.Comparisons = append(set.Comparisons,
				replicat(128, hasPointerField, index, -3.0, -2.5, 1.5, true))
		}
	}
	if mutate != nil {
		mutate(&set)
	}
	return Evidence{
		Campaign:       models.Campaign{ID: "C-9"},
		ComparisonSet:  &set,
		ComparisonPath: "results/campaigns/C-9/comparison.json",
	}
}

func TestUC005_H012_MainFlow(t *testing.T) {
	t.Parallel()
	got := evaluateH012(h012Evidence(t, nil))
	if got.Outcome != models.OutcomeConfirmed {
		t.Fatalf("verdict = %s, CONFIRMED attendu — %s", got.Outcome, got.Rationale)
	}
	// Le critère exige que le verdict consigne le plancher, la barrière, le seuil et les ciHigh de
	// chaque cellule jugée, plus la cellule écartée.
	for _, needle := range []string{"plancher", "barrière", "seuil", "ciHigh", "écartée du jugement"} {
		if !strings.Contains(got.Rationale, needle) {
			t.Fatalf("%q absent de la rationale : %s", needle, got.Rationale)
		}
	}
}

// Une seule cellule jugée suffit à infirmer.
func TestUC005_H012_UneCelluleSuffitAInfirmer(t *testing.T) {
	t.Parallel()
	e := h012Evidence(t, func(set *models.ComparisonSet) {
		for i, c := range set.Comparisons {
			if c.SizeBytes == 16 && !c.HasPointerField {
				set.Comparisons[i] = replicat(16, false, replicateIndexOf(c), -0.4, -0.3, 1.0, true)
			}
		}
	})
	got := evaluateH012(e)
	if got.Outcome != models.OutcomeRefuted {
		t.Fatalf("verdict = %s, REFUTED attendu — %s", got.Outcome, got.Rationale)
	}
	if !strings.Contains(got.Rationale, "16 octets sans champ pointeur") {
		t.Fatalf("la cellule fautive doit être nommée : %s", got.Rationale)
	}
}

// La cellule de 8 octets avec champ pointeur est mesurée et consignée mais n'entre dans aucune
// clause : son type n'a qu'un champ, qui est le pointeur lui-même.
func TestUC005_H012_CelluleEcarteeNInfirmePas(t *testing.T) {
	t.Parallel()
	e := h012Evidence(t, func(set *models.ComparisonSet) {
		for i, c := range set.Comparisons {
			if c.SizeBytes == 8 && c.HasPointerField {
				set.Comparisons[i] = replicat(8, true, replicateIndexOf(c), -0.5, -0.4, 1.0, true)
			}
		}
	})
	got := evaluateH012(e)
	if got.Outcome != models.OutcomeConfirmed {
		t.Fatalf("verdict = %s, CONFIRMED attendu : la cellule écartée ne juge pas — %s", got.Outcome, got.Rationale)
	}
	if !strings.Contains(got.Rationale, "écartée du jugement") {
		t.Fatalf("la cellule écartée doit rester consignée : %s", got.Rationale)
	}
}

// Clause (c) : à défaut d'infirmation, une cellule dont le plancher dépasse la barrière relative
// rend le verdict non concluant. Le bruit y a pu avaler l'écart à trancher.
func TestUC005_H012_CelluleNonResolue(t *testing.T) {
	t.Parallel()
	e := h012Evidence(t, func(set *models.ComparisonSet) {
		// Trois réplicats de rang médian très dispersés : le plancher dépasse 10 % de la médiane.
		deltas := []float64{-0.5, -0.2, 0.0, 0.2, 0.5}
		n := 0
		for i, c := range set.Comparisons {
			if c.SizeBytes == 24 && !c.HasPointerField {
				set.Comparisons[i] = replicat(24, false, replicateIndexOf(c), deltas[n], 0.05, 1.0, false)
				n++
			}
		}
	})
	got := evaluateH012(e)
	if got.Outcome != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu — %s", got.Outcome, got.Rationale)
	}
	if !strings.Contains(got.Rationale, "24 octets sans champ pointeur") {
		t.Fatalf("la cellule non résolue doit être nommée : %s", got.Rationale)
	}
}

// Clauses (a), (b) et (d), et l'ordre dans lequel elles s'évaluent.
func TestUC005_H012_ClausesDeNonConclusion(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		mutate func(set *models.ComparisonSet)
		id     string
		needle string
	}{
		"(a) réplicats incomplets": {
			mutate: func(set *models.ComparisonSet) {
				out := set.Comparisons[:0]
				vus := 0
				for _, c := range set.Comparisons {
					if c.SizeBytes == 16 && !c.HasPointerField {
						vus++
						if vus > 3 {
							continue
						}
					}
					out = append(out, c)
				}
				set.Comparisons = out
			},
			needle: "réplicat",
		},
		"(a) témoin de sensibilité absent": {
			mutate: func(set *models.ComparisonSet) {
				out := set.Comparisons[:0]
				for _, c := range set.Comparisons {
					if c.SizeBytes >= RegisterArgumentBytes {
						continue
					}
					out = append(out, c)
				}
				set.Comparisons = out
			},
			needle: "≥ 80 octets",
		},
		"(b) témoin de sensibilité muet": {
			mutate: func(set *models.ComparisonSet) {
				for i, c := range set.Comparisons {
					if c.SizeBytes >= RegisterArgumentBytes {
						set.Comparisons[i] = replicat(c.SizeBytes, c.HasPointerField, replicateIndexOf(c), 0.01, 0.05, 1.5, false)
					}
				}
			},
			needle: "témoin de sensibilité manque",
		},
		"(d) campagne antérieure au critère": {
			id:     "C-2026-09-10-2",
			needle: "précèdent la rédaction",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := h012Evidence(t, tc.mutate)
			if tc.id != "" {
				e.Campaign.ID = tc.id
			}
			got := evaluateH012(e)
			if got.Outcome != models.OutcomeInconclusive {
				t.Fatalf("verdict = %s, INCONCLUSIVE attendu — %s", got.Outcome, got.Rationale)
			}
			if !strings.Contains(got.Rationale, tc.needle) {
				t.Fatalf("%q absent de : %s", tc.needle, got.Rationale)
			}
		})
	}
}

// Le plancher écarte le plus petit et le plus grand écart : un réplicat dissident ne le fixe pas.
func TestUC005_H012_PlancherEcarteLesExtremes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		values []float64
		want   float64
	}{
		"cinq valeurs, un dissident": {[]float64{0.01, 0.02, 0.03, 0.04, 5.0}, 0.02},
		"cinq valeurs groupées":      {[]float64{0.10, 0.11, 0.12, 0.13, 0.14}, 0.02},
		"une seule valeur":           {[]float64{0.5}, 0},
		"aucune valeur":              {nil, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := trimmedRange(tc.values)
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("trimmedRange(%v) = %.4f, %.4f attendu", tc.values, got, tc.want)
			}
		})
	}
}

// La barrière relative retient le plus grand des deux bras : c'est le coût de l'opération et non
// celui d'un seul bras.
func TestUC005_H012_BarriereRetientLePlusGrandBras(t *testing.T) {
	t.Parallel()
	e := h012Evidence(t, func(set *models.ComparisonSet) {
		for i, c := range set.Comparisons {
			if c.SizeBytes == 24 && !c.HasPointerField {
				// Bras valeur assignable aux registres, bras pointeur deux fois plus cher.
				set.Comparisons[i] = replicat(24, false, replicateIndexOf(c), 0.44, 0.05, 0.44, false)
			}
		}
	})
	cells := h012CellsOf(e)
	cell := cells[h012Key{24, false}]
	// medianPointer = 0,44 + 0,44 = 0,88 ; la barrière vaut donc 0,088 et non 0,044.
	if diff := cell.relativeBar - 0.088; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("barrière = %.4f, 0,0880 attendue : le plus grand bras doit être retenu", cell.relativeBar)
	}
}

// replicateIndexOf retrouve le rang d'un réplicat depuis son identifiant de cellule valeur.
func replicateIndexOf(c models.Comparison) int {
	if i := strings.Index(c.ValueCellID, "_X"); i >= 0 {
		rest := c.ValueCellID[i+2:]
		if j := strings.Index(rest, "/"); j >= 0 {
			rest = rest[:j]
		}
		n := 0
		for _, ch := range rest {
			n = n*10 + int(ch-'0')
		}
		return n
	}
	return 1
}

// UC-003, C-009 : H-007 indexe ses Comparison par taille et n'en retient qu'une, arbitrairement.
// Sur une matrice à réplicats elle jugerait un réplicat tiré de l'ordre du fichier. La campagne est
// refusée plutôt que de laisser sortir ce verdict.
func TestUC003_H007RefuseeSurMatriceRepliquee(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	params := smallParameters()
	params.Layouts = []models.Layout{models.LayoutNamedFields}
	params.Profiles = []models.LifetimeProfile{models.ProfileLocal}
	params.Replicates = models.ReplicateCount
	matrix, err := models.NewMatrix(params, "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.store.matrices[matrix.ID] = matrix
	f.store.escapes[matrix.ID] = []models.EscapeReport{{MatrixID: matrix.ID, Provenance: f.provenance.provenance}}
	f.store.escapePaths[matrix.ID] = []string{"results/escape/" + matrix.ID + "/0001.json"}

	opts := defaultOptions(matrix.ID)
	opts.HypothesisIDs = []string{"H-001", "H-007"}
	if _, err := f.service.Run(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "H-007") {
		t.Fatalf("erreur = %v, un refus nommant H-007 était attendu", err)
	}
	// Sans H-007, la même matrice est acceptée.
	opts.HypothesisIDs = []string{"H-001"}
	if _, err := f.service.Run(context.Background(), opts); err != nil {
		t.Fatalf("la campagne doit être acceptée sans H-007 : %v", err)
	}
}

// UC-004, C-009 : le point de bascule d'une série répliquée n'est pas défini, le balayage
// descendant dépendant de l'ordre du fichier. Il n'est donc pas publié.
func TestUC004_PointDeBasculeNonPublieSurSerieRepliquee(t *testing.T) {
	t.Parallel()
	simple := []models.Comparison{
		{SizeBytes: 8, Profile: models.ProfileLocal, Layout: models.LayoutNamedFields, DeltaNsPerOp: -1, CIHigh: -0.5, Significant: true},
		{SizeBytes: 16, Profile: models.ProfileLocal, Layout: models.LayoutNamedFields, DeltaNsPerOp: -1, CIHigh: -0.5, Significant: true},
	}
	key := models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutNamedFields}
	points, excluded := TippingPoints(simple)
	if got, ok := points[key]; !ok || got != 8 {
		t.Fatalf("série simple : point de bascule = %d, présent = %v", got, ok)
	}
	if len(excluded) != 0 {
		t.Fatalf("aucune série n'est exclue ici : %v", excluded)
	}
	// La même série, répliquée : deux Comparison portent la taille 8.
	repliquee := append(append([]models.Comparison(nil), simple...), simple[0])
	points, excluded = TippingPoints(repliquee)
	if _, ok := points[key]; ok {
		t.Fatal("une série répliquée ne doit pas publier de point de bascule")
	}
	// A-032 : l'exclusion est consignée avec sa raison, jamais silencieuse.
	if len(excluded) != 1 || excluded[0].Key != key || excluded[0].Reason == "" {
		t.Fatalf("l'exclusion doit être consignée avec sa raison : %+v", excluded)
	}
	// Une série non répliquée du même fichier reste publiée.
	autre := models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill}
	repliquee = append(repliquee, models.Comparison{SizeBytes: 8, Profile: models.ProfileLocal,
		Layout: models.LayoutArrayFill, DeltaNsPerOp: -1, CIHigh: -0.5, Significant: true})
	points, _ = TippingPoints(repliquee)
	if _, ok := points[autre]; !ok {
		t.Fatal("les séries non répliquées du même fichier doivent rester publiées")
	}
}

// UC-005, C-009 : le tableau de bord agrège toutes les campagnes. Une campagne ne couvre que les
// hypothèses qu'elle a gelées, et H-007 comme H-012 ne peuvent pas cohabiter dans la même. Ne lire
// que le dernier rapport effacerait donc du tableau les verdicts des campagnes précédentes.
func TestUC005_LeTableauDeBordAgregeLesCampagnes(t *testing.T) {
	t.Parallel()
	f := newVerdictFixture(t)
	f.store.verdicts = []models.VerdictReport{
		{CampaignID: "C-1", Verdicts: []models.Verdict{
			{HypothesisID: "H-001", CampaignID: "C-1", Outcome: models.OutcomeRefuted, Rationale: "r"},
			{HypothesisID: "H-003", CampaignID: "C-1", Outcome: models.OutcomeConfirmed, Rationale: "r"},
		}},
		{CampaignID: "C-2", Verdicts: []models.Verdict{
			{HypothesisID: "H-006", CampaignID: "C-2", Outcome: models.OutcomeConfirmed, Rationale: "r"},
		}},
	}
	if _, err := f.service.Dashboard(context.Background()); err != nil {
		t.Fatalf("Dashboard : %v", err)
	}
	rows := map[string]string{}
	for _, h := range f.dashboard.data.Hypotheses {
		rows[h.ID] = h.CampaignID + "/" + h.Outcome
	}
	// Le verdict de la première campagne survit à la seconde, qui ne le couvre pas.
	if rows["H-001"] != "C-1/REFUTED" {
		t.Fatalf("H-001 = %q, la campagne précédente doit survivre", rows["H-001"])
	}
	if rows["H-003"] != "C-1/CONFIRMED" {
		t.Fatalf("H-003 = %q", rows["H-003"])
	}
	// La seconde campagne ne couvre qu'une hypothèse ; c'est elle qui l'emporte pour celle-là.
	if rows["H-006"] != "C-2/CONFIRMED" {
		t.Fatalf("H-006 = %q", rows["H-006"])
	}
	// Une hypothèse qu'aucune campagne n'a couverte reste sans verdict.
	if rows["H-002"] != "/" {
		t.Fatalf("H-002 = %q, aucun verdict attendu", rows["H-002"])
	}
}
