// Package corpus porte les cas mesurés par LeakLab : un fichier par cas, fautif par construction
// quand le catalogue le déclare. Le tableau « Corpus de référence » de docs/requirements.md fait
// autorité ; un test du module principal vérifie que Catalog lui est identique (BR-001-1).
package corpus

import (
	"reflect"
	"runtime"
	"strings"
)

// AntiPattern classe un cas selon la section du livre qu'il illustre.
type AntiPattern string

const (
	GoroutineLeak    AntiPattern = "GOROUTINE_LEAK"
	ChannelDeadlock  AntiPattern = "CHANNEL_DEADLOCK"
	ForgottenCancel  AntiPattern = "FORGOTTEN_CANCEL"
	ContextIgnoredIO AntiPattern = "CONTEXT_IGNORED_IO"
	DataRace         AntiPattern = "DATA_RACE"
	Witness          AntiPattern = "WITNESS"
)

// Primitive nomme ce qui bloque la goroutine fuitée ou le scénario.
type Primitive string

const (
	ChanSend  Primitive = "CHAN_SEND"
	ChanRecv  Primitive = "CHAN_RECV"
	Select    Primitive = "SELECT"
	WaitGroup Primitive = "WAITGROUP"
	Cond      Primitive = "COND"
	Mutex     Primitive = "MUTEX"
	NetRead   Primitive = "NET_READ"
	NoBlock   Primitive = "NONE"
)

// Static nomme le défaut qu'un analyseur statique peut viser.
type Static string

const (
	LostCancel Static = "LOST_CANCEL"
	CtxIO      Static = "CTX_IO"
	NoStatic   Static = "NONE"
)

// Case est un cas du corpus et sa vérité terrain (modèle d'entités, Case).
type Case struct {
	ID          string
	AntiPattern AntiPattern
	Page        int
	Faulty      bool
	Leak        bool
	Blocks      bool
	Race        bool
	Primitive   Primitive
	Reachable   bool
	Static      Static
	Fixes       string
	// Worker est la fonction dont le nom apparaît dans la pile de la goroutine fuitée, ou qui y
	// apparaîtrait si le cas fuyait ; nil pour le témoin.
	Worker   any
	Scenario func() error
}

// File rend le fichier source du cas, relatif au paquet.
func (c Case) File() string { return strings.ReplaceAll(c.ID, "-", "_") + ".go" }

// WorkerName rend le nom qualifié de Worker tel que runtime.Stack l'imprime, ou "".
func (c Case) WorkerName() string {
	if c.Worker == nil {
		return ""
	}
	return runtime.FuncForPC(reflect.ValueOf(c.Worker).Pointer()).Name()
}

// Lookup rend le cas d'identifiant id.
func Lookup(id string) (Case, bool) {
	for _, c := range Catalog() {
		if c.ID == id {
			return c, true
		}
	}
	return Case{}, false
}

func leak(id string, page int, p Primitive, reachable bool, static Static, ap AntiPattern, worker any, scenario func() error) Case {
	return Case{ID: id, AntiPattern: ap, Page: page, Faulty: true, Leak: true, Primitive: p, Reachable: reachable, Static: static, Worker: worker, Scenario: scenario}
}

func fix(id string, page int, fixes string, ap AntiPattern, worker any, scenario func() error) Case {
	return Case{ID: id, AntiPattern: ap, Page: page, Primitive: NoBlock, Static: NoStatic, Fixes: fixes, Worker: worker, Scenario: scenario}
}

