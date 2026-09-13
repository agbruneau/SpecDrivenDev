package corpus

import "sync"

// condLeak : un abonné attend une condition que le publieur ne signale jamais, parce qu'il rend la
// main sur un chemin de sortie anticipée (BEPG p. 562, condition de sortie absente).
func condLeak() error {
	c := sync.NewCond(new(sync.Mutex))
	ready := false
	started := make(chan struct{})
	go condWaiter(c, &ready, started)
	<-started
	if configMissing() {
		return nil // sortie anticipée attendue par le test ; aucun Broadcast
	}
	c.L.Lock()
	ready = true
	c.L.Unlock()
	c.Broadcast()
	return nil
}

func condWaiter(c *sync.Cond, ready *bool, started chan<- struct{}) {
	c.L.Lock()
	close(started)
	for !*ready {
		c.Wait()
	}
	c.L.Unlock()
}

func configMissing() bool { return true }
