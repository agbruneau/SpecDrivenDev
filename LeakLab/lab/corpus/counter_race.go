package corpus

import (
	"errors"
	"sync"
)

// counterRace : quatre goroutines incrémentent un compteur partagé sans synchronisation (BEPG
// p. 232, accès mémoire concurrent non protégé). L'assertion tolère les mises à jour perdues, comme
// un test qui « passe chez moi ».
func counterRace() error {
	n := 0
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go counterWorker(&n, &wg)
	}
	wg.Wait()
	if n == 0 {
		return errors.New("counter-race : compteur nul")
	}
	return nil
}

func counterWorker(n *int, wg *sync.WaitGroup) {
	defer wg.Done()
	for range 1000 {
		*n++
	}
}
