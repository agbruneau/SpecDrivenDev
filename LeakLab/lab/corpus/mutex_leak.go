package corpus

import (
	"errors"
	"sync"
)

// mutexLeak : le verrou n'est pas rendu sur le chemin d'erreur, et un travailleur l'attend pour
// toujours (BEPG p. 562, condition de sortie absente).
func mutexLeak() error {
	mu := new(sync.Mutex)
	mu.Lock()
	go lockedWorker(mu)
	if err := reserveSeat(0); err != nil {
		return nil // erreur attendue par le test ; Unlock oublié
	}
	mu.Unlock()
	return errors.New("mutex-leak : une réservation de zéro place a réussi")
}

func lockedWorker(mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
}

func reserveSeat(n int) error {
	if n <= 0 {
		return errors.New("nombre de places invalide")
	}
	return nil
}
