// Package verdict produit les verdicts d'une campagne LeakLab (UC-002) : cellules majoritaires,
// un évaluateur par critère gelé, matrice de détectabilité.
package verdict

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// Outcome est l'issue d'une hypothèse.
type Outcome string

const (
	Confirmed    Outcome = "CONFIRMED"
	Refuted      Outcome = "REFUTED"
	Inconclusive Outcome = "INCONCLUSIVE"
)

// Verdict est le jugement d'une hypothèse sur une campagne.
type Verdict struct {
	HypothesisID string   `json:"hypothesisId"`
	RunID        string   `json:"runId"`
	Outcome      Outcome  `json:"outcome"`
	Rationale    string   `json:"rationale"`
	Evidence     []string `json:"evidence"`
}

// Cell regroupe les observations d'un couple cas × détecteur.
type Cell struct {
	Obs []results.Observation
}

func (c Cell) count(match func(results.Outcome) bool) int {
	n := 0
	for _, o := range c.Obs {
		if match(o.Outcome) {
			n++
		}
	}
	return n
}

// Diagnosed : plus de la moitié des répétitions ont l'issue o.
func (c Cell) Diagnosed(o results.Outcome) bool {
	return 2*c.count(func(x results.Outcome) bool { return x == o }) > len(c.Obs)
}

// Detected : plus de la moitié des répétitions ont une issue autre que PASS.
func (c Cell) Detected() bool {
	return 2*c.count(func(x results.Outcome) bool { return x != results.OutcomePass }) > len(c.Obs)
}

// Unstable : les répétitions n'ont pas toutes la même issue.
func (c Cell) Unstable() bool {
	return len(c.Obs) > 0 && c.count(func(x results.Outcome) bool { return x == c.Obs[0].Outcome }) != len(c.Obs)
}

// Majority rend l'issue majoritaire, ou "" s'il n'y en a pas.
func (c Cell) Majority() results.Outcome {
	for _, o := range c.Obs {
		if c.Diagnosed(o.Outcome) {
			return o.Outcome
		}
	}
	return ""
}

// eval porte les données d'une évaluation et note ce qui manque (A2).
type eval struct {
	cells    map[string]*Cell
	probes   map[string][]results.ProbeResult
	cases    []corpus.Case
	missing  []string
	evidence []string
}

func newEval(run results.Run) *eval {
	e := &eval{cells: map[string]*Cell{}, probes: map[string][]results.ProbeResult{}, cases: corpus.Catalog()}
	for _, o := range run.Observations {
		k := o.CaseID + "/" + string(o.Detector)
		if e.cells[k] == nil {
			e.cells[k] = &Cell{}
		}
		e.cells[k].Obs = append(e.cells[k].Obs, o)
	}
	for _, p := range run.Probes {
		k := p.Probe + "/" + p.Arm
		e.probes[k] = append(e.probes[k], p)
	}
	return e
}

func (e *eval) cell(caseID string, det results.Detector) Cell {
	k := caseID + "/" + string(det)
	e.evidence = append(e.evidence, k)
	c, ok := e.cells[k]
	if !ok || len(c.Obs) == 0 {
		e.missing = append(e.missing, k)
		return Cell{}
	}
	return *c
}

// median rend la médiane d'une grandeur d'un bras de sonde.
func (e *eval) median(probe, arm string, value func(results.ProbeResult) float64) float64 {
	k := probe + "/" + arm
	e.evidence = append(e.evidence, k)
	ps := e.probes[k]
	if len(ps) == 0 {
		e.missing = append(e.missing, k)
		return 0
	}
	vs := make([]float64, len(ps))
	for i, p := range ps {
		vs[i] = value(p)
	}
	sort.Float64s(vs)
	n := len(vs)
	if n%2 == 1 {
		return vs[n/2]
	}
	return (vs[n/2-1] + vs[n/2]) / 2
}

func (e *eval) set(keep func(corpus.Case) bool) []corpus.Case {
	var out []corpus.Case
	for _, c := range e.cases {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

// Ensembles nommés de docs/requirements.md.
func setL(c corpus.Case) bool { return c.Faulty && c.Leak && !c.Blocks }
func setS(c corpus.Case) bool { return !c.Leak && !c.Blocks && c.AntiPattern != corpus.Witness }
func setR(c corpus.Case) bool { return c.Race }
func setN(c corpus.Case) bool { return c.Faulty && !c.Race }

func ids(cs []corpus.Case) string {
	if len(cs) == 0 {
		return "aucun"
	}
	s := make([]string, len(cs))
	for i, c := range cs {
		s[i] = c.ID
	}
	return strings.Join(s, ", ")
}

type evaluator func(e *eval) (Outcome, string)

// Evaluate juge chaque hypothèse de current sur run (UC-002). Une empreinte changée depuis la
// campagne est un refus (A1) ; une hypothèse absente de la campagne est non concluante.
func Evaluate(run results.Run, current map[string]string) ([]Verdict, error) {
	var changed []string
	for id, digest := range current {
		if old, ok := run.CriteriaDigests[id]; ok && old != digest {
			changed = append(changed, id)
		}
	}
	if len(changed) > 0 {
		slices.Sort(changed)
		return nil, fmt.Errorf("UC-002 A1, critère modifié depuis la campagne %s : %s ; créer une hypothèse successeur", run.ID, strings.Join(changed, ", "))
	}
	idsSorted := make([]string, 0, len(current))
	for id := range current {
		idsSorted = append(idsSorted, id)
	}
	slices.Sort(idsSorted)
	var vs []Verdict
	for _, id := range idsSorted {
		v := Verdict{HypothesisID: id, RunID: run.ID}
		ev, known := evaluators[id]
		_, frozen := run.CriteriaDigests[id]
		switch {
		case !frozen:
			v.Outcome, v.Rationale = Inconclusive, "critère absent de la campagne, donc écrit après elle"
		case !known:
			v.Outcome, v.Rationale = Inconclusive, "aucun évaluateur pour ce critère (BR-002-1)"
		default:
			e := newEval(run)
			v.Outcome, v.Rationale = ev(e)
			if len(e.missing) > 0 {
				v.Outcome, v.Rationale = Inconclusive, "données absentes de la campagne (A2) : "+strings.Join(e.missing, ", ")
			}
			seen := map[string]bool{}
			for _, k := range e.evidence {
				if !seen[k] {
					seen[k] = true
					v.Evidence = append(v.Evidence, k)
				}
			}
		}
		vs = append(vs, v)
	}
	return vs, nil
}
