//go:build !windows && !unix

package gotool

import (
	"os/exec"
	"time"
)

// Sur une plateforme qui n'expose ni Job Object ni groupe de processus POSIX, l'arbre n'est ni
// mesuré ni terminé en bloc. La mesure de quiétude se déclare alors non faite et H-013 rend non
// concluant, ce qui vaut mieux qu'un chiffre faux (C-010).
type treeCPUTracker struct{}

// prepareTree ne prépare rien ; cmd.WaitDelay, posé par l'appelant, reste la seule garde.
func prepareTree(*exec.Cmd) *treeCPUTracker { return nil }

// attach ne retient rien.
func (t *treeCPUTracker) attach(*exec.Cmd) {}

// total déclare la mesure non faite.
func (t *treeCPUTracker) total() (time.Duration, bool) { return 0, false }
