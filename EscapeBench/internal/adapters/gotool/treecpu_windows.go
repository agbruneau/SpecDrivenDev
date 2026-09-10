//go:build windows

package gotool

import (
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// Le temps processeur d'un `go test` ne se lit pas sur son seul processus : la commande compile le
// sujet, lie le binaire de test puis l'exécute, chaque étape dans un processus enfant. Retrancher
// le seul temps propre de `go test` laisserait tout le travail légitime de la campagne dans la
// fraction d'occupation de C-010, qui refuserait alors les campagnes saines.
//
// Windows agrège l'arbre par un Job Object : les processus enfants héritent du job, et le compteur
// du job inclut ceux qui se sont déjà terminés.

const (
	jobObjectBasicAccountingInformation = 1
	processSetQuota                     = 0x0100
	processTerminate                    = 0x0001
	filetimeUnit                        = 100 * time.Nanosecond
)

// jobAccounting reflète JOBOBJECT_BASIC_ACCOUNTING_INFORMATION.
type jobAccounting struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// treeCPUTracker suit le temps processeur d'un arbre de processus.
type treeCPUTracker struct {
	job syscall.Handle
}

// startTreeCPU place le processus déjà démarré dans un job neuf. L'assignation suit immédiatement
// le démarrage : `go test` n'engendre ses propres enfants qu'après avoir analysé sa ligne de
// commande, bien après cet appel. Une erreur n'est pas fatale — la mesure de quiétude se déclare
// alors non faite et H-013 rend non concluant, ce qui vaut mieux qu'un chiffre faux.
func startTreeCPU(cmd *exec.Cmd) *treeCPUTracker {
	if cmd.Process == nil {
		return nil
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	create := kernel32.NewProc("CreateJobObjectW")
	assign := kernel32.NewProc("AssignProcessToJobObject")
	job, _, _ := create.Call(0, 0)
	if job == 0 {
		return nil
	}
	handle, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(cmd.Process.Pid))
	if err != nil {
		syscall.CloseHandle(syscall.Handle(job))
		return nil
	}
	defer syscall.CloseHandle(handle)
	if ret, _, _ := assign.Call(job, uintptr(handle)); ret == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return nil
	}
	return &treeCPUTracker{job: syscall.Handle(job)}
}

// total rend le temps processeur cumulé de l'arbre, puis libère le job.
func (t *treeCPUTracker) total() (time.Duration, bool) {
	if t == nil {
		return 0, false
	}
	defer syscall.CloseHandle(t.job)
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	query := kernel32.NewProc("QueryInformationJobObject")
	var info jobAccounting
	ret, _, _ := query.Call(uintptr(t.job), jobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), 0)
	if ret == 0 {
		return 0, false
	}
	return time.Duration(info.TotalUserTime+info.TotalKernelTime) * filetimeUnit, true
}
