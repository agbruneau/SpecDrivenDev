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
	jobObjectExtendedLimitInformation   = 9
	jobObjectLimitKillOnJobClose        = 0x2000
	processSetQuota                     = 0x0100
	processTerminate                    = 0x0001
	filetimeUnit                        = 100 * time.Nanosecond
)

// jobBasicLimit reflète JOBOBJECT_BASIC_LIMIT_INFORMATION.
type jobBasicLimit struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// ioCounters reflète IO_COUNTERS.
type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// jobExtendedLimit reflète JOBOBJECT_EXTENDED_LIMIT_INFORMATION.
type jobExtendedLimit struct {
	BasicLimitInformation jobBasicLimit
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

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
	job      syscall.Handle
	assigned bool
}

// prepareTree crée le job avant le démarrage du processus et le règle pour que la fermeture du
// handle termine tout l'arbre.
//
// Révision du 2026-09-12 (A-062) : le job n'était créé qu'après le démarrage et ne servait qu'à
// compter. JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE le fait aussi terminer l'arbre — à l'annulation,
// mais également si le processus parent est tué sans pouvoir exécuter quoi que ce soit
// (`taskkill /F`), ce qu'aucune fonction Cancel ne couvre. Sans cela, le binaire de benchmark
// survivait à l'annulation, bloquait Wait sur les tubes de sortie jusqu'à la fin de sa benchtime
// et consommait un cœur pendant ce temps.
//
// CreateJobObjectW ne demande aucun pid : seul AssignProcessToJobObject attend le processus.
func prepareTree(*exec.Cmd) *treeCPUTracker {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	create := kernel32.NewProc("CreateJobObjectW")
	setInfo := kernel32.NewProc("SetInformationJobObject")
	job, _, _ := create.Call(0, 0)
	if job == 0 {
		return nil
	}
	limits := jobExtendedLimit{}
	limits.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	if ret, _, _ := setInfo.Call(job, jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits)); ret == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return nil
	}
	return &treeCPUTracker{job: syscall.Handle(job)}
}

// attach place le processus démarré dans le job. L'assignation suit immédiatement le démarrage :
// `go test` n'engendre ses propres enfants qu'après avoir analysé sa ligne de commande, bien après
// cet appel. Une erreur n'est pas fatale — la mesure de quiétude se déclare alors non faite et
// H-013 rend non concluant, ce qui vaut mieux qu'un chiffre faux.
func (t *treeCPUTracker) attach(cmd *exec.Cmd) {
	if t == nil || cmd.Process == nil {
		return
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	assign := kernel32.NewProc("AssignProcessToJobObject")
	handle, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(cmd.Process.Pid))
	if err != nil {
		t.assigned = false
		return
	}
	defer syscall.CloseHandle(handle)
	ret, _, _ := assign.Call(uintptr(t.job), uintptr(handle))
	t.assigned = ret != 0
}

// total rend le temps processeur cumulé de l'arbre, puis libère le job.
func (t *treeCPUTracker) total() (time.Duration, bool) {
	if t == nil {
		return 0, false
	}
	// La fermeture du handle termine l'arbre resté vivant (KILL_ON_JOB_CLOSE).
	defer syscall.CloseHandle(t.job)
	if !t.assigned {
		return 0, false
	}
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
