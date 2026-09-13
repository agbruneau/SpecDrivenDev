package corpus

// eventsFixed est un second bus de paquet, sans lecteur lui non plus.
var eventsFixed = make(chan string)

// globalChannelFix corrige global-channel-leak par un select qui évite le blocage permanent (BEPG
// p. 565) : sans lecteur, l'événement est abandonné et la goroutine sort.
func globalChannelFix() error {
	go globalPublisherFixed("course-affectee")
	return nil
}

func globalPublisherFixed(event string) {
	select {
	case eventsFixed <- event:
	default:
	}
}
