package service

import (
	"fmt"

	"github.com/agbruneau/escapebench/internal/models"
)

// Évaluateur de H-013, l'hypothèse que C-010 rend mesurable. Il applique le texte gelé de
// `docs/requirements.md` et rien d'autre. La différence avec H-008 tient en deux points, et
// l'implémentation les suit à la lettre : les trois bandes sont pincées des deux côtés, et le
// verdict est adossé à une attestation de quiétude extérieure aux sondes.

// LastLevelRatioMin est le rapport minimal entre le dernier niveau de cache et le cache L1 de
// données. Un rapport moindre signale qu'un niveau n'a pas été détecté, et que la bande dite non
// résidente serait encore servie par un cache.
const LastLevelRatioMin = 64

// CacheHitCeilingNs est la latence qu'annonce BEPG p. 254 pour un succès de cache. Une sonde
// résidente plus lente ne mesure pas le succès dont le livre parle.
const CacheHitCeilingNs = 10

// h013Band nomme une bande de résidence et les Probe qui y tombent.
type h013Band struct {
	name       string
	parameters []int
}

// evaluateH013 — la latence d'un accès dépendant non résident dépasse 100 ns et ne vaut pas plus de
// 200 fois celle d'un accès résident, sur une machine dont la quiétude est attestée.
func evaluateH013(e Evidence) evaluation {
	provenance := e.Campaign.Provenance
	if provenance.L1DataCacheBytes <= 0 || provenance.LastLevelCacheBytes <= 0 {
		return inconclusive("la provenance de %s ne porte pas les tailles de cache : l1DataCacheBytes = %d, lastLevelCacheBytes = %d",
			e.Campaign.ID, provenance.L1DataCacheBytes, provenance.LastLevelCacheBytes)
	}
	if provenance.LastLevelCacheBytes < LastLevelRatioMin*provenance.L1DataCacheBytes {
		return inconclusive("dernier niveau de cache %d octets pour un L1 de données de %d : le rapport est inférieur à %d, un niveau n'a probablement pas été détecté et la bande non résidente serait encore servie par un cache",
			provenance.LastLevelCacheBytes, provenance.L1DataCacheBytes, LastLevelRatioMin)
	}

	// Les trois bandes sont pincées des deux côtés : un paramètre libre sur un large intervalle
	// déciderait seul du verdict.
	resident := h013Band{name: fmt.Sprintf("résidente (≤ %d octets)", provenance.L1DataCacheBytes/2)}
	intermediate := h013Band{name: fmt.Sprintf("intermédiaire (de %d à %d octets)",
		2*provenance.L1DataCacheBytes, 8*provenance.L1DataCacheBytes)}
	nonResident := h013Band{name: fmt.Sprintf("non résidente (de %d à %d octets)",
		4*provenance.LastLevelCacheBytes, 16*provenance.LastLevelCacheBytes)}

	for _, subjectID := range sortedSubjectIDs(e.Measurements) {
		kind, parameter, ok := models.ParseProbeID(subjectID)
		if !ok || kind != models.ProbePointerChase {
			continue
		}
		if _, complete := completeMeasurement(e, subjectID); !complete {
			continue
		}
		size := int64(parameter)
		switch {
		case size <= provenance.L1DataCacheBytes/2:
			resident.parameters = append(resident.parameters, parameter)
		case size >= 2*provenance.L1DataCacheBytes && size <= 8*provenance.L1DataCacheBytes:
			intermediate.parameters = append(intermediate.parameters, parameter)
		case size >= 4*provenance.LastLevelCacheBytes && size <= 16*provenance.LastLevelCacheBytes:
			nonResident.parameters = append(nonResident.parameters, parameter)
		}
	}
	for _, band := range []h013Band{resident, intermediate, nonResident} {
		if len(band.parameters) != 1 {
			return inconclusive("bande %s de %s : %d sonde %s au statut COMPLETE, exactement une est exigée",
				band.name, e.Campaign.ID, len(band.parameters), models.ProbePointerChase)
		}
	}

	residentID := probeIDOf(resident.parameters[0])
	intermediateID := probeIDOf(intermediate.parameters[0])
	nonResidentID := probeIDOf(nonResident.parameters[0])
	residentM, _ := completeMeasurement(e, residentID)
	intermediateM, _ := completeMeasurement(e, intermediateID)
	nonResidentM, _ := completeMeasurement(e, nonResidentID)
	m1, m2, m3 := residentM.MedianNs(), intermediateM.MedianNs(), nonResidentM.MedianNs()
	files := []string{e.MeasurementPaths[residentID], e.MeasurementPaths[intermediateID], e.MeasurementPaths[nonResidentID]}

	if m1 <= 0 || m1 >= CacheHitCeilingNs {
		return inconclusive("latence résidente %.4f ns : elle doit être strictement positive et sous les %d ns qu'annonce le livre pour un succès de cache",
			m1, CacheHitCeilingNs)
	}
	// Témoin d'ordonnancement : il n'exclut qu'un banc muet dont les trois sondes se vaudraient, et
	// ne dit rien de la charge de la machine, une contention gonflant les trois en conservant leur
	// ordre. C'est l'attestation de quiétude qui porte cette seconde garde.
	if !(m2 > m1 && m2 < m3) {
		return inconclusive("le banc ne résout pas la hiérarchie mémoire : résidente %.4f ns, intermédiaire %.4f ns, non résidente %.4f ns, l'ordre strict est exigé",
			m1, m2, m3)
	}

	// Attestation de quiétude (C-010), sur la fenêtre de chacune des trois sondes.
	for _, id := range []string{residentID, intermediateID, nonResidentID} {
		m := e.Measurements[id]
		if !m.QuietudeMeasured {
			return inconclusive("la mesure de %s ne porte pas d'attestation de quiétude : sans elle une infirmation ne se distingue pas d'un artefact de contention (C-010)", id)
		}
		if m.QuietudeOccupancy > models.QuietudeThreshold {
			return inconclusive("machine occupée à %.1f %% hors du sujet pendant la fenêtre de %s, le seuil de C-010 est %.1f %% : la latence relevée n'est pas imputable à la seule hiérarchie mémoire",
				100*m.QuietudeOccupancy, id, 100*models.QuietudeThreshold)
		}
	}

	ratio := m3 / m1
	detail := fmt.Sprintf("résident %d o : %.2f ns/accès ; intermédiaire %d o : %.2f ns/accès ; non résident %d o : %.2f ns/accès ; rapport ×%.1f ; occupation %s",
		resident.parameters[0], m1, intermediate.parameters[0], m2, nonResident.parameters[0], m3, ratio,
		quietudeDetail(e, residentID, intermediateID, nonResidentID))

	if m3 <= CacheMissFloorNs {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("la latence non résidente ne dépasse pas %d ns — %s", CacheMissFloorNs, detail),
			Files:     files,
		}
	}
	if ratio > CacheRatioMax {
		return evaluation{
			Outcome:   models.OutcomeRefuted,
			Rationale: fmt.Sprintf("le rapport dépasse %d — %s", CacheRatioMax, detail),
			Files:     files,
		}
	}
	return evaluation{
		Outcome:   models.OutcomeConfirmed,
		Rationale: fmt.Sprintf("la latence non résidente dépasse %d ns et le rapport ne dépasse pas %d, machine attestée au repos — %s", CacheMissFloorNs, CacheRatioMax, detail),
		Files:     files,
	}
}

// CacheMissFloorNs et CacheRatioMax sont les deux ancres de BEPG p. 254 que H-013 reprend.
const (
	CacheMissFloorNs = 100
	CacheRatioMax    = 200
)

// probeIDOf rend l'identifiant d'une Probe POINTER_CHASE de paramètre donné.
func probeIDOf(parameter int) string {
	return models.Probe{Kind: models.ProbePointerChase, Parameter: parameter}.ID()
}

// quietudeDetail met en forme les trois fractions d'occupation relevées, que le critère exige de
// consigner.
func quietudeDetail(e Evidence, ids ...string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%.1f %%", 100*e.Measurements[id].QuietudeOccupancy))
	}
	return join(parts)
}
