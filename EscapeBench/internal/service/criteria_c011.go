package service

import (
	"fmt"
	"slices"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
)

// Évaluateurs de H-014, H-015 et H-016, successeurs de H-001, H-002 et H-012 (D-63). Ils
// appliquent le texte de `docs/requirements.md` et rien d'autre. Les trois partagent les grandeurs
// de H-012 (plancher, barrière relative, seuil, résolution, franchissement d'un réplicat) et une
// règle de multiplicité : une cellule donne l'avantage au pointeur à quatre réplicats franchissants
// sur cinq, elle est indéterminée à trois, et un verdict qui dépendrait d'elle est non concluant.

// informedSeriesDigest est l'empreinte de harnais des campagnes qui ont informé ces trois critères
// (préenregistrement séquentiel informé) : elles ne peuvent pas les mettre à l'épreuve.
const informedSeriesDigest = "551ce66be89b2dda720d9fd5ea7a3fb840977f09aed9bbbdf2ed5d112796257b"

// Règle de multiplicité commune à H-014, H-015 et H-016.
const (
	successorWinCrossings          = 4
	successorUndeterminedCrossings = 3
)

// wins applique la règle de multiplicité : au moins quatre réplicats sur cinq franchissent.
func (c h012Cell) wins() bool { return c.crossing >= successorWinCrossings }

// undetermined : exactement trois réplicats franchissent, la règle ne tranche pas la cellule.
func (c h012Cell) undetermined() bool { return c.crossing == successorUndeterminedCrossings }

// complete indique si la cellule fournit exactement les cinq réplicats qu'exige le critère.
func (c h012Cell) complete() bool { return len(c.deltas) == models.ReplicateCount }

// informedByPriorSeries applique la clause qui prime sur toutes : une campagne de la série qui a
// informé le critère ne peut pas le mettre à l'épreuve.
func informedByPriorSeries(e Evidence, id string) (evaluation, bool) {
	if e.Campaign.HarnessDigest != informedSeriesDigest {
		return evaluation{}, false
	}
	return inconclusive("la campagne %s porte l'empreinte de harnais %s…, dont les campagnes ont informé le critère de %s et ne peuvent pas le mettre à l'épreuve",
		e.Campaign.ID, informedSeriesDigest[:8], id), true
}

// sensitivityWitnesses rend les cellules complètes de 80 octets ou plus d'une série.
func sensitivityWitnesses(cells map[h012Key]h012Cell, hasPointerField bool) []h012Cell {
	var out []h012Cell
	for key, cell := range cells {
		if key.hasPointerField == hasPointerField && key.size >= RegisterArgumentBytes && cell.complete() {
			out = append(out, cell)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].sizeBytes < out[j].sizeBytes })
	return out
}

// checkWitness applique les clauses (a) et (b) du témoin de sensibilité d'une série.
func checkWitness(cells map[h012Key]h012Cell, hasPointerField bool, layout models.Layout) (evaluation, bool) {
	witnesses := sensitivityWitnesses(cells, hasPointerField)
	if len(witnesses) == 0 {
		return inconclusive("série %s : aucune taille ≥ %d octets en %s ne fournit %d réplicats complets",
			seriesLabel(hasPointerField), RegisterArgumentBytes, layout, models.ReplicateCount), true
	}
	if !slices.ContainsFunc(witnesses, h012Cell.wins) {
		return inconclusive("série %s : aucune taille ≥ %d octets en %s ne donne l'avantage au pointeur à %d réplicats sur %d, le témoin de sensibilité manque",
			seriesLabel(hasPointerField), RegisterArgumentBytes, layout, successorWinCrossings, models.ReplicateCount), true
	}
	return evaluation{}, false
}

