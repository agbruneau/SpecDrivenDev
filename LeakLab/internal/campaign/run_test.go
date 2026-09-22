package campaign

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/lab/corpus"
)

// synthCases sont les cas de la campagne synthétique : une fuite, une cible de go vet, une cible
// de ctxvet.
var synthCases = []string{"dispatch-leak", "io-without-context-leak", "forgotten-cancel"}

// synthRoot rend une racine LeakLab jetable et restreint le catalogue à synthCases : la vraie
// spécification, réduite aux lignes de ces cas dans le corpus de référence, et à la place de lab/
// le module synthétique de testdata/synthlab, dont les pilotes répondent aussitôt. La campagne y
// parcourt toutes ses étapes en quelques secondes ; rien n'est écrit sous le results/ du dépôt.
func synthRoot(t *testing.T) string {
	t.Helper()
	restore := catalog
	t.Cleanup(func() { catalog = restore })
	catalog = func() []corpus.Case {
		return slices.DeleteFunc(corpus.Catalog(), func(c corpus.Case) bool { return !slices.Contains(synthCases, c.ID) })
	}
	root := t.TempDir()
	full, err := os.ReadFile(filepath.Join("..", "..", "docs", "requirements.md"))
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, l := range strings.Split(string(full), "\n") {
		id, _, _ := strings.Cut(strings.TrimPrefix(l, "| "), " |")
		if _, known := corpus.Lookup(id); known && !slices.Contains(synthCases, id) {
			continue
		}
		kept = append(kept, l)
	}
	doc := []byte(strings.Join(kept, "\n"))
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "requirements.md"), doc, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, "lab"), os.DirFS(filepath.Join("testdata", "synthlab"))); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestUC001_CampagneSynthetique parcourt le scénario principal de UC-001 de bout en bout : oracle,
// provenance, compilation des trois binaires, matrice dynamique, détecteurs statiques, sondes,
// écriture unique (BR-001-5).
func TestUC001_CampagneSynthetique(t *testing.T) {
	root := synthRoot(t)
	run, path, err := Run(context.Background(), Config{Root: root, Reps: MinReps, Timeout: 10 * time.Second, Log: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "results", "runs", run.ID+".json"); path != want {
		t.Fatalf("chemin %s, %s attendu", path, want)
	}
	stored, err := results.LoadRun(path)
	if err != nil || stored.ID != run.ID || len(stored.Observations) != len(run.Observations) {
		t.Fatalf("relecture : %v", err)
	}
	p := run.Provenance
	if p.GoVersion == "" || p.GOOS == "" || p.GOARCH == "" || p.NumCPU < 1 {
		t.Fatalf("provenance incomplète : %+v", p)
	}
	cases := len(synthCases)
	if want := cases*6*MinReps + cases*2; len(run.Observations) != want {
		t.Fatalf("%d observations, %d attendues", len(run.Observations), want)
	}
	if want := len(probeArms()) * MinReps; len(run.Probes) != want {
		t.Fatalf("%d mesures de sonde, %d attendues", len(run.Probes), want)
	}
	if len(run.CriteriaDigests) == 0 {
		t.Fatal("aucune empreinte de critère (C-008)")
	}
	got := map[[2]string]results.Outcome{}
	for _, o := range run.Observations {
		got[[2]string{o.CaseID, string(o.Detector)}] = o.Outcome
	}
	for cell, want := range map[[2]string]results.Outcome{
		{"dispatch-leak", "NUMGOROUTINE"}:     results.OutcomeLeak,
		{"forgotten-cancel", "NUMGOROUTINE"}:  results.OutcomePass,
		{"forgotten-cancel", "VET"}:           results.OutcomeDiagnostic,
		{"io-without-context-leak", "CTXVET"}: results.OutcomeDiagnostic,
		{"io-without-context-leak", "VET"}:    results.OutcomePass,
		{"dispatch-leak", "PROGRAM"}:          results.OutcomePass,
	} {
		if got[cell] != want {
			t.Errorf("%v : %s, %s attendu", cell, got[cell], want)
		}
	}
}

// TestUC001_NFR001_ProvenanceSysteme verrouille la révision du 2026-09-22 du modèle d'entités :
// nom commercial du processeur et version du système, lus sans dépendance.
func TestUC001_NFR001_ProvenanceSysteme(t *testing.T) {
	cpuinfo := "processor\t: 0\nvendor_id\t: GenuineIntel\nmodel name\t: Intel(R) Core(TM) Ultra 9 285K\nprocessor\t: 1\nmodel name\t: Intel(R) Core(TM) Ultra 9 285K\n"
	if got := cpuInfoModel(cpuinfo); got != "Intel(R) Core(TM) Ultra 9 285K" {
		t.Errorf("cpuInfoModel = %q", got)
	}
	if got := cpuInfoModel("processor\t: 0\nCPU part\t: 0xd0c\n"); got != "" {
		t.Errorf("cpuinfo sans model name : %q, vide attendu", got)
	}
	tests := []struct{ release, kernel, want string }{
		{"NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 24.04.1 LTS\"\n", "6.6.87.2-microsoft-standard-WSL2\n", "Ubuntu 24.04.1 LTS, noyau 6.6.87.2-microsoft-standard-WSL2"},
		{"", "6.1.0\n", "noyau 6.1.0"},
		{"PRETTY_NAME='Debian GNU/Linux 12'\n", "", "Debian GNU/Linux 12"},
		{"", "", ""},
	}
	for _, tt := range tests {
		if got := linuxOSVersion(tt.release, tt.kernel); got != tt.want {
			t.Errorf("linuxOSVersion(%q, %q) = %q, %q attendu", tt.release, tt.kernel, got, tt.want)
		}
	}
	if runtime.GOOS == "windows" {
		if name := cpuName(); name == "" || strings.HasPrefix(name, "Intel64 Family") || strings.HasPrefix(name, "AMD64 Family") {
			t.Errorf("cpuName = %q : signature CPUID au lieu du nom commercial", name)
		}
		if v := platformOSVersion(); !strings.HasPrefix(v, "Windows ") {
			t.Errorf("platformOSVersion = %q", v)
		}
	}
	t.Logf("cpu = %q, osVersion = %q", cpuName(), platformOSVersion())
}
