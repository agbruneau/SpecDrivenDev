// Package corpus est le corpus synthétique de TestUC001_CampagneSynthetique : un diagnostic go vet
// sur forgotten_cancel.go, un diagnostic ctxvet sur io_without_context_leak.go, rien d'autre.
package corpus

import (
	"context"
	"time"
)

func forgottenCancel() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	_ = ctx
}
