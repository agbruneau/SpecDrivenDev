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
	"time"
	"unicode/utf8"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// Result est le résultat d'une commande externe.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	// TreeCPU est le temps processeur cumulé de l'arbre de processus de la commande (C-010).
	TreeCPU         time.Duration
	TreeCPUMeasured bool
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

// ExecRunner exécute réellement la commande. Le temps processeur de tout l'arbre de processus est
// relevé au passage : `go test` compile, lie puis exécute dans des processus enfants, et C-010 doit
// retrancher ce travail de la fraction d'occupation, faute de quoi la garde de quiétude refuserait
// les campagnes saines.
func ExecRunner(ctx context.Context, dir string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// A-062 : les sorties sont collectées dans des tampons, donc par des tubes. Un descendant qui
	// survit à l'annulation les garde ouverts et Wait attend leur fermeture, c'est-à-dire la fin
	// du benchmark orphelin. WaitDelay borne cette attente ; prepareTree termine l'arbre.
	cmd.WaitDelay = 2 * time.Second
	tracker := prepareTree(cmd)
	err := cmd.Start()
	if err == nil {
		tracker.attach(cmd)
		err = cmd.Wait()
	}
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	result.TreeCPU, result.TreeCPUMeasured = tracker.total()
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
	// quietude relève les temps processeur de la machine autour de chaque mesure (C-010). Elle est
	// injectée depuis la racine de composition : un adaptateur n'en importe pas un autre. Nulle,
	// la mesure se déclare non faite et H-013 rend non concluant.
	quietude QuietudeSampler
	cpus     int
}

// QuietudeSampler rend le temps processeur cumulé de la machine passé hors de la boucle
// d'inactivité, et si la plateforme sait le produire (C-010).
type QuietudeSampler func() (busy time.Duration, ok bool)

// WithQuietude branche la sonde de quiétude sur la chaîne d'outils.
func (t *Toolchain) WithQuietude(sample QuietudeSampler, cpus int) *Toolchain {
	t.quietude = sample
	t.cpus = cpus
	return t
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
	// A-141 : un `go build` tué par l'annulation sort avec un code non nul. Sans cette garde il
	// devenait un ports.CompileError, la cellule était classée COMPILE_ERROR et consignée comme
	// telle dans le fichier d'échappement, alors que le compilateur n'a rien dit de ce sujet.
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("analyse d'échappement de %s interrompue : %w", subjectID, err)
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
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("compilation de %s interrompue : %w", subjectID, err)
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

	// C-010 : la fenêtre de mesure est encadrée par deux relevés des temps processeur de la
	// machine. Le travail de la campagne elle-même en est retranché ; ce qui reste est l'occupation
	// des cœurs par tout ce qui n'est pas le sujet.
	busyBefore, quietudeOK := t.sampleQuietude()
	startedAt := time.Now()
	result, err := t.run(ctx, matrixDir, t.goBin, args...)
	elapsed := time.Since(startedAt)
	busyAfter, afterOK := t.sampleQuietude()
	// A-261 : une interruption n'est pas un résultat de mesure. Sans cette garde, le processus tué
	// par le signal rendait une Measurement FAILED avec une erreur Go nulle : la boucle de mesure
	// ne voyait pas l'annulation, écrivait le sujet en échec, puis échouait en chaîne sur tous les
	// suivants et clôturait la campagne COMPLETED. Le flux A4 devenait inatteignable.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return models.Measurement{}, fmt.Errorf("mesure de %s interrompue : %w", subjectID, ctxErr)
	}
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
	if quietudeOK && afterOK && result.TreeCPUMeasured {
		m.QuietudeOccupancy, m.QuietudeMeasured = occupancyOf(
			busyBefore, busyAfter, result.TreeCPU, elapsed, t.cpus)
	}
	return m, nil
}

// sampleQuietude relève les temps processeur de la machine, si la sonde est branchée.
func (t *Toolchain) sampleQuietude() (time.Duration, bool) {
	if t.quietude == nil || t.cpus <= 0 {
		return 0, false
	}
	return t.quietude()
}

// occupancyOf rend la fraction d'occupation des cœurs non mesurés (C-010). Elle est bornée à
// [0, 1] : un dépassement ne peut venir que d'un arrondi ou d'un compteur qui recule.
func occupancyOf(before, after, own, elapsed time.Duration, cpus int) (float64, bool) {
	if elapsed <= 0 || cpus <= 0 {
		return 0, false
	}
	busy := after - before
	if busy < 0 {
		return 0, false
	}
	other := busy - own
	if other < 0 {
		other = 0
	}
	fraction := float64(other) / float64(time.Duration(cpus)*elapsed)
	switch {
	case fraction < 0:
		return 0, true
	case fraction > 1:
		return 1, true
	default:
		return fraction, true
	}
}

// failed construit une Measurement au statut FAILED (UC-003, A3).
func failed(subjectID, reason string) models.Measurement {
	return models.Measurement{SubjectID: subjectID, Status: models.MeasurementFailed, FailureReason: truncate(reason, 2000)}
}

// truncate borne la longueur d'un message consigné dans un fichier de résultats.
// A-063 : couper à l'octet laissait une séquence UTF-8 incomplète dans FailureReason, donc un
// octet invalide dans un fichier de résultats JSON. La coupe recule jusqu'à la dernière frontière
// de rune.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "… (tronqué)"
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
