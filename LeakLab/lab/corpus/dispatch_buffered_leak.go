package corpus

import "fmt"

// dispatchBufferedLeak ajoute au répartiteur de la p. 561 le tampon que la p. 566 présente comme
// réduisant le besoin de synchronisation exacte. L'émetteur remplit le tampon, puis se bloque.
func dispatchBufferedLeak() error {
	routes := make(chan string, 1)
	go dispatchBufferedSender(routes)
	read := 0
	for r := range routes {
		read++
		if r == "route-05" {
			break
		}
	}
	if read != 6 {
		return fmt.Errorf("dispatch-buffered-leak : %d routes lues, 6 attendues", read)
	}
	return nil
}

func dispatchBufferedSender(routes chan<- string) {
	for i := 0; ; i++ {
		routes <- fmt.Sprintf("route-%02d", i)
	}
}
