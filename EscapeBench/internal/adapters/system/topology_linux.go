//go:build linux

package system

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// parseLinuxCacheSize lit une taille de cache écrite par le noyau sous la forme « 48K », « 3M » ou
// un nombre d'octets. La fonction est pure et éprouvée sans accès disque.
func parseLinuxCacheSize(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	multiplier := int64(1)
	switch value[len(value)-1] {
	case 'K', 'k':
		multiplier = 1024
		value = value[:len(value)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		value = value[:len(value)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		value = value[:len(value)-1]
	}
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n * multiplier
}

// readLinuxCaches lit les instances de cache exposées par sysfs pour un processeur donné.
func readLinuxCaches(cpuDir string) []CacheLevel {
	entries, err := filepath.Glob(filepath.Join(cpuDir, "cache", "index*"))
	if err != nil {
		return nil
	}
	read := func(dir, name string) string {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(content))
	}
	var out []CacheLevel
	for _, dir := range entries {
		level, err := strconv.Atoi(read(dir, "level"))
		if err != nil {
			continue
		}
		kind := read(dir, "type")
		out = append(out, CacheLevel{
			Level:     level,
			Data:      kind == "Data" || kind == "Unified",
			SizeBytes: parseLinuxCacheSize(read(dir, "size")),
		})
	}
	return out
}

// detectTopology relève la hiérarchie de caches du premier processeur. Une lecture impossible rend
// une topologie vide : H-008 se déclare alors non concluante plutôt que de supposer une taille.
func detectTopology() Topology {
	return topologyFrom(readLinuxCaches("/sys/devices/system/cpu/cpu0"))
}
