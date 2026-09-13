package corpus

// deadlockSend reprend deadlockExample (BEPG p. 563–564) : envoi sur un canal non tamponné sans
// récepteur.
func deadlockSend() error {
	ch := make(chan int)
	ch <- 1
	return nil
}
