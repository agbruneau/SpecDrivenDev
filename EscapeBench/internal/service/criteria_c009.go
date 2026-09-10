package service

import (
	"fmt"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
)

// Évaluateur de H-012, l'hypothèse que C-009 rend mesurable. Il applique le texte gelé de
// `docs/requirements.md` et rien d'autre. Le critère y est plus explicite que ses aînés sur deux
// points, et l'implémentation les suit à la lettre : l'ordre d'évaluation des clauses, et le fait
// qu'aucune grandeur relevée sur une autre cellule n'entre dans le seuil d'une cellule.

// h012Cell rassemble les cinq réplicats d'un couple taille-série et les grandeurs qui en dérivent.
type h012Cell struct {
	sizeBytes       int
	hasPointerField bool
	deltas          []float64
	ciHighs         []float64
	significant     []bool
	medianValue     float64
	medianPointer   float64
	floor           float64
	relativeBar     float64
	threshold       float64
	crossing        int
}

// pointerWins012 applique la règle de H-012 : au moins trois des cinq réplicats significatifs et de
// ciHigh au plus égal à l'opposé du seuil.
func (c h012Cell) pointerWins012() bool { return c.crossing >= 3 }

// resolved applique la définition de « résolue » : c'est la barrière postulée et non le bruit
// mesuré qui fixe le seuil.
func (c h012Cell) resolved() bool { return c.floor <= c.relativeBar }

// h012JudgedCells énumère les cinq couples taille-série que le critère juge. La taille de 8 octets
// de la série avec champ pointeur en est absente : son type n'a qu'un champ et ce champ est un
// pointeur, de sorte que les deux bras y passent un pointeur et que la règle de la page 253 n'y
// compare rien. Elle reste mesurée et consignée.
func h012JudgedCells() []struct {
	size            int
	hasPointerField bool
} {
	return []struct {
		size            int
		hasPointerField bool
	}{
		{8, false}, {16, false}, {24, false},
		{16, true}, {24, true},
	}
}

// evaluateH012 — la valeur ne se fait pas devancer par le pointeur de 1 à 3 mots machine, éprouvée
// sur un seuil apparié à la cellule et mesuré sur cinq réplicats de la paire réelle.
func evaluateH012(e Evidence) evaluation {
	// Clause (d), qui prime sur toutes.
	switch e.Campaign.ID {
	case "C-2026-09-10-1", "C-2026-09-10-2":
		return inconclusive("les mesures de %s précèdent la rédaction du critère de H-012 et ne peuvent pas le mettre à l'épreuve",
			e.Campaign.ID)
	}
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	files := []string{e.ComparisonPath}
	cells := h012CellsOf(e)

	// Clause (a) : exhaustivité des réplicats, sur les cinq cellules jugées et sur au moins un
	// témoin de sensibilité par série.
	for _, judged := range h012JudgedCells() {
		cell, ok := cells[h012Key{judged.size, judged.hasPointerField}]
		if !ok || len(cell.deltas) != models.ReplicateCount {
			return inconclusive("cellule %d octets %s : %d réplicat(s) complet(s) en %s, %d exigés (C-009)",
				judged.size, seriesLabel(judged.hasPointerField), replicateCount(cell, ok),
				models.LayoutNamedFields, models.ReplicateCount)
		}
	}
	witnesses := map[bool][]h012Cell{}
	for key, cell := range cells {
		if key.size >= RegisterArgumentBytes && len(cell.deltas) == models.ReplicateCount {
			witnesses[key.hasPointerField] = append(witnesses[key.hasPointerField], cell)
		}
	}
	for _, hasPointerField := range []bool{false, true} {
		if len(witnesses[hasPointerField]) == 0 {
			return inconclusive("série %s : aucune taille ≥ %d octets en %s ne fournit %d réplicats complets",
				seriesLabel(hasPointerField), RegisterArgumentBytes, models.LayoutNamedFields, models.ReplicateCount)
		}
	}

	// Clause (b) : témoin de sensibilité, dans chacune des deux séries, contre le seuil de sa
	// propre cellule. Son existence suffit ; aucune clause ne lit sa valeur.
	for _, hasPointerField := range []bool{false, true} {
		seen := false
		for _, cell := range witnesses[hasPointerField] {
			if cell.pointerWins012() {
				seen = true
				break
			}
		}
		if !seen {
			return inconclusive("série %s : aucune taille ≥ %d octets ne donne l'avantage au pointeur, le témoin de sensibilité manque",
				seriesLabel(hasPointerField), RegisterArgumentBytes)
		}
	}

	// Infirmation : une seule cellule jugée suffit.
	var winning []string
	var unresolved []string
	details := make([]string, 0, len(h012JudgedCells()))
	for _, judged := range h012JudgedCells() {
		cell := cells[h012Key{judged.size, judged.hasPointerField}]
		details = append(details, cell.describe())
		if cell.pointerWins012() {
			winning = append(winning, cell.label())
		}
		if !cell.resolved() {
			unresolved = append(unresolved, cell.label())
		}
	}
	// La cellule écartée est consignée, jamais jugée.
	if excluded, ok := cells[h012Key{8, true}]; ok {
		details = append(details, excluded.describe()+" [écartée du jugement : le type n'a qu'un champ, qui est le pointeur]")
	}

	if len(winning) > 0 {
		return evaluation{
			Outcome: models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le pointeur devance la valeur au-delà du seuil sur %s : %s",
				join(winning), join(details)),
			Files: files,
		}
	}
	// Clause (c) : à défaut d'infirmation, toute cellule non résolue rend le verdict non concluant.
	if len(unresolved) > 0 {
		return inconclusive("le banc ne résout pas %s : le plancher de bruit y dépasse la barrière relative, une confirmation n'y serait qu'un artefact — %s",
			join(unresolved), join(details))
	}
	return evaluation{
		Outcome: models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("aucune des %d cellules jugées ne donne l'avantage au pointeur, et toutes sont résolues : %s",
			len(h012JudgedCells()), join(details)),
		Files: files,
	}
}

