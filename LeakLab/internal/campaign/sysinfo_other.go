//go:build !windows

package campaign

import "os"

// platformCPU lit le champ « model name » de /proc/cpuinfo ; "" hors Linux ou si le champ manque
// (cas des processeurs arm64, qui ne le portent pas).
func platformCPU() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	return cpuInfoModel(string(data))
}

// platformOSVersion rend la distribution et le noyau sous Linux ; "" ailleurs.
func platformOSVersion() string {
	release, _ := os.ReadFile("/etc/os-release")
	kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	return linuxOSVersion(string(release), string(kernel))
}
