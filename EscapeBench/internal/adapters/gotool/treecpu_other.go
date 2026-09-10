//go:build !windows

package gotool

import (
	"os/exec"
	"time"
)

// treeCPUTracker suit le temps processeur d'un arbre de processus. Hors de Windows, il se rabat
// sur le temps propre du processus et de ses enfants attendus, que le système d'exploitation
// expose déjà dans l'état du processus.
type treeCPUTracker struct{ cmd *exec.Cmd }

// startTreeCPU retient la commande ; rien à préparer.
func startTreeCPU(cmd *exec.Cmd) *treeCPUTracker { return &treeCPUTracker{cmd: cmd} }

// total rend le temps processeur du processus une fois terminé.
func (t *treeCPUTracker) total() (time.Duration, bool) {
	if t == nil || t.cmd == nil || t.cmd.ProcessState == nil {
		return 0, false
	}
	return t.cmd.ProcessState.UserTime() + t.cmd.ProcessState.SystemTime(), true
}
