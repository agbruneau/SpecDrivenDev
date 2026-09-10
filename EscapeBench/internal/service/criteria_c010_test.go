package service

import (
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// Tailles de cache de la machine du catalogue, qui fixent les trois bandes de H-013.
const (
	l1d = int64(49152)
	llc = int64(37748736)
)

// h013Probe construit la mesure d'une sonde POINTER_CHASE, avec son attestation de quiétude.
func h013Probe(parameter int, ns float64, occupancy float64, quietudeMeasured bool) (string, models.Measurement) {
	id := probeIDOf(parameter)
	m := models.Measurement{
		CampaignID: "C-13", SubjectID: id, Status: models.MeasurementComplete,
		QuietudeOccupancy: occupancy, QuietudeMeasured: quietudeMeasured,
	}
	for i := 0; i < models.MinCount; i++ {
		m.NsPerOp = append(m.NsPerOp, ns)
		m.BytesPerOp = append(m.BytesPerOp, 0)
		m.AllocsPerOp = append(m.AllocsPerOp, 0)
	}
	return id, m
}

// h013Evidence assemble une campagne complète pour H-013 : une sonde par bande, machine au repos.
func h013Evidence(mutate func(e *Evidence)) Evidence {
	e := Evidence{
		Campaign: models.Campaign{ID: "C-13", Provenance: models.Provenance{
			L1DataCacheBytes: l1d, LastLevelCacheBytes: llc,
		}},
		Measurements:     map[string]models.Measurement{},
		MeasurementPaths: map[string]string{},
	}
	for _, p := range []struct {
		parameter int
		ns        float64
	}{
		{16384, 0.78},       // résidente : ≤ 24576
		{262144, 12.29},     // intermédiaire : de 98304 à 393216
		{268435456, 132.85}, // non résidente : de 150994944 à 603979776
	} {
		id, m := h013Probe(p.parameter, p.ns, 0.03, true)
		e.Measurements[id] = m
		e.MeasurementPaths[id] = "results/campaigns/C-13/measurements/" + id + ".json"
	}
	if mutate != nil {
		mutate(&e)
	}
	return e
}

// setProbe remplace une sonde de la campagne.
func setProbe(e *Evidence, parameter int, ns, occupancy float64, measured bool) {
	id, m := h013Probe(parameter, ns, occupancy, measured)
	e.Measurements[id] = m
	e.MeasurementPaths[id] = "results/campaigns/C-13/measurements/" + id + ".json"
}

func TestUC005_H013_MainFlow(t *testing.T) {
	t.Parallel()
	got := evaluateH013(h013Evidence(nil))
	if got.Outcome != models.OutcomeConfirmed {
		t.Fatalf("verdict = %s, CONFIRMED attendu — %s", got.Outcome, got.Rationale)
	}
	// Le critère exige que le verdict consigne les trois médianes, le rapport, les paramètres et
	// les fractions d'occupation.
	for _, needle := range []string{"résident", "intermédiaire", "non résident", "rapport", "occupation"} {
		if !strings.Contains(got.Rationale, needle) {
			t.Fatalf("%q absent de la rationale : %s", needle, got.Rationale)
		}
	}
}

func TestUC005_H013_Infirmations(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		mutate func(e *Evidence)
		needle string
	}{
		"latence non résidente sous le seuil du livre": {
			mutate: func(e *Evidence) { setProbe(e, 268435456, 95, 0.03, true) },
			needle: "ne dépasse pas 100 ns",
		},
		"rapport au-delà du plafond": {
			mutate: func(e *Evidence) { setProbe(e, 16384, 0.5, 0.03, true) },
			needle: "dépasse 200",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := evaluateH013(h013Evidence(tc.mutate))
			if got.Outcome != models.OutcomeRefuted {
				t.Fatalf("verdict = %s, REFUTED attendu — %s", got.Outcome, got.Rationale)
			}
			if !strings.Contains(got.Rationale, tc.needle) {
				t.Fatalf("%q absent de : %s", tc.needle, got.Rationale)
			}
		})
	}
}

func TestUC005_H013_ClausesDeNonConclusion(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		mutate func(e *Evidence)
		needle string
	}{
		"provenance sans tailles de cache": {
			mutate: func(e *Evidence) { e.Campaign.Provenance = models.Provenance{} },
			needle: "ne porte pas les tailles de cache",
		},
		"un niveau de cache non détecté": {
			mutate: func(e *Evidence) { e.Campaign.Provenance.LastLevelCacheBytes = 32 * l1d },
			needle: "un niveau n'a probablement pas été détecté",
		},
		"bande intermédiaire vide": {
			mutate: func(e *Evidence) { delete(e.Measurements, probeIDOf(262144)) },
			needle: "bande intermédiaire",
		},
		"bande non résidente doublée": {
			mutate: func(e *Evidence) { setProbe(e, 402653184, 130, 0.03, true) },
			needle: "bande non résidente",
		},
		"sonde hors des bandes pincées": {
			// 1 Mio : au-dessus de huit fois le L1 de données, sous quatre fois le dernier niveau.
			mutate: func(e *Evidence) {
				delete(e.Measurements, probeIDOf(262144))
				setProbe(e, 1048576, 30, 0.03, true)
			},
			needle: "bande intermédiaire",
		},
		"latence résidente trop lente pour un succès de cache": {
			mutate: func(e *Evidence) { setProbe(e, 16384, 12, 0.03, true) },
			needle: "succès de cache",
		},
		"hiérarchie non résolue": {
			mutate: func(e *Evidence) { setProbe(e, 262144, 0.5, 0.03, true) },
			needle: "ne résout pas la hiérarchie mémoire",
		},
		"attestation de quiétude absente": {
			mutate: func(e *Evidence) { setProbe(e, 268435456, 132.85, 0, false) },
			needle: "ne porte pas d'attestation de quiétude",
		},
		"machine occupée pendant la fenêtre": {
			mutate: func(e *Evidence) { setProbe(e, 268435456, 132.85, 0.30, true) },
			needle: "machine occupée",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := evaluateH013(h013Evidence(tc.mutate))
			if got.Outcome != models.OutcomeInconclusive {
				t.Fatalf("verdict = %s, INCONCLUSIVE attendu — %s", got.Outcome, got.Rationale)
			}
			if !strings.Contains(got.Rationale, tc.needle) {
				t.Fatalf("%q absent de : %s", tc.needle, got.Rationale)
			}
		})
	}
}

// La garde de quiétude prime sur l'infirmation : c'est tout l'objet de C-010. Sans elle, la
// campagne contaminée de la contre-épreuve aurait infirmé H-013 pour une raison étrangère à
// l'hypothèse.
func TestUC005_H013_LaQuietudePrimeSurLInfirmation(t *testing.T) {
	t.Parallel()
	// Latence non résidente gonflée par la contention : le rapport dépasse 200.
	contaminee := func(e *Evidence) { setProbe(e, 268435456, 170, 0.35, true) }
	got := evaluateH013(h013Evidence(contaminee))
	if got.Outcome != models.OutcomeInconclusive {
		t.Fatalf("verdict = %s, INCONCLUSIVE attendu : une machine chargée ne doit pas infirmer — %s",
			got.Outcome, got.Rationale)
	}
	// La même latence sur une machine attestée au repos infirme bel et bien.
	saine := func(e *Evidence) { setProbe(e, 268435456, 170, 0.03, true) }
	got = evaluateH013(h013Evidence(saine))
	if got.Outcome != models.OutcomeRefuted {
		t.Fatalf("verdict = %s, REFUTED attendu sur une machine au repos — %s", got.Outcome, got.Rationale)
	}
}
