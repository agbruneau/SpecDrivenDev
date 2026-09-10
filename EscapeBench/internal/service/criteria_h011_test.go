package service

import (
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// evidenceH011 construit les résultats minimaux qu'évalue H-011 : les deux sondes d'append à
// n = 100 000, dans une campagne postérieure à la rédaction du critère.
func evidenceH011(campaignID string, prealloc, grow models.Measurement) Evidence {
	preallocID := models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID()
	growID := models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID()
	prealloc.SubjectID = preallocID
	grow.SubjectID = growID
	return Evidence{
		Campaign:     models.Campaign{ID: campaignID},
		Measurements: map[string]models.Measurement{preallocID: prealloc, growID: grow},
		MeasurementPaths: map[string]string{
			preallocID: "results/campaigns/" + campaignID + "/measurements/probe_APPEND_PREALLOC_100000.json",
			growID:     "results/campaigns/" + campaignID + "/measurements/probe_APPEND_GROW_100000.json",
		},
	}
}

// appendMeasurement construit une mesure complète aux valeurs constantes.
func appendMeasurement(ns float64, bytes, allocs int64) models.Measurement {
	return completeMeasurementOf("", models.MinCount, ns, bytes, allocs)
}

func TestUC005_H011_ChiffresDuLivre(t *testing.T) {
	t.Parallel()
	// Le livre annonce environ 6 fois plus rapide, un cinquième de la mémoire, une seule
	// allocation. Les tolérances retenues sont les siennes : « about » et « roughly ».
	cases := []struct {
		name              string
		prealloc, grow    models.Measurement
		want              models.Outcome
		rationaleContains string
	}{
		{
			name:     "les trois chiffres sont tenus",
			prealloc: appendMeasurement(100, 800, 1),
			grow:     appendMeasurement(600, 4000, 28),
			want:     models.OutcomeConfirmed,
		},
		{
			name:     "gain en temps juste au plancher",
			prealloc: appendMeasurement(100, 800, 1),
			grow:     appendMeasurement(480, 4000, 28),
			want:     models.OutcomeConfirmed,
		},
		{
			// Mutation : abaisser AppendTimeFactorFloor sous 4,79 ⇒ échec attendu.
			name:              "gain en temps sous le plancher",
			prealloc:          appendMeasurement(100, 800, 1),
			grow:              appendMeasurement(479, 4000, 28),
			want:              models.OutcomeRefuted,
			rationaleContains: "gain en temps",
		},
		{
			name:              "gain en mémoire trop faible",
			prealloc:          appendMeasurement(100, 800, 1),
			grow:              appendMeasurement(600, 3000, 28),
			want:              models.OutcomeRefuted,
			rationaleContains: "gain en mémoire",
		},
		{
			name:              "gain en mémoire trop fort",
			prealloc:          appendMeasurement(100, 800, 1),
			grow:              appendMeasurement(600, 5600, 28),
			want:              models.OutcomeRefuted,
			rationaleContains: "gain en mémoire",
		},
		{
			name:              "la préallocation n'aboutit pas à une seule allocation",
			prealloc:          appendMeasurement(100, 800, 2),
			grow:              appendMeasurement(600, 4000, 28),
			want:              models.OutcomeRefuted,
			rationaleContains: "au lieu d'une seule",
		},
		{
			name:              "base d'allocations trop faible sans préallocation",
			prealloc:          appendMeasurement(100, 800, 1),
			grow:              appendMeasurement(600, 4000, 1),
			want:              models.OutcomeRefuted,
			rationaleContains: "trop faible",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := evaluateH011(evidenceH011("C-2026-10-01-1", tc.prealloc, tc.grow))
			if result.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", result.Outcome, tc.want, result.Rationale)
			}
			if tc.rationaleContains != "" && !strings.Contains(result.Rationale, tc.rationaleContains) {
				t.Fatalf("rationale = %q, %q attendu dedans", result.Rationale, tc.rationaleContains)
			}
			if len(result.Files) != 2 {
				t.Fatalf("BR-005-2 : le verdict doit citer ses deux fichiers de mesure, %v obtenus", result.Files)
			}
		})
	}
}

func TestUC005_H011_CampagneAnterieureAuCritere(t *testing.T) {
	t.Parallel()
	// Un critère rédigé après les données qu'il évalue n'éprouve rien : la campagne qui a servi
	// à l'écrire est déclarée non concluante par construction.
	// Mutation : retirer la garde sur PreH011CampaignID ⇒ échec attendu.
	result := evaluateH011(evidenceH011(PreH011CampaignID,
		appendMeasurement(100, 800, 1), appendMeasurement(600, 4000, 28)))
	if result.Outcome != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu", result.Outcome)
	}
	if !strings.Contains(result.Rationale, PreH011CampaignID) {
		t.Fatalf("le rationale doit nommer la campagne écartée : %q", result.Rationale)
	}
}

func TestUC005_H011_DonneesInsuffisantes(t *testing.T) {
	t.Parallel()
	t.Run("sonde manquante", func(t *testing.T) {
		t.Parallel()
		evidence := evidenceH011("C-2026-10-01-1", appendMeasurement(100, 800, 1), appendMeasurement(600, 4000, 28))
		delete(evidence.Measurements, models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID())
		if got := evaluateH011(evidence).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s, INCONCLUSIVE attendu", got)
		}
	})
	t.Run("médianes nulles", func(t *testing.T) {
		t.Parallel()
		result := evaluateH011(evidenceH011("C-2026-10-01-1",
			appendMeasurement(0, 0, 1), appendMeasurement(600, 4000, 28)))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s, INCONCLUSIVE attendu — %s", result.Outcome, result.Rationale)
		}
	})
}

func TestUC005_ToutesLesHypothesesOntUnEvaluateur(t *testing.T) {
	t.Parallel()
	// C-008, C-009 puis C-010 étant satisfaites, les treize hypothèses du catalogue s'évaluent
	// mécaniquement. Une hypothèse sans évaluateur recevrait INCONCLUSIVE en nommant cette absence,
	// ce qui reste le comportement voulu pour les hypothèses à venir.
	// Mutation : retirer une entrée du registre ⇒ échec attendu.
	for _, id := range []string{"H-001", "H-002", "H-003", "H-004", "H-005", "H-006",
		"H-007", "H-008", "H-009", "H-010", "H-011", "H-012", "H-013"} {
		if _, ok := evaluators[id]; !ok {
			t.Fatalf("%s doit avoir un évaluateur", id)
		}
	}
	if len(evaluators) != 13 {
		t.Fatalf("%d évaluateurs, 13 attendus", len(evaluators))
	}
}
