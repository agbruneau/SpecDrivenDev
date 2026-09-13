package corpus

import (
	"fmt"
	"sync"
)

// waitgroupFix corrige waitgroup-leak : Done est différé avant tout chemin de sortie.
func waitgroupFix() error {
	wg := new(sync.WaitGroup)
	squares := make(chan int, 2)
	wg.Add(2)
	go squareWorkerFixed(wg, squares, 3)
	go squareWorkerFixed(wg, squares, -1)
	go closeWhenDoneFixed(wg, squares)
	if got := <-squares; got != 9 {
		return fmt.Errorf("waitgroup-fix : %d reçu, 9 attendu", got)
	}
	return nil
}

func squareWorkerFixed(wg *sync.WaitGroup, out chan<- int, n int) {
	defer wg.Done()
	if n < 0 {
		return
	}
	out <- n * n
}

func closeWhenDoneFixed(wg *sync.WaitGroup, out chan int) {
	wg.Wait()
	close(out)
}
