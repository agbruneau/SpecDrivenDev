//go:build windows

package campaign

import (
	"strconv"
	"syscall"
	"unsafe"
)

// platformCPU lit le nom commercial du processeur dans le registre, comme la provenance
// d'EscapeBench (internal/adapters/system/cpumodel_windows.go, A-065) ; "" si la clé manque.
func platformCPU() string {
	return registryValue(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "ProcessorNameString")
}

// platformOSVersion rend « Windows <majeure>.<mineure>.<build>.<UBR> (<DisplayVersion>) », ou "" si
// le build est illisible. ProductName n'est pas lu : il vaut « Windows 10 » sur Windows 11.
func platformOSVersion() string {
	const key = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	build := registryValue(key, "CurrentBuildNumber")
	if build == "" {
		return ""
	}
	v := "Windows " + registryValue(key, "CurrentMajorVersionNumber") + "." + registryValue(key, "CurrentMinorVersionNumber") + "." + build
	if ubr := registryValue(key, "UBR"); ubr != "" {
		v += "." + ubr
	}
	if display := registryValue(key, "DisplayVersion"); display != "" {
		v += " (" + display + ")"
	}
	return v
}

// registryValue lit une valeur REG_SZ ou REG_DWORD sous HKEY_LOCAL_MACHINE, rendue en texte ; ""
// si la clé, la valeur ou son type ne conviennent pas.
func registryValue(path, name string) string {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, p, 0, syscall.KEY_READ, &key); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	buffer := make([]uint16, 256) // 512 octets : le plus long nom commercial observé tient en 64 caractères
	size := uint32(len(buffer) * 2)
	var kind uint32
	if err := syscall.RegQueryValueEx(key, n, nil, &kind, (*byte)(unsafe.Pointer(&buffer[0])), &size); err != nil {
		return ""
	}
	switch kind {
	case syscall.REG_SZ:
		return syscall.UTF16ToString(buffer[:size/2])
	case syscall.REG_DWORD:
		if size == 4 {
			return strconv.FormatUint(uint64(buffer[0])|uint64(buffer[1])<<16, 10)
		}
	}
	return ""
}
