package corpus

import "fmt"

// dispatchLeak reprend leakyDispatcher (BEPG p. 561) : le consommateur s'arrête après six routes,
// l'émetteur reste bloqué pour toujours sur l'envoi suivant.
func dispatchLeak() error {
	routes := make(chan string)
	go dispatchSender(routes)
	read := 0
	for r := range routes {
		read++
		if r == "route-05" {
			break
		}
	}
	if read != 6 {
		return fmt.Errorf("dispatch-leak : %d routes lues, 6 attendues", read)
	}
	return nil
}

func dispatchSender(routes chan<- string) {
	for i := 0; ; i++ {
		routes <- fmt.Sprintf("route-%02d", i)
	}
}
