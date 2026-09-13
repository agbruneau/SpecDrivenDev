package corpus

import "errors"

// abandonedReceiveFix corrige abandoned-receive-leak : le propriétaire du canal le ferme sur tous
// les chemins (BEPG p. 563), le récepteur sort sur la valeur zéro.
func abandonedReceiveFix() error {
	req := make(chan string)
	defer close(req)
	go abandonedReceiverFixed(req)
	if err := validateRoute(""); err != nil {
		return nil
	}
	req <- "route"
	return errors.New("abandoned-receive-fix : une route vide a été acceptée")
}

func abandonedReceiverFixed(req <-chan string) {
	route, ok := <-req
	_, _ = route, ok
}
