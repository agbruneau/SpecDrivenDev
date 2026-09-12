//go:build !windows

package system

import "os"

// platformCPUModel lit le champ « model name » de /proc/cpuinfo quand il existe.
//
// A-065 : le repli sur PROCESSOR_IDENTIFIER est réservé à Windows. Lu partout, il laissait une
// variable d'environnement d'un poste quelconque décider de la provenance d'une campagne.
func platformCPUModel() string {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	return cpuModelFromCPUInfo(string(content))
}
