//go:build linux

package system

import (
	"testing"
	"time"
)

// parseProcStat doit compter iowait comme de l'inactivité et ignorer les deux champs d'invité, déjà
// comptés dans user et nice.
func TestC010_ParseProcStat(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		content      string
		wantBusy     time.Duration
		wantTotal    time.Duration
		wantMeasured bool
	}{
		"ligne agrégée": {
			content:      "cpu  100 0 100 700 100 0 0 0 0 0\ncpu0 10 0 10 70 10 0 0 0 0 0\n",
			wantBusy:     200 * clockTick,
			wantTotal:    1000 * clockTick,
			wantMeasured: true,
		},
		"les champs d'invité sont ignorés": {
			content:      "cpu  100 0 100 800 0 0 0 0 500 500\n",
			wantBusy:     200 * clockTick,
			wantTotal:    1000 * clockTick,
			wantMeasured: true,
		},
		"ligne absente":     {content: "intr 1234\n", wantMeasured: false},
		"champ illisible":   {content: "cpu  100 x 100 700 100\n", wantMeasured: false},
		"ligne trop courte": {content: "cpu  1 2\n", wantMeasured: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := parseProcStat(tc.content)
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
