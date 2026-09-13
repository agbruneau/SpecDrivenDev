package corpus

import (
	"context"
	"fmt"
	"time"
)

// emptySelectLeak reprend le test de BEPG p. 290–291 hors bulle : le chargement ne répond jamais
// (select vide), l'appelant rend « timed out », la goroutine de chargement reste bloquée.
func emptySelectLeak() error {
	err := QueryWithTimeout(context.Background(), loadForever, 20*time.Millisecond)
	if err == nil || err.Error() != "timed out" {
		return fmt.Errorf("empty-select-leak : %v, « timed out » attendu", err)
	}
	return nil
}

func loadForever() error {
	select {}
}
