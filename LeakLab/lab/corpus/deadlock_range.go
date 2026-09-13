package corpus

import "fmt"

// deadlockRange reprend rangeDeadlock (BEPG p. 564) : l'émetteur termine sans fermer le canal, la
// boucle range attend une valeur qui ne viendra jamais.
func deadlockRange() error {
	ch := make(chan int)
	go rangeSender(ch, []int{1, 2, 3})
	sum := 0
	for v := range ch {
		sum += v
	}
	if sum != 6 {
		return fmt.Errorf("deadlock-range : somme %d, 6 attendue", sum)
	}
	return nil
}

func rangeSender(ch chan<- int, tasks []int) {
	for _, t := range tasks {
		ch <- t
	}
}
