package service

import (
	"fmt"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
)

// evaluators associe chaque hypothèse de docs/requirements.md à l'évaluation mécanique de son
// critère de réfutation. Le texte du critère reste la référence : son empreinte est gelée à la
// création de la campagne (BR-003-5) et UC-005 refuse tout verdict si elle a changé. Modifier un
// critère oblige donc à créer une nouvelle H-### et à écrire son évaluateur ici.
var evaluators = map[string]evaluator{
	"H-001": evaluateH001,
	"H-002": evaluateH002,
	"H-003": evaluateH003,
	"H-004": evaluateH004,
	"H-005": evaluateH005,
	"H-006": evaluateH006,
	"H-007": evaluateH007,
	"H-008": evaluateH008,
	"H-009": evaluateH009,
	"H-010": evaluateH010,
	"H-011": evaluateH011,
}

// SmallStructBytes est la borne « 1 à 3 mots machine » de BEPG p. 253, en octets sur 64 bits.
const SmallStructBytes = models.SmallStructBytes

// inconclusive construit une évaluation non concluante (UC-005, A2 et étape 5).
func inconclusive(format string, args ...any) evaluation {
	return evaluation{Outcome: models.OutcomeInconclusive, Rationale: fmt.Sprintf(format, args...)}
}

// evaluateH001 — « copier une petite struct peut être moins cher que passer un pointeur ».
// Infirmée si, pour au moins deux tailles ≤ 24 octets sans champ pointeur en profil LOCAL, la
// Comparison est significative avec ciHigh < 0.
//
// Les Comparison sont restreintes à la disposition ARRAY_FILL : c'est la seule qui existait quand
// le critère a été gelé, et la seule qu'il désigne donc. Une campagne à plusieurs dispositions
// verserait sinon dans le même décompte les paires NAMED_FIELDS et le témoin nul, dont l'écart est
// nul par construction, et évaluerait le critère gelé sur un corpus qu'il ne nomme pas.
func evaluateH001(e Evidence) evaluation {
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	var eligible, refuting []models.Comparison
	for _, comparison := range e.ComparisonSet.Comparisons {
		if comparison.Profile != models.ProfileLocal || comparison.HasPointerField || comparison.SizeBytes > SmallStructBytes {
			continue
		}
		if comparison.EffectiveLayout() != models.LayoutArrayFill {
			continue
		}
		eligible = append(eligible, comparison)
		if comparison.Significant && comparison.CIHigh < 0 {
			refuting = append(refuting, comparison)
		}
	}
	if len(eligible) == 0 {
		return inconclusive("aucune paire LOCAL %s sans champ pointeur de taille ≤ %d octets dans la campagne %s",
			models.LayoutArrayFill, SmallStructBytes, e.Campaign.ID)
	}
	files := []string{e.ComparisonPath}
	if len(refuting) >= 2 {
		return evaluation{
			Outcome: models.OutcomeRefuted,
			Rationale: fmt.Sprintf("%d paires ≤ %d octets ont significant vrai et ciHigh < 0 : %s (sur %d paires éligibles)",
				len(refuting), SmallStructBytes, join(pairLabels(refuting)), len(eligible)),
			Files: files,
		}
	}
	return evaluation{
		Outcome: models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("%d paire(s) sur %d ont significant vrai et ciHigh < 0 ; le critère exige au moins 2 : %s",
			len(refuting), len(eligible), join(pairLabels(eligible))),
		Files: files,
	}
}

// evaluateH002 — le point de bascule est supérieur à 24 octets pour le profil LOCAL.
// Infirmée si l'une des deux séries LOCAL a un point de bascule ≤ 24 octets ; « non observé »
// n'infirme pas.
//
// « L'une des deux séries » est le texte gelé : ce sont les deux séries ARRAY_FILL, avec et sans
// champ pointeur. Les séries des dispositions ajoutées par C-008 relèvent de H-007, pas d'ici.
func evaluateH002(e Evidence) evaluation {
	if e.ComparisonSet == nil {
		return inconclusive("aucun fichier de comparaison pour la campagne %s ; exécuter `escapebench compare`", e.Campaign.ID)
	}
	var observed []string
	var refuting []string
	found := false
	for _, hasPointerField := range []bool{false, true} {
		key := models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill, HasPointerField: hasPointerField}
		size, ok := e.ComparisonSet.TippingPoints[key]
		if !ok {
			continue
		}
		found = true
		label := seriesLabel(hasPointerField)
		if size == models.TippingNotObserved {
			observed = append(observed, label+" : non observé")
			continue
		}
		observed = append(observed, fmt.Sprintf("%s : %d octets", label, size))
		if size <= SmallStructBytes {
			refuting = append(refuting, fmt.Sprintf("%s à %d octets", label, size))
		}
	}
	if !found {
		return inconclusive("aucun point de bascule LOCAL %s dans le fichier de comparaison de %s",
			models.LayoutArrayFill, e.Campaign.ID)
	}
	files := []string{e.ComparisonPath}
	if len(refuting) > 0 {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("point de bascule ≤ %d octets pour %s (séries : %s)", SmallStructBytes, join(refuting), join(observed)),
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("aucun point de bascule LOCAL ≤ %d octets (séries : %s)", SmallStructBytes, join(observed)),
		Files:     files,
	}
}

