package verdict

import (
	"fmt"
	"strings"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// cancelN et les seuils ci-dessous recopient le texte des critères gelés de docs/requirements.md.
const cancelN = 20000

// evaluators code chaque critère ; le commentaire de chaque fonction en résume la lettre.
var evaluators = map[string]evaluator{
	"H-001": h001, "H-002": h002, "H-003": h003, "H-004": h004, "H-005": h005,
	"H-006": h006, "H-007": h007, "H-008": h008, "H-009": h009, "H-010": h010,
	"H-011": h011, "H-012": h012, "H-013": h013,
}

func verdictOf(refuted bool) Outcome {
	if refuted {
		return Refuted
	}
	return Confirmed
}

// cellsWhere rend les cas de cs dont la cellule par det satisfait keep.
func (e *eval) cellsWhere(cs []corpus.Case, det results.Detector, keep func(Cell) bool) []corpus.Case {
	var out []corpus.Case
	for _, c := range cs {
		if keep(e.cell(c.ID, det)) {
			out = append(out, c)
		}
	}
	return out
}

// H-001 : infirmée si BARE détecte au moins la moitié de L ou dispatch-leak ; non concluante si
// BARE ne détecte pas witness-fail.
func h001(e *eval) (Outcome, string) {
	witness := e.cell("witness-fail", results.DetectorBare).Detected()
	l := e.set(setL)
	det := e.cellsWhere(l, results.DetectorBare, Cell.Detected)
	book := e.cell("dispatch-leak", results.DetectorBare).Detected()
	r := fmt.Sprintf("BARE détecte %d cas de L sur %d (%s) ; dispatch-leak détecté : %t ; witness-fail détecté : %t", len(det), len(l), ids(det), book, witness)
	if !witness {
		return Inconclusive, r
	}
	return verdictOf(2*len(det) >= len(l) || book), r
}

// H-002 : signalement dans la sortie BARE des cas de L ; infirmée si aucune exécution n'en
// contient, confirmée si toutes, non concluante sinon ou si la capture perd LEAKLAB-WITNESS.
func h002(e *eval) (Outcome, string) {
	witness := false
	for _, o := range e.cell("witness-fail", results.DetectorBare).Obs {
		witness = witness || o.MentionsWitness
	}
	total, mentions := 0, 0
	for _, c := range e.set(setL) {
		for _, o := range e.cell(c.ID, results.DetectorBare).Obs {
			total++
			if o.MentionsLeak {
				mentions++
			}
		}
	}
	r := fmt.Sprintf("%d exécutions BARE de cas de L sur %d contiennent un signalement ; LEAKLAB-WITNESS capturé : %t", mentions, total, witness)
	switch {
	case !witness:
		return Inconclusive, r
	case mentions == 0:
		return Refuted, r
	case mentions == total:
		return Confirmed, r
	}
	return Inconclusive, r
}

// H-003 : infirmée si un cas de R n'est pas diagnostiqué RACE par RACE, ou si un cas ni race ni
// blocks l'est ; non concluante si R compte moins de deux cas.
func h003(e *eval) (Outcome, string) {
	r := e.set(setR)
	isRace := func(c Cell) bool { return c.Diagnosed(results.OutcomeRace) }
	missed := e.cellsWhere(r, results.DetectorRace, func(c Cell) bool { return !isRace(c) })
	fps := e.cellsWhere(e.set(func(c corpus.Case) bool { return !c.Race && !c.Blocks }), results.DetectorRace, isRace)
	rat := fmt.Sprintf("cas de R non diagnostiqués : %s ; cas sans course diagnostiqués RACE : %s", ids(missed), ids(fps))
	if len(r) < 2 {
		return Inconclusive, rat
	}
	return verdictOf(len(missed) > 0 || len(fps) > 0), rat
}

// H-004 : un cas de N est attribuable à -race s'il est diagnostiqué RACE par RACE, ou détecté par
// RACE sans l'être par BARE. Infirmée si aucun ; confirmée si au moins la moitié.
func h004(e *eval) (Outcome, string) {
	n := e.set(setN)
	var attr []corpus.Case
	for _, c := range n {
		race, bare := e.cell(c.ID, results.DetectorRace), e.cell(c.ID, results.DetectorBare)
		if race.Diagnosed(results.OutcomeRace) || (race.Detected() && !bare.Detected()) {
			attr = append(attr, c)
		}
	}
	r := fmt.Sprintf("%d cas de N attribuables à -race sur %d (%s)", len(attr), len(n), ids(attr))
	switch {
	case len(attr) == 0:
		return Refuted, r
	case 2*len(attr) >= len(n):
		return Confirmed, r
	}
	return Inconclusive, r
}

// H-005 : infirmée si un cas de L n'est pas diagnostiqué DEADLOCK par SYNCTEST ; non concluante si
// aucun ne l'est.
func h005(e *eval) (Outcome, string) {
	l := e.set(setL)
	var missed []string
	diag := 0
	for _, c := range l {
		cell := e.cell(c.ID, results.DetectorSynctest)
		if cell.Diagnosed(results.OutcomeDeadlock) {
			diag++
		} else {
			missed = append(missed, fmt.Sprintf("%s (%s)", c.ID, majorityLabel(cell)))
		}
	}
	r := fmt.Sprintf("SYNCTEST diagnostique DEADLOCK sur %d cas de L sur %d ; non diagnostiqués : %s", diag, len(l), strings.Join(missed, ", "))
	switch {
	case diag == 0:
		return Inconclusive, r
	case diag < len(l):
		return Refuted, r
	}
	return Confirmed, r
}

func wall(p results.ProbeResult) float64  { return float64(p.WallNs) }
func bytes(p results.ProbeResult) float64 { return p.BytesPerOp }
func delta(p results.ProbeResult) float64 { return float64(p.GoroutineDelta) }

// H-006 : non concluante si REAL < 450 ms ; confirmée si SYNCTEST < 50 ms ; infirmée si ≥ 250 ms.
func h006(e *eval) (Outcome, string) {
	unbubbled := e.median("SYNCTEST_TIMEOUT", "REAL", wall)
	bubble := e.median("SYNCTEST_TIMEOUT", "SYNCTEST", wall)
	r := fmt.Sprintf("médiane REAL %.1f ms ; médiane SYNCTEST %.3f ms", unbubbled/1e6, bubble/1e6)
	switch {
	case unbubbled < 450e6:
		return Inconclusive, r
	case bubble < 50e6:
		return Confirmed, r
	case bubble >= 250e6:
		return Refuted, r
	}
	return Inconclusive, r
}

func deadlockBoth(e *eval, det results.Detector) (Outcome, string) {
	var parts []string
	refuted := false
	for _, id := range []string{"deadlock-send", "deadlock-range"} {
		cell := e.cell(id, det)
		refuted = refuted || !cell.Diagnosed(results.OutcomeDeadlock)
		parts = append(parts, fmt.Sprintf("%s/%s : %s", id, det, majorityLabel(cell)))
	}
	return verdictOf(refuted), strings.Join(parts, " ; ")
}

// H-007 : infirmée si l'un des deux cas n'est pas diagnostiqué DEADLOCK par PROGRAM.
func h007(e *eval) (Outcome, string) { return deadlockBoth(e, results.DetectorProgram) }

// H-008 : infirmée si l'un des deux cas n'est pas diagnostiqué DEADLOCK par BARE.
func h008(e *eval) (Outcome, string) { return deadlockBoth(e, results.DetectorBare) }

// H-009 : retenue = FORGOTTEN − CANCELLED, résidu = EXPIRED − CANCELLED, pour BACKGROUND et
// CANCELABLE. Confirmée si retenue ≥ 32 et résidu < 8 pour les deux ; infirmée si retenue < 8 pour
// les deux, ou résidu ≥ 32 pour l'un.
func h009(e *eval) (Outcome, string) {
	allHeld, allResidueLow, allHeldLow, anyResidueHigh := true, true, true, false
	var parts []string
	for _, p := range []string{"BACKGROUND", "CANCELABLE"} {
		c := e.median("CANCEL_RETENTION", p+"/CANCELLED", bytes)
		held := e.median("CANCEL_RETENTION", p+"/FORGOTTEN", bytes) - c
		residue := e.median("CANCEL_RETENTION", p+"/EXPIRED", bytes) - c
		allHeld = allHeld && held >= 32
		allResidueLow = allResidueLow && residue < 8
		allHeldLow = allHeldLow && held < 8
		anyResidueHigh = anyResidueHigh || residue >= 32
		parts = append(parts, fmt.Sprintf("%s : retenue %.1f o, résidu %.1f o", p, held, residue))
	}
	r := strings.Join(parts, " ; ")
	switch {
	case allHeld && allResidueLow:
		return Confirmed, r
	case allHeldLow || anyResidueHigh:
		return Refuted, r
	}
	return Inconclusive, r
}

// H-010 : delta = FORGOTTEN − CANCELLED en goroutines. Non concluante si OPAQUE < N/2 ; infirmée si
// ≤ 1 pour BACKGROUND et CANCELABLE ; confirmée si ≥ N/2 pour les deux.
func h010(e *eval) (Outcome, string) {
	d := func(p string) float64 {
		return e.median("CANCEL_RETENTION", p+"/FORGOTTEN", delta) - e.median("CANCEL_RETENTION", p+"/CANCELLED", delta)
	}
	bg, cc, op := d("BACKGROUND"), d("CANCELABLE"), d("OPAQUE")
	r := fmt.Sprintf("delta de goroutines : BACKGROUND %.0f, CANCELABLE %.0f, OPAQUE (témoin) %.0f, N = %d", bg, cc, op, cancelN)
	switch {
	case op < cancelN/2:
		return Inconclusive, r
	case bg <= 1 && cc <= 1:
		return Refuted, r
	case bg >= cancelN/2 && cc >= cancelN/2:
		return Confirmed, r
	}
	return Inconclusive, r
}

// H-011 : croissance = AFTER_IN_LOOP − REUSED_TIMER, référence = RETAINED_WITNESS − REUSED_TIMER.
// Non concluante si référence < 64 ; infirmée si croissance < 10 % de la référence ; confirmée si
// elle en atteint 50 %.
func h011(e *eval) (Outcome, string) {
	reused := e.median("TIMER_GROWTH", "REUSED_TIMER", bytes)
	growth := e.median("TIMER_GROWTH", "AFTER_IN_LOOP", bytes) - reused
	ref := e.median("TIMER_GROWTH", "RETAINED_WITNESS", bytes) - reused
	r := fmt.Sprintf("croissance %.2f o par itération ; référence %.2f o", growth, ref)
	switch {
	case ref < 64:
		return Inconclusive, r
	case growth < 0.1*ref:
		return Refuted, r
	case growth >= 0.5*ref:
		return Confirmed, r
	}
	return Inconclusive, r
}

// H-012 : infirmée si un cas de L n'est pas diagnostiqué LEAK par NUMGOROUTINE, ou si un cas de S
// l'est.
func h012(e *eval) (Outcome, string) {
	isLeak := func(c Cell) bool { return c.Diagnosed(results.OutcomeLeak) }
	missed := e.cellsWhere(e.set(setL), results.DetectorNumGoroutine, func(c Cell) bool { return !isLeak(c) })
	fps := e.cellsWhere(e.set(setS), results.DetectorNumGoroutine, isLeak)
	return verdictOf(len(missed) > 0 || len(fps) > 0), fmt.Sprintf("fuites manquées : %s ; cas sains signalés : %s", ids(missed), ids(fps))
}

// H-013 : domaine = cas de L bloqués sur une primitive de concurrence inaccessible. Infirmée si un
// cas du domaine n'est pas diagnostiqué LEAK par LEAKPROFILE, ou si un cas de S l'est.
func h013(e *eval) (Outcome, string) {
	concurrency := map[corpus.Primitive]bool{corpus.ChanSend: true, corpus.ChanRecv: true, corpus.Select: true, corpus.WaitGroup: true, corpus.Cond: true, corpus.Mutex: true}
	inDomain := func(c corpus.Case) bool { return setL(c) && concurrency[c.Primitive] && !c.Reachable }
	isLeak := func(c Cell) bool { return c.Diagnosed(results.OutcomeLeak) }
	missed := e.cellsWhere(e.set(inDomain), results.DetectorLeakProfile, func(c Cell) bool { return !isLeak(c) })
	fps := e.cellsWhere(e.set(setS), results.DetectorLeakProfile, isLeak)
	var outside []string
	for _, c := range e.set(func(c corpus.Case) bool { return setL(c) && !inDomain(c) }) {
		outside = append(outside, fmt.Sprintf("%s (%s)", c.ID, majorityLabel(e.cell(c.ID, results.DetectorLeakProfile))))
	}
	r := fmt.Sprintf("domaine manqué : %s ; cas sains signalés : %s ; hors domaine, sans décider : %s", ids(missed), ids(fps), strings.Join(outside, ", "))
	return verdictOf(len(missed) > 0 || len(fps) > 0), r
}

// majorityLabel rend l'issue majoritaire d'une cellule, « MIXTE » sans majorité, suivie de « ~ » si
// la cellule est instable.
func majorityLabel(c Cell) string {
	if len(c.Obs) == 0 {
		return "—"
	}
	label := string(c.Majority())
	if label == "" {
		label = "MIXTE"
	}
	if c.Unstable() {
		label += " ~"
	}
	return label
}
