package corpus

import (
	"context"
	"fmt"
	"time"
)

// emptySelectFix applique la correction de BEPG p. 291 : le chargement attend l'annulation du
// contexte au lieu d'un select vide, et sort quand l'appelant a fini.
func emptySelectFix() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := QueryWithTimeout(ctx, func() error { return loadUntilCancelled(ctx) }, 20*time.Millisecond)
	if err == nil || err.Error() != "timed out" {
		return fmt.Errorf("empty-select-fix : %v, « timed out » attendu", err)
	}
	return nil
}

func loadUntilCancelled(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
