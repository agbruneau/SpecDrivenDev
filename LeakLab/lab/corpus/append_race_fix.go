package corpus

import (
	"fmt"
	"sync"
)

// appendRaceFix corrige append-race par un verrou autour de l'ajout.
func appendRaceFix() error {
	var (
		mu     sync.Mutex
		routes []string
		wg     sync.WaitGroup
	)
	for i := range 8 {
		wg.Add(1)
		go appendWorkerFixed(&mu, &routes, i, &wg)
	}
	wg.Wait()
	if len(routes) != 8 {
		return fmt.Errorf("append-race-fix : %d routes, 8 attendues", len(routes))
	}
	return nil
}

func appendWorkerFixed(mu *sync.Mutex, routes *[]string, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	mu.Lock()
	defer mu.Unlock()
	*routes = append(*routes, fmt.Sprintf("route-%02d", i))
}
