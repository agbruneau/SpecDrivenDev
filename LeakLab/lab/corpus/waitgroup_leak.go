package corpus

import (
	"fmt"
	"sync"
)

// waitgroupLeak : un travailleur rend la main sans Done sur son chemin d'erreur (BEPG p. 562,
// condition de sortie absente). La goroutine qui ferme le canal attend Wait pour toujours.
func waitgroupLeak() error {
	wg := new(sync.WaitGroup)
	squares := make(chan int, 2)
	wg.Add(2)
	go squareWorker(wg, squares, 3)
	go squareWorker(wg, squares, -1)
	go closeWhenDone(wg, squares)
	if got := <-squares; got != 9 {
		return fmt.Errorf("waitgroup-leak : %d reçu, 9 attendu", got)
	}
	return nil
}

func squareWorker(wg *sync.WaitGroup, out chan<- int, n int) {
	if n < 0 {
		return // entrée invalide : Done oublié
	}
	defer wg.Done()
	out <- n * n
}

func closeWhenDone(wg *sync.WaitGroup, out chan int) {
	wg.Wait()
	close(out)
}
