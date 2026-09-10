//go:build windows

package system

import (
	"syscall"
	"time"
	"unsafe"
)

// filetimeUnit est l'unité des FILETIME que rend GetSystemTimes : cent nanosecondes.
const filetimeUnit = 100 * time.Nanosecond

// sampleCPUTimes lit les temps processeur cumulés de la machine.
//
// GetSystemTimes rend trois FILETIME cumulés sur tous les processeurs : le temps d'inactivité, le
// temps noyau et le temps utilisateur. Le temps noyau INCLUT le temps d'inactivité, ce qui est le
// piège de cette interface : le total est donc noyau plus utilisateur, et le temps occupé est ce
// total moins l'inactivité.
func sampleCPUTimes() CPUTimes {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetSystemTimes")
	var idle, kernel, user syscall.Filetime
	ret, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return CPUTimes{}
	}
	return cpuTimesFrom(filetimeTicks(idle), filetimeTicks(kernel), filetimeTicks(user))
}

// filetimeTicks assemble les deux moitiés d'un FILETIME.
func filetimeTicks(f syscall.Filetime) uint64 {
	return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime)
}

// cpuTimesFrom dérive un relevé des trois compteurs. La fonction est pure : elle est éprouvée sur
// des valeurs construites à la main, sans appeler le système.
func cpuTimesFrom(idle, kernel, user uint64) CPUTimes {
	total := kernel + user
	if total < idle {
		return CPUTimes{}
	}
	return CPUTimes{
		Measured: true,
		Busy:     time.Duration(total-idle) * filetimeUnit,
		Total:    time.Duration(total) * filetimeUnit,
	}
}
