package gotool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// recorder capture les commandes demandées et rend une réponse programmée.
type recorder struct {
	result Result
	err    error
	calls  [][]string
	dirs   []string
}

func (r *recorder) run(_ context.Context, dir, name string, args ...string) (Result, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	r.dirs = append(r.dirs, dir)
	return r.result, r.err
}

func (r *recorder) last() string {
	if len(r.calls) == 0 {
		return ""
	}
	return strings.Join(r.calls[len(r.calls)-1], " ")
}

func TestResultCombined(t *testing.T) {
	t.Parallel()
	cases := []struct {
		result Result
		want   string
	}{
		{Result{Stdout: "a", Stderr: "b"}, "a\nb"},
		{Result{Stdout: "a"}, "a"},
		{Result{Stderr: "b"}, "b"},
		{Result{}, ""},
	}
	for _, tc := range cases {
		if got := tc.result.Combined(); got != tc.want {
			t.Fatalf("Combined() = %q, attendu %q", got, tc.want)
		}
	}
}

func TestEscapeAnalysis(t *testing.T) {
	t.Parallel()
	rec := &recorder{result: Result{Stderr: "subjects/x/subject.go:5:2: moved to heap: t\n\n  autre ligne  "}}
	lines, err := New(rec.run, "").EscapeAnalysis(context.Background(), "/matrix", "Size0024Plain/LOCAL/VALUE")
	if err != nil {
		t.Fatalf("EscapeAnalysis : %v", err)
	}
	if len(lines) != 2 || lines[1] != "autre ligne" {
		t.Fatalf("lignes = %q", lines)
	}
	// C-003 : la classification passe par `go build -gcflags=-m` sur le seul paquet du sujet.
	// Mutation : retirer -gcflags=-m ⇒ échec attendu.
	if got := rec.last(); !strings.Contains(got, "-gcflags=-m") || !strings.HasSuffix(got, "./subjects/Size0024Plain_LOCAL_VALUE") {
		t.Fatalf("commande = %q", got)
	}
	if rec.dirs[0] != "/matrix" {
		t.Fatalf("répertoire = %q", rec.dirs[0])
	}
}

func TestEscapeAnalysisCompileError(t *testing.T) {
	t.Parallel()
	rec := &recorder{result: Result{Stderr: "subject.go:3:1: syntax error", ExitCode: 1}}
	_, err := New(rec.run, "").EscapeAnalysis(context.Background(), "/matrix", "s")
	var compileErr *ports.CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("erreur = %v, *ports.CompileError attendue", err)
	}
	if compileErr.SubjectID != "s" || !strings.Contains(compileErr.Error(), "syntax error") {
		t.Fatalf("erreur mal renseignée : %+v", compileErr)
	}
}

func TestEscapeAnalysisErreurDExecution(t *testing.T) {
	t.Parallel()
	rec := &recorder{err: errors.New("go introuvable")}
	if _, err := New(rec.run, "").EscapeAnalysis(context.Background(), "/m", "s"); err == nil {
		t.Fatal("une erreur d'exécution doit remonter")
	}
	if err := New(rec.run, "").Build(context.Background(), "/m", "s"); err == nil {
		t.Fatal("une erreur d'exécution doit remonter")
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	if err := New(rec.run, "").Build(context.Background(), "/m", "probe/APPEND_GROW/1000"); err != nil {
		t.Fatalf("Build : %v", err)
	}
	if got := rec.last(); got != "go build ./subjects/probe_APPEND_GROW_1000" {
		t.Fatalf("commande = %q", got)
	}
	failing := &recorder{result: Result{Stderr: "boom", ExitCode: 2}}
	var compileErr *ports.CompileError
	if err := New(failing.run, "").Build(context.Background(), "/m", "s"); !errors.As(err, &compileErr) {
		t.Fatalf("erreur = %v, *ports.CompileError attendue", err)
	}
}

func TestParseBenchmarkOutput(t *testing.T) {
	t.Parallel()
	out := `goos: windows
goarch: amd64
BenchmarkSubject   	 1000000	         1.234 ns/op	       0 B/op	       0 allocs/op
BenchmarkSubject   	  900000	      2 ns/op	      16 B/op	       1 allocs/op
PASS
ok  	escapebench.local/matrix/M-1/subjects/s	0.5s`
	samples, err := ParseBenchmarkOutput(out)
	if err != nil {
		t.Fatalf("ParseBenchmarkOutput : %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("%d répétitions, 2 attendues", len(samples))
	}
	if samples[0].NsPerOp != 1.234 || samples[0].BytesPerOp != 0 || samples[0].AllocsPerOp != 0 {
		t.Fatalf("première répétition = %+v", samples[0])
	}
	if samples[1].BytesPerOp != 16 || samples[1].AllocsPerOp != 1 {
		t.Fatalf("seconde répétition = %+v", samples[1])
	}
	if _, err := ParseBenchmarkOutput("PASS\nok\n"); err == nil {
		t.Fatal("une sortie sans ligne de benchmark doit être une erreur")
	}
}

func TestRunMesureUnSujet(t *testing.T) {
	t.Parallel()
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("BenchmarkSubject   \t 100\t %d ns/op\t 8 B/op\t 1 allocs/op", i+1))
	}
	rec := &recorder{result: Result{Stdout: strings.Join(lines, "\n")}}
	m, err := New(rec.run, "").Run(context.Background(), "/m", "s", ports.RunOptions{Count: 20, BenchTime: "250ms", CPU: 1})
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if m.Status != models.MeasurementComplete {
		t.Fatalf("statut = %s : %s", m.Status, m.FailureReason)
	}
	if m.SubjectID != "s" {
		t.Fatalf("subjectId = %q", m.SubjectID)
	}
	// campaignId est renseigné par le service : la mesure est valide une fois rattachée.
	m.CampaignID = "C-1"
	if err := m.Validate(20); err != nil {
		t.Fatalf("NFR-003 : %v", err)
	}
	// C-003 : chaque sujet est mesuré dans un processus `go test` distinct, avec -benchmem,
	// -count, -benchtime et -cpu. Mutation : retirer -benchmem ⇒ échec attendu.
	command := rec.last()
	for _, flag := range []string{"-benchmem", "-count 20", "-benchtime 250ms", "-cpu 1", "-bench ^BenchmarkSubject$", "-run ^$"} {
		if !strings.Contains(command, flag) {
			t.Fatalf("%q absent de la commande %q", flag, command)
		}
	}
}

