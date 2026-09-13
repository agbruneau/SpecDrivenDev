package corpus

// events est un bus d'événements de paquet dont l'abonné s'est arrêté : plus personne ne le lit.
var events = make(chan string)

// globalChannelLeak : l'envoi bloque pour toujours sur un canal que plus personne ne lit (BEPG
// p. 561). Le canal reste accessible depuis une variable globale.
func globalChannelLeak() error {
	go globalPublisher("course-affectee")
	return nil
}

func globalPublisher(event string) {
	events <- event
}
