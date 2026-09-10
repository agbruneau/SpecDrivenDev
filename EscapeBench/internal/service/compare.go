package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// BootstrapResamples est le nombre de rééchantillonnages du bootstrap percentile. La méthode et
// ce paramètre sont consignés dans chaque fichier de comparaison (UC-004, notes de revue).
const BootstrapResamples = 2000

// ComparisonMethod décrit la méthode d'estimation, écrite dans le fichier de comparaison.
const ComparisonMethod = "bootstrap percentile de la différence des médianes, 2000 rééchantillonnages, IC 95 %, graine dérivée des identifiants de la paire"

// ComparisonReport est ce que UC-004 rend observable (étape 7).
type ComparisonReport struct {
	CampaignID    string
	Path          string
	Set           models.ComparisonSet
	ExcludedCount int
	NotObserved   []models.TippingKey
}

// Comparator met en œuvre UC-004 Comparer valeur et pointeur.
type Comparator struct {
	repo     ports.MatrixRepository
	store    ports.CampaignStore
	digester ports.Digester
	clock    ports.Clock
}

// NewComparator câble le service UC-004.
func NewComparator(repo ports.MatrixRepository, store ports.CampaignStore, digester ports.Digester, clock ports.Clock) *Comparator {
	return &Comparator{repo: repo, store: store, digester: digester, clock: clock}
}

// Compare exécute UC-004 sur une campagne complétée.
//
// UC-004 Comparer valeur et pointeur — étapes 1 à 7.
func (c *Comparator) Compare(ctx context.Context, campaignID string) (ComparisonReport, error) {
	campaign, err := c.store.LoadCampaign(ctx, campaignID)
	if err != nil {
		return ComparisonReport{}, err
	}
	// Étape 2 et A1 : la campagne doit être COMPLETED.
	if campaign.Status != models.CampaignCompleted {
		return ComparisonReport{}, fmt.Errorf("%w : la campagne %s est au statut %s", ErrPrecondition, campaignID, campaign.Status)
	}
	matrix, err := c.repo.Load(ctx, campaign.MatrixID)
	if err != nil {
		return ComparisonReport{}, err
	}
	if campaign.HarnessDigest != matrix.HarnessDigest {
		return ComparisonReport{}, fmt.Errorf("%w : campagne %s (%s) et matrice %s (%s)",
			ErrHarnessChanged, campaign.ID, campaign.HarnessDigest, matrix.ID, matrix.HarnessDigest)
	}
	measurements, err := c.store.LoadMeasurements(ctx, campaignID)
	if err != nil {
		return ComparisonReport{}, err
	}
	bySubject := make(map[string]models.Measurement, len(measurements))
	for _, m := range measurements {
		bySubject[m.SubjectID] = m
	}

	// Étape 3 : appariement des cellules VALUE et POINTER.
	pairs := matrix.ValuePointerPairs()
	if len(pairs) == 0 {
		return ComparisonReport{}, fmt.Errorf("%w : la matrice %s ne contient aucune paire (VALUE, POINTER)", ErrPrecondition, matrix.ID)
	}

	set := models.ComparisonSet{
		CampaignID: campaignID, MatrixID: matrix.ID, ComputedAt: c.clock.Now(), Method: ComparisonMethod,
		TippingPoints: map[models.TippingKey]int{}, ExcludedPairs: []models.ExcludedPair{},
	}
	for _, pair := range pairs {
		value, pointer := pair[0], pair[1]
		// A2 : paire incomplète — exclusion consignée avec sa raison (BR-004-1).
		valueM, okValue := bySubject[value.ID()]
		pointerM, okPointer := bySubject[pointer.ID()]
		if reason := pairProblem(value.ID(), pointer.ID(), valueM, okValue, pointerM, okPointer); reason != "" {
			set.ExcludedPairs = append(set.ExcludedPairs,
				models.ExcludedPair{ValueCellID: value.ID(), PointerCellID: pointer.ID(), Reason: reason})
			continue
		}
		// Étape 4 : delta des médianes, intervalle de confiance et drapeau significant (BR-004-2).
		set.Comparisons = append(set.Comparisons, compare(value, pointer, valueM, pointerM))
	}
	if len(set.Comparisons) == 0 {
		return ComparisonReport{}, fmt.Errorf("%w : aucune paire complète dans la campagne %s", ErrPrecondition, campaignID)
	}

	// Étape 5 et A3 : point de bascule par couple (profil, champ pointeur).
	set.TippingPoints = TippingPoints(set.Comparisons)

	// Étape 6 : écriture d'un nouveau fichier horodaté (BR-004-3).
	path, err := c.store.WriteComparisonSet(ctx, set)
	if err != nil {
		return ComparisonReport{}, err
	}
	var notObserved []models.TippingKey
	for _, key := range sortedKeys(set.TippingPoints) {
		if set.TippingPoints[key] == models.TippingNotObserved {
			notObserved = append(notObserved, key)
		}
	}
	return ComparisonReport{CampaignID: campaignID, Path: path, Set: set,
		ExcludedCount: len(set.ExcludedPairs), NotObserved: notObserved}, nil
}

