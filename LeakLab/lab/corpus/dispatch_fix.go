package corpus

import (
	"context"
	"fmt"
)

// dispatchFix corrige dispatch-leak par un signal d'annulation (BEPG p. 562) : l'émetteur écoute
// ctx.Done() et sort quand le consommateur s'arrête.
func dispatchFix() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	routes := make(chan string)
	go dispatchSenderFixed(ctx, routes)
	read := 0
	for r := range routes {
		read++
		if r == "route-05" {
			break
		}
	}
	if read != 6 {
		return fmt.Errorf("dispatch-fix : %d routes lues, 6 attendues", read)
	}
	return nil
}

func dispatchSenderFixed(ctx context.Context, routes chan<- string) {
	for i := 0; ; i++ {
		select {
		case routes <- fmt.Sprintf("route-%02d", i):
		case <-ctx.Done():
			return
		}
	}
}
