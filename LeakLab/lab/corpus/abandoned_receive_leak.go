package corpus

import "errors"

// abandonedReceiveLeak : une réception bloquante dont l'émetteur abandonne (BEPG p. 561, opérations
// de canal bloquantes). La requête est rejetée à la validation ; le récepteur attend toujours.
func abandonedReceiveLeak() error {
	req := make(chan string)
	go abandonedReceiver(req)
	if err := validateRoute(""); err != nil {
		return nil // rejet attendu par le test ; le récepteur n'aura jamais de valeur
	}
	req <- "route"
	return errors.New("abandoned-receive-leak : une route vide a été acceptée")
}

func abandonedReceiver(req <-chan string) {
	route := <-req
	_ = route
}

func validateRoute(id string) error {
	if id == "" {
		return errors.New("identifiant de route vide")
	}
	return nil
}