// h012Key identifie un couple taille-série.
type h012Key struct {
	size            int
	hasPointerField bool
}

// replicateCount rend le nombre de réplicats trouvés, zéro si la cellule est absente.
func replicateCount(cell h012Cell, ok bool) int {
	if !ok {
		return 0
	}
	return len(cell.deltas)
}

// label nomme une cellule.
func (c h012Cell) label() string {
	return fmt.Sprintf("%d octets %s", c.sizeBytes, seriesLabel(c.hasPointerField))
}

// describe rend ce que le critère exige de consigner pour chaque cellule.
func (c h012Cell) describe() string {
	return fmt.Sprintf("%s : plancher %.4f, barrière %.4f, seuil %.4f, ciHigh %s, %d/%d réplicats franchissent",
		c.label(), c.floor, c.relativeBar, c.threshold, formatFloats(c.ciHighs), c.crossing, len(c.ciHighs))
}

// h012CellsOf regroupe les Comparison NAMED_FIELDS en profil LOCAL par couple taille-série et
// calcule leurs grandeurs. Le seuil d'une cellule ne dépend que d'elle-même.
func h012CellsOf(e Evidence) map[h012Key]h012Cell {
	grouped := map[h012Key][]models.Comparison{}
	for _, c := range e.ComparisonSet.Comparisons {
		if c.Profile != models.ProfileLocal || c.EffectiveLayout() != models.LayoutNamedFields {
			continue
		}
		key := h012Key{c.SizeBytes, c.HasPointerField}
		grouped[key] = append(grouped[key], c)
	}
	out := make(map[h012Key]h012Cell, len(grouped))
	for key, group := range grouped {
		// L'ordre du fichier ne doit décider de rien : les réplicats sont triés par identifiant.
		sort.Slice(group, func(i, j int) bool { return group[i].ValueCellID < group[j].ValueCellID })
		cell := h012Cell{sizeBytes: key.size, hasPointerField: key.hasPointerField}
		var values, pointers []float64
		for _, c := range group {
			cell.deltas = append(cell.deltas, c.DeltaNsPerOp)
			cell.ciHighs = append(cell.ciHighs, c.CIHigh)
			cell.significant = append(cell.significant, c.Significant)
			values = append(values, c.MedianValueNs)
			pointers = append(pointers, c.MedianPointerNs)
		}
		cell.medianValue = medianOf(values)
		cell.medianPointer = medianOf(pointers)
		cell.floor = trimmedRange(cell.deltas)
		cell.relativeBar = NoiseFloorRatio * maxOf(cell.medianValue, cell.medianPointer)
		cell.threshold = maxOf(cell.floor, cell.relativeBar)
		for i := range cell.ciHighs {
			if cell.significant[i] && cell.ciHighs[i] <= -cell.threshold {
				cell.crossing++
			}
		}
		out[key] = cell
	}
	return out
}

// trimmedRange rend l'étendue des trois valeurs de rang médian, le plus petit et le plus grand
// écartés. Un réplicat dissident ne fixe donc pas le plancher. Sur un nombre de valeurs autre que
// celui qu'exige le critère, l'étendue complète est rendue : la clause (a) aura de toute façon
// déjà refusé la campagne.
func trimmedRange(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) > 2 {
		sorted = sorted[1 : len(sorted)-1]
	}
	return sorted[len(sorted)-1] - sorted[0]
}

// medianOf rend la médiane d'une liste non vide.
func medianOf(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// maxOf rend la plus grande de deux valeurs.
func maxOf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// formatFloats met en forme une liste de valeurs pour une rationale.
func formatFloats(values []float64) string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, fmt.Sprintf("%.4f", v))
	}
	return "[" + join(out) + "]"
}