// evaluateH014 — copier une petite struct n'est pas plus cher que passer un pointeur, sur
// ARRAY_FILL, cinq réplicats par cellule et au plus une taille sur trois favorable au pointeur.
func evaluateH014(e Evidence) evaluation {
	// Clause (e), qui prime sur toutes.
	if ev, ok := informedByPriorSeries(e, "H-014"); ok {
		return ev
	}
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	cells := replicatedCells(e, models.LayoutArrayFill)
	judged := []int{8, 16, 24}
	// Clause (a).
	for _, size := range judged {
		cell, ok := cells[h012Key{size, false}]
		if !ok || !cell.complete() {
			return inconclusive("cellule %d octets sans champ pointeur : %d réplicat(s) en %s, %d exigés (C-011)",
				size, replicateCount(cell, ok), models.LayoutArrayFill, models.ReplicateCount)
		}
	}
	// Clauses (a) et (b) du témoin.
	if ev, ok := checkWitness(cells, false, models.LayoutArrayFill); ok {
		return ev
	}
	var winning, undetermined, unresolved, details []string
	for _, size := range judged {
		cell := cells[h012Key{size, false}]
		details = append(details, cell.describe())
		switch {
		case cell.wins():
			winning = append(winning, cell.label())
			continue
		case cell.undetermined():
			undetermined = append(undetermined, cell.label())
		}
		if !cell.resolved() {
			unresolved = append(unresolved, cell.label())
		}
	}
	files := []string{e.ComparisonPath}
	if len(winning) >= 2 {
		return evaluation{
			Outcome: models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le pointeur devance la valeur au-delà du seuil sur %d des 3 tailles (%s) : %s",
				len(winning), join(winning), join(details)),
			Files: files,
		}
	}
	// Clause (c) : multiplicité.
	if len(winning)+len(undetermined) >= 2 {
		return inconclusive("le verdict dépend d'une cellule indéterminée (%s, %d réplicats sur %d) : %s",
			join(undetermined), successorUndeterminedCrossings, models.ReplicateCount, join(details))
	}
	// Clause (d) : résolution.
	if len(unresolved) > 0 {
		return inconclusive("le banc ne résout pas %s : le plancher de bruit y dépasse la barrière relative — %s",
			join(unresolved), join(details))
	}
	return evaluation{
		Outcome: models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("le pointeur devance la valeur sur %d des 3 tailles, le critère en exige 2, et les autres sont résolues : %s",
			len(winning), join(details)),
		Files: files,
	}
}

// h015Series rend les cellules d'une série ARRAY_FILL par taille croissante, cellule écartée exclue.
func h015Series(cells map[h012Key]h012Cell, hasPointerField bool) []h012Cell {
	var out []h012Cell
	for key, cell := range cells {
		if key.hasPointerField != hasPointerField || (hasPointerField && key.size == 8) {
			continue
		}
		out = append(out, cell)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].sizeBytes < out[j].sizeBytes })
	return out
}

// replicatedTipping rend le point de bascule répliqué d'une série : la plus petite taille dont la
// cellule et celles de toutes les tailles supérieures satisfont favorable. observed est faux si la
// plus grande taille ne le satisfait pas.
func replicatedTipping(series []h012Cell, favorable func(h012Cell) bool) (size int, observed bool) {
	for i := len(series) - 1; i >= 0; i-- {
		if !favorable(series[i]) {
			break
		}
		size, observed = series[i].sizeBytes, true
	}
	return size, observed
}

// evaluateH015 — le point de bascule valeur → pointeur est supérieur à 24 octets en ARRAY_FILL,
// établi cellule par cellule sur cinq réplicats.
func evaluateH015(e Evidence) evaluation {
	// Clause (e), qui prime sur toutes.
	if ev, ok := informedByPriorSeries(e, "H-015"); ok {
		return ev
	}
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	cells := replicatedCells(e, models.LayoutArrayFill)
	judged := map[bool][]int{false: {8, 16, 24}, true: {16, 24}}
	series := map[bool][]h012Cell{}
	for _, hasPointerField := range []bool{false, true} {
		// Clause (a) : cellules jugées présentes, toutes les cellules complètes, un témoin.
		for _, size := range judged[hasPointerField] {
			if _, ok := cells[h012Key{size, hasPointerField}]; !ok {
				return inconclusive("série %s : aucune cellule de %d octets en %s", seriesLabel(hasPointerField), size, models.LayoutArrayFill)
			}
		}
		series[hasPointerField] = h015Series(cells, hasPointerField)
		for _, cell := range series[hasPointerField] {
			if !cell.complete() {
				return inconclusive("cellule %s : %d réplicat(s) en %s, %d exigés (C-011)",
					cell.label(), len(cell.deltas), models.LayoutArrayFill, models.ReplicateCount)
			}
		}
		// Clauses (a) et (b) du témoin.
		if ev, ok := checkWitness(cells, hasPointerField, models.LayoutArrayFill); ok {
			return ev
		}
	}

	var details, refuting, blurred, unresolved []string
	for _, hasPointerField := range []bool{false, true} {
		s := series[hasPointerField]
		tipping, observed := replicatedTipping(s, h012Cell.wins)
		label := seriesLabel(hasPointerField)
		if observed {
			details = append(details, fmt.Sprintf("série %s : point de bascule répliqué %d octets", label, tipping))
		} else {
			details = append(details, fmt.Sprintf("série %s : point de bascule répliqué non observé", label))
		}
		if observed && tipping <= smallStructBytes {
			refuting = append(refuting, fmt.Sprintf("%s à %d octets", label, tipping))
		}
		if loose, ok := replicatedTipping(s, func(c h012Cell) bool { return c.wins() || c.undetermined() }); ok && loose <= smallStructBytes {
			blurred = append(blurred, fmt.Sprintf("%s à %d octets", label, loose))
		}
		for _, cell := range s {
			details = append(details, cell.describe())
			if cell.sizeBytes <= smallStructBytes && !cell.wins() && !cell.resolved() {
				unresolved = append(unresolved, cell.label())
			}
		}
	}
	if excluded, ok := cells[h012Key{8, true}]; ok {
		details = append(details, excluded.describe()+" [écartée du jugement : le type n'a qu'un champ, qui est le pointeur]")
	}
	files := []string{e.ComparisonPath}
	if len(refuting) > 0 {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("point de bascule répliqué ≤ %d octets pour %s : %s", smallStructBytes, join(refuting), join(details)),
			Files:     files,
		}
	}
	// Clause (c) : multiplicité.
	if len(blurred) > 0 {
		return inconclusive("en comptant les cellules indéterminées comme favorables au pointeur, le point de bascule tomberait à ≤ %d octets pour %s : %s",
			smallStructBytes, join(blurred), join(details))
	}
	// Clause (d) : résolution.
	if len(unresolved) > 0 {
		return inconclusive("le banc ne résout pas %s : le plancher de bruit y dépasse la barrière relative — %s",
			join(unresolved), join(details))
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("aucun point de bascule répliqué ≤ %d octets : %s", smallStructBytes, join(details)),
		Files:     files,
	}
}

