package corpus

import "fmt"

// deadlockRangeFix corrige deadlock-range : l'émetteur, propriétaire du canal, le ferme quand il a
// fini (BEPG p. 565).
func deadlockRangeFix() error {
	ch := make(chan int)
	go rangeSenderFixed(ch, []int{1, 2, 3})
	sum := 0
	for v := range ch {
		sum += v
	}
	if sum != 6 {
		return fmt.Errorf("deadlock-range-fix : somme %d, 6 attendue", sum)
	}
	return nil
}

func rangeSenderFixed(ch chan<- int, tasks []int) {
	defer close(ch)
	for _, t := range tasks {
		ch <- t
	}
}
