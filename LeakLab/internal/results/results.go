// Package results porte les entités persistées de LeakLab (Run, Observation, ProbeResult) et leur
// écriture exclusive sous results/ (NFR-004).
package results

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Outcome est l'issue d'une observation (C-005).
type Outcome string

const (
	OutcomePass       Outcome = "PASS"
	OutcomeFail       Outcome = "FAIL"
	OutcomeHang       Outcome = "HANG"
	OutcomeDeadlock   Outcome = "DEADLOCK"
	OutcomeRace       Outcome = "RACE"
	OutcomeLeak       Outcome = "LEAK"
	OutcomeDiagnostic Outcome = "DIAGNOSTIC"
)

// Detector nomme un détecteur (C-006).
type Detector string

const (
	DetectorBare         Detector = "BARE"
	DetectorRace         Detector = "RACE"
	DetectorSynctest     Detector = "SYNCTEST"
	DetectorNumGoroutine Detector = "NUMGOROUTINE"
	DetectorLeakProfile  Detector = "LEAKPROFILE"
	DetectorProgram      Detector = "PROGRAM"
	DetectorVet          Detector = "VET"
	DetectorCtxvet       Detector = "CTXVET"
)

// Detectors rend tous les détecteurs, dynamiques d'abord, dans l'ordre de la matrice.
func Detectors() []Detector {
	return []Detector{DetectorBare, DetectorRace, DetectorSynctest, DetectorNumGoroutine, DetectorLeakProfile, DetectorProgram, DetectorVet, DetectorCtxvet}
}

// Provenance identifie la machine et la toolchain d'une campagne (NFR-001).
type Provenance struct {
	GoVersion string `json:"goVersion"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPU       string `json:"cpu"`
	NumCPU    int    `json:"numCPU"`
}

// Observation est une exécution d'un cas par un détecteur.
type Observation struct {
	CaseID          string   `json:"caseId"`
	Detector        Detector `json:"detector"`
	Rep             int      `json:"rep"`
	Outcome         Outcome  `json:"outcome"`
	MentionsLeak    bool     `json:"mentionsLeak"`
	MentionsWitness bool     `json:"mentionsWitness"`
	DurationMs      int64    `json:"durationMs"`
	Detail          string   `json:"detail,omitempty"`
}

// ProbeResult est une mesure d'un bras de sonde (C-007).
type ProbeResult struct {
	Probe          string  `json:"probe"`
	Arm            string  `json:"arm"`
	Rep            int     `json:"rep"`
	WallNs         int64   `json:"wallNs"`
	BytesPerOp     float64 `json:"bytesPerOp"`
	GoroutineDelta int     `json:"goroutineDelta"`
}

// Run est une campagne complète (UC-001).
type Run struct {
	ID              string            `json:"id"`
	Provenance      Provenance        `json:"provenance"`
	Reps            int               `json:"reps"`
	TimeoutMs       int64             `json:"timeoutMs"`
	CriteriaDigests map[string]string `json:"criteriaDigests"`
	Observations    []Observation     `json:"observations"`
	Probes          []ProbeResult     `json:"probes"`
	StartedAt       time.Time         `json:"startedAt"`
	FinishedAt      time.Time         `json:"finishedAt"`
}

// WriteExclusive crée path et y écrit data ; un fichier existant n'est jamais écrasé (NFR-004).
func WriteExclusive(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("création de %s : %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("écriture exclusive de %s : %w", path, err)
	}
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(path) // un fichier partiel n'est pas un résultat
		return fmt.Errorf("écriture de %s : %w", path, werr)
	}
	return nil
}

// WriteJSONExclusive encode v en JSON indenté et l'écrit par WriteExclusive.
func WriteJSONExclusive(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage de %s : %w", path, err)
	}
	return WriteExclusive(path, append(data, '\n'))
}

// LoadRun lit une campagne.
func LoadRun(path string) (Run, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Run{}, fmt.Errorf("lecture de la campagne : %w", err)
	}
	var r Run
	if err := json.Unmarshal(data, &r); err != nil {
		return Run{}, fmt.Errorf("décodage de %s : %w", path, err)
	}
	return r, nil
}

// NextRunID rend R-<date>-<n>, n étant un de plus que le plus grand numéro du jour sous dir.
func NextRunID(dir string, day time.Time) (string, error) {
	prefix := "R-" + day.UTC().Format("2006-01-02") + "-"
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("lecture de %s : %w", dir, err)
	}
	next := 1
	for _, e := range entries {
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(e.Name(), prefix), ".json"))
		if err == nil && strings.HasPrefix(e.Name(), prefix) && n >= next {
			next = n + 1
		}
	}
	return prefix + strconv.Itoa(next), nil
}
