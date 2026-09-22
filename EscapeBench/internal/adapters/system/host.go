package system

import (
	"sort"
	"strconv"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
)

// Host rassemble l'état de la machine que la Provenance consigne depuis le 2026-09-22 (D-60, E-14).
// Chaque champ est vide quand la plateforme ne l'expose pas : le banc constate, il ne suppose rien
// et ne configure rien.
type Host struct {
	OSVersion   string
	PowerPlan   string
	CPUAffinity string
	CoreTypes   models.CoreTypes
}

// Unpinned est la valeur de cpuAffinity quand le masque couvre tous les processeurs de la machine.
const Unpinned = "non épinglé"

// parseCPUList lit une liste de processeurs au format du noyau Linux (« 0-3,8,10-11 ») et rend les
// numéros triés, sans doublon. Une liste illisible rend nil : l'appelant laisse alors le champ vide.
func parseCPUList(list string) []int {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	seen := map[int]bool{}
	for _, part := range strings.Split(list, ",") {
		lo, hi, isRange := strings.Cut(strings.TrimSpace(part), "-")
		first, err := strconv.Atoi(lo)
		if err != nil || first < 0 {
			return nil
		}
		last := first
		if isRange {
			if last, err = strconv.Atoi(hi); err != nil || last < first {
				return nil
			}
		}
		for cpu := first; cpu <= last; cpu++ {
			seen[cpu] = true
		}
	}
	out := make([]int, 0, len(seen))
	for cpu := range seen {
		out = append(out, cpu)
	}
	sort.Ints(out)
	return out
}

// coreTypesFrom réduit un décompte de cœurs par classe d'efficacité (plus la classe est haute, plus
// le cœur est rapide) aux deux nombres consignés : les cœurs de la classe la plus haute sont de
// performance, tous les autres d'efficacité. Une seule classe rend la valeur nulle : un processeur
// homogène ne distingue pas ses cœurs.
func coreTypesFrom(coresByClass map[int]int) models.CoreTypes {
	if len(coresByClass) < 2 {
		return models.CoreTypes{}
	}
	top := -1
	for class := range coresByClass {
		if class > top {
			top = class
		}
	}
	var c models.CoreTypes
	for class, n := range coresByClass {
		if class == top {
			c.Performance += n
		} else {
			c.Efficiency += n
		}
	}
	return c
}

// osReleaseName extrait PRETTY_NAME d'un contenu /etc/os-release.
func osReleaseName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "PRETTY_NAME="); ok {
			return strings.Trim(value, `"'`)
		}
	}
	return ""
}
