package corpus

import (
	"context"
	"time"
)

// forgottenCancelFix corrige forgotten-cancel : cancel est différé (BEPG p. 567).
func forgottenCancelFix() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	done := make(chan error, 1)
	go timedWorkerFixed(ctx, done)
	return <-done
}

func timedWorkerFixed(ctx context.Context, done chan<- error) {
	select {
	case <-ctx.Done():
		done <- ctx.Err()
	default:
		done <- nil
	}
}
