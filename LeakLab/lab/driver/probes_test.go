package driver

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/agbruneau/leaklab/lab/corpus"
)

// TestProbeSynctestTimeout est la sonde SYNCTEST_TIMEOUT (H-006) : le test de délai de 500 ms de
// BEPG p. 290–291, dans une bulle ou hors bulle, chronométré hors de la bulle.
func TestProbeSynctestTimeout(t *testing.T) {
	which := arm(t, "SYNCTEST_TIMEOUT")
	body := func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		load := func() error { <-ctx.Done(); return ctx.Err() }
		if err := corpus.QueryWithTimeout(ctx, load, 500*time.Millisecond); err == nil || err.Error() != "timed out" {
			t.Fatalf("%v, « timed out » attendu", err)
		}
	}
	start := time.Now()
	switch which {
	case "SYNCTEST":
		synctest.Test(t, body)
	case "REAL":
		body(t)
	default:
		t.Fatalf("bras inconnu : %q", which)
	}
	metric("wallNs", time.Since(start).Nanoseconds())
}

// opaque enveloppe un parent annulable en masquant sa valeur interne : context ne le reconnaît
// plus et surveille chaque enfant par une goroutine. Témoin de sensibilité de H-010.
type opaque struct{ context.Context }

func (opaque) Value(any) any { return nil }

const cancelN = 20000

// TestProbeCancelRetention est la sonde CANCEL_RETENTION (H-009, H-010).
func TestProbeCancelRetention(t *testing.T) {
	parentName, mode, _ := strings.Cut(arm(t, "CANCEL_RETENTION"), "/")
	var parent context.Context
	switch parentName {
	case "BACKGROUND":
		parent = context.Background()
	case "CANCELABLE", "OPAQUE":
		p, cancel := context.WithCancel(context.Background())
		defer cancel()
		parent = p
		if parentName == "OPAQUE" {
			parent = opaque{p}
		}
	default:
		t.Fatalf("parent inconnu : %q", parentName)
	}
	timeout := time.Hour
	switch mode {
	case "FORGOTTEN", "CANCELLED":
	case "EXPIRED":
		timeout = time.Millisecond
	default:
		t.Fatalf("mode inconnu : %q", mode)
	}
	g0, h0 := runtime.NumGoroutine(), heapAlloc()
	for range cancelN {
		_, cancel := context.WithTimeout(parent, timeout)
		if mode == "CANCELLED" {
			cancel()
		}
	}
	if mode == "EXPIRED" {
		time.Sleep(500 * time.Millisecond)
	}
	h1, g1 := heapAlloc(), runtime.NumGoroutine()
	runtime.KeepAlive(parent)
	metric("bytesPerOp", float64(h1-h0)/cancelN)
	metric("goroutineDelta", g1-g0)
}

const timerN = 100000

// TestProbeTimerGrowth est la sonde TIMER_GROWTH (H-011) : chaque itération attend dans un select
// le résultat d'une goroutine, comme l'exemple de BEPG p. 276.
func TestProbeTimerGrowth(t *testing.T) {
	which := arm(t, "TIMER_GROWTH")
	switch which {
	case "AFTER_IN_LOOP", "REUSED_TIMER", "RETAINED_WITNESS":
	default:
		t.Fatalf("bras inconnu : %q", which)
	}
	reused := time.NewTimer(time.Hour)
	defer reused.Stop()
	var keep []*time.Timer
	h0 := heapAlloc()
	for i := range timerN {
		res := make(chan int, 1)
		go func() { res <- i }()
		switch which {
		case "AFTER_IN_LOOP":
			select {
			case <-res:
			case <-time.After(time.Hour):
			}
		case "REUSED_TIMER":
			select {
			case <-res:
			case <-reused.C:
			}
		case "RETAINED_WITNESS":
			tm := time.NewTimer(time.Hour)
			keep = append(keep, tm)
			select {
			case <-res:
			case <-tm.C:
			}
		}
	}
	h1 := heapAlloc()
	runtime.KeepAlive(keep)
	metric("bytesPerOp", float64(h1-h0)/timerN)
}
