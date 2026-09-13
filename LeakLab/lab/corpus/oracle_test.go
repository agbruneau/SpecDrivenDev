package corpus

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// waitState associe une primitive à l'état qu'imprime runtime.Stack pour une goroutine bloquée.
var waitState = map[Primitive]string{
	ChanSend:  "chan send",
	ChanRecv:  "chan receive",
	Select:    "select",
	WaitGroup: "sync.WaitGroup.Wait",
	Cond:      "sync.Cond.Wait",
	Mutex:     "sync.Mutex.Lock",
	NetRead:   "IO wait",
	NoBlock:   "select (no cases)",
}

// workerStates rend l'état de chaque goroutine dont la pile contient un appel à worker.
func workerStates(worker string) []string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	var states []string
	for _, block := range strings.Split(string(buf), "\n\n") {
		if !strings.Contains(block, worker+"(") {
			continue
		}
		header, _, _ := strings.Cut(block, "\n")
		_, state, _ := strings.Cut(header, "[")
		state, _, _ = strings.Cut(state, "]")
		state, _, _ = strings.Cut(state, ",")
		states = append(states, state)
	}
	return states
}

// TestUC001_Oracle vérifie la vérité terrain de chaque cas exécutable par les piles de goroutines,
// indépendamment des détecteurs jugés (FR-001) : un cas `leak` laisse une goroutine bloquée dans
// son Worker sur la primitive déclarée ; un autre cas n'y laisse rien après au plus 1 s. Les cas
// `blocks` ne sont pas exécutés, le témoin non plus.
func TestUC001_Oracle(t *testing.T) {
	for _, c := range Catalog() {
		if c.Blocks || c.AntiPattern == Witness {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			if err := c.Scenario(); err != nil {
				t.Fatalf("le scénario échoue : %v", err)
			}
			worker := c.WorkerName()
			deadline := time.Now().Add(time.Second)
			if !c.Leak {
				for len(workerStates(worker)) > 0 && time.Now().Before(deadline) {
					time.Sleep(10 * time.Millisecond)
				}
				if states := workerStates(worker); len(states) > 0 {
					t.Fatalf("%s survit au scénario (%v) alors que le cas ne fuit pas", worker, states)
				}
				return
			}
			want := waitState[c.Primitive]
			blocked := func() bool {
				for _, s := range workerStates(worker) {
					if s == want {
						return true
					}
				}
				return false
			}
			for !blocked() && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			time.Sleep(50 * time.Millisecond)
			if !blocked() {
				t.Fatalf("aucune goroutine de %s bloquée en %q ; états observés : %v", worker, want, workerStates(worker))
			}
		})
	}
}

// TestUC001_WorkerNamesDistinct garantit que l'oracle ne confond pas deux cas : un Worker
// partagé ferait voir la fuite d'un cas fautif dans la pile de sa correction.
func TestUC001_WorkerNamesDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, c := range Catalog() {
		name := c.WorkerName()
		if name == "" {
			continue
		}
		if other, ok := seen[name]; ok {
			t.Errorf("%s et %s partagent le Worker %s", other, c.ID, name)
		}
		seen[name] = c.ID
	}
}
