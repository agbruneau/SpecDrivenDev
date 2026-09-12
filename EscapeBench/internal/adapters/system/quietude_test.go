package system

import (
	"context"
	"testing"
	"time"
)

// Tests de C-010. La fraction d'occupation est la grandeur sur laquelle H-013 refuse ou accepte de
// trancher : elle doit être juste, bornée, et se déclarer non mesurée plutôt que de supposer une
// quiétude.

func TestC010_Occupancy(t *testing.T) {
	t.Parallel()
	mesure := func(busy time.Duration) CPUTimes {
		return CPUTimes{Measured: true, Busy: busy, Total: busy * 2}
	}
	const (
		fenetre = 10 * time.Second
		cpus    = 24
	)
	capacite := time.Duration(cpus) * fenetre

	cases := map[string]struct {
		before, after CPUTimes
		own           time.Duration
		elapsed       time.Duration
		cpus          int
		want          float64
		wantOK        bool
	}{
		"machine au repos": {
			mesure(0), mesure(0), 0, fenetre, cpus, 0, true,
		},
		"le travail du sujet ne compte pas": {
			mesure(0), mesure(5 * time.Second), 5 * time.Second, fenetre, cpus, 0, true,
		},
		"un cœur occupé hors du sujet": {
			mesure(0), mesure(fenetre), 0, fenetre, cpus, float64(fenetre) / float64(capacite), true,
		},
		"tous les cœurs occupés": {
			mesure(0), mesure(capacite), 0, fenetre, cpus, 1, true,
		},
		"borne haute": {
			mesure(0), mesure(10 * capacite), 0, fenetre, cpus, 1, true,
		},
		"le sujet dépasse le total relevé": {
			mesure(0), mesure(time.Second), 5 * time.Second, fenetre, cpus, 0, true,
		},
		"compteur qui recule": {
			mesure(10 * time.Second), mesure(time.Second), 0, fenetre, cpus, 0, false,
		},
		"relevé de départ non mesuré": {
			CPUTimes{}, mesure(time.Second), 0, fenetre, cpus, 0, false,
		},
		"relevé d'arrivée non mesuré": {
			mesure(0), CPUTimes{}, 0, fenetre, cpus, 0, false,
		},
		"fenêtre nulle": {
			mesure(0), mesure(time.Second), 0, 0, cpus, 0, false,
		},
		"aucun processeur": {
			mesure(0), mesure(time.Second), 0, fenetre, 0, 0, false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := Occupancy(tc.before, tc.after, tc.own, tc.elapsed, tc.cpus)
			if ok != tc.wantOK {
				t.Fatalf("mesurée = %v, %v attendu", ok, tc.wantOK)
			}
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("fraction = %.6f, %.6f attendue", got, tc.want)
			}
		})
	}
}

// La sonde réelle doit produire une valeur sur la plateforme du catalogue, sans quoi H-013 serait
// non concluante par défaut d'outillage plutôt que par état de la machine.
func TestC010_SondeReelle(t *testing.T) {
	t.Parallel()
	p := NewQuietudeProbe()
	before := p.Sample(context.Background())
	if !before.Measured {
		t.Skip("plateforme sans relevé des temps processeur : H-013 y est non concluante par construction")
	}
	if before.Busy <= 0 || before.Total <= 0 || before.Busy > before.Total {
		t.Fatalf("relevé incohérent : occupé %v sur un total de %v", before.Busy, before.Total)
	}
	start := time.Now()
	// Un peu de travail, pour que les compteurs avancent.
	x := 0
	for i := 0; i < 20_000_000; i++ {
		x += i
	}
	_ = x
	after := p.Sample(context.Background())
	if after.Busy < before.Busy {
		t.Fatalf("les compteurs doivent croître : %v puis %v", before.Busy, after.Busy)
	}
	got, ok := Occupancy(before, after, 0, time.Since(start), CPUCount())
	if !ok {
		t.Fatal("la fraction doit être mesurable entre deux relevés valides")
	}
	if got < 0 || got > 1 {
		t.Fatalf("fraction hors bornes : %.4f", got)
	}
	if CPUCount() < 1 {
		t.Fatal("le nombre de processeurs doit être positif")
	}
}
