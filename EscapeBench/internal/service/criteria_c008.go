package service

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
)

// Évaluateurs des hypothèses que C-008 rend mesurables. Chacun applique le texte gelé de
// `docs/requirements.md` et rien d'autre ; les gardes d'évaluabilité passent avant l'infirmation,
// comme dans tous les évaluateurs du banc : un verdict ne se rend pas sur un corpus incomplet.

// RegisterArgumentBytes est la taille au-delà de laquelle même une disposition à champs nommés
// dépasse les registres d'argument entiers de l'ABI amd64, et où le pointeur doit donc l'emporter.
// C'est le témoin de sensibilité de H-007.
const RegisterArgumentBytes = 80

// NoiseFloorRatio est la barrière relative de H-007, retenue quand le témoin nul mesure un
// plancher proche de zéro.
const NoiseFloorRatio = 0.10

// smallStructSizes sont les trois tailles de 1 à 3 mots machine sur lesquelles porte H-007.
func smallStructSizes() []int { return []int{8, 16, 24} }

// evaluateH007 — la règle « 1 à 3 mots machine » sur une disposition assignable aux registres.
// Infirmée si, dans au moins une des deux séries LOCAL, au moins deux des trois petites tailles en
// NAMED_FIELDS donnent l'avantage au pointeur au-delà du plancher de bruit mesuré.
func evaluateH007(e Evidence) evaluation {
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	files := []string{e.ComparisonPath}

	// Témoin de sensibilité : sans une taille où le pointeur l'emporte réellement, rien
	// n'établit que le banc sache détecter un écart.
	sensitivity := false
	for _, c := range e.ComparisonSet.Comparisons {
		if c.Profile != models.ProfileLocal || c.Layout != models.LayoutNamedFields || c.SizeBytes < RegisterArgumentBytes {
			continue
		}
		if pointerWins(c, noiseFloor(e, c.HasPointerField)) {
			sensitivity = true
			break
		}
	}
	if !sensitivity {
		return inconclusive("aucune taille ≥ %d octets en %s ne donne l'avantage au pointeur : le témoin de sensibilité manque",
			RegisterArgumentBytes, models.LayoutNamedFields)
	}

	var details []string
	refutingSeries := ""
	for _, hasPointerField := range []bool{false, true} {
		label := seriesLabel(hasPointerField)
		named := smallBySize(e, models.LayoutNamedFields, hasPointerField)
		sham := smallBySize(e, models.LayoutNamedFieldsSham, hasPointerField)
		for _, size := range smallStructSizes() {
			if _, ok := named[size]; !ok {
				return inconclusive("série %s : taille %d octets absente en %s", label, size, models.LayoutNamedFields)
			}
			if _, ok := sham[size]; !ok {
				return inconclusive("série %s : taille %d octets absente en %s, le plancher de bruit ne peut pas être mesuré",
					label, size, models.LayoutNamedFieldsSham)
			}
		}
		floor := noiseFloor(e, hasPointerField)
		var winning []string
		for _, size := range smallStructSizes() {
			c := named[size]
			if pointerWins(c, floor) {
				winning = append(winning, fmt.Sprintf("%d o", size))
			}
		}
		details = append(details, fmt.Sprintf("%s : plancher %.4f ns, avantage pointeur sur %d des 3 petites tailles%s",
			label, floor, len(winning), suffixList(winning)))
		if len(winning) >= 2 && refutingSeries == "" {
			refutingSeries = label
		}
	}

	if refutingSeries != "" {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("la série « %s » donne l'avantage au pointeur sur au moins deux des trois tailles de 1 à 3 mots (%s)", refutingSeries, join(details)),
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: "aucune série ne donne l'avantage au pointeur sur plus d'une des trois petites tailles (" + join(details) + ")",
		Files:     files,
	}
}

// noiseFloor rend le plancher de bruit d'une série : le plus grand écart observé sur le témoin
// nul, dont le delta vrai est nul par construction.
func noiseFloor(e Evidence, hasPointerField bool) float64 {
	floor := 0.0
	for _, c := range e.ComparisonSet.Comparisons {
		if c.Profile != models.ProfileLocal || c.Layout != models.LayoutNamedFieldsSham || c.HasPointerField != hasPointerField {
			continue
		}
		if delta := math.Abs(c.DeltaNsPerOp); delta > floor {
			floor = delta
		}
	}
	return floor
}

