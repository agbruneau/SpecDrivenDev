package corpus

import (
	"net"
	"sync"
)

// held garde les connexions acceptées par unresponsiveServer : le service distant reste vivant,
// seulement muet. Sans elles, le ramasse-miettes fermerait la connexion et débloquerait le client.
var (
	heldMu sync.Mutex
	held   []net.Conn
)

// unresponsiveServer écoute sur la boucle locale, accepte une connexion et ne répond jamais.
// Outillage commun aux cas io-without-context ; ce fichier n'est attribué à aucun cas.
func unresponsiveServer() (addr string, stop func(), err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		heldMu.Lock()
		held = append(held, conn)
		heldMu.Unlock()
	}()
	return ln.Addr().String(), func() { ln.Close() }, nil
}
