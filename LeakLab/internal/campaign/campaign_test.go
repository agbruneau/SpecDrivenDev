package campaign

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/leaklab/internal/results"
)

func TestUC001_BR2_Classification(t *testing.T) {
	tests := []struct {
		name   string
		output string
		exit   int
		killed bool
		want   results.Outcome
	}{
		{"tué au délai, même avec un rapport", "WARNING: DATA RACE\n", 0, true, results.OutcomeHang},
		{"course avant interblocage", "WARNING: DATA RACE\nfatal error: all goroutines are asleep - deadlock!\n", 2, false, results.OutcomeRace},
		{"erreur fatale du runtime", "fatal error: all goroutines are asleep - deadlock!\n\ngoroutine 1 [chan send]:\n", 2, false, results.OutcomeDeadlock},
		{"panique de synctest", "panic: deadlock: main bubble goroutine has exited but blocked goroutines remain [recovered]\n", 2, false, results.OutcomeDeadlock},
		{"CRLF", "=== RUN   TestSynctest\r\npanic: deadlock: all goroutines in bubble are blocked\r\n", 2, false, results.OutcomeDeadlock},
		{"deadlock hors ligne fatale", "    corpus.deadlockSend()\n", 1, false, results.OutcomeFail},
		{"fuite signalée par un pilote", "=== RUN   TestNumGoroutine\nLEAKLAB-LEAK goroutines=10\n--- PASS\n", 0, false, results.OutcomeLeak},
		{"marqueur hors début de ligne", "  LEAKLAB-LEAK\n", 0, false, results.OutcomePass},
		{"échec", "--- FAIL: TestBare\n", 1, false, results.OutcomeFail},
		{"succès", "=== RUN   TestBare\n--- PASS: TestBare (0.00s)\nPASS\n", 0, false, results.OutcomePass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.output, tt.exit, tt.killed); got != tt.want {
				t.Fatalf("Classify = %s, %s attendu", got, tt.want)
			}
		})
	}
}

func TestH002_MentionsLeak(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"=== RUN   TestBare\n--- PASS: TestBare (0.00s)\nPASS\n", false},
		{"=== RUN   TestBare\n    testing: 1 leaked goroutine\n--- PASS: TestBare (0.00s)\n", true},
		{"goroutine 7 [chan send]:\n", true},
		{"--- FAIL: TestLeakyThing\nFAIL\n", false},
	}
	for _, tt := range tests {
		if got := MentionsLeak(tt.output); got != tt.want {
			t.Errorf("MentionsLeak(%q) = %t", tt.output, got)
		}
	}
}

func TestUC001_Detail(t *testing.T) {
	out := "=== RUN   TestBare\n    driver_test.go:38: LEAKLAB-WITNESS : assertion fausse\n--- FAIL: TestBare (0.00s)\nFAIL\n"
	if got := Detail(out); !strings.Contains(got, "LEAKLAB-WITNESS") {
		t.Fatalf("Detail = %q", got)
	}
	if got := Detail(strings.Repeat("é", 300)); len([]rune(got)) != 200 {
		t.Fatalf("Detail non tronqué à 200 caractères : %d", len([]rune(got)))
	}
}

func TestUC001_A4_RepetitionsInsuffisantes(t *testing.T) {
	root := t.TempDir()
	if _, _, err := Run(context.Background(), Config{Root: root, Reps: 4, Timeout: time.Second}); err == nil || !strings.Contains(err.Error(), "NFR-002") {
		t.Fatalf("Run = %v, refus NFR-002 attendu", err)
	}
	if _, err := os.Stat(filepath.Join(root, "results")); !os.IsNotExist(err) {
		t.Fatal("un refus a écrit sous results/")
	}
}

