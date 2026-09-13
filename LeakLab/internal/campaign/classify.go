package campaign

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/agbruneau/leaklab/internal/results"
)

var (
	deadlockLine = regexp.MustCompile(`(?m)^(fatal error|panic):.*deadlock`)
	leakWord     = regexp.MustCompile(`(?i)leak|goroutine`)
)

func normalize(output string) string { return strings.ReplaceAll(output, "\r\n", "\n") }

func hasLinePrefix(output, prefix string) bool {
	for _, l := range strings.Split(output, "\n") {
		if strings.HasPrefix(l, prefix) {
			return true
		}
	}
	return false
}

// Classify applique l'ordre de C-005 à la sortie et à la fin d'un processus (BR-001-2).
func Classify(output string, exitCode int, killed bool) results.Outcome {
	output = normalize(output)
	switch {
	case killed:
		return results.OutcomeHang
	case strings.Contains(output, "WARNING: DATA RACE"):
		return results.OutcomeRace
	case deadlockLine.MatchString(output):
		return results.OutcomeDeadlock
	case hasLinePrefix(output, "LEAKLAB-LEAK"):
		return results.OutcomeLeak
	case exitCode != 0:
		return results.OutcomeFail
	}
	return results.OutcomePass
}

// MentionsLeak applique la définition du signalement de H-002.
func MentionsLeak(output string) bool {
	for _, l := range strings.Split(normalize(output), "\n") {
		if strings.HasPrefix(l, "=== ") || strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "PASS") || strings.HasPrefix(l, "FAIL") || strings.HasPrefix(l, "ok") {
			continue
		}
		if leakWord.MatchString(l) {
			return true
		}
	}
	return false
}

// Detail rend la première ligne significative de la sortie, 200 caractères au plus.
func Detail(output string) string {
	var last string
	for _, l := range strings.Split(normalize(output), "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		for _, p := range []string{"WARNING: DATA RACE", "fatal error:", "panic:", "LEAKLAB-LEAK"} {
			if strings.HasPrefix(t, p) {
				return truncate(t)
			}
		}
		if strings.Contains(t, "_test.go:") {
			return truncate(t)
		}
		if !strings.HasPrefix(t, "=== ") && !strings.HasPrefix(t, "--- ") && t != "PASS" && t != "FAIL" {
			last = t
		}
	}
	return truncate(last)
}

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= 200 {
		return s
	}
	return string([]rune(s)[:200])
}

type process struct {
	output   string
	exitCode int
	killed   bool
	duration time.Duration
}

// runProcess lance un processus neuf (BR-001-4) et le tue au-delà de timeout (C-004).
func runProcess(ctx context.Context, timeout time.Duration, dir string, env []string, name string, args ...string) (process, error) {
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(pctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	cmd.WaitDelay = 2 * time.Second
	start := time.Now()
	err := cmd.Run()
	p := process{output: buf.String(), duration: time.Since(start)}
	if ctx.Err() != nil {
		return p, fmt.Errorf("campagne interrompue : %w", ctx.Err())
	}
	if err != nil && pctx.Err() != nil {
		p.killed = true
		return p, nil
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		p.exitCode = exitErr.ExitCode()
	default:
		return p, fmt.Errorf("lancement de %s : %w", name, err)
	}
	return p, nil
}
