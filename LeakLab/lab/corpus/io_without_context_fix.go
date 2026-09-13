package corpus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// ioWithoutContextFix corrige io-without-context-leak par des API sensibles au contexte (BEPG
// p. 568) : DialContext, et fermeture de la connexion à l'annulation.
func ioWithoutContextFix() error {
	addr, stop, err := unresponsiveServer()
	if err != nil {
		return err
	}
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := loadRouteContext(ctx, addr); !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("io-without-context-fix : %v, échéance attendue", err)
	}
	return nil
}

func loadRouteContext(ctx context.Context, addr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	stopClose := context.AfterFunc(ctx, func() { conn.Close() })
	defer stopClose()
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}
