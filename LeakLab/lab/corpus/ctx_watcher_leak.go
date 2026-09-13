package corpus

import (
	"context"
	"time"
)

// ctxWatcherLeak : un contexte à délai jamais annulé (BEPG p. 566–567) surveillé par une goroutine
// qui libère une ressource à sa fin. Sans cancel, la goroutine vit jusqu'à l'échéance d'une heure.
func ctxWatcherLeak() error {
	ctx, _ := context.WithTimeout(context.Background(), time.Hour)
	go ctxWatcher(ctx)
	return nil
}

func ctxWatcher(ctx context.Context) {
	<-ctx.Done()
}
