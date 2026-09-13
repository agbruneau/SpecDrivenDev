//go:build unix

package gotool

import (
	"os/exec"
	"syscall"
	"time"
)

// treeCPUTracker suit le temps processeur d'un arbre de processus. Hors de Windows, il se rabat
// sur le temps propre du processus et de ses enfants attendus, que le système d'exploitation
// expose déjà dans l'état du processus.
type treeCPUTracker struct{ cmd *exec.Cmd }

// prepareTree place la commande dans son propre groupe de processus et règle l'annulation pour
// qu'elle tue le groupe entier.
//
// Révision du 2026-09-12 (A-062) : exec.CommandContext ne tue par défaut que le processus lancé.
// `go test` ayant déjà engendré le binaire de benchmark, celui-ci survivait à l'annulation, gardait
// les tubes de sortie ouverts — donc bloquait Wait jusqu'à la fin de sa benchtime — et consommait
// un cœur pendant ce temps, faussant l'attestation de quiétude de toute mesure concurrente (C-010).
func prepareTree(cmd *exec.Cmd) *treeCPUTracker {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Le pid négatif désigne le groupe. Si le groupe n'existe plus, retomber sur le seul
		// processus vaut mieux que de ne rien tuer.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return &treeCPUTracker{}
}

// attach retient la commande démarrée.
func (t *treeCPUTracker) attach(cmd *exec.Cmd) {
	if t != nil {
		t.cmd = cmd
	}
}

// total rend le temps processeur du processus une fois terminé.
func (t *treeCPUTracker) total() (time.Duration, bool) {
	if t == nil || t.cmd == nil || t.cmd.ProcessState == nil {
		return 0, false
	}
	return t.cmd.ProcessState.UserTime() + t.cmd.ProcessState.SystemTime(), true
}
