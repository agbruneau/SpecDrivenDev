//go:build !windows && !linux

package system

// sampleCPUTimes ne sait rien produire sur les autres plateformes. Un relevé non mesuré rend H-013
// non concluante, ce qui est préférable à supposer une quiétude que rien n'atteste (C-010).
func sampleCPUTimes() CPUTimes { return CPUTimes{} }
