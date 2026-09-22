// Package driver imite les pilotes de lab/driver : chaque test rend immédiatement une sortie que
// la campagne sait classer et lire.
package driver

import (
	"fmt"
	"os"
	"testing"
)

func TestBare(t *testing.T) {}

func TestSynctest(t *testing.T) {}

// TestNumGoroutine signale une fuite pour un seul cas, pour que la matrice ne soit pas uniforme.
func TestNumGoroutine(t *testing.T) {
	if os.Getenv("LEAKLAB_CASE") == "dispatch-leak" {
		fmt.Println("LEAKLAB-LEAK goroutines=10")
	}
}

func TestLeakProfile(t *testing.T) {}

func TestProbeSynctestTimeout(t *testing.T) { fmt.Println("LEAKLAB-METRIC wallNs=1") }

func TestProbeCancelRetention(t *testing.T) {
	fmt.Println("LEAKLAB-METRIC bytesPerOp=1.5")
	fmt.Println("LEAKLAB-METRIC goroutineDelta=0")
}

func TestProbeTimerGrowth(t *testing.T) { fmt.Println("LEAKLAB-METRIC bytesPerOp=2") }