// Catalog rend le corpus de référence, dans l'ordre du tableau de docs/requirements.md.
func Catalog() []Case {
	return []Case{
		leak("dispatch-leak", 561, ChanSend, false, NoStatic, GoroutineLeak, dispatchSender, dispatchLeak),
		leak("dispatch-buffered-leak", 566, ChanSend, false, NoStatic, GoroutineLeak, dispatchBufferedSender, dispatchBufferedLeak),
		leak("abandoned-receive-leak", 561, ChanRecv, false, NoStatic, GoroutineLeak, abandonedReceiver, abandonedReceiveLeak),
		leak("select-no-cancel-leak", 562, Select, false, NoStatic, GoroutineLeak, pollLoop, selectNoCancelLeak),
		leak("waitgroup-leak", 562, WaitGroup, false, NoStatic, GoroutineLeak, closeWhenDone, waitgroupLeak),
		leak("cond-leak", 562, Cond, false, NoStatic, GoroutineLeak, condWaiter, condLeak),
		leak("mutex-leak", 562, Mutex, false, NoStatic, GoroutineLeak, lockedWorker, mutexLeak),
		leak("global-channel-leak", 561, ChanSend, true, NoStatic, GoroutineLeak, globalPublisher, globalChannelLeak),
		leak("empty-select-leak", 290, NoBlock, false, NoStatic, GoroutineLeak, loadForever, emptySelectLeak),
		leak("ctx-watcher-leak", 567, ChanRecv, true, LostCancel, ForgottenCancel, ctxWatcher, ctxWatcherLeak),
		leak("io-without-context-leak", 568, NetRead, false, CtxIO, ContextIgnoredIO, loadRoute, ioWithoutContextLeak),
		{ID: "forgotten-cancel", AntiPattern: ForgottenCancel, Page: 566, Faulty: true, Primitive: NoBlock, Static: LostCancel, Worker: timedWorker, Scenario: forgottenCancel},
		{ID: "counter-race", AntiPattern: DataRace, Page: 232, Faulty: true, Race: true, Primitive: NoBlock, Static: NoStatic, Worker: counterWorker, Scenario: counterRace},
		{ID: "append-race", AntiPattern: DataRace, Page: 232, Faulty: true, Race: true, Primitive: NoBlock, Static: NoStatic, Worker: appendWorker, Scenario: appendRace},
		{ID: "deadlock-send", AntiPattern: ChannelDeadlock, Page: 564, Faulty: true, Blocks: true, Primitive: ChanSend, Static: NoStatic, Worker: deadlockSend, Scenario: deadlockSend},
		{ID: "deadlock-range", AntiPattern: ChannelDeadlock, Page: 564, Faulty: true, Blocks: true, Primitive: ChanRecv, Static: NoStatic, Worker: rangeSender, Scenario: deadlockRange},
		fix("dispatch-fix", 562, "dispatch-leak", GoroutineLeak, dispatchSenderFixed, dispatchFix),
		fix("abandoned-receive-fix", 563, "abandoned-receive-leak", GoroutineLeak, abandonedReceiverFixed, abandonedReceiveFix),
		fix("select-no-cancel-fix", 562, "select-no-cancel-leak", GoroutineLeak, pollLoopFixed, selectNoCancelFix),
		fix("waitgroup-fix", 562, "waitgroup-leak", GoroutineLeak, closeWhenDoneFixed, waitgroupFix),
		fix("cond-fix", 562, "cond-leak", GoroutineLeak, condWaiterFixed, condFix),
		fix("mutex-fix", 562, "mutex-leak", GoroutineLeak, lockedWorkerFixed, mutexFix),
		fix("global-channel-fix", 565, "global-channel-leak", GoroutineLeak, globalPublisherFixed, globalChannelFix),
		fix("empty-select-fix", 291, "empty-select-leak", GoroutineLeak, loadUntilCancelled, emptySelectFix),
		fix("ctx-watcher-fix", 567, "ctx-watcher-leak", ForgottenCancel, ctxWatcherFixed, ctxWatcherFix),
		fix("io-without-context-fix", 568, "io-without-context-leak", ContextIgnoredIO, loadRouteContext, ioWithoutContextFix),
		fix("forgotten-cancel-fix", 567, "forgotten-cancel", ForgottenCancel, timedWorkerFixed, forgottenCancelFix),
		fix("counter-race-fix", 232, "counter-race", DataRace, counterWorkerFixed, counterRaceFix),
		fix("append-race-fix", 232, "append-race", DataRace, appendWorkerFixed, appendRaceFix),
		fix("deadlock-send-fix", 566, "deadlock-send", ChannelDeadlock, dedicatedSender, deadlockSendFix),
		fix("deadlock-range-fix", 565, "deadlock-range", ChannelDeadlock, rangeSenderFixed, deadlockRangeFix),
		{ID: "witness-fail", AntiPattern: Witness, Primitive: NoBlock, Static: NoStatic, Scenario: witnessFail},
	}
}