// evaluateH003 — le passage au pointeur double les allocations en profil RETURNED.
// Infirmée si le rapport des médianes allocsPerOp pointeur / valeur reste < 2 pour toutes les
// tailles. Une paire dont la médiane valeur est 0 compte comme rapport ≥ 2 si la médiane pointeur
// est ≥ 1.
func evaluateH003(e Evidence) evaluation {
	var files []string
	var details []string
	confirming := 0
	pairs := 0
	for _, pair := range e.Matrix.ValuePointerPairs() {
		value, pointer := pair[0], pair[1]
		if value.Profile != models.ProfileReturned {
			continue
		}
		valueM, okValue := completeMeasurement(e, value.ID())
		pointerM, okPointer := completeMeasurement(e, pointer.ID())
		if !okValue || !okPointer {
			continue
		}
		pairs++
		files = append(files, e.MeasurementPaths[value.ID()], e.MeasurementPaths[pointer.ID()])
		medianValue := valueM.MedianAllocs()
		medianPointer := pointerM.MedianAllocs()
		doubled := false
		switch {
		case medianValue == 0:
			doubled = medianPointer >= 1
		default:
			doubled = medianPointer/medianValue >= 2
		}
		details = append(details, fmt.Sprintf("%d o%s : %.0f → %.0f allocs/op",
			value.TypeSpec.SizeBytes, pointerSuffix(value.TypeSpec.HasPointerField), medianValue, medianPointer))
		if doubled {
			confirming++
		}
	}
	if pairs == 0 {
		return inconclusive("aucune paire RETURNED complète dans la campagne %s", e.Campaign.ID)
	}
	sort.Strings(files)
	if confirming == 0 {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le rapport des médianes allocs/op reste < 2 sur les %d paires RETURNED : %s", pairs, join(details)),
			Files:     dedupe(files),
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("%d paire(s) RETURNED sur %d atteignent un facteur ≥ 2 : %s", confirming, pairs, join(details)),
		Files:     dedupe(files),
	}
}

// L2Threshold est le jeu de travail au-delà duquel le critère de H-004 s'applique : 32 MiB.
const L2Threshold = 32 * 1024 * 1024

// evaluateH004 — le rapport dispersé/séquentiel se situe dans [10, 200] au-delà du cache L2.
// Infirmée si, pour toutes les Probe de jeu de travail ≥ 32 MiB, le rapport est hors de [10, 200].
func evaluateH004(e Evidence) evaluation {
	sequential := probeMedians(e, models.ProbeSequentialScan)
	scattered := probeMedians(e, models.ProbeScatteredScan)
	var files, details []string
	inRange, total := 0, 0
	var parameters []int
	for parameter := range sequential {
		if parameter >= L2Threshold {
			parameters = append(parameters, parameter)
		}
	}
	sort.Ints(parameters)
	for _, parameter := range parameters {
		scatteredNs, ok := scattered[parameter]
		if !ok || sequential[parameter] == 0 {
			continue
		}
		total++
		ratio := scatteredNs / sequential[parameter]
		details = append(details, fmt.Sprintf("%d MiB : ×%.1f", parameter/(1024*1024), ratio))
		if ratio >= 10 && ratio <= 200 {
			inRange++
		}
		files = append(files,
			e.MeasurementPaths[models.Probe{Kind: models.ProbeSequentialScan, Parameter: parameter}.ID()],
			e.MeasurementPaths[models.Probe{Kind: models.ProbeScatteredScan, Parameter: parameter}.ID()])
	}
	if total == 0 {
		return inconclusive("aucune paire de sondes de parcours ≥ %d MiB complète dans la campagne %s",
			L2Threshold/(1024*1024), e.Campaign.ID)
	}
	if inRange == 0 {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le rapport dispersé/séquentiel est hors de [10, 200] pour les %d jeux de travail ≥ %d MiB : %s", total, L2Threshold/(1024*1024), join(details)),
			Files:     dedupe(files),
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("%d jeu(x) de travail sur %d ont un rapport dans [10, 200] : %s", inRange, total, join(details)),
		Files:     dedupe(files),
	}
}

