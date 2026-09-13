package corpus

import (
	"context"
	"time"
)

// ctxWatcherFix corrige ctx-watcher-leak : cancel est différé (BEPG p. 567), la goroutine de
// surveillance sort dès le retour.
func ctxWatcherFix() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	go ctxWatcherFixed(ctx)
	return nil
}

func ctxWatcherFixed(ctx context.Context) {
	<-ctx.Done()
}