// evaluateH016 — H-012 avec la règle de multiplicité : quatre réplicats sur cinq pour donner
// l'avantage au pointeur, et une cellule indéterminée rend le verdict non concluant.
func evaluateH016(e Evidence) evaluation {
	// Clause (d) de H-016, qui remplace celle de H-012 et prime sur toutes.
	if ev, ok := informedByPriorSeries(e, "H-016"); ok {
		return ev
	}
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	cells := h012CellsOf(e)
	// Clause (a) de H-012 : les cinq cellules jugées.
	for _, judged := range h012JudgedCells() {
		cell, ok := cells[h012Key{judged.size, judged.hasPointerField}]
		if !ok || !cell.complete() {
			return inconclusive("cellule %d octets %s : %d réplicat(s) complet(s) en %s, %d exigés (C-009)",
				judged.size, seriesLabel(judged.hasPointerField), replicateCount(cell, ok),
				models.LayoutNamedFields, models.ReplicateCount)
		}
	}
	// Clauses (a) et (b) de H-012 pour le témoin, sous la règle de quatre.
	for _, hasPointerField := range []bool{false, true} {
		if ev, ok := checkWitness(cells, hasPointerField, models.LayoutNamedFields); ok {
			return ev
		}
	}
	var winning, undetermined, unresolved, details []string
	for _, judged := range h012JudgedCells() {
		cell := cells[h012Key{judged.size, judged.hasPointerField}]
		details = append(details, cell.describe())
		if cell.wins() {
			winning = append(winning, cell.label())
		}
		if cell.undetermined() {
			undetermined = append(undetermined, cell.label())
		}
		if !cell.resolved() {
			unresolved = append(unresolved, cell.label())
		}
	}
	if excluded, ok := cells[h012Key{8, true}]; ok {
		details = append(details, excluded.describe()+" [écartée du jugement : le type n'a qu'un champ, qui est le pointeur]")
	}
	files := []string{e.ComparisonPath}
	if len(winning) > 0 {
		return evaluation{
			Outcome: models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le pointeur devance la valeur à %d réplicats sur %d au moins sur %s : %s",
				successorWinCrossings, models.ReplicateCount, join(winning), join(details)),
			Files: files,
		}
	}
	// Clause (m) : multiplicité.
	if len(undetermined) > 0 {
		return inconclusive("cellule(s) indéterminée(s) à %d réplicats sur %d : %s — %s",
			successorUndeterminedCrossings, models.ReplicateCount, join(undetermined), join(details))
	}
	// Clause (c) de H-012 : résolution.
	if len(unresolved) > 0 {
		return inconclusive("le banc ne résout pas %s : le plancher de bruit y dépasse la barrière relative — %s",
			join(unresolved), join(details))
	}
	return evaluation{
		Outcome: models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("aucune des %d cellules jugées ne donne l'avantage au pointeur ni n'est indéterminée, et toutes sont résolues : %s",
			len(h012JudgedCells()), join(details)),
		Files: files,
	}
}
