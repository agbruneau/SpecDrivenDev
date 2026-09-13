package verdict

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// grid donne l'issue de toutes les répétitions d'une cellule « cas/DÉTECTEUR » ; PASS par défaut.
type grid map[string]results.Outcome

func digests() map[string]string {
	m := map[string]string{}
	for i := 1; i <= 13; i++ {
		m[fmt.Sprintf("H-%03d", i)] = "gel"
	}
	return m
}

// makeRun construit une campagne synthétique de cinq répétitions ; mentions désigne les cellules
// dont la sortie contient un signalement (H-002). Le témoin échoue et sa sortie est capturée.
func makeRun(g grid, mentions map[string]bool, probes map[string][3]float64) results.Run {
	run := results.Run{ID: "R-test", Reps: 5, CriteriaDigests: digests()}
	if _, ok := g["witness-fail/BARE"]; !ok {
		g["witness-fail/BARE"] = results.OutcomeFail
	}
	for _, c := range corpus.Catalog() {
		for _, d := range results.Detectors() {
			k := c.ID + "/" + string(d)
			o := g[k]
			if o == "" {
				o = results.OutcomePass
			}
			for rep := 1; rep <= 5; rep++ {
				run.Observations = append(run.Observations, results.Observation{CaseID: c.ID, Detector: d, Rep: rep, Outcome: o, MentionsLeak: mentions[k], MentionsWitness: k == "witness-fail/BARE"})
			}
		}
	}
	for k, v := range probes {
		probe, arm, _ := strings.Cut(k, "/")
		for rep := 1; rep <= 5; rep++ {
			run.Probes = append(run.Probes, results.ProbeResult{Probe: probe, Arm: arm, Rep: rep, WallNs: int64(v[0]), BytesPerOp: v[1], GoroutineDelta: int(v[2])})
		}
	}
	return run
}

func verdictOfRun(t *testing.T, id string, run results.Run) Verdict {
	t.Helper()
	vs, err := Evaluate(run, digests())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if v.HypothesisID == id {
			return v
		}
	}
	t.Fatalf("%s absent des verdicts", id)
	return Verdict{}
}

func forCases(g grid, keep func(corpus.Case) bool, det results.Detector, o results.Outcome) grid {
	for _, c := range corpus.Catalog() {
		if keep(c) {
			g[c.ID+"/"+string(det)] = o
		}
	}
	return g
}

type hcase struct {
	name string
	run  results.Run
	want Outcome
}

func runCases(t *testing.T, id string, cases []hcase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if v := verdictOfRun(t, id, tc.run); v.Outcome != tc.want {
				t.Fatalf("%s = %s, %s attendu ; %s", id, v.Outcome, tc.want, v.Rationale)
			}
		})
	}
}

func TestUC002_BR1_Cellule(t *testing.T) {
	mk := func(os ...results.Outcome) Cell {
		var c Cell
		for _, o := range os {
			c.Obs = append(c.Obs, results.Observation{Outcome: o})
		}
		return c
	}
	p, l, h := results.OutcomePass, results.OutcomeLeak, results.OutcomeHang
	c := mk(l, l, l, p, p)
	if !c.Diagnosed(l) || !c.Detected() || !c.Unstable() || c.Majority() != l || majorityLabel(c) != "LEAK ~" {
		t.Fatalf("3 sur 5 : %+v", c)
	}
	c = mk(l, l, p, p, h)
	if c.Diagnosed(l) || !c.Detected() || c.Majority() != "" || majorityLabel(c) != "MIXTE ~" {
		t.Fatalf("sans majorité : %s", majorityLabel(c))
	}
	if c := mk(p, p, p, p, p); c.Detected() || c.Unstable() || majorityLabel(c) != "PASS" {
		t.Fatalf("stable : %s", majorityLabel(c))
	}
}

func TestUC002_A1_CritereModifie(t *testing.T) {
	run := makeRun(grid{}, nil, nil)
	cur := digests()
	cur["H-005"] = "autre"
	if _, err := Evaluate(run, cur); err == nil || !strings.Contains(err.Error(), "H-005") {
		t.Fatalf("Evaluate = %v, refus nommant H-005 attendu", err)
	}
}

func TestUC002_A2_DonneesManquantes(t *testing.T) {
	cur := digests()
	cur["H-014"] = "neuf"
	vs, err := Evaluate(results.Run{ID: "R-vide", CriteriaDigests: digests()}, cur)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if v.Outcome != Inconclusive {
			t.Errorf("%s = %s sur une campagne vide", v.HypothesisID, v.Outcome)
		}
	}
	if last := vs[len(vs)-1]; last.HypothesisID != "H-014" || !strings.Contains(last.Rationale, "après") {
		t.Fatalf("hypothèse postérieure : %+v", last)
	}
}

