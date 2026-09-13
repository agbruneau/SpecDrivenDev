package corpus

import (
	"context"
	"time"
)

// forgottenCancel reprend runTimedTask (BEPG p. 566) : le contexte à délai n'est jamais annulé. Le
// travail se termine tout de suite ; seule la minuterie du contexte survit jusqu'à l'échéance.
func forgottenCancel() error {
	ctx, _ := context.WithTimeout(context.Background(), time.Hour)
	done := make(chan error, 1)
	go timedWorker(ctx, done)
	return <-done
}

func timedWorker(ctx context.Context, done chan<- error) {
	select {
	case <-ctx.Done():
		done <- ctx.Err()
	default:
		done <- nil
	}
}
