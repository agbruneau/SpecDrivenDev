//go:build windows

package system

import (
	"os"
	"syscall"
	"unsafe"
)

// platformCPUModel lit le nom commercial du processeur dans le registre.
//
// Révision du 2026-09-12 (A-065) : la provenance consignait PROCESSOR_IDENTIFIER, c'est-à-dire la
// signature CPUID — « Intel64 Family 6 Model 198 Stepping 2, GenuineIntel ». Cette chaîne est
// partagée par tous les SKU d'une même génération : deux machines de fréquences et de caches
// différents portent la même, et NFR-001 exige le modèle du processeur, que les résultats publiés
// doivent identifier. `ProcessorNameString` porte le nom commercial ; PROCESSOR_IDENTIFIER reste
// le repli, de sorte qu'un poste où la clé est absente consigne encore quelque chose.
//
// Les provenances antérieures restent valides : SameToolchain ne compare pas cpuModel.
func platformCPUModel() string {
	if model := registryProcessorName(); model != "" {
		return model
	}
	return os.Getenv("PROCESSOR_IDENTIFIER")
}

// registryProcessorName lit HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0\ProcessorNameString.
func registryProcessorName() string {
	path, err := syscall.UTF16PtrFromString(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`)
	if err != nil {
		return ""
	}
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, path, 0, syscall.KEY_READ, &key); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)

	name, err := syscall.UTF16PtrFromString("ProcessorNameString")
	if err != nil {
		return ""
	}
	// Le nom commercial le plus long observé tient en 64 caractères ; 256 laisse de la marge.
	buffer := make([]uint16, 256)
	size := uint32(len(buffer) * 2)
	var kind uint32
	err = syscall.RegQueryValueEx(key, name, nil, &kind,
		(*byte)(unsafe.Pointer(&buffer[0])), &size)
	if err != nil || kind != syscall.REG_SZ {
		return ""
	}
	return syscall.UTF16ToString(buffer[:size/2])
}
