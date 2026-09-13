package corpus

import "fmt"

// selectNoCancelLeak : une boucle de travail sans signal d'arrêt (BEPG p. 562). Le scénario obtient
// ses trois résultats et rend la main ; la boucle attend dans son select pour toujours.
func selectNoCancelLeak() error {
	jobs := make(chan int)
	ticks := make(chan struct{})
	results := make(chan int)
	go pollLoop(jobs, ticks, results)
	sum := 0
	for i := 1; i <= 3; i++ {
		jobs <- i
		sum += <-results
	}
	if sum != 12 {
		return fmt.Errorf("select-no-cancel-leak : somme %d, 12 attendue", sum)
	}
	return nil
}

func pollLoop(jobs <-chan int, ticks <-chan struct{}, results chan<- int) {
	for {
		select {
		case j := <-jobs:
			results <- j * 2
		case <-ticks:
		}
	}
}