// pointerWins indique qu'une Comparison donne l'avantage au pointeur au-delà du seuil : l'écart
// est significatif et sa borne haute dépasse le plancher de bruit, ou la barrière relative si
// celle-ci est plus exigeante.
// A-230 : le paramètre `ref` n'a jamais reçu autre chose que `c` à ses deux sites d'appel. Le
// garder laissait croire que le plancher relatif pouvait se calculer sur une autre comparaison que
// celle jugée, ce qu'aucun critère gelé ne prévoit.
func pointerWins(c models.Comparison, floor float64) bool {
	threshold := max(floor, NoiseFloorRatio*c.MedianValueNs)
	return c.Significant && c.CIHigh <= -threshold
}

// smallBySize indexe par taille les comparaisons LOCAL d'une disposition et d'une série données,
// restreintes aux trois petites tailles.
func smallBySize(e Evidence, layout models.Layout, hasPointerField bool) map[int]models.Comparison {
	out := map[int]models.Comparison{}
	for _, c := range e.ComparisonSet.Comparisons {
		if c.Profile != models.ProfileLocal || c.Layout != layout || c.HasPointerField != hasPointerField {
			continue
		}
		for _, size := range smallStructSizes() {
			if c.SizeBytes == size {
				out[size] = c
			}
		}
	}
	return out
}

// evaluateH008 — l'écart de 10 à 200 fois entre un succès de cache et un défaut servi par la
// mémoire, mesuré sur une chaîne de pointeurs dépendante.
func evaluateH008(e Evidence) evaluation {
	provenance := e.Campaign.Provenance
	if provenance.L1DataCacheBytes <= 0 || provenance.LastLevelCacheBytes <= 0 {
		return inconclusive("la provenance de %s ne porte pas les tailles de cache : l1DataCacheBytes = %d, lastLevelCacheBytes = %d",
			e.Campaign.ID, provenance.L1DataCacheBytes, provenance.LastLevelCacheBytes)
	}
	residentMax := provenance.L1DataCacheBytes / 2
	nonResidentMin := 4 * provenance.LastLevelCacheBytes

	var resident, nonResident []int
	for _, subjectID := range sortedSubjectIDs(e.Measurements) {
		kind, parameter, ok := models.ParseProbeID(subjectID)
		if !ok || kind != models.ProbePointerChase {
			continue
		}
		if _, complete := completeMeasurement(e, subjectID); !complete {
			continue
		}
		switch {
		case int64(parameter) <= residentMax:
			resident = append(resident, parameter)
		case int64(parameter) >= nonResidentMin:
			nonResident = append(nonResident, parameter)
		}
	}
	if len(resident) != 1 || len(nonResident) != 1 {
		return inconclusive("bandes de %s mal peuplées : %d sonde(s) résidente(s) en L1 (≤ %d octets) et %d non résidente(s) (≥ %d octets), une de chaque est exigée",
			models.ProbePointerChase, len(resident), residentMax, len(nonResident), nonResidentMin)
	}

	residentID := models.Probe{Kind: models.ProbePointerChase, Parameter: resident[0]}.ID()
	nonResidentID := models.Probe{Kind: models.ProbePointerChase, Parameter: nonResident[0]}.ID()
	residentM, _ := completeMeasurement(e, residentID)
	nonResidentM, _ := completeMeasurement(e, nonResidentID)
	m1 := residentM.MedianNs()
	m3 := nonResidentM.MedianNs()
	if m1 <= 0 {
		return inconclusive("latence résidente nulle pour %s : le rapport est indéfini", residentID)
	}
	ratio := m3 / m1
	files := []string{e.MeasurementPaths[residentID], e.MeasurementPaths[nonResidentID]}
	detail := fmt.Sprintf("résident %d o : %.2f ns/accès ; non résident %d o : %.2f ns/accès ; rapport ×%.1f",
		resident[0], m1, nonResident[0], m3, ratio)

	var breaches []string
	if m3 <= 100 {
		breaches = append(breaches, fmt.Sprintf("latence non résidente de %.2f ns, au plus 100 ns", m3))
	}
	if ratio < 10 {
		breaches = append(breaches, fmt.Sprintf("rapport ×%.1f sous 10", ratio))
	}
	if ratio > 200 {
		breaches = append(breaches, fmt.Sprintf("rapport ×%.1f au-dessus de 200", ratio))
	}
	if len(breaches) > 0 {
		return evaluation{Outcome: models.OutcomeRefuted, Rationale: join(breaches) + " — " + detail, Files: files}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: "la latence non résidente dépasse 100 ns et le rapport reste dans [10, 200] : " + detail,
		Files:     files,
	}
}