// pairProblem rend la raison d'exclusion d'une paire, ou la chaîne vide si elle est complète.
func pairProblem(valueID, pointerID string, valueM models.Measurement, okValue bool,
	pointerM models.Measurement, okPointer bool) string {
	switch {
	case !okValue && !okPointer:
		return "aucune mesure pour " + valueID + " ni " + pointerID
	case !okValue:
		return "aucune mesure pour " + valueID
	case !okPointer:
		return "aucune mesure pour " + pointerID
	case valueM.Status != models.MeasurementComplete:
		return valueID + " est FAILED : " + valueM.FailureReason
	case pointerM.Status != models.MeasurementComplete:
		return pointerID + " est FAILED : " + pointerM.FailureReason
	}
	return ""
}

// compare calcule la Comparison d'une paire complète.
func compare(value, pointer models.Cell, valueM, pointerM models.Measurement) models.Comparison {
	medianValue := valueM.MedianNs()
	medianPointer := pointerM.MedianNs()
	low, high := bootstrapDeltaCI(valueM.NsPerOp, pointerM.NsPerOp, seedFor(value.ID(), pointer.ID()))
	return models.Comparison{
		CampaignID:      valueM.CampaignID,
		ValueCellID:     value.ID(),
		PointerCellID:   pointer.ID(),
		SizeBytes:       value.TypeSpec.SizeBytes,
		HasPointerField: value.TypeSpec.HasPointerField,
		Profile:         value.Profile,
		DeltaNsPerOp:    medianPointer - medianValue,
		CILow:           low,
		CIHigh:          high,
		// BR-004-2 : significant est vrai si et seulement si l'intervalle exclut zéro.
		Significant:     low > 0 || high < 0,
		MedianValueNs:   medianValue,
		MedianPointerNs: medianPointer,
	}
}

// seedFor dérive une graine déterministe des identifiants d'une paire : deux exécutions de UC-004
// sur la même campagne produisent le même intervalle.
func seedFor(valueID, pointerID string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(valueID + "\x00" + pointerID))
	return h.Sum64()
}

// bootstrapDeltaCI estime l'intervalle de confiance à 95 % de la différence des médianes
// (médiane pointeur − médiane valeur) par bootstrap percentile.
func bootstrapDeltaCI(value, pointer []float64, seed uint64) (low, high float64) {
	if len(value) == 0 || len(pointer) == 0 {
		return 0, 0
	}
	generator := rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))
	deltas := make([]float64, 0, BootstrapResamples)
	valueSample := make([]float64, len(value))
	pointerSample := make([]float64, len(pointer))
	for i := 0; i < BootstrapResamples; i++ {
		for j := range valueSample {
			valueSample[j] = value[generator.IntN(len(value))]
		}
		for j := range pointerSample {
			pointerSample[j] = pointer[generator.IntN(len(pointer))]
		}
		deltas = append(deltas, models.MedianFloat(pointerSample)-models.MedianFloat(valueSample))
	}
	sort.Float64s(deltas)
	return percentile(deltas, 0.025), percentile(deltas, 0.975)
}

// percentile rend le quantile d'un échantillon déjà trié.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(p * float64(len(sorted)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// TippingPoints rend, pour chaque couple (profil, champ pointeur), la plus petite taille telle que
// pour cette taille et toutes les tailles supérieures mesurées du couple, la Comparison est
// significative et deltaNsPerOp est négatif ; TippingNotObserved sinon (UC-004, étape 5 et A3).
func TippingPoints(comparisons []models.Comparison) map[models.TippingKey]int {
	bySeries := map[models.TippingKey][]models.Comparison{}
	for _, comparison := range comparisons {
		key := models.TippingKey{Profile: comparison.Profile, HasPointerField: comparison.HasPointerField}
		bySeries[key] = append(bySeries[key], comparison)
	}
	out := make(map[models.TippingKey]int, len(bySeries))
	for key, series := range bySeries {
		sort.Slice(series, func(i, j int) bool { return series[i].SizeBytes < series[j].SizeBytes })
		tipping := models.TippingNotObserved
		// Parcours descendant : la bascule est le début du plus long suffixe entièrement favorable
		// au pointeur.
		for i := len(series) - 1; i >= 0; i-- {
			if series[i].Significant && series[i].DeltaNsPerOp < 0 {
				tipping = series[i].SizeBytes
				continue
			}
			break
		}
		out[key] = tipping
	}
	return out
}

// sortedKeys rend les clés de points de bascule dans un ordre stable.
func sortedKeys(m map[models.TippingKey]int) []models.TippingKey {
	keys := make([]models.TippingKey, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Profile != keys[j].Profile {
			return keys[i].Profile < keys[j].Profile
		}
		return !keys[i].HasPointerField && keys[j].HasPointerField
	})
	return keys
}
