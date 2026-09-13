// Command leaklab est le composition root : run (UC-001), verdict (UC-002), ctxvet (UC-003).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/agbruneau/leaklab/internal/campaign"
	"github.com/agbruneau/leaklab/internal/ctxvet"
	"github.com/agbruneau/leaklab/internal/results"
	"github.com/agbruneau/leaklab/internal/spec"
	"github.com/agbruneau/leaklab/internal/verdict"
)

const usage = `usage :
  leaklab run [-root .] [-reps 5] [-timeout 5s]   exécuter une campagne (UC-001)
  leaklab verdict [-root .] -run <runId>          produire les verdicts (UC-002)
  leaklab ctxvet <répertoire>                     analyser un paquet (UC-003)`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "répertoire LeakLab")
	switch args[0] {
	case "run":
		reps := fs.Int("reps", campaign.MinReps, "répétitions par cellule et par bras (NFR-002)")
		timeout := fs.Duration("timeout", 5*time.Second, "délai réel par observation dynamique (C-004)")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		r, path, err := campaign.Run(ctx, campaign.Config{Root: *root, Reps: *reps, Timeout: *timeout, Log: stderr})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		counts := map[results.Outcome]int{}
		for _, o := range r.Observations {
			counts[o.Outcome]++
		}
		fmt.Fprintf(stdout, "campagne %s écrite dans %s\n", r.ID, path)
		for _, o := range []results.Outcome{results.OutcomePass, results.OutcomeFail, results.OutcomeHang, results.OutcomeDeadlock, results.OutcomeRace, results.OutcomeLeak, results.OutcomeDiagnostic} {
			fmt.Fprintf(stdout, "  %-10s %d\n", o, counts[o])
		}
		return 0
	case "verdict":
		runID := fs.String("run", "", "identifiant de la campagne")
		if fs.Parse(args[1:]) != nil || *runID == "" {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		r, err := results.LoadRun(filepath.Join(*root, "results", "runs", *runID+".json"))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		doc, err := os.ReadFile(filepath.Join(*root, "docs", "requirements.md"))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		hs, err := spec.Hypotheses(doc)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		vs, err := verdict.Evaluate(r, spec.Digests(hs))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		jsonPath, mdPath, err := verdict.Write(*root, r, vs, time.Now())
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for _, v := range vs {
			fmt.Fprintf(stdout, "%s  %-12s %s\n", v.HypothesisID, v.Outcome, v.Rationale)
		}
		fmt.Fprintf(stdout, "verdicts écrits dans %s et %s\n", jsonPath, mdPath)
		return 0
	case "ctxvet":
		if len(args) != 2 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		diags, err := ctxvet.Analyze(args[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		for _, d := range diags {
			fmt.Fprintln(stdout, d)
		}
		if len(diags) > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "sous-commande inconnue : %s\n%s\n", strings.TrimSpace(args[0]), usage)
	return 2
}