// containerProfiles sont les trois conteneurs que le livre nomme à sa quatrième cause.
func containerProfiles() []models.LifetimeProfile {
	return []models.LifetimeProfile{models.ProfileStoredInMap, models.ProfileStoredInSlice, models.ProfileStoredInStruct}
}

// evaluateH009 — la généralisation de la quatrième cause aux tranches et aux structs.
// Infirmée si, pour l'un des trois conteneurs, deux cellules de tailles distinctes n'échappent pas.
func evaluateH009(e Evidence) evaluation {
	if e.EscapeReport == nil {
		return inconclusive("aucun fichier de verdicts d'échappement pour la toolchain de la campagne %s", e.Campaign.ID)
	}
	files := []string{e.EscapePath}

	// Témoin de spécificité : sans cellules locales qui n'échappent pas, un échappement partout
	// ne prouverait rien.
	witness := 0
	for _, verdict := range e.EscapeReport.Verdicts {
		cell, known := e.Matrix.Cell(verdict.CellID)
		if !known || cell.Profile != models.ProfileLocal || cell.PassingMode != models.PassingPointer {
			continue
		}
		if verdict.Status == models.EscapeStatusOK && !verdict.Escapes {
			witness++
		}
	}
	if witness < 2 {
		return inconclusive("témoin manquant : %d cellule(s) LOCAL/POINTER au statut OK n'échappent pas, deux sont exigées", witness)
	}

	var details []string
	refuting := ""
	for _, profile := range containerProfiles() {
		sizes := map[int]bool{}
		escaping, notEscaping := 0, map[int]bool{}
		for _, verdict := range e.EscapeReport.Verdicts {
			cell, known := e.Matrix.Cell(verdict.CellID)
			if !known || cell.Profile != profile || cell.PassingMode != models.PassingPointer || verdict.Status != models.EscapeStatusOK {
				continue
			}
			sizes[cell.TypeSpec.SizeBytes] = true
			if verdict.Escapes {
				escaping++
			} else {
				notEscaping[cell.TypeSpec.SizeBytes] = true
			}
		}
		if len(sizes) < 2 {
			return inconclusive("le profil %s ne fournit que %d taille(s) de cellule POINTER au statut OK, deux sont exigées", profile, len(sizes))
		}
		details = append(details, fmt.Sprintf("%s : %d échappent, %d taille(s) n'échappent pas", profile, escaping, len(notEscaping)))
		if len(notEscaping) >= 2 && refuting == "" {
			refuting = string(profile)
		}
	}
	if refuting != "" {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le conteneur %s ne fait pas échapper la locale sur au moins deux tailles distinctes (%s)", refuting, join(details)),
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: "les trois conteneurs font échapper la locale (" + join(details) + ")",
		Files:     files,
	}
}

// returnedFamily indique si un profil appartient à la famille des retours, sur laquelle porte
// H-010. Le profil d'origine `RETURNED` n'y produit jamais de base non nulle : seule la variante
// qui accompagne chaque instance d'une charge allouée fournit des paires éligibles.
func returnedFamily(profile models.LifetimeProfile) bool {
	return strings.HasPrefix(string(profile), string(models.ProfileReturned))
}

// pairAllocs porte une paire éligible de H-010 : son étiquette, ses deux médianes d'allocations et
// les fichiers de mesure dont elles viennent.
type pairAllocs struct {
	label   string
	value   float64
	pointer float64
	files   []string
}

