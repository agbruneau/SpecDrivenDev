package service

import (
	"context"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// Tests des successeurs H-014, H-015 et H-016 (D-63) et de C-011. Les campagnes portent par défaut
// l'empreinte de la série arm64 : la clause de préenregistrement ne s'applique qu'à la série
// 551ce66b…, dont les campagnes ont informé les critères.

const arm64SeriesDigest = "fd4a470c1fea2dc1a366d5ffb26921cf5c14f8a40540c67a2fcf91a962d06100"

// replicatIn construit un réplicat de H-012 dans une disposition donnée.
func replicatIn(layout models.Layout, size int, hasPointerField bool, index int, delta, ciHigh, medianValue float64, significant bool) models.Comparison {
	c := replicat(size, hasPointerField, index, delta, ciHigh, medianValue, significant)
	c.Layout = layout
	if layout != models.LayoutNamedFields {
		c.ValueCellID = strings.Replace(c.ValueCellID, "Fields", "", 1)
		c.PointerCellID = strings.Replace(c.PointerCellID, "Fields", "", 1)
	}
	return c
}

// setCrossings réécrit les cinq réplicats d'une cellule pour que exactement k franchissent : tous
// ont le même écart, donc un plancher quasi nul et un seuil égal à la barrière de 0,1 ns ; seuls les
// k premiers sont significatifs, avec un ciHigh bien en deçà de l'opposé du seuil.
func setCrossings(set *models.ComparisonSet, layout models.Layout, size int, hasPointerField bool, k int) {
	for i, c := range set.Comparisons {
		if c.EffectiveLayout() != layout || c.SizeBytes != size || c.HasPointerField != hasPointerField {
			continue
		}
		index := replicateIndexOf(c)
		delta := -0.45 - float64(index)*0.001
		if index <= k {
			set.Comparisons[i] = replicatIn(layout, size, hasPointerField, index, delta, -0.35, 1.0, true)
		} else {
			set.Comparisons[i] = replicatIn(layout, size, hasPointerField, index, delta, 0.01, 1.0, false)
		}
	}
}

// arrayFillEvidence assemble une campagne ARRAY_FILL répliquée conforme à ce que la série
// 551ce66b… laisse attendre sur amd64 : la valeur tient à 8 et 16 octets, le pointeur l'emporte à
// 24 et 128 octets, dans les deux séries.
func arrayFillEvidence(t *testing.T, mutate func(set *models.ComparisonSet)) Evidence {
	t.Helper()
	set := models.ComparisonSet{CampaignID: "C-A"}
	for _, hasPointerField := range []bool{false, true} {
		for _, size := range []int{8, 16, 24, 128} {
			for index := 1; index <= models.ReplicateCount; index++ {
				set.Comparisons = append(set.Comparisons,
					replicatIn(models.LayoutArrayFill, size, hasPointerField, index, 0.02+float64(index)*0.001, 0.05, 1.0, false))
			}
		}
		setCrossings(&set, models.LayoutArrayFill, 24, hasPointerField, 5)
		setCrossings(&set, models.LayoutArrayFill, 128, hasPointerField, 5)
	}
	if mutate != nil {
		mutate(&set)
	}
	return Evidence{
		Campaign:       models.Campaign{ID: "C-A", HarnessDigest: arm64SeriesDigest},
		ComparisonSet:  &set,
		ComparisonPath: "results/campaigns/C-A/comparison.json",
	}
}

func TestUC005_H014(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(set *models.ComparisonSet)
		want   models.Outcome
		needle string
	}{
		{"une seule taille favorable au pointeur", nil, models.OutcomeConfirmed, "sur 1 des 3 tailles"},
		{"deux tailles à quatre réplicats sur cinq", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 16, false, 4)
		}, models.OutcomeRefuted, "16 octets sans champ pointeur"},
		// Règle de multiplicité : trois sur cinq ne tranche pas. Mutation : ramener
		// successorWinCrossings à 3 ⇒ REFUTED au lieu d'INCONCLUSIVE.
		{"une taille indéterminée fait basculer le décompte", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 16, false, 3)
		}, models.OutcomeInconclusive, "indéterminée"},
		{"réplicat manquant", func(s *models.ComparisonSet) {
			s.Comparisons = s.Comparisons[1:]
		}, models.OutcomeInconclusive, "réplicat"},
		{"témoin de sensibilité absent", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 128, false, 2)
		}, models.OutcomeInconclusive, "témoin de sensibilité"},
		{"cellule non résolue", func(s *models.ComparisonSet) {
			for i, c := range s.Comparisons {
				if c.EffectiveLayout() == models.LayoutArrayFill && c.SizeBytes == 8 && !c.HasPointerField {
					index := replicateIndexOf(c)
					s.Comparisons[i] = replicatIn(models.LayoutArrayFill, 8, false, index, 0.3*float64(index), 1.5, 1.0, false)
				}
			}
		}, models.OutcomeInconclusive, "ne résout pas"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := evaluateH014(arrayFillEvidence(t, tc.mutate))
			if got.Outcome != tc.want || !strings.Contains(got.Rationale, tc.needle) {
				t.Fatalf("verdict = %s, %s attendu, avec %q — %s", got.Outcome, tc.want, tc.needle, got.Rationale)
			}
		})
	}
	// NAMED_FIELDS n'est pas lue : une campagne sans ARRAY_FILL est non concluante.
	named := h012Evidence(t, nil)
	named.Campaign.HarnessDigest = arm64SeriesDigest
	if got := evaluateH014(named); got.Outcome != models.OutcomeInconclusive {
		t.Fatalf("sans ARRAY_FILL : %s — %s", got.Outcome, got.Rationale)
	}
}

