package system

import "sort"

// CacheLevel décrit une instance de cache relevée sur la machine.
type CacheLevel struct {
	Level int
	// Data vaut vrai pour un cache de données ou unifié : un cache d'instructions ne sert pas
	// l'hypothèse qui compare des accès mémoire.
	Data      bool
	SizeBytes int64
}

// Topology rassemble ce que le banc doit connaître de la hiérarchie mémoire pour interpréter une
// hypothèse qui porte sur les caches (C-008). Une valeur nulle signifie « non détecté » : aucun
// verdict ne se rend sur une taille supposée.
type Topology struct {
	L1DataCacheBytes    int64
	LastLevelCacheBytes int64
}

// topologyFrom réduit les instances relevées aux deux tailles retenues : la plus grande instance
// de cache de données de niveau 1, qui est celle qu'un cœur voit, et la plus grande instance du
// niveau le plus élevé, qui est le dernier niveau avant la mémoire.
func topologyFrom(levels []CacheLevel) Topology {
	var t Topology
	highest := 0
	for _, c := range levels {
		if c.SizeBytes <= 0 || !c.Data {
			continue
		}
		if c.Level == 1 && c.SizeBytes > t.L1DataCacheBytes {
			t.L1DataCacheBytes = c.SizeBytes
		}
		if c.Level > highest {
			highest = c.Level
		}
	}
	if highest == 0 {
		return t
	}
	for _, c := range levels {
		if c.Data && c.Level == highest && c.SizeBytes > t.LastLevelCacheBytes {
			t.LastLevelCacheBytes = c.SizeBytes
		}
	}
	return t
}

// sortedLevels rend les niveaux relevés dans un ordre stable ; utilisé par les tests et les
// diagnostics.
func sortedLevels(levels []CacheLevel) []CacheLevel {
	out := append([]CacheLevel(nil), levels...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level < out[j].Level
		}
		return out[i].SizeBytes < out[j].SizeBytes
	})
	return out
}
