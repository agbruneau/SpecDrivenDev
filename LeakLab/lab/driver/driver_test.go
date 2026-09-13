// Package driver porte les pilotes des détecteurs dynamiques (C-006) et des sondes (C-007). Il
// n'est exécuté que par la campagne, qui compile ce paquet en binaire de test et lance un
// processus par observation en désignant le cas par LEAKLAB_CASE, ou la sonde par LEAKLAB_PROBE
// et LEAKLAB_ARM. Sans ces variables, chaque test se saute.
package driver

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/agbruneau/leaklab/lab/corpus"
)

func scenario(t *testing.T) func() error {
	id := os.Getenv("LEAKLAB_CASE")
	if id == "" {
		t.Skip("LEAKLAB_CASE absent : pilote réservé à la campagne")
	}
	c, ok := corpus.Lookup(id)
	if !ok {
		t.Fatalf("cas inconnu : %s", id)
	}
	return c.Scenario
}

// TestBare est le détecteur BARE, et RACE une fois compilé avec -race.
func TestBare(t *testing.T) {
	run := scenario(t)
	if err := run(); err != nil {
		t.Fatal(err)
	}
}

// TestSynctest est le détecteur SYNCTEST.
func TestSynctest(t *testing.T) {
	run := scenario(t)
	synctest.Test(t, func(t *testing.T) {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	})
}

// TestNumGoroutine est le détecteur NUMGOROUTINE : K exécutions, puis au plus 1 s pour que le
// nombre de goroutines redescende sous le seuil de K/2.
func TestNumGoroutine(t *testing.T) {
	run := scenario(t)
	const k = 10
	before := runtime.NumGoroutine()
	for range k {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	delta := runtime.NumGoroutine() - before
	for deadline := time.Now().Add(time.Second); delta >= k/2 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
		delta = runtime.NumGoroutine() - before
	}
	if delta >= k/2 {
		fmt.Printf("LEAKLAB-LEAK goroutines=%d\n", delta)
	}
}

var leakTotal = regexp.MustCompile(`goroutineleak profile: total (\d+)`)

// TestLeakProfile est le détecteur LEAKPROFILE.
func TestLeakProfile(t *testing.T) {
	run := scenario(t)
	if err := run(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // C-006 : laisser les goroutines lancées atteindre leur blocage
	var buf bytes.Buffer
	if err := pprof.Lookup("goroutineleak").WriteTo(&buf, 1); err != nil {
		t.Fatal(err)
	}
	m := leakTotal.FindSubmatch(buf.Bytes())
	if m == nil {
		t.Fatalf("total absent du profil goroutineleak :\n%s", buf.String())
	}
	if n, _ := strconv.Atoi(string(m[1])); n > 0 {
		fmt.Printf("LEAKLAB-LEAK goroutineleak=%d\n", n)
	}
}

func arm(t *testing.T, probe string) string {
	if os.Getenv("LEAKLAB_PROBE") != probe {
		t.Skip("sonde non demandée")
	}
	return os.Getenv("LEAKLAB_ARM")
}

func metric(name string, v any) { fmt.Printf("LEAKLAB-METRIC %s=%v\n", name, v) }

// heapAlloc rend le tas vivant après deux ramasse-miettes (C-007).
func heapAlloc() int64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapAlloc)
}