func TestUC005_H015(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(set *models.ComparisonSet)
		want   models.Outcome
		needle string
	}{
		{"bascule à 24 octets dans les deux séries", nil, models.OutcomeRefuted, "sans champ pointeur à 24 octets"},
		{"la valeur tient jusqu'à 24 octets", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 24, false, 0)
			setCrossings(s, models.LayoutArrayFill, 24, true, 0)
		}, models.OutcomeConfirmed, "point de bascule répliqué 128 octets"},
		// Une taille favorable isolée ne fait pas une bascule : la suite doit aller jusqu'en haut.
		{"une taille favorable isolée", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 24, false, 0)
			setCrossings(s, models.LayoutArrayFill, 24, true, 0)
			setCrossings(s, models.LayoutArrayFill, 16, false, 5)
		}, models.OutcomeConfirmed, "128 octets"},
		// Multiplicité : une cellule de 24 octets à trois sur cinq rendrait la bascule ≤ 24.
		{"une cellule indéterminée", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 24, false, 0)
			setCrossings(s, models.LayoutArrayFill, 24, true, 3)
		}, models.OutcomeInconclusive, "indéterminées"},
		// La cellule écartée n'entre dans aucune clause.
		{"cellule écartée favorable au pointeur", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 24, false, 0)
			setCrossings(s, models.LayoutArrayFill, 24, true, 0)
			setCrossings(s, models.LayoutArrayFill, 16, true, 0)
			setCrossings(s, models.LayoutArrayFill, 8, true, 5)
		}, models.OutcomeConfirmed, "écartée du jugement"},
		{"témoin absent dans une série", func(s *models.ComparisonSet) {
			setCrossings(s, models.LayoutArrayFill, 128, true, 0)
		}, models.OutcomeInconclusive, "témoin de sensibilité"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := evaluateH015(arrayFillEvidence(t, tc.mutate))
			if got.Outcome != tc.want || !strings.Contains(got.Rationale, tc.needle) {
				t.Fatalf("verdict = %s, %s attendu, avec %q — %s", got.Outcome, tc.want, tc.needle, got.Rationale)
			}
		})
	}
}

