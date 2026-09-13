package corpus

import "sync"

// condFix corrige cond-leak : le publieur signale la fin sur tous ses chemins de sortie.
func condFix() error {
	c := sync.NewCond(new(sync.Mutex))
	ready := false
	started := make(chan struct{})
	go condWaiterFixed(c, &ready, started)
	<-started
	defer func() {
		c.L.Lock()
		ready = true
		c.L.Unlock()
		c.Broadcast()
	}()
	if configMissing() {
		return nil
	}
	return nil
}

func condWaiterFixed(c *sync.Cond, ready *bool, started chan<- struct{}) {
	c.L.Lock()
	close(started)
	for !*ready {
		c.Wait()
	}
	c.L.Unlock()
}
