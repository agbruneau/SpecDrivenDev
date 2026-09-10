//go:build windows

package system

import (
	"testing"
	"time"
)

// cpuTimesFrom est le seul endroit où le piège de GetSystemTimes se joue : le temps noyau inclut
// le temps d'inactivité. Le confondre ferait passer une machine chargée pour une machine au repos.
// Mutation : écrire total = kernel + user - idle sans retrancher ⇒ échec attendu.
func TestC010_CPUTimesFrom(t *testing.T) {
	t.Parallel()
	const tick = uint64(1) // une unité de 100 ns
	cases := map[string]struct {
		idle, kernel, user uint64
		wantBusy           time.Duration
		wantTotal          time.Duration
		wantMeasured       bool
	}{
		"machine au repos": {
			idle: 100 * tick, kernel: 100 * tick, user: 0,
			wantBusy: 0, wantTotal: 100 * filetimeUnit, wantMeasured: true,
		},
		"noyau et utilisateur": {
			idle: 60 * tick, kernel: 80 * tick, user: 20 * tick,
			wantBusy: 40 * filetimeUnit, wantTotal: 100 * filetimeUnit, wantMeasured: true,
		},
		"machine pleine": {
			idle: 0, kernel: 50 * tick, user: 50 * tick,
			wantBusy: 100 * filetimeUnit, wantTotal: 100 * filetimeUnit, wantMeasured: true,
		},
		"inactivité supérieure au total, incohérent": {
			idle: 200 * tick, kernel: 100 * tick, user: 0,
			wantMeasured: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := cpuTimesFrom(tc.idle, tc.kernel, tc.user)
			if got.Measured != tc.wantMeasured {
				t.Fatalf("mesurée = %v, %v attendu", got.Measured, tc.wantMeasured)
			}
			if !tc.wantMeasured {
				return
			}
			if got.Busy != tc.wantBusy || got.Total != tc.wantTotal {
				t.Fatalf("occupé %v sur %v, attendu %v sur %v", got.Busy, got.Total, tc.wantBusy, tc.wantTotal)
			}
		})
	}
}
