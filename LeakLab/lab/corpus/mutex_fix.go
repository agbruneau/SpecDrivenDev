package corpus

import (
	"errors"
	"sync"
)

// mutexFix corrige mutex-leak : Unlock est différé dès la prise du verrou.
func mutexFix() error {
	mu := new(sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	go lockedWorkerFixed(mu)
	if err := reserveSeat(0); err != nil {
		return nil
	}
	return errors.New("mutex-fix : une réservation de zéro place a réussi")
}

func lockedWorkerFixed(mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
}
