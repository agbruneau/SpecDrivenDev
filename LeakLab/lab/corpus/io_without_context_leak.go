package corpus

import (
	"context"
	"fmt"
	"net"
	"time"
)

// ioWithoutContextLeak reprend loadRoute (BEPG p. 568) : l'I/O ne reçoit aucun contexte. L'appelant
// abandonne à l'échéance ; la goroutine reste bloquée en lecture. Adaptation : TCP brut plutôt que
// http.Get, pour que les goroutines internes du transport HTTP ne brouillent pas les comptes.
func ioWithoutContextLeak() error {
	addr, stop, err := unresponsiveServer()
	if err != nil {
		return err
	}
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	res := make(chan error, 1)
	go func() { res <- loadRoute(addr) }()
	select {
	case err := <-res:
		return fmt.Errorf("io-without-context-leak : réponse inattendue (%v)", err)
	case <-ctx.Done():
		return nil // abandon attendu par le test
	}
}

func loadRoute(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	return err
}
