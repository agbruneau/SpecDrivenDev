// Package campaign exécute une campagne LeakLab (UC-001) : oracle, compilation des binaires de
// mesure, matrice des détecteurs dynamiques, détecteurs statiques, sondes, écriture unique.
package campaign

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/internal/spec"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// MinReps est le plancher de NFR-002.
const MinReps = 5

// Config paramètre une campagne.
type Config struct {
	Root    string        // répertoire LeakLab (docs/, lab/, results/)
	Reps    int           // répétitions par cellule et par bras
	Timeout time.Duration // délai réel par observation dynamique (C-004)
	Log     io.Writer     // progression ; io.Discard accepté
}

// goTestTimeout est le délai que go test transmet par défaut à un binaire de test ; lancé
// directement, le binaire n'en aurait aucun (C-004, précision du 2026-09-13).
const goTestTimeout = "-test.timeout=10m0s"

// binaries sont les exécutables de mesure compilés à l'étape 4.
type binaries struct{ driver, driverRace, program string }

// command rend l'exécutable et les arguments d'un détecteur dynamique (C-006).
func (b binaries) command(det results.Detector, caseID string) (string, []string) {
	test := func(bin, name string) (string, []string) {
		return bin, []string{"-test.run=^" + name + "$", "-test.v", "-test.count=1", goTestTimeout}
	}
	switch det {
	case results.DetectorBare:
		return test(b.driver, "TestBare")
	case results.DetectorRace:
		return test(b.driverRace, "TestBare")
	case results.DetectorSynctest:
		return test(b.driver, "TestSynctest")
	case results.DetectorNumGoroutine:
		return test(b.driver, "TestNumGoroutine")
	case results.DetectorLeakProfile:
		return test(b.driver, "TestLeakProfile")
	}
	return b.program, []string{caseID}
}

