//go:build linux

package system

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// clockTick est la durée d'un jiffy sur les noyaux courants. /proc/stat compte en jiffies, et
// sysconf(_SC_CLK_TCK) vaut 100 sur toutes les architectures que le catalogue vise (C-006).
const clockTick = 10 * time.Millisecond

// sampleCPUTimes lit la première ligne de /proc/stat, qui agrège tous les processeurs.
func sampleCPUTimes() CPUTimes {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUTimes{}
	}
	return parseProcStat(string(content))
}

// parseProcStat dérive un relevé de la ligne agrégée de /proc/stat. La fonction est pure : elle est
// éprouvée sur un contenu construit à la main, sans lire le système de fichiers.
//
// Les champs sont, dans l'ordre : user, nice, system, idle, iowait, irq, softirq, steal, puis deux
// champs d'invité qui sont déjà comptés dans user et nice et ne doivent donc pas être ajoutés. Le
// temps inactif est idle plus iowait : un processeur qui attend un disque n'exécute rien.
func parseProcStat(content string) CPUTimes {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		var total, idle uint64
		for i, field := range fields[1:] {
			if i >= 8 {
				break
			}
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return CPUTimes{}
			}
			total += value
			if i == 3 || i == 4 {
				idle += value
			}
		}
		if total < idle {
			return CPUTimes{}
		}
		return CPUTimes{
			Measured: true,
			Busy:     time.Duration(total-idle) * clockTick,
			Total:    time.Duration(total) * clockTick,
		}
	}
	return CPUTimes{}
}