func TestH001(t *testing.T) {
	runCases(t, "H-001", []hcase{
		{"silencieux", makeRun(grid{}, nil, nil), Confirmed},
		{"exemple du livre détecté", makeRun(grid{"dispatch-leak/BARE": results.OutcomeHang}, nil, nil), Refuted},
		{"moitié détectée", makeRun(forCases(grid{}, func(c corpus.Case) bool { return setL(c) && c.ID != "dispatch-leak" && !c.Reachable }, results.DetectorBare, results.OutcomeFail), nil, nil), Refuted},
		{"témoin muet", makeRun(grid{"witness-fail/BARE": results.OutcomePass}, nil, nil), Inconclusive},
	})
}

func TestH002(t *testing.T) {
	all := map[string]bool{}
	for _, c := range corpus.Catalog() {
		if setL(c) {
			all[c.ID+"/BARE"] = true
		}
	}
	witnessLost := makeRun(grid{}, all, nil)
	for i := range witnessLost.Observations {
		witnessLost.Observations[i].MentionsWitness = false
	}
	runCases(t, "H-002", []hcase{
		{"aucun signalement", makeRun(grid{}, nil, nil), Refuted},
		{"tous signalés", makeRun(grid{}, all, nil), Confirmed},
		{"un seul", makeRun(grid{}, map[string]bool{"dispatch-leak/BARE": true}, nil), Inconclusive},
		{"capture perdue", witnessLost, Inconclusive},
	})
}

func TestH003(t *testing.T) {
	races := grid{"counter-race/RACE": results.OutcomeRace, "append-race/RACE": results.OutcomeRace}
	withFP := grid{"counter-race/RACE": results.OutcomeRace, "append-race/RACE": results.OutcomeRace, "dispatch-fix/RACE": results.OutcomeRace}
	runCases(t, "H-003", []hcase{
		{"courses diagnostiquées", makeRun(races, nil, nil), Confirmed},
		{"course manquée", makeRun(grid{"counter-race/RACE": results.OutcomeRace}, nil, nil), Refuted},
		{"faux positif", makeRun(withFP, nil, nil), Refuted},
	})
}

func TestH004(t *testing.T) {
	var n []corpus.Case
	for _, c := range corpus.Catalog() {
		if setN(c) {
			n = append(n, c)
		}
	}
	half, few := grid{}, grid{}
	for i, c := range n {
		if 2*i < len(n) {
			half[c.ID+"/RACE"] = results.OutcomeHang
		}
		if i < 2 {
			few[c.ID+"/RACE"] = results.OutcomeHang
		}
	}
	both := grid{n[0].ID + "/RACE": results.OutcomeHang, n[0].ID + "/BARE": results.OutcomeHang}
	runCases(t, "H-004", []hcase{
		{"rien d'attribuable", makeRun(grid{}, nil, nil), Refuted},
		{"détecté aussi par BARE", makeRun(both, nil, nil), Refuted},
		{"moitié attribuable", makeRun(half, nil, nil), Confirmed},
		{"entre les deux", makeRun(few, nil, nil), Inconclusive},
	})
}

func TestH005(t *testing.T) {
	all := forCases(grid{}, setL, results.DetectorSynctest, results.OutcomeDeadlock)
	one := forCases(grid{}, setL, results.DetectorSynctest, results.OutcomeDeadlock)
	one["mutex-leak/SYNCTEST"] = results.OutcomeHang
	runCases(t, "H-005", []hcase{
		{"toutes diagnostiquées", makeRun(all, nil, nil), Confirmed},
		{"une manquée", makeRun(one, nil, nil), Refuted},
		{"bulle muette", makeRun(grid{}, nil, nil), Inconclusive},
	})
}

func TestH006(t *testing.T) {
	p := func(bubble, real float64) map[string][3]float64 {
		return map[string][3]float64{"SYNCTEST_TIMEOUT/SYNCTEST": {bubble}, "SYNCTEST_TIMEOUT/REAL": {real}}
	}
	runCases(t, "H-006", []hcase{
		{"instantané", makeRun(grid{}, nil, p(1e6, 501e6)), Confirmed},
		{"lent", makeRun(grid{}, nil, p(300e6, 501e6)), Refuted},
		{"entre les deux", makeRun(grid{}, nil, p(100e6, 501e6)), Inconclusive},
		{"témoin trop rapide", makeRun(grid{}, nil, p(1e6, 100e6)), Inconclusive},
		{"sonde absente", makeRun(grid{}, nil, nil), Inconclusive},
	})
}

func TestH007H008(t *testing.T) {
	for id, det := range map[string]string{"H-007": "PROGRAM", "H-008": "BARE"} {
		both := grid{"deadlock-send/" + det: results.OutcomeDeadlock, "deadlock-range/" + det: results.OutcomeDeadlock}
		runCases(t, id, []hcase{
			{"les deux fatals", makeRun(both, nil, nil), Confirmed},
			{"un blocage", makeRun(grid{"deadlock-send/" + det: results.OutcomeDeadlock, "deadlock-range/" + det: results.OutcomeHang}, nil, nil), Refuted},
		})
	}
}