// Run exécute la campagne et rend le chemin du fichier écrit. En cas d'erreur, rien n'est écrit.
func Run(ctx context.Context, cfg Config) (results.Run, string, error) {
	if cfg.Reps < MinReps {
		return results.Run{}, "", fmt.Errorf("UC-001 A4 : %d répétitions demandées, NFR-002 en exige au moins %d", cfg.Reps, MinReps)
	}
	logf := func(format string, args ...any) { fmt.Fprintf(cfg.Log, format+"\n", args...) }
	labDir := filepath.Join(cfg.Root, "lab")

	// Étape 2 : empreintes des critères et conformité du corpus (A2).
	doc, err := os.ReadFile(filepath.Join(cfg.Root, "docs", "requirements.md"))
	if err != nil {
		return results.Run{}, "", fmt.Errorf("lecture de la spécification : %w", err)
	}
	hs, err := spec.Hypotheses(doc)
	if err != nil {
		return results.Run{}, "", err
	}
	rows, err := spec.Corpus(doc)
	if err != nil {
		return results.Run{}, "", err
	}
	if diffs := spec.CompareCorpus(rows, corpus.Catalog()); len(diffs) > 0 {
		return results.Run{}, "", fmt.Errorf("UC-001 A2, catalogue et spécification divergent :\n%s", strings.Join(diffs, "\n"))
	}

	// Étape 3 : oracle (A1).
	logf("oracle du corpus…")
	p, err := runProcess(ctx, 5*time.Minute, labDir, nil, "go", "test", "-count=1", "./corpus")
	if err != nil {
		return results.Run{}, "", err
	}
	if p.killed || p.exitCode != 0 {
		return results.Run{}, "", fmt.Errorf("UC-001 A1, l'oracle contredit la vérité terrain :\n%s", p.output)
	}

	started := time.Now().UTC()
	prov, err := provenance(ctx, labDir)
	if err != nil {
		return results.Run{}, "", err
	}

	// Étape 4 : binaires (A3).
	binDir, err := os.MkdirTemp("", "leaklab-bin-")
	if err != nil {
		return results.Run{}, "", err
	}
	defer os.RemoveAll(binDir)
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	bins := binaries{
		driver:     filepath.Join(binDir, "driver"+exe),
		driverRace: filepath.Join(binDir, "driver_race"+exe),
		program:    filepath.Join(binDir, "scenario"+exe),
	}
	for _, args := range [][]string{
		{"test", "-c", "-o", bins.driver, "./driver"},
		{"test", "-race", "-c", "-o", bins.driverRace, "./driver"},
		{"build", "-o", bins.program, "./cmd/scenario"},
	} {
		logf("go %s", strings.Join(args, " "))
		p, err := runProcess(ctx, 10*time.Minute, labDir, nil, "go", args...)
		if err != nil {
			return results.Run{}, "", err
		}
		if p.killed || p.exitCode != 0 {
			return results.Run{}, "", fmt.Errorf("UC-001 A3, compilation refusée (go %s) :\n%s", strings.Join(args, " "), p.output)
		}
	}

	// Étape 5 : matrice dynamique, répétitions en boucle externe pour étaler la dérive de la machine.
	var obs []results.Observation
	dynamic := []results.Detector{results.DetectorBare, results.DetectorRace, results.DetectorSynctest, results.DetectorNumGoroutine, results.DetectorLeakProfile, results.DetectorProgram}
	for rep := 1; rep <= cfg.Reps; rep++ {
		for _, c := range corpus.Catalog() {
			for _, det := range dynamic {
				name, args := bins.command(det, c.ID)
				p, err := runProcess(ctx, cfg.Timeout, labDir, []string{"LEAKLAB_CASE=" + c.ID}, name, args...)
				if err != nil {
					return results.Run{}, "", err
				}
				obs = append(obs, results.Observation{
					CaseID: c.ID, Detector: det, Rep: rep,
					Outcome:         Classify(p.output, p.exitCode, p.killed),
					MentionsLeak:    MentionsLeak(p.output),
					MentionsWitness: strings.Contains(p.output, "LEAKLAB-WITNESS"),
					DurationMs:      p.duration.Milliseconds(),
					Detail:          Detail(p.output),
				})
			}
		}
		logf("répétition %d/%d : %d observations", rep, cfg.Reps, len(obs))
	}

	// Étape 6 : détecteurs statiques.
	static, err := staticObservations(ctx, labDir)
	if err != nil {
		return results.Run{}, "", err
	}
	obs = append(obs, static...)

	// Étape 7 : sondes.
	var probes []results.ProbeResult
	for rep := 1; rep <= cfg.Reps; rep++ {
		for _, a := range probeArms() {
			env := []string{"LEAKLAB_PROBE=" + a.probe, "LEAKLAB_ARM=" + a.arm}
			p, err := runProcess(ctx, time.Minute, labDir, env, bins.driver, "-test.run=^"+a.test+"$", "-test.count=1", goTestTimeout)
			if err != nil {
				return results.Run{}, "", err
			}
			if p.killed || p.exitCode != 0 {
				return results.Run{}, "", fmt.Errorf("sonde %s/%s en échec :\n%s", a.probe, a.arm, p.output)
			}
			r, err := parseMetrics(a, rep, p.output)
			if err != nil {
				return results.Run{}, "", err
			}
			probes = append(probes, r)
		}
		logf("sondes, répétition %d/%d", rep, cfg.Reps)
	}

	// Étape 8 : écriture unique (BR-001-5).
	runsDir := filepath.Join(cfg.Root, "results", "runs")
	id, err := results.NextRunID(runsDir, started)
	if err != nil {
		return results.Run{}, "", err
	}
	run := results.Run{
		ID: id, Provenance: prov, Reps: cfg.Reps, TimeoutMs: cfg.Timeout.Milliseconds(),
		CriteriaDigests: spec.Digests(hs), Observations: obs, Probes: probes,
		StartedAt: started, FinishedAt: time.Now().UTC(),
	}
	path := filepath.Join(runsDir, id+".json")
	if err := results.WriteJSONExclusive(path, run); err != nil {
		return results.Run{}, "", err
	}
	return run, path, nil
}

// provenance relève la toolchain qui compile les binaires de mesure et la machine (NFR-001).
func provenance(ctx context.Context, labDir string) (results.Provenance, error) {
	p, err := runProcess(ctx, time.Minute, labDir, nil, "go", "env", "GOVERSION", "GOOS", "GOARCH")
	if err != nil {
		return results.Provenance{}, err
	}
	f := strings.Fields(p.output)
	if p.exitCode != 0 || len(f) != 3 {
		return results.Provenance{}, fmt.Errorf("go env illisible : %q", p.output)
	}
	return results.Provenance{GoVersion: f[0], GOOS: f[1], GOARCH: f[2], CPU: cpuName(), NumCPU: runtime.NumCPU()}, nil
}

// cpuName rend l'identifiant du processeur que le système expose sans dépendance, ou "".
func cpuName() string {
	if name := os.Getenv("PROCESSOR_IDENTIFIER"); name != "" {
		return name
	}
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if k, v, ok := strings.Cut(s.Text(), ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
