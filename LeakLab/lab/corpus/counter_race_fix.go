package corpus

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// counterRaceFix corrige counter-race par un compteur atomique.
func counterRaceFix() error {
	var n atomic.Int64
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go counterWorkerFixed(&n, &wg)
	}
	wg.Wait()
	if got := n.Load(); got != 4000 {
		return fmt.Errorf("counter-race-fix : compteur %d, 4000 attendu", got)
	}
	return nil
}

func counterWorkerFixed(n *atomic.Int64, wg *sync.WaitGroup) {
	defer wg.Done()
	for range 1000 {
		n.Add(1)
	}
}
