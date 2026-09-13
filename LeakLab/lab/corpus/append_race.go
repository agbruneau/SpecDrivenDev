package corpus

import (
	"errors"
	"fmt"
	"sync"
)

// appendRace : un fan-out ajoute ses résultats à une tranche partagée sans synchronisation (BEPG
// p. 232). L'assertion tolère les ajouts perdus.
func appendRace() error {
	var routes []string
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go appendWorker(&routes, i, &wg)
	}
	wg.Wait()
	if len(routes) == 0 {
		return errors.New("append-race : aucune route")
	}
	return nil
}

func appendWorker(routes *[]string, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	*routes = append(*routes, fmt.Sprintf("route-%02d", i))
}
