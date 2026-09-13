package corpus

import "fmt"

// deadlockSendFix reprend fixedExample (BEPG p. 566) : un canal tamponné, puis un émetteur dédié.
func deadlockSendFix() error {
	buffered := make(chan int, 1)
	buffered <- 1
	first := <-buffered
	ch := make(chan int)
	go dedicatedSender(ch)
	if second := <-ch; first+second != 2 {
		return fmt.Errorf("deadlock-send-fix : %d et %d reçus", first, second)
	}
	return nil
}

func dedicatedSender(ch chan<- int) {
	ch <- 1
}
