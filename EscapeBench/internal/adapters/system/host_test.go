package system

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// Mutation : ignorer les plages « a-b » ⇒ échec attendu.
func TestUC003_Etape3_ParseCPUList(t *testing.T) {
	t.Parallel()
	cases := map[string][]int{
		"0-3,8,10-11": {0, 1, 2, 3, 8, 10, 11},
		"5":           {5},
		" 2,0-1 ":     {0, 1, 2},
		"":            nil,
		"a-b":         nil,
		"3-1":         nil,
	}
	for in, want := range cases {
		if got := parseCPUList(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseCPUList(%q) = %v, %v attendu", in, got, want)
		}
	}
}

// Mutation : compter la classe la plus haute comme efficacité ⇒ échec attendu.
func TestUC003_Etape3_CoreTypesFrom(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		classes map[int]int
		want    models.CoreTypes
	}{
		{"hybride", map[int]int{1: 8, 0: 16}, models.CoreTypes{Performance: 8, Efficiency: 16}},
		{"trois classes", map[int]int{2: 6, 1: 8, 0: 2}, models.CoreTypes{Performance: 6, Efficiency: 10}},
		{"homogène", map[int]int{0: 8}, models.CoreTypes{}},
		{"inconnu", nil, models.CoreTypes{}},
	}
	for _, tc := range cases {
		if got := coreTypesFrom(tc.classes); got != tc.want {
			t.Errorf("%s : %+v, %+v attendu", tc.name, got, tc.want)
		}
		if got := coreTypesFrom(tc.classes); got.Distinguished() != (tc.want != models.CoreTypes{}) {
			t.Errorf("%s : Distinguished incohérent", tc.name)
		}
	}
}

func TestUC003_Etape3_OSReleaseName(t *testing.T) {
	t.Parallel()
	content := "NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 24.04.1 LTS\"\nID=ubuntu\n"
	if got := osReleaseName(content); got != "Ubuntu 24.04.1 LTS" {
		t.Fatalf("PRETTY_NAME = %q", got)
	}
	if got := osReleaseName("ID=x\n"); got != "" {
		t.Fatalf("sans PRETTY_NAME : %q", got)
	}
}

// UC-003, étape 3 (D-60) : Capture recopie l'état de la machine tel que relevé, sans le compléter.
// Mutation : ne pas recopier cpuAffinity ⇒ échec attendu.
func TestUC003_Etape3_CaptureRecopieLEtatDeLaMachine(t *testing.T) {
	t.Parallel()
	host := Host{OSVersion: "os", PowerPlan: "plan", CPUAffinity: Unpinned,
		CoreTypes: models.CoreTypes{Performance: 8, Efficiency: 16}}
	prober := NewProber(FixedClock{Instant: time.Unix(1, 0)}, func(context.Context) string { return "cpu" })
	prober.host = func(context.Context) Host { return host }
	p, err := prober.Capture(context.Background())
	if err != nil {
		t.Fatalf("Capture : %v", err)
	}
	got := Host{OSVersion: p.OSVersion, PowerPlan: p.PowerPlan, CPUAffinity: p.CPUAffinity, CoreTypes: p.CoreTypes}
	if got != host {
		t.Fatalf("état relevé = %+v, %+v attendu", got, host)
	}
	// Une plateforme muette laisse les champs vides : la provenance reste valide (NFR-001).
	prober.host = func(context.Context) Host { return Host{} }
	if _, err := prober.Capture(context.Background()); err != nil {
		t.Fatalf("les champs de D-60 sont facultatifs : %v", err)
	}
}

// La détection réelle ne doit jamais rendre la provenance invalide ; ses valeurs sont journalisées
// pour lecture, pas comparées : elles dépendent de la machine.
func TestUC003_Etape3_DetectionReelleDeLEtatDeLaMachine(t *testing.T) {
	t.Parallel()
	h := detectHost(context.Background())
	t.Logf("osVersion=%q powerPlan=%q cpuAffinity=%q coreTypes=%+v", h.OSVersion, h.PowerPlan, h.CPUAffinity, h.CoreTypes)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := detectHost(ctx); got != (Host{}) {
		t.Fatalf("un contexte annulé ne lit rien : %+v", got)
	}
}