func retention(forgotten, expired float64, gForgotten float64) map[string][3]float64 {
	m := map[string][3]float64{}
	for _, p := range []string{"BACKGROUND", "CANCELABLE"} {
		m["CANCEL_RETENTION/"+p+"/FORGOTTEN"] = [3]float64{0, forgotten, gForgotten}
		m["CANCEL_RETENTION/"+p+"/CANCELLED"] = [3]float64{0, 10, 0}
		m["CANCEL_RETENTION/"+p+"/EXPIRED"] = [3]float64{0, expired, 0}
	}
	m["CANCEL_RETENTION/OPAQUE/FORGOTTEN"] = [3]float64{0, 400, cancelN}
	m["CANCEL_RETENTION/OPAQUE/CANCELLED"] = [3]float64{0, 10, 0}
	m["CANCEL_RETENTION/OPAQUE/EXPIRED"] = [3]float64{0, 10, 0}
	return m
}

func TestH009(t *testing.T) {
	runCases(t, "H-009", []hcase{
		{"retenue puis libérée", makeRun(grid{}, nil, retention(200, 12, 0)), Confirmed},
		{"rien de retenu", makeRun(grid{}, nil, retention(12, 12, 0)), Refuted},
		{"retenue au-delà de l'échéance", makeRun(grid{}, nil, retention(200, 100, 0)), Refuted},
		{"retenue faible", makeRun(grid{}, nil, retention(30, 12, 0)), Inconclusive},
	})
}

func TestH010(t *testing.T) {
	blind := retention(200, 12, 0)
	blind["CANCEL_RETENTION/OPAQUE/FORGOTTEN"] = [3]float64{0, 400, 0}
	runCases(t, "H-010", []hcase{
		{"aucune goroutine", makeRun(grid{}, nil, retention(200, 12, 0)), Refuted},
		{"une goroutine par contexte", makeRun(grid{}, nil, retention(200, 12, cancelN)), Confirmed},
		{"quelques-unes", makeRun(grid{}, nil, retention(200, 12, 50)), Inconclusive},
		{"témoin aveugle", makeRun(grid{}, nil, blind), Inconclusive},
	})
}

func TestH011(t *testing.T) {
	p := func(after, reused, witness float64) map[string][3]float64 {
		return map[string][3]float64{"TIMER_GROWTH/AFTER_IN_LOOP": {0, after}, "TIMER_GROWTH/REUSED_TIMER": {0, reused}, "TIMER_GROWTH/RETAINED_WITNESS": {0, witness}}
	}
	runCases(t, "H-011", []hcase{
		{"pas de croissance", makeRun(grid{}, nil, p(3, 2, 150)), Refuted},
		{"croissance", makeRun(grid{}, nil, p(120, 2, 150)), Confirmed},
		{"entre les deux", makeRun(grid{}, nil, p(40, 2, 150)), Inconclusive},
		{"témoin aveugle", makeRun(grid{}, nil, p(3, 2, 30)), Inconclusive},
	})
}

func TestH012(t *testing.T) {
	leaks := func() grid { return forCases(grid{}, setL, results.DetectorNumGoroutine, results.OutcomeLeak) }
	fp := leaks()
	fp["dispatch-fix/NUMGOROUTINE"] = results.OutcomeLeak
	missed := leaks()
	missed["cond-leak/NUMGOROUTINE"] = results.OutcomePass
	runCases(t, "H-012", []hcase{
		{"séparation parfaite", makeRun(leaks(), nil, nil), Confirmed},
		{"faux positif", makeRun(fp, nil, nil), Refuted},
		{"fuite manquée", makeRun(missed, nil, nil), Refuted},
	})
}

func TestH013(t *testing.T) {
	domain := func() grid {
		return forCases(grid{}, func(c corpus.Case) bool {
			return setL(c) && !c.Reachable && c.Primitive != corpus.NetRead && c.Primitive != corpus.NoBlock
		}, results.DetectorLeakProfile, results.OutcomeLeak)
	}
	missed := domain()
	missed["mutex-leak/LEAKPROFILE"] = results.OutcomePass
	fp := domain()
	fp["waitgroup-fix/LEAKPROFILE"] = results.OutcomeLeak
	runCases(t, "H-013", []hcase{
		{"domaine couvert, hors domaine manqué", makeRun(domain(), nil, nil), Confirmed},
		{"domaine manqué", makeRun(missed, nil, nil), Refuted},
		{"faux positif", makeRun(fp, nil, nil), Refuted},
	})
}

func TestUC002_BR3_Rapport(t *testing.T) {
	root := t.TempDir()
	run := makeRun(grid{}, nil, retention(200, 12, 0))
	vs, err := Evaluate(run, digests())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	_, md, err := Write(root, run, vs, now)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(md)
	for _, want := range []string{"## Matrice de détectabilité", "| dispatch-leak | oui | PASS |", "| witness-fail | non | FAIL |", "CANCEL_RETENTION | BACKGROUND/FORGOTTEN"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("rapport sans %q", want)
		}
	}
	if _, _, err := Write(root, run, vs, now); err == nil {
		t.Fatal("un rapport existant a été écrasé")
	}
}
