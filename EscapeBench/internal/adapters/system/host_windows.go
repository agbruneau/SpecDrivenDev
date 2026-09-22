//go:build windows

package system

import (
	"context"
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32Host = syscall.NewLazyDLL("kernel32.dll")
	powrprof     = syscall.NewLazyDLL("powrprof.dll")
)

// detectHost relève l'état de la machine sous Windows (D-60). Chaque lecture qui échoue laisse son
// champ vide.
func detectHost(ctx context.Context) Host {
	if ctx.Err() != nil {
		return Host{}
	}
	return Host{
		OSVersion:   windowsVersion(),
		PowerPlan:   activePowerScheme(),
		CPUAffinity: processAffinity(),
		CoreTypes:   coreTypesFrom(windowsCoreClasses()),
	}
}

// windowsVersion lit version, build et révision dans le registre. ProductName n'est pas repris : il
// porte encore « Windows 10 » sur Windows 11, alors que le numéro de build les distingue.
func windowsVersion() string {
	const path = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	major, okMajor := registryDWORD(path, "CurrentMajorVersionNumber")
	minor, okMinor := registryDWORD(path, "CurrentMinorVersionNumber")
	build := registryString(path, "CurrentBuildNumber")
	if !okMajor || !okMinor || build == "" {
		return ""
	}
	version := fmt.Sprintf("Windows %d.%d.%s", major, minor, build)
	if ubr, ok := registryDWORD(path, "UBR"); ok {
		version += fmt.Sprintf(".%d", ubr)
	}
	if display := registryString(path, "DisplayVersion"); display != "" {
		version += " (" + display + ")"
	}
	return version
}

// windowsGUID reproduit la structure GUID de Windows.
type windowsGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g windowsGUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x", g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// activePowerScheme rend le nom et le GUID du plan d'alimentation actif. Le curseur de mode
// d'alimentation de Windows 11, qui se superpose au plan, n'est pas relevé.
func activePowerScheme() string {
	getActive := powrprof.NewProc("PowerGetActiveScheme")
	readName := powrprof.NewProc("PowerReadFriendlyName")
	if getActive.Find() != nil || readName.Find() != nil {
		return ""
	}
	var guid *windowsGUID
	if ret, _, _ := getActive.Call(0, uintptr(unsafe.Pointer(&guid))); ret != 0 || guid == nil {
		return ""
	}
	defer kernel32Host.NewProc("LocalFree").Call(uintptr(unsafe.Pointer(guid)))
	id := guid.String()
	var size uint32
	if ret, _, _ := readName.Call(0, uintptr(unsafe.Pointer(guid)), 0, 0, 0, uintptr(unsafe.Pointer(&size))); ret != 0 || size < 2 || size > 4096 {
		return id
	}
	buffer := make([]uint16, size/2)
	if ret, _, _ := readName.Call(0, uintptr(unsafe.Pointer(guid)), 0, 0,
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size))); ret != 0 {
		return id
	}
	if name := syscall.UTF16ToString(buffer); name != "" {
		return name + " (" + id + ")"
	}
	return id
}

// processAffinity compare le masque du processus à celui de la machine. Au-delà de 64 processeurs
// logiques, Windows répartit les processeurs en groupes et ces masques ne couvrent que le groupe
// courant : la valeur rendue ne vaut alors que pour ce groupe.
func processAffinity() string {
	var process, system uintptr
	ret, _, _ := kernel32Host.NewProc("GetProcessAffinityMask").Call(uintptr(currentProcess()),
		uintptr(unsafe.Pointer(&process)), uintptr(unsafe.Pointer(&system)))
	if ret == 0 || process == 0 {
		return ""
	}
	return affinityLabel(uint64(process), uint64(system))
}

func currentProcess() syscall.Handle {
	handle, _ := syscall.GetCurrentProcess()
	return handle
}

// affinityLabel rend « non épinglé » quand le processus peut s'exécuter sur tous les processeurs de
// la machine, le masque en hexadécimal sinon.
func affinityLabel(process, system uint64) string {
	if process == system {
		return Unpinned
	}
	return fmt.Sprintf("0x%x", process)
}

// relationProcessorCore est la relation de GetLogicalProcessorInformationEx qui décrit un cœur.
const relationProcessorCore = 0

// windowsCoreClasses compte les cœurs physiques par classe d'efficacité.
func windowsCoreClasses() map[int]int {
	proc := kernel32Host.NewProc("GetLogicalProcessorInformationEx")
	if proc.Find() != nil {
		return nil
	}
	var length uint32
	proc.Call(relationProcessorCore, 0, uintptr(unsafe.Pointer(&length)))
	if length == 0 || length > 1<<20 {
		return nil
	}
	buffer := make([]byte, length)
	if ret, _, _ := proc.Call(relationProcessorCore, uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&length))); ret == 0 {
		return nil
	}
	return parseWindowsCoreClasses(buffer[:length])
}

// parseWindowsCoreClasses lit les enregistrements SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX d'un
// tampon : relation (4 octets), taille de l'enregistrement (4 octets), puis PROCESSOR_RELATIONSHIP,
// dont le deuxième octet est EfficiencyClass. La fonction est pure : elle est éprouvée sur un
// tampon construit à la main.
func parseWindowsCoreClasses(buffer []byte) map[int]int {
	classes := map[int]int{}
	for offset := 0; offset+10 <= len(buffer); {
		size := int(binary.LittleEndian.Uint32(buffer[offset+4 : offset+8]))
		if size < 10 || offset+size > len(buffer) {
			break
		}
		if binary.LittleEndian.Uint32(buffer[offset:offset+4]) == relationProcessorCore {
			classes[int(buffer[offset+9])]++
		}
		offset += size
	}
	return classes
}

// registryString lit une valeur REG_SZ de HKLM ; vide si absente.
func registryString(path, name string) string {
	key, ok := openHKLM(path)
	if !ok {
		return ""
	}
	defer syscall.RegCloseKey(key)
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	buffer := make([]uint16, 256)
	size := uint32(len(buffer) * 2)
	var kind uint32
	if err := syscall.RegQueryValueEx(key, namePtr, nil, &kind, (*byte)(unsafe.Pointer(&buffer[0])), &size); err != nil || kind != syscall.REG_SZ {
		return ""
	}
	return syscall.UTF16ToString(buffer[:size/2])
}

// registryDWORD lit une valeur REG_DWORD de HKLM.
func registryDWORD(path, name string) (uint32, bool) {
	key, ok := openHKLM(path)
	if !ok {
		return 0, false
	}
	defer syscall.RegCloseKey(key)
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, false
	}
	var value, kind uint32
	size := uint32(4)
	if err := syscall.RegQueryValueEx(key, namePtr, nil, &kind, (*byte)(unsafe.Pointer(&value)), &size); err != nil || kind != syscall.REG_DWORD {
		return 0, false
	}
	return value, true
}

func openHKLM(path string) (syscall.Handle, bool) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, pathPtr, 0, syscall.KEY_READ, &key); err != nil {
		return 0, false
	}
	return key, true
}
