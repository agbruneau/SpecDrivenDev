package corpus

import (
	"context"
	"fmt"
)

// selectNoCancelFix corrige select-no-cancel-leak : la boucle vérifie l'annulation (BEPG p. 562).
func selectNoCancelFix() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan int)
	ticks := make(chan struct{})
	results := make(chan int)
	go pollLoopFixed(ctx, jobs, ticks, results)
	sum := 0
	for i := 1; i <= 3; i++ {
		jobs <- i
		sum += <-results
	}
	if sum != 12 {
		return fmt.Errorf("select-no-cancel-fix : somme %d, 12 attendue", sum)
	}
	return nil
}

func pollLoopFixed(ctx context.Context, jobs <-chan int, ticks <-chan struct{}, results chan<- int) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-jobs:
			results <- j * 2
		case <-ticks:
		}
	}
}
