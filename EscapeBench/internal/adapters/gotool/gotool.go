// Package gotool encapsule les appels à la chaîne d'outils Go : compilation avec diagnostic
// d'échappement (C-003), exécution d'un benchmark par sujet dans un processus distinct
// (BR-003-4) et exécution des tests d'un cas d'utilisation (FR-007).
package gotool

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// Result est le résultat d'une commande externe.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Combined rend la sortie standard et la sortie d'erreur concaténées.
func (r Result) Combined() string {
	if r.Stdout == "" {
		return r.Stderr
	}
	if r.Stderr == "" {
		return r.Stdout
	}
	return r.Stdout + "\n" + r.Stderr
}

// CommandRunner exécute une commande dans un répertoire donné. Il est injecté pour que les tests
// n'aient jamais besoin de la chaîne d'outils réelle.
type CommandRunner func(ctx context.Context, dir string, name string, args ...string) (Result, error)

// ExecRunner exécute réellement la commande.
func ExecRunner(ctx context.Context, dir string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return result, nil
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	default:
		return result, fmt.Errorf("exécution de %s : %w", name, err)
	}
}

// Toolchain met en œuvre les ports Compiler, BenchmarkRunner et TestReporter.
type Toolchain struct {
	run     CommandRunner
	goBin   string
	rootDir string
}

// New construit une Toolchain. rootDir est la racine du module principal, utilisée par le
// rapporteur de tests ; il peut être vide pour les usages qui n'en ont pas besoin.
func New(run CommandRunner, rootDir string) *Toolchain {
	if run == nil {
		run = ExecRunner
	}
	return &Toolchain{run: run, goBin: "go", rootDir: rootDir}
}

// packagePath rend le chemin de paquet relatif d'un sujet dans le module de la Matrix.
func packagePath(subjectID string) string {
	return "./subjects/" + models.SubjectDir(subjectID)
}

// EscapeAnalysis compile le paquet du sujet avec `-gcflags=-m` (C-003) et rend les lignes brutes.
func (t *Toolchain) EscapeAnalysis(ctx context.Context, matrixDir, subjectID string) ([]string, error) {
	result, err := t.run(ctx, matrixDir, t.goBin, "build", "-gcflags=-m", packagePath(subjectID))
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, &ports.CompileError{SubjectID: subjectID, Output: strings.TrimSpace(result.Combined())}
	}
	return splitLines(result.Combined()), nil
}

// Build compile le paquet du sujet et rend un *ports.CompileError s'il ne compile pas.
func (t *Toolchain) Build(ctx context.Context, matrixDir, subjectID string) error {
	result, err := t.run(ctx, matrixDir, t.goBin, "build", packagePath(subjectID))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &ports.CompileError{SubjectID: subjectID, Output: strings.TrimSpace(result.Combined())}
	}
	return nil
}

// splitLines découpe une sortie en lignes non vides.
func splitLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// benchLineRe capture une ligne de résultat de benchmark avec -benchmem.
var benchLineRe = regexp.MustCompile(`^Benchmark\S*\s+(\d+)\s+([0-9.eE+-]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

// Sample est une répétition unique d'un benchmark.
type Sample struct {
	NsPerOp     float64
	BytesPerOp  int64
	AllocsPerOp int64
}

// ParseBenchmarkOutput extrait les répétitions d'une sortie `go test -bench -benchmem`.
func ParseBenchmarkOutput(out string) ([]Sample, error) {
	var samples []Sample
	for _, line := range strings.Split(out, "\n") {
		match := benchLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		ns, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			return nil, fmt.Errorf("ns/op illisible dans %q : %w", line, err)
		}
		bytes, err := strconv.ParseInt(match[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("B/op illisible dans %q : %w", line, err)
		}
		allocs, err := strconv.ParseInt(match[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("allocs/op illisible dans %q : %w", line, err)
		}
		samples = append(samples, Sample{NsPerOp: ns, BytesPerOp: bytes, AllocsPerOp: allocs})
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("aucune ligne de benchmark dans la sortie")
	}
	return samples, nil
}

// Run mesure un sujet dans un processus `go test` distinct (BR-003-4) selon les drapeaux de C-003.
func (t *Toolchain) Run(ctx context.Context, matrixDir, subjectID string, opts ports.RunOptions) (models.Measurement, error) {
	args := []string{
		"test",
		"-run", "^$",
		"-bench", "^BenchmarkSubject$",
		"-benchmem",
		"-count", strconv.Itoa(opts.Count),
		"-benchtime", opts.BenchTime,
	}
	if opts.CPU > 0 {
		args = append(args, "-cpu", strconv.Itoa(opts.CPU))
	}
	args = append(args, packagePath(subjectID))

	result, err := t.run(ctx, matrixDir, t.goBin, args...)
	if err != nil {
		return failed(subjectID, err.Error()), nil
	}
	if result.ExitCode != 0 {
		return failed(subjectID, strings.TrimSpace(result.Combined())), nil
	}
	samples, err := ParseBenchmarkOutput(result.Combined())
	if err != nil {
		return failed(subjectID, err.Error()), nil
	}
	if len(samples) != opts.Count {
		return failed(subjectID, fmt.Sprintf("%d répétitions mesurées, %d demandées", len(samples), opts.Count)), nil
	}
	m := models.Measurement{SubjectID: subjectID, Status: models.MeasurementComplete}
	for _, s := range samples {
		m.NsPerOp = append(m.NsPerOp, s.NsPerOp)
		m.BytesPerOp = append(m.BytesPerOp, s.BytesPerOp)
		m.AllocsPerOp = append(m.AllocsPerOp, s.AllocsPerOp)
	}
	return m, nil
}

// failed construit une Measurement au statut FAILED (UC-003, A3).
func failed(subjectID, reason string) models.Measurement {
	return models.Measurement{SubjectID: subjectID, Status: models.MeasurementFailed, FailureReason: truncate(reason, 2000)}
}

// truncate borne la longueur d'un message consigné dans un fichier de résultats.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "… (tronqué)"
}

// runLineRe capture les tests de premier niveau exécutés.
var runLineRe = regexp.MustCompile(`(?m)^=== RUN\s+(Test\w+)$`)

// RunUseCaseTests exécute les tests nommés d'après un cas d'utilisation (FR-007, étape 7).
func (t *Toolchain) RunUseCaseTests(ctx context.Context, useCaseID string) (ports.TestOutcome, error) {
	pattern := "^Test" + strings.ReplaceAll(useCaseID, "-", "") + "_"
	return t.runTests(ctx, "-run", pattern, "-v", "-count", "1", "./...")
}

// RunAll exécute la suite complète (colonne Regression du tableau de bord).
func (t *Toolchain) RunAll(ctx context.Context) (ports.TestOutcome, error) {
	return t.runTests(ctx, "-count", "1", "./...")
}

// runTests exécute `go test` à la racine du module principal.
func (t *Toolchain) runTests(ctx context.Context, args ...string) (ports.TestOutcome, error) {
	result, err := t.run(ctx, t.rootDir, t.goBin, append([]string{"test"}, args...)...)
	if err != nil {
		return ports.TestOutcome{}, err
	}
	out := result.Combined()
	return ports.TestOutcome{
		Selected: len(runLineRe.FindAllStringSubmatch(out, -1)),
		Passed:   result.ExitCode == 0,
		Output:   out,
	}, nil
}