func TestRunSansCPU(t *testing.T) {
	t.Parallel()
	rec := &recorder{result: Result{Stdout: "BenchmarkSubject\t1\t1 ns/op\t0 B/op\t0 allocs/op"}}
	if _, err := New(rec.run, "").Run(context.Background(), "/m", "s", ports.RunOptions{Count: 1, BenchTime: "1ms"}); err != nil {
		t.Fatalf("Run : %v", err)
	}
	if strings.Contains(rec.last(), "-cpu") {
		t.Fatalf("-cpu ne doit pas être passé quand il n'est pas demandé : %q", rec.last())
	}
}

func TestRunEchecs(t *testing.T) {
	t.Parallel()
	// UC-003 A3 : un sujet qui échoue est consigné FAILED, la campagne se poursuit — Run ne
	// rend donc jamais d'erreur pour un échec de mesure.
	cases := []struct {
		name   string
		rec    *recorder
		opts   ports.RunOptions
		reason string
	}{
		{"processus en erreur", &recorder{err: errors.New("boom")}, ports.RunOptions{Count: 1}, "boom"},
		{"code de sortie non nul", &recorder{result: Result{Stderr: "panic", ExitCode: 2}}, ports.RunOptions{Count: 1}, "panic"},
		{"sortie illisible", &recorder{result: Result{Stdout: "rien"}}, ports.RunOptions{Count: 1}, "aucune ligne"},
		{"répétitions manquantes", &recorder{result: Result{Stdout: "BenchmarkSubject\t1\t1 ns/op\t0 B/op\t0 allocs/op"}}, ports.RunOptions{Count: 20}, "20 demandées"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := New(tc.rec.run, "").Run(context.Background(), "/m", "s", tc.opts)
			if err != nil {
				t.Fatalf("Run ne doit pas rendre d'erreur : %v", err)
			}
			if m.Status != models.MeasurementFailed {
				t.Fatalf("statut = %s", m.Status)
			}
			if !strings.Contains(m.FailureReason, tc.reason) {
				t.Fatalf("raison = %q, %q attendu", m.FailureReason, tc.reason)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got := truncate("abc", 10); got != "abc" {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncate("abcdef", 3); got != "abc… (tronqué)" {
		t.Fatalf("truncate = %q", got)
	}
}

func TestRunUseCaseTests(t *testing.T) {
	t.Parallel()
	rec := &recorder{result: Result{Stdout: "=== RUN   TestUC003_MainFlow\n=== RUN   TestUC003_A1_Insuffisant\n--- PASS: TestUC003_MainFlow\nPASS\n"}}
	outcome, err := New(rec.run, "/root").RunUseCaseTests(context.Background(), "UC-003")
	if err != nil {
		t.Fatalf("RunUseCaseTests : %v", err)
	}
	if outcome.Selected != 2 || !outcome.Passed {
		t.Fatalf("outcome = %+v", outcome)
	}
	// Les tests d'un cas d'utilisation sont nommés TestUC###_ : le motif doit refléter cette
	// convention, sinon le tableau de bord ne verrait jamais de test.
	if !strings.Contains(rec.last(), "^TestUC003_") {
		t.Fatalf("motif = %q", rec.last())
	}
	if rec.dirs[0] != "/root" {
		t.Fatalf("les tests tournent à la racine du module : %q", rec.dirs[0])
	}
}

func TestRunAll(t *testing.T) {
	t.Parallel()
	rec := &recorder{result: Result{Stdout: "ok\n", ExitCode: 1}}
	outcome, err := New(rec.run, "/root").RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll : %v", err)
	}
	if outcome.Passed {
		t.Fatal("un code de sortie non nul signale un échec")
	}
	failing := &recorder{err: errors.New("boom")}
	if _, err := New(failing.run, "/root").RunAll(context.Background()); err == nil {
		t.Fatal("une erreur d'exécution doit remonter")
	}
}

func TestNewSansRunnerUtiliseLExecutionReelle(t *testing.T) {
	t.Parallel()
	toolchain := New(nil, "")
	if toolchain.run == nil {
		t.Fatal("un runner nil doit retomber sur ExecRunner")
	}
}

func TestExecRunner(t *testing.T) {
	t.Parallel()
	result, err := ExecRunner(context.Background(), "", "go", "env", "GOARCH")
	if err != nil {
		t.Fatalf("ExecRunner : %v", err)
	}
	if result.ExitCode != 0 || strings.TrimSpace(result.Stdout) == "" {
		t.Fatalf("résultat = %+v", result)
	}
	failing, err := ExecRunner(context.Background(), "", "go", "cette-sous-commande-nexiste-pas")
	if err != nil {
		t.Fatalf("un code de sortie non nul n'est pas une erreur d'exécution : %v", err)
	}
	if failing.ExitCode == 0 {
		t.Fatal("code de sortie non nul attendu")
	}
	if _, err := ExecRunner(context.Background(), "", "binaire-inexistant-escapebench"); err == nil {
		t.Fatal("un binaire absent doit rendre une erreur")
	}
}
