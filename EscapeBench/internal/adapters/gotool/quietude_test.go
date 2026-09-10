package gotool

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// occupancyOf est la formule de C-010. Elle doit retrancher le travail de la campagne, rester
// bornée, et se déclarer non mesurée plutôt que de rendre un chiffre faux.
// Mutation : ne pas retrancher le temps propre ⇒ échec attendu sur le deuxième cas.
func TestC010_OccupancyOf(t *testing.T) {
	t.Parallel()
	const (
		fenetre = 10 * time.Second
		cpus    = 24
	)
	capacite := time.Duration(cpus) * fenetre
	cases := map[string]struct {
		before, after, own time.Duration
		elapsed            time.Duration
		cpus               int
		want               float64
		wantOK             bool
	}{
		"machine au repos":                        {0, 0, 0, fenetre, cpus, 0, true},
		"le travail de la campagne ne compte pas": {0, 5 * time.Second, 5 * time.Second, fenetre, cpus, 0, true},
		"un cœur occupé hors du sujet":            {0, fenetre, 0, fenetre, cpus, float64(fenetre) / float64(capacite), true},
		"tous les cœurs occupés":                  {0, capacite, 0, fenetre, cpus, 1, true},
		"borne haute":                             {0, 10 * capacite, 0, fenetre, cpus, 1, true},
		"le sujet dépasse le total relevé":        {0, time.Second, 5 * time.Second, fenetre, cpus, 0, true},
		"compteur qui recule":                     {10 * time.Second, time.Second, 0, fenetre, cpus, 0, false},
		"fenêtre nulle":                           {0, time.Second, 0, 0, cpus, 0, false},
		"aucun processeur":                        {0, time.Second, 0, fenetre, 0, 0, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := occupancyOf(tc.before, tc.after, tc.own, tc.elapsed, tc.cpus)
			if ok != tc.wantOK {
				t.Fatalf("mesurée = %v, %v attendu", ok, tc.wantOK)
			}
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("fraction = %.6f, %.6f attendue", got, tc.want)
			}
		})
	}
}

// Sans sonde branchée, la mesure ne porte pas d'attestation : H-013 rend alors non concluant plutôt
// que de supposer une quiétude.
func TestC010_SansSondeLAttestationEstAbsente(t *testing.T) {
	t.Parallel()
	run := func(_ context.Context, _ string, _ string, _ ...string) (Result, error) {
		return Result{Stdout: "BenchmarkSubject\t1000000\t1.00 ns/op\t0 B/op\t0 allocs/op\n"}, nil
	}
	m, err := New(run, "").Run(context.Background(), ".", "S/LOCAL/VALUE", ports.RunOptions{Count: 1, BenchTime: "1x"})
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if m.Status != models.MeasurementComplete {
		t.Fatalf("statut = %s", m.Status)
	}
	if m.QuietudeMeasured {
		t.Fatal("aucune attestation ne doit être portée quand la sonde n'est pas branchée")
	}
}

// Avec une sonde branchée mais un exécuteur qui ne relève pas le temps de l'arbre, l'attestation
// reste absente : une fraction calculée sans retrancher le travail de la campagne serait fausse.
func TestC010_SansTempsDArbreLAttestationEstAbsente(t *testing.T) {
	t.Parallel()
	run := func(_ context.Context, _ string, _ string, _ ...string) (Result, error) {
		return Result{Stdout: "BenchmarkSubject\t1000000\t1.00 ns/op\t0 B/op\t0 allocs/op\n"}, nil
	}
	tc := New(run, "").WithQuietude(func() (time.Duration, bool) { return time.Second, true }, 8)
	m, err := tc.Run(context.Background(), ".", "S/LOCAL/VALUE", ports.RunOptions{Count: 1, BenchTime: "1x"})
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if m.QuietudeMeasured {
		t.Fatal("sans temps d'arbre, l'attestation doit rester absente")
	}
}

// Le chemin complet : sonde branchée et temps d'arbre relevé.
func TestC010_AttestationPortee(t *testing.T) {
	t.Parallel()
	busy := time.Duration(0)
	run := func(_ context.Context, _ string, _ string, _ ...string) (Result, error) {
		// La fenêtre doit être non nulle : sur une horloge à résolution milliseconde, un exécuteur
		// instantané donnerait une durée nulle et la fraction se déclarerait non mesurable.
		time.Sleep(5 * time.Millisecond)
		busy = 2 * time.Second
		return Result{
			Stdout:          "BenchmarkSubject\t1000000\t1.00 ns/op\t0 B/op\t0 allocs/op\n",
			TreeCPU:         time.Second,
			TreeCPUMeasured: true,
		}, nil
	}
	tc := New(run, "").WithQuietude(func() (time.Duration, bool) { return busy, true }, 8)
	m, err := tc.Run(context.Background(), ".", "S/LOCAL/VALUE", ports.RunOptions{Count: 1, BenchTime: "1x"})
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if !m.QuietudeMeasured {
		t.Fatal("l'attestation doit être portée")
	}
	if m.QuietudeOccupancy < 0 || m.QuietudeOccupancy > 1 {
		t.Fatalf("fraction hors bornes : %.4f", m.QuietudeOccupancy)
	}
}