// AppendProbeSize est le paramètre des sondes d'append visé par H-005.
const AppendProbeSize = 100000

// evaluateH005 — la préallocation divise le temps et la mémoire par au moins quatre.
// Infirmée si l'un des deux rapports de médianes PREALLOC / GROW dépasse 1/2.
func evaluateH005(e Evidence) evaluation {
	prealloc := models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID()
	grow := models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID()
	preallocM, okPrealloc := completeMeasurement(e, prealloc)
	growM, okGrow := completeMeasurement(e, grow)
	if !okPrealloc || !okGrow {
		return inconclusive("sondes d'append n = %d incomplètes dans la campagne %s", AppendProbeSize, e.Campaign.ID)
	}
	if growM.MedianNs() == 0 || growM.MedianBytes() == 0 {
		return inconclusive("médianes nulles pour %s : rapport indéfini", grow)
	}
	nsRatio := preallocM.MedianNs() / growM.MedianNs()
	bytesRatio := preallocM.MedianBytes() / growM.MedianBytes()
	files := []string{e.MeasurementPaths[prealloc], e.MeasurementPaths[grow]}
	detail := fmt.Sprintf("ns/op ×%.3f, B/op ×%.3f (n = %d)", nsRatio, bytesRatio, AppendProbeSize)
	if nsRatio > 0.5 || bytesRatio > 0.5 {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: "un rapport PREALLOC / GROW dépasse 1/2 : " + detail,
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: "les deux rapports PREALLOC / GROW restent ≤ 1/2 : " + detail,
		Files:     files,
	}
}

// evaluateH006 — les quatre causes d'échappement du livre sont exhaustives.
// Infirmée si au moins une cellule échappe pour une raison classée hors des quatre catégories.
func evaluateH006(e Evidence) evaluation {
	if e.EscapeReport == nil {
		return inconclusive("aucun fichier de verdicts d'échappement pour la toolchain de la campagne %s", e.Campaign.ID)
	}
	var others []string
	escaping := 0
	for _, verdict := range e.EscapeReport.Verdicts {
		if verdict.Status != models.EscapeStatusOK || !verdict.Escapes {
			continue
		}
		escaping++
		if !verdict.Category.InBook() {
			others = append(others, verdict.CellID+" ("+verdict.CompilerReason+")")
		}
	}
	files := []string{e.EscapePath}
	if escaping == 0 {
		return inconclusive("aucune cellule n'échappe dans %s : le critère est inévaluable", e.EscapePath)
	}
	sort.Strings(others)
	if len(others) > 0 {
		shown := others
		if len(shown) > 5 {
			shown = shown[:5]
		}
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("%d cellule(s) sur %d échappent hors des quatre causes du livre : %s", len(others), escaping, join(shown)),
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("les %d cellules qui échappent se classent toutes dans les quatre causes du livre", escaping),
		Files:     files,
	}
}

// AppendTimeFactorFloor est le facteur de gain en temps minimal exigé par H-011 : la borne basse
// de l'« about 6× » du livre, à 20 % près.
const AppendTimeFactorFloor = 4.8

// Bornes du facteur de gain en mémoire de H-011, autour du « one-fifth » du livre, à 20 % près.
const (
	AppendMemoryFactorLow  = 4.0
	AppendMemoryFactorHigh = 6.0
)

// PreH011CampaignID désigne la campagne dont les mesures précèdent la rédaction du critère de
// H-011. Un critère écrit après les données qu'il évalue n'éprouve rien : cette campagne rend
// H-011 non concluante, par construction et non par accident.
const PreH011CampaignID = "C-2026-09-10-1"

