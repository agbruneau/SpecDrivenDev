package verdict

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// Markdown rend le tableau des verdicts, la matrice de détectabilité et les médianes des sondes
// (FR-006, BR-002-3).
func Markdown(run results.Run, vs []Verdict) string {
	var b strings.Builder
	p := run.Provenance
	fmt.Fprintf(&b, "# Verdicts — %s\n\n", run.ID)
	fmt.Fprintf(&b, "Produit par `leaklab verdict` (UC-002) ; ne pas éditer. Campagne du %s au %s, %s %s/%s, %s, %d cœurs logiques, %d répétitions, délai %d ms.\n\n",
		run.StartedAt.Format(time.RFC3339), run.FinishedAt.Format(time.RFC3339), p.GoVersion, p.GOOS, p.GOARCH, p.CPU, p.NumCPU, run.Reps, run.TimeoutMs)

	b.WriteString("## Verdicts\n\n| Hypothèse | Verdict | Rationale |\n|---|---|---|\n")
	for _, v := range vs {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", v.HypothesisID, v.Outcome, strings.ReplaceAll(v.Rationale, "|", "/"))
	}

	e := newEval(run)
	b.WriteString("\n## Matrice de détectabilité\n\nIssue majoritaire par cellule ; « ~ » : répétitions discordantes ; « MIXTE » : aucune majorité.\n\n| Cas | faulty | ")
	dets := results.Detectors()
	for _, d := range dets {
		fmt.Fprintf(&b, "%s | ", d)
	}
	b.WriteString("\n|---|---|" + strings.Repeat("---|", len(dets)) + "\n")
	for _, c := range corpus.Catalog() {
		faulty := "non"
		if c.Faulty {
			faulty = "oui"
		}
		fmt.Fprintf(&b, "| %s | %s | ", c.ID, faulty)
		for _, d := range dets {
			fmt.Fprintf(&b, "%s | ", majorityLabel(e.cell(c.ID, d)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Sondes (médianes)\n\n| Sonde | Bras | Temps réel (ms) | Octets par opération | Écart de goroutines |\n|---|---|---|---|---|\n")
	seen := map[string]bool{}
	for _, pr := range run.Probes {
		k := pr.Probe + "/" + pr.Arm
		if seen[k] {
			continue
		}
		seen[k] = true
		fmt.Fprintf(&b, "| %s | %s | %.3f | %.2f | %.0f |\n", pr.Probe, pr.Arm,
			e.median(pr.Probe, pr.Arm, wall)/1e6, e.median(pr.Probe, pr.Arm, bytes), e.median(pr.Probe, pr.Arm, delta))
	}
	return b.String()
}

// Write écrit results/verdicts/<runId>-<horodatage>.json et .md en écriture exclusive.
func Write(root string, run results.Run, vs []Verdict, now time.Time) (jsonPath, mdPath string, err error) {
	base := filepath.Join(root, "results", "verdicts", run.ID+"-"+now.UTC().Format("20060102T150405Z"))
	jsonPath, mdPath = base+".json", base+".md"
	if err := results.WriteJSONExclusive(jsonPath, vs); err != nil {
		return "", "", err
	}
	if err := results.WriteExclusive(mdPath, []byte(Markdown(run, vs))); err != nil {
		return "", "", err
	}
	return jsonPath, mdPath, nil
}