func TestUC001_A2_CorpusDivergent(t *testing.T) {
	root := t.TempDir()
	doc := "## Corpus de référence\n\n| Cas | a | 1 | oui | oui | non | non | CHAN_SEND | non | NONE | — |\n\n## Hypothèses à éprouver\n\n| H-001 | s | e | c | UC-001 |\n"
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc = strings.Replace(doc, "| Cas |", "| inconnu |", 1)
	if err := os.WriteFile(filepath.Join(root, "docs", "requirements.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Run(context.Background(), Config{Root: root, Reps: 5, Timeout: time.Second}); err == nil || !strings.Contains(err.Error(), "A2") {
		t.Fatalf("Run = %v, refus A2 attendu", err)
	}
}

func TestUC001_BR3_AttributionStatique(t *testing.T) {
	out := "# github.com/agbruneau/leaklab/lab/corpus\n" +
		"corpus\\ctx_watcher_leak.go:12:7: the cancel function returned by context.WithTimeout should be called, not discarded, to avoid a context leak\n" +
		"vet: corpus/forgotten_cancel.go:12:7: the cancel function should be called\n" +
		"corpus/fixture.go:3:1: diagnostic sans cas\n"
	fs := parseVet(out)
	if len(fs) != 3 {
		t.Fatalf("parseVet = %v", fs)
	}
	obs := attribute(results.DetectorVet, fs, time.Second)
	got := map[string]results.Outcome{}
	for _, o := range obs {
		got[o.CaseID] = o.Outcome
	}
	if got["ctx-watcher-leak"] != results.OutcomeDiagnostic || got["forgotten-cancel"] != results.OutcomeDiagnostic || got["dispatch-leak"] != results.OutcomePass {
		t.Fatalf("attribution = %v", got)
	}
	diag := 0
	for _, o := range got {
		if o == results.OutcomeDiagnostic {
			diag++
		}
	}
	if diag != 2 {
		t.Fatalf("%d cas diagnostiqués, 2 attendus : le fichier sans cas ne s'attribue à personne", diag)
	}
}

func TestUC001_Sondes(t *testing.T) {
	a := probeArm{"CANCEL_RETENTION", "BACKGROUND/FORGOTTEN", "TestProbeCancelRetention"}
	r, err := parseMetrics(a, 2, "LEAKLAB-METRIC bytesPerOp=123.5\r\nLEAKLAB-METRIC goroutineDelta=0\r\nPASS\r\n")
	if err != nil || r.BytesPerOp != 123.5 || r.GoroutineDelta != 0 || r.Rep != 2 || r.Arm != a.arm {
		t.Fatalf("parseMetrics = %+v, %v", r, err)
	}
	if _, err := parseMetrics(a, 1, "LEAKLAB-METRIC bytesPerOp=1\n"); err == nil {
		t.Fatal("une grandeur manquante doit être une erreur")
	}
	if n := len(probeArms()); n != 15 {
		t.Fatalf("%d bras, 15 attendus (C-007)", n)
	}
}

// TestUC001_C004_DelaiParDefaut verrouille la précision de C-004 : un binaire de test lancé
// directement n'a aucun délai ; le banc lui passe celui que go test transmet.
func TestUC001_C004_DelaiParDefaut(t *testing.T) {
	b := binaries{driver: "d", driverRace: "r", program: "p"}
	for _, det := range []results.Detector{results.DetectorBare, results.DetectorRace, results.DetectorSynctest, results.DetectorNumGoroutine, results.DetectorLeakProfile} {
		_, args := b.command(det, "dispatch-leak")
		if !slices.Contains(args, goTestTimeout) {
			t.Errorf("%s : arguments %v sans %s", det, args, goTestTimeout)
		}
	}
	if name, args := b.command(results.DetectorProgram, "dispatch-leak"); name != "p" || len(args) != 1 {
		t.Errorf("PROGRAM : %s %v", name, args)
	}
	if goTestTimeout != "-test.timeout=10m0s" {
		t.Errorf("délai %s, celui de go test attendu", goTestTimeout)
	}
}

// TestUC001_BR4_ProcessusReels vérifie la chaîne processus → classification sur de vrais processus,
// dont un processus tué au délai et une erreur fatale d'interblocage du runtime.
func TestUC001_BR4_ProcessusReels(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fixture")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, "./testdata/fixture").CombinedOutput(); err != nil {
		t.Fatalf("compilation de la fixture : %v\n%s", err, out)
	}
	tests := []struct {
		mode string
		want results.Outcome
	}{
		{"pass", results.OutcomePass},
		{"fail", results.OutcomeFail},
		{"hang", results.OutcomeHang},
		{"deadlock", results.OutcomeDeadlock},
		{"leak", results.OutcomeLeak},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			p, err := runProcess(context.Background(), 2*time.Second, ".", nil, bin, tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if got := Classify(p.output, p.exitCode, p.killed); got != tt.want {
				t.Fatalf("%s classé %s, %s attendu\n%s", tt.mode, got, tt.want, p.output)
			}
			if tt.mode == "hang" && p.duration > 10*time.Second {
				t.Fatalf("le processus tué a rendu la main après %s", p.duration)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runProcess(ctx, time.Second, ".", nil, bin, "pass"); err == nil {
		t.Fatal("une campagne interrompue doit rendre une erreur")
	}
}
