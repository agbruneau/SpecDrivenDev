//go:build linux

package system

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// detectHost relève l'état de la machine sous Linux (D-60). Chaque lecture qui échoue laisse son
// champ vide ; sous WSL2, cpufreq et la topologie hybride ne sont en général pas exposés.
func detectHost(ctx context.Context) Host {
	if ctx.Err() != nil {
		return Host{}
	}
	h := Host{PowerPlan: readTrimmed("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor")}
	name := osReleaseName(readTrimmed("/etc/os-release"))
	kernel := readTrimmed("/proc/sys/kernel/osrelease")
	switch {
	case name != "" && kernel != "":
		h.OSVersion = name + ", noyau " + kernel
	case kernel != "":
		h.OSVersion = "Linux " + kernel
	}
	h.CPUAffinity = linuxAffinity(procStatusField(readTrimmed("/proc/self/status"), "Cpus_allowed_list"),
		readTrimmed("/sys/devices/system/cpu/online"))
	h.CoreTypes = coreTypesFrom(linuxCoreClasses())
	return h
}

func readTrimmed(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// procStatusField extrait un champ de /proc/self/status.
func procStatusField(status, field string) string {
	for _, line := range strings.Split(status, "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == field {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// linuxAffinity rend « non épinglé » quand les processeurs permis sont tous les processeurs en
// ligne, la liste permise sinon.
func linuxAffinity(allowed, online string) string {
	a, o := parseCPUList(allowed), parseCPUList(online)
	switch {
	case a == nil:
		return ""
	case o != nil && slices.Equal(a, o):
		return Unpinned
	default:
		return allowed
	}
}

// linuxCoreClasses compte les cœurs physiques des deux PMU d'un processeur hybride Intel : cpu_core
// (performance) et cpu_atom (efficacité). Leur absence rend nil : le processeur est homogène, ou le
// noyau ne le dit pas.
func linuxCoreClasses() map[int]int {
	perf := physicalCores(readTrimmed("/sys/devices/cpu_core/cpus"))
	eff := physicalCores(readTrimmed("/sys/devices/cpu_atom/cpus"))
	if perf == 0 || eff == 0 {
		return nil
	}
	return map[int]int{1: perf, 0: eff}
}

// physicalCores compte les cœurs distincts d'une liste de processeurs logiques, deux fils d'un même
// cœur partageant la même liste core_cpus_list.
func physicalCores(list string) int {
	cores := map[string]bool{}
	for _, cpu := range parseCPUList(list) {
		dir := filepath.Join("/sys/devices/system/cpu", "cpu"+strconv.Itoa(cpu), "topology")
		siblings := readTrimmed(filepath.Join(dir, "core_cpus_list"))
		if siblings == "" {
			siblings = readTrimmed(filepath.Join(dir, "thread_siblings_list"))
		}
		if siblings == "" {
			siblings = strconv.Itoa(cpu)
		}
		cores[siblings] = true
	}
	return len(cores)
}