// evaluateH011 — les trois chiffres de la préallocation annoncés par BEPG p. 114, aux tolérances
// que le livre s'accorde lui-même. Infirmée si le gain en temps passe sous 4,8, si le gain en
// mémoire sort de [4, 6], si la préallocation ne ramène pas les allocations à exactement une, ou
// si la version sans préallocation en compte moins de deux.
func evaluateH011(e Evidence) evaluation {
	if e.Campaign.ID == PreH011CampaignID {
		return inconclusive("les mesures de la campagne %s précèdent la rédaction du critère de H-011 : elles ne peuvent pas le mettre à l'épreuve", PreH011CampaignID)
	}
	prealloc := models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: AppendProbeSize}.ID()
	grow := models.Probe{Kind: models.ProbeAppendGrow, Parameter: AppendProbeSize}.ID()
	preallocM, okPrealloc := completeMeasurement(e, prealloc)
	growM, okGrow := completeMeasurement(e, grow)
	if !okPrealloc || !okGrow {
		return inconclusive("sondes d'append n = %d incomplètes dans la campagne %s", AppendProbeSize, e.Campaign.ID)
	}
	if preallocM.MedianNs() == 0 || preallocM.MedianBytes() == 0 {
		return inconclusive("médianes nulles pour %s : les facteurs sont indéfinis", prealloc)
	}
	timeFactor := growM.MedianNs() / preallocM.MedianNs()
	memoryFactor := growM.MedianBytes() / preallocM.MedianBytes()
	preallocAllocs := preallocM.MedianAllocs()
	growAllocs := growM.MedianAllocs()
	detail := fmt.Sprintf("temps ×%.2f, mémoire ×%.2f, allocations %.0f → %.0f (n = %d)",
		timeFactor, memoryFactor, growAllocs, preallocAllocs, AppendProbeSize)
	files := []string{e.MeasurementPaths[prealloc], e.MeasurementPaths[grow]}

	var breaches []string
	if timeFactor < AppendTimeFactorFloor {
		breaches = append(breaches, fmt.Sprintf("gain en temps ×%.2f sous le plancher de %.1f", timeFactor, AppendTimeFactorFloor))
	}
	if memoryFactor < AppendMemoryFactorLow || memoryFactor > AppendMemoryFactorHigh {
		breaches = append(breaches, fmt.Sprintf("gain en mémoire ×%.2f hors de [%.0f, %.0f]", memoryFactor, AppendMemoryFactorLow, AppendMemoryFactorHigh))
	}
	if preallocAllocs != 1 {
		breaches = append(breaches, fmt.Sprintf("%.0f allocation(s) avec préallocation au lieu d'une seule", preallocAllocs))
	}
	if growAllocs < 2 {
		breaches = append(breaches, fmt.Sprintf("%.0f allocation(s) sans préallocation, la base est trop faible pour parler de réduction", growAllocs))
	}
	if len(breaches) > 0 {
		return evaluation{Outcome: models.OutcomeRefuted, Rationale: join(breaches) + " — " + detail, Files: files}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: "les trois chiffres de la page 114 sont tenus : " + detail,
		Files:     files,
	}
}

// completeMeasurement rend la mesure d'un sujet si elle est complète.
func completeMeasurement(e Evidence, subjectID string) (models.Measurement, bool) {
	m, ok := e.Measurements[subjectID]
	if !ok || m.Status != models.MeasurementComplete {
		return models.Measurement{}, false
	}
	return m, true
}

// probeMedians rend, par paramètre, la médiane nsPerOp des sondes complètes d'un genre donné.
func probeMedians(e Evidence, kind models.ProbeKind) map[int]float64 {
	out := map[int]float64{}
	for _, subjectID := range sortedSubjectIDs(e.Measurements) {
		probeKind, parameter, ok := models.ParseProbeID(subjectID)
		if !ok || probeKind != kind {
			continue
		}
		if m, complete := completeMeasurement(e, subjectID); complete {
			out[parameter] = m.MedianNs()
		}
	}
	return out
}

// pairLabels rend une étiquette lisible par comparaison.
func pairLabels(comparisons []models.Comparison) []string {
	labels := make([]string, 0, len(comparisons))
	for _, comparison := range comparisons {
		labels = append(labels, fmt.Sprintf("%d o (Δ %.2f ns, IC [%.2f, %.2f])",
			comparison.SizeBytes, comparison.DeltaNsPerOp, comparison.CILow, comparison.CIHigh))
	}
	return labels
}

// seriesLabel nomme une série de points de bascule.
func seriesLabel(hasPointerField bool) string {
	if hasPointerField {
		return "avec champ pointeur"
	}
	return "sans champ pointeur"
}

// pointerSuffix marque la présence d'un champ pointeur dans un détail de rationale.
func pointerSuffix(hasPointerField bool) string {
	if hasPointerField {
		return " (champ pointeur)"
	}
	return ""
}

// dedupe retire les doublons et les chaînes vides d'une liste de chemins.
func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
