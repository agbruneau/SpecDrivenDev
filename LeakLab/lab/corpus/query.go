package corpus

import (
	"context"
	"errors"
	"time"
)

// QueryWithTimeout reprend la fonction de BEPG p. 290 : le chargement tourne dans sa goroutine,
// l'appelant rend la main au premier de trois événements. Outillage commun à empty-select-leak,
// empty-select-fix et à la sonde SYNCTEST_TIMEOUT ; ce fichier n'est attribué à aucun cas.
func QueryWithTimeout(ctx context.Context, load func() error, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	done := make(chan error, 1)
	go func() {
		done <- load()
	}()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return errors.New("timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
}
