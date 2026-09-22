//go:build !windows && !linux

package system

import "context"

// detectHost ne relève rien sur les autres systèmes : les champs de D-60 restent vides.
func detectHost(context.Context) Host { return Host{} }