// evaluateH010 — le doublement des allocations sur une base non nulle qui varie.
func evaluateH010(e Evidence) evaluation {
	var eligible []pairAllocs
	for _, pair := range e.Matrix.ValuePointerPairs() {
		value, pointer := pair[0], pair[1]
		if !returnedFamily(value.Profile) {
			continue
		}
		valueM, okValue := completeMeasurement(e, value.ID())
		pointerM, okPointer := completeMeasurement(e, pointer.ID())
		if !okValue || !okPointer || !constantAllocs(valueM) || !constantAllocs(pointerM) {
			continue
		}
		if valueM.MedianAllocs() < 1 {
			continue
		}
		eligible = append(eligible, pairAllocs{
			label:   fmt.Sprintf("%d o ×%d charge %d", value.TypeSpec.SizeBytes, value.Repetitions(), value.Payloads()),
			value:   valueM.MedianAllocs(),
			pointer: pointerM.MedianAllocs(),
			files:   []string{e.MeasurementPaths[value.ID()], e.MeasurementPaths[pointer.ID()]},
		})
	}
	if len(eligible) < 4 {
		return inconclusive("%d paire(s) éligible(s) dans la campagne %s, quatre sont exigées : une base d'allocation non nulle est nécessaire",
			len(eligible), e.Campaign.ID)
	}
	bases := map[float64]bool{}
	for _, p := range eligible {
		bases[p.value] = true
	}
	if len(bases) < 2 {
		return inconclusive("les %d paires éligibles n'offrent qu'une seule base d'allocation : un doublement ne se distingue pas d'un terme additif", len(eligible))
	}
	low, high := 0.0, 0.0
	for base := range bases {
		if low == 0 || base < low {
			low = base
		}
		if base > high {
			high = base
		}
	}
	if high < 4*low {
		return inconclusive("bases d'allocation de %.0f à %.0f : l'étalement est inférieur au facteur quatre exigé", low, high)
	}

	var files []string
	var offending, conforming []pairAllocs
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].value < eligible[j].value })
	for _, p := range eligible {
		files = append(files, p.files...)
		if p.pointer != 2*p.value {
			offending = append(offending, p)
		} else {
			conforming = append(conforming, p)
		}
	}
	if len(offending) > 0 {
		return evaluation{
			Outcome: models.OutcomeRefuted,
			Rationale: fmt.Sprintf("%d des %d paires éligibles ne doublent pas : %s ; les %d autres doublent : %s",
				len(offending), len(eligible), summarizeAllocPairs(offending),
				len(conforming), summarizeAllocPairs(conforming)),
			Files: dedupe(files),
		}
	}
	return evaluation{
		Outcome: models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("les %d paires éligibles doublent exactement, sur %d bases distinctes : %s",
			len(eligible), len(bases), summarizeAllocPairs(eligible)),
		Files: dedupe(files),
	}
}

// summarizeAllocPairs regroupe les paires par couple de bases d'allocation. Énumérer les quatre-
// vingts paires d'une campagne à plusieurs répétitions et plusieurs charges rendait la rationale
// illisible au tableau de bord, et surtout muette sur ce qui compte : le rapport ne dépend que du
// couple, pas de la taille de la structure. Chaque groupe cite une paire, ce qui suffit à remonter
// aux mesures, le champ resultFiles portant de toute façon tous les fichiers.
func summarizeAllocPairs(pairs []pairAllocs) string {
	type bases struct{ value, pointer float64 }
	var order []bases
	labels := map[bases][]string{}
	for _, p := range pairs {
		key := bases{p.value, p.pointer}
		if _, seen := labels[key]; !seen {
			order = append(order, key)
		}
		labels[key] = append(labels[key], p.label)
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].value != order[j].value {
			return order[i].value < order[j].value
		}
		return order[i].pointer < order[j].pointer
	})
	out := make([]string, 0, len(order))
	for _, key := range order {
		group := labels[key]
		sort.Strings(group)
		cite := group[0]
		if len(group) > 1 {
			cite = fmt.Sprintf("%s et %d autres", group[0], len(group)-1)
		}
		out = append(out, fmt.Sprintf("%.0f → %.0f allocs/op sur %d paire(s) dont %s",
			key.value, key.pointer, len(group), cite))
	}
	return join(out)
}

// constantAllocs indique que le compte d'allocations n'a pas varié d'une répétition à l'autre :
// la médiane est alors un entier exact et non une valeur bruitée.
func constantAllocs(m models.Measurement) bool {
	if len(m.AllocsPerOp) == 0 {
		return false
	}
	for _, v := range m.AllocsPerOp {
		if v != m.AllocsPerOp[0] {
			return false
		}
	}
	return true
}

// abs rend la valeur absolue d'un flottant.
// suffixList rend une énumération entre parenthèses, ou la chaîne vide si elle est vide.
func suffixList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return " (" + strings.Join(values, ", ") + ")"
}
