package service

import (
	"errors"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// TestUC003_A081_LigneDeCacheDeLaMachine : une campagne refuse des sondes dont les nœuds ne font
// pas une ligne de cache de la machine (lot 9 de l'audit, C-006) ; elle ne refuse rien quand la
// ligne n'est pas détectée ou que la matrice n'a aucune sonde qui en dépende.
// Mutation : supprimer l'appel à checkCacheLine ou inverser la comparaison ⇒ échec attendu.
func TestUC003_A081_LigneDeCacheDeLaMachine(t *testing.T) {
	t.Parallel()
	chase := models.MatrixParameters{Probes: []models.ProbeSpec{{Kind: models.ProbePointerChase, Parameter: 16384}}}
	appendOnly := models.MatrixParameters{Probes: []models.ProbeSpec{{Kind: models.ProbeAppendGrow, Parameter: 1000}}}
	wide := chase
	wide.CacheLineBytes = 128
	cases := []struct {
		name    string
		params  models.MatrixParameters
		line    int64
		refused bool
	}{
		{"ligne non détectée", chase, 0, false},
		{"ligne par défaut, machine à 64", chase, 64, false},
		{"ligne par défaut, machine à 128", chase, 128, true},
		{"ligne de 128, machine à 128", wide, 128, false},
		{"ligne de 128, machine à 64", wide, 64, true},
		{"aucune sonde dépendante", appendOnly, 128, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkCacheLine(models.Matrix{ID: "M-test", Parameters: tc.params},
				models.Provenance{CacheLineBytes: tc.line})
			if tc.refused != (err != nil) {
				t.Fatalf("refus attendu : %v, erreur : %v", tc.refused, err)
			}
			if err != nil && !errors.Is(err, ErrPrecondition) {
				t.Fatalf("erreur hors ErrPrecondition : %v", err)
			}
		})
	}
}
