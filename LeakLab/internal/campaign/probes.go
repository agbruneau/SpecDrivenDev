package campaign

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/agbruneau/leaklab/internal/results"
)

type probeArm struct{ probe, arm, test string }

// probeArms rend les bras de C-007, dans l'ordre d'exécution.
func probeArms() []probeArm {
	arms := []probeArm{
		{"SYNCTEST_TIMEOUT", "SYNCTEST", "TestProbeSynctestTimeout"},
		{"SYNCTEST_TIMEOUT", "REAL", "TestProbeSynctestTimeout"},
	}
	for _, parent := range []string{"BACKGROUND", "CANCELABLE", "OPAQUE"} {
		for _, mode := range []string{"FORGOTTEN", "CANCELLED", "EXPIRED"} {
			arms = append(arms, probeArm{"CANCEL_RETENTION", parent + "/" + mode, "TestProbeCancelRetention"})
		}
	}
	for _, a := range []string{"AFTER_IN_LOOP", "REUSED_TIMER", "RETAINED_WITNESS"} {
		arms = append(arms, probeArm{"TIMER_GROWTH", a, "TestProbeTimerGrowth"})
	}
	return arms
}

// parseMetrics lit les lignes LEAKLAB-METRIC d'une sonde. Une grandeur absente ou illisible est
// une erreur : une sonde muette n'est pas une mesure nulle.
func parseMetrics(a probeArm, rep int, output string) (results.ProbeResult, error) {
	values := map[string]string{}
	for _, l := range strings.Split(normalize(output), "\n") {
		if kv, ok := strings.CutPrefix(strings.TrimSpace(l), "LEAKLAB-METRIC "); ok {
			k, v, _ := strings.Cut(kv, "=")
			values[k] = v
		}
	}
	r := results.ProbeResult{Probe: a.probe, Arm: a.arm, Rep: rep}
	var err error
	switch a.probe {
	case "SYNCTEST_TIMEOUT":
		r.WallNs, err = strconv.ParseInt(values["wallNs"], 10, 64)
	case "CANCEL_RETENTION":
		if r.BytesPerOp, err = strconv.ParseFloat(values["bytesPerOp"], 64); err == nil {
			r.GoroutineDelta, err = strconv.Atoi(values["goroutineDelta"])
		}
	case "TIMER_GROWTH":
		r.BytesPerOp, err = strconv.ParseFloat(values["bytesPerOp"], 64)
	}
	if err != nil {
		return r, fmt.Errorf("sonde %s/%s, répétition %d : grandeur illisible (%w)\n%s", a.probe, a.arm, rep, err, output)
	}
	return r, nil
}
