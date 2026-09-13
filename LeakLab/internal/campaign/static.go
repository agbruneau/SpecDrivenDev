package campaign

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/agbruneau/leaklab/internal/ctxvet"
	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

var vetDiagnostic = regexp.MustCompile(`^(?:vet: )?(.+?\.go):\d+:\d+: (.+)$`)

// finding est un diagnostic statique réduit à son fichier et à son message.
type finding struct{ file, message string }

// parseVet extrait les diagnostics de la sortie de go vet.
func parseVet(output string) []finding {
	var fs []finding
	for _, l := range strings.Split(normalize(output), "\n") {
		if m := vetDiagnostic.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			fs = append(fs, finding{file: filepath.Base(m[1]), message: m[2]})
		}
	}
	return fs
}

// attribute produit une Observation statique par cas (BR-001-3) : DIAGNOSTIC si un diagnostic vise
// le fichier du cas. Un diagnostic sur un fichier sans cas n'est attribué à personne.
func attribute(det results.Detector, fs []finding, d time.Duration) []results.Observation {
	var obs []results.Observation
	for _, c := range corpus.Catalog() {
		o := results.Observation{CaseID: c.ID, Detector: det, Rep: 1, Outcome: results.OutcomePass, DurationMs: d.Milliseconds()}
		for _, f := range fs {
			if f.file == c.File() {
				o.Outcome, o.Detail = results.OutcomeDiagnostic, truncate(f.message)
				break
			}
		}
		obs = append(obs, o)
	}
	return obs
}

// staticObservations passe le paquet du corpus à go vet puis à ctxvet (FR-003).
func staticObservations(ctx context.Context, labDir string) ([]results.Observation, error) {
	p, err := runProcess(ctx, 5*time.Minute, labDir, nil, "go", "vet", "./corpus")
	if err != nil {
		return nil, err
	}
	fs := parseVet(p.output)
	if p.killed || (p.exitCode != 0 && len(fs) == 0) {
		return nil, fmt.Errorf("go vet a échoué sans diagnostic :\n%s", p.output)
	}
	obs := attribute(results.DetectorVet, fs, p.duration)

	start := time.Now()
	diags, err := ctxvet.Analyze(filepath.Join(labDir, "corpus"))
	if err != nil {
		return nil, err
	}
	var cfs []finding
	for _, d := range diags {
		cfs = append(cfs, finding{file: filepath.Base(d.File), message: d.Call + " ignore le contexte ; utiliser " + d.Replacement})
	}
	return append(obs, attribute(results.DetectorCtxvet, cfs, time.Since(start))...), nil
}