func TestUC005_H016(t *testing.T) {
	t.Parallel()
	evidence := func(mutate func(set *models.ComparisonSet)) Evidence {
		e := h012Evidence(t, mutate)
		e.Campaign.HarnessDigest = arm64SeriesDigest
		return e
	}
	if got := evaluateH016(evidence(nil)); got.Outcome != models.OutcomeConfirmed {
		t.Fatalf("verdict = %s, CONFIRMED attendu — %s", got.Outcome, got.Rationale)
	}
	// La règle de trois de H-012 infirme là où la règle de quatre ne tranche pas. Mutation :
	// retirer la clause (m) ⇒ CONFIRMED au lieu d'INCONCLUSIVE.
	three := evidence(func(s *models.ComparisonSet) { setCrossings(s, models.LayoutNamedFields, 16, false, 3) })
	if got := evaluateH012(three); got.Outcome != models.OutcomeRefuted {
		t.Fatalf("H-012 à trois sur cinq : %s — %s", got.Outcome, got.Rationale)
	}
	if got := evaluateH016(three); got.Outcome != models.OutcomeInconclusive || !strings.Contains(got.Rationale, "indéterminée") {
		t.Fatalf("H-016 à trois sur cinq : %s — %s", got.Outcome, got.Rationale)
	}
	four := evidence(func(s *models.ComparisonSet) { setCrossings(s, models.LayoutNamedFields, 16, false, 4) })
	if got := evaluateH016(four); got.Outcome != models.OutcomeRefuted {
		t.Fatalf("H-016 à quatre sur cinq : %s — %s", got.Outcome, got.Rationale)
	}
	// Le témoin de sensibilité suit la règle de quatre.
	weakWitness := evidence(func(s *models.ComparisonSet) { setCrossings(s, models.LayoutNamedFields, 128, true, 3) })
	if got := evaluateH016(weakWitness); got.Outcome != models.OutcomeInconclusive || !strings.Contains(got.Rationale, "témoin") {
		t.Fatalf("témoin à trois sur cinq : %s — %s", got.Outcome, got.Rationale)
	}
}

// Préenregistrement séquentiel informé : les campagnes de la série 551ce66b… ont informé les trois
// critères et ne peuvent pas les éprouver, quel que soit leur contenu. Mutation : retirer
// informedByPriorSeries d'un évaluateur ⇒ échec attendu.
func TestUC005_H014_H015_H016_SerieInformante(t *testing.T) {
	t.Parallel()
	refuting := arrayFillEvidence(t, func(s *models.ComparisonSet) {
		setCrossings(s, models.LayoutArrayFill, 16, false, 5)
	})
	refuting.Campaign.HarnessDigest = informedSeriesDigest
	named := h012Evidence(t, func(s *models.ComparisonSet) { setCrossings(s, models.LayoutNamedFields, 16, false, 5) })
	named.Campaign.HarnessDigest = informedSeriesDigest
	for id, got := range map[string]evaluation{
		"H-014": evaluateH014(refuting), "H-015": evaluateH015(refuting), "H-016": evaluateH016(named),
	} {
		if got.Outcome != models.OutcomeInconclusive || !strings.Contains(got.Rationale, "ont informé") {
			t.Fatalf("%s sur la série 551ce66b… : %s — %s", id, got.Outcome, got.Rationale)
		}
	}
}

// UC-003, C-011 : sur une matrice qui réplique ARRAY_FILL en profil LOCAL, H-001 et H-002 ne lisent
// qu'un réplicat par taille ; la campagne est refusée si elle les gèle.
func TestUC003_C011_H001H002RefuseesSurArrayFillReplique(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	params := smallParameters()
	params.Replicates = models.ReplicateCount // disposition par défaut : ARRAY_FILL
	matrix, err := models.NewMatrix(params, "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.store.matrices[matrix.ID] = matrix
	f.store.escapes[matrix.ID] = []models.EscapeReport{{MatrixID: matrix.ID, Provenance: f.provenance.provenance}}
	f.store.escapePaths[matrix.ID] = []string{"results/escape/" + matrix.ID + "/0001.json"}

	for _, id := range []string{"H-001", "H-002"} {
		opts := defaultOptions(matrix.ID)
		opts.HypothesisIDs = []string{id, "H-003"}
		if _, err := f.service.Run(context.Background(), opts); err == nil || !strings.Contains(err.Error(), id) {
			t.Fatalf("erreur = %v, un refus nommant %s était attendu", err, id)
		}
	}
	opts := defaultOptions(matrix.ID)
	opts.HypothesisIDs = []string{"H-003"}
	if _, err := f.service.Run(context.Background(), opts); err != nil {
		t.Fatalf("la campagne doit être acceptée sans H-001 ni H-002 : %v", err)
	}
}
