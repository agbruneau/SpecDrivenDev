// Package adapters regroupe les implémentations des ports : compilateur (go build -gcflags=-m),
// runner (go test -bench), système de fichiers de résultats, provenance (go version, GOOS/GOARCH,
// CPU). Un sous-paquet par adapter ; aucune logique métier (BEPG ch. 14, p. 369).
package adapters
