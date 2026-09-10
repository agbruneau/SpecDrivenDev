// Command escapebench est le composition root du banc (BEPG ch. 14, p. 361 ; ch. 16, p. 414).
// Chaque sous-commande correspond à un cas d'utilisation de docs/use-cases :
//
//	matrix    UC-001  Générer la matrice de cellules
//	escape    UC-002  Classer l'échappement
//	campaign  UC-003  Exécuter une campagne de mesure
//	compare   UC-004  Comparer valeur et pointeur
//	verdict   UC-005  Produire les verdicts
//	dashboard UC-005 (étape 7) / FR-007
//
// Ce fichier ne contient aucune logique métier : il assemble adapters et services, puis met en
// forme ce que le cas d'utilisation rend observable.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbruneau/escapebench/internal/adapters/cli"
	"github.com/agbruneau/escapebench/internal/adapters/dashboard"
	"github.com/agbruneau/escapebench/internal/adapters/escape"
	"github.com/agbruneau/escapebench/internal/adapters/gotool"
	"github.com/agbruneau/escapebench/internal/adapters/specs"
	"github.com/agbruneau/escapebench/internal/adapters/store"
	"github.com/agbruneau/escapebench/internal/adapters/system"
	"github.com/agbruneau/escapebench/internal/harness"
	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/service"
)

const usage = `usage: escapebench <matrix|escape|campaign|compare|verdict|dashboard> [options]

Sous-commandes (une par cas d'utilisation, voir docs/use-cases) :
  matrix     UC-001  --reference | --params <spec>
  escape     UC-002  --matrix <id>
  campaign   UC-003  --matrix <id> --count <n≥20> [--hypotheses H-001,...] [--benchtime 250ms] [--cpu 1]
                     --resume <campaignId>
  compare    UC-004  --campaign <id>
  verdict    UC-005  --campaign <id> [--escape <fichier>]
  dashboard  UC-005  (régénère docs/dashboard.md)

Option commune : --root <répertoire du projet> (par défaut, remonte depuis le répertoire courant).

Spécification de matrice (--params) :
  sizes=8,16,24;pointer=false,true;profiles=LOCAL,RETURNED;modes=VALUE,POINTER;probes=SEQUENTIAL_SCAN:65536
  Clés reconnues : sizes, pointer, profiles, modes, layouts, repeats, payloads, replicates, probes.
  Les clés absentes prennent la valeur de la matrice de référence (BR-001-4).
`

// exit codes : 0 succès, 2 usage, 3 échec d'un cas d'utilisation.
const (
	exitUsage   = 2
	exitFailure = 3
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(exitUsage)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1], os.Args[2:]); err != nil {
		if errors.Is(err, cli.ErrUsage) {
			fmt.Fprintf(os.Stderr, "%v\n\n%s", err, usage)
			os.Exit(exitUsage)
		}
		fmt.Fprintf(os.Stderr, "escapebench %s : %v\n", os.Args[1], err)
		os.Exit(exitFailure)
	}
}

// run exécute une sous-commande.
func run(ctx context.Context, command string, args []string) error {
	switch command {
	case "matrix":
		return runMatrix(ctx, args)
	case "escape":
		return runEscape(ctx, args)
	case "campaign":
		return runCampaign(ctx, args)
	case "compare":
		return runCompare(ctx, args)
	case "verdict":
		return runVerdict(ctx, args)
	case "dashboard":
		return runDashboard(ctx, args)
	default:
		return fmt.Errorf("%w : sous-commande %q inconnue", cli.ErrUsage, command)
	}
}

// deps rassemble les adapters du projet.
type deps struct {
	root       string
	store      *store.Store
	renderer   *harness.Renderer
	toolchain  *gotool.Toolchain
	classifier escape.Classifier
	specs      *specs.Reader
	index      *specs.Index
	dashboard  *dashboard.Writer
	prober     *system.Prober
	clock      system.Clock
}

// HarnessDigest rend l'empreinte du harnais embarqué (BR-003-1).
func (d *deps) HarnessDigest(context.Context) (string, error) { return d.renderer.Digest(), nil }

// newDeps construit les adapters à partir de la racine du projet.
func newDeps(rootFlag string) (*deps, error) {
	start := rootFlag
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		start = cwd
	}
	root, err := cli.FindRoot(start)
	if err != nil {
		return nil, err
	}
	renderer, err := harness.NewRenderer()
	if err != nil {
		return nil, err
	}
	clock := system.Clock{}
	return &deps{
		root:       root,
		store:      store.New(root),
		renderer:   renderer,
		toolchain:  gotool.New(nil, root),
		classifier: escape.New(),
		specs:      specs.NewReader(root),
		index:      specs.NewIndex(root),
		dashboard:  dashboard.NewWriter(root),
		prober:     system.NewProber(clock, nil),
		clock:      clock,
	}, nil
}

// addRoot déclare l'option commune --root.
func addRoot(fs *flag.FlagSet) *string {
	return fs.String("root", "", "répertoire du projet (par défaut : remontée depuis le répertoire courant)")
}

// parse analyse les options d'une sous-commande.
func parse(fs *flag.FlagSet, args []string) error {
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w : %s", cli.ErrUsage, err)
	}
	return nil
}

// runMatrix exécute UC-001.
func runMatrix(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("matrix", flag.ContinueOnError)
	root := addRoot(fs)
	reference := fs.Bool("reference", false, "générer la matrice de référence (BR-001-4)")
	params := fs.String("params", "", "spécification de matrice")
	if err := parse(fs, args); err != nil {
		return err
	}
	if *reference == (*params != "") {
		return fmt.Errorf("%w : indiquer exactement l'une des options --reference ou --params", cli.ErrUsage)
	}
	parameters := models.ReferenceParameters()
	if *params != "" {
		parsed, err := cli.ParseParameters(*params)
		if err != nil {
			return err
		}
		parameters = parsed
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	generator := service.NewMatrixGenerator(d.store, d.renderer, d.toolchain, d, d.store, d.clock)
	report, err := generator.Generate(ctx, parameters)
	fmt.Print(cli.RenderMatrix(report))
	return err
}

// runEscape exécute UC-002.
func runEscape(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("escape", flag.ContinueOnError)
	root := addRoot(fs)
	matrixID := fs.String("matrix", "", "identifiant de la matrice")
	if err := parse(fs, args); err != nil {
		return err
	}
	if *matrixID == "" {
		return fmt.Errorf("%w : --matrix est obligatoire", cli.ErrUsage)
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	svc := service.NewEscapeService(d.store, d.toolchain, d.classifier, d.store, d.prober, d, d.clock)
	summary, err := svc.Classify(ctx, *matrixID)
	if err != nil {
		return err
	}
	fmt.Print(cli.RenderEscape(summary))
	return nil
}

// runCampaign exécute UC-003.
func runCampaign(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("campaign", flag.ContinueOnError)
	root := addRoot(fs)
	matrixID := fs.String("matrix", "", "identifiant de la matrice")
	count := fs.Int("count", models.MinCount, "nombre de répétitions par sujet (≥ 20, NFR-003)")
	hypotheses := fs.String("hypotheses", "", "hypothèses couvertes, séparées par des virgules")
	benchTime := fs.String("benchtime", "250ms", "durée de mesure par répétition (C-003)")
	cpu := fs.Int("cpu", 1, "valeur de -cpu pour les cellules (C-003)")
	resume := fs.String("resume", "", "reprendre une campagne interrompue (UC-003 A4)")
	if err := parse(fs, args); err != nil {
		return err
	}
	if *matrixID == "" && *resume == "" {
		return fmt.Errorf("%w : --matrix ou --resume est obligatoire", cli.ErrUsage)
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	svc := service.NewCampaignService(d.store, d.store, d.store, d.toolchain, d, d.prober, d.specs, specs.Digest, d.clock)
	report, err := svc.Run(ctx, service.CampaignOptions{
		MatrixID: *matrixID, Count: *count, HypothesisIDs: cli.ParseHypotheses(*hypotheses),
		BenchTime: *benchTime, CPU: *cpu, Resume: *resume,
	})
	fmt.Print(cli.RenderCampaign(report))
	return err
}

// runCompare exécute UC-004.
func runCompare(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	root := addRoot(fs)
	campaignID := fs.String("campaign", "", "identifiant de la campagne")
	if err := parse(fs, args); err != nil {
		return err
	}
	if *campaignID == "" {
		return fmt.Errorf("%w : --campaign est obligatoire", cli.ErrUsage)
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	svc := service.NewComparator(d.store, d.store, d, d.clock)
	report, err := svc.Compare(ctx, *campaignID)
	if err != nil {
		return err
	}
	fmt.Print(cli.RenderComparison(report))
	return nil
}

// runVerdict exécute UC-005.
func runVerdict(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("verdict", flag.ContinueOnError)
	root := addRoot(fs)
	campaignID := fs.String("campaign", "", "identifiant de la campagne")
	escapeFile := fs.String("escape", "", "fichier de verdicts d'échappement (H-006)")
	if err := parse(fs, args); err != nil {
		return err
	}
	if *campaignID == "" {
		return fmt.Errorf("%w : --campaign est obligatoire", cli.ErrUsage)
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	svc := newVerdictService(d)
	summary, err := svc.Produce(ctx, *campaignID, *escapeFile)
	if err != nil {
		return err
	}
	fmt.Print(cli.RenderVerdicts(summary))
	return nil
}

// runDashboard régénère docs/dashboard.md (UC-005, étape 7 ; FR-007).
func runDashboard(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	root := addRoot(fs)
	if err := parse(fs, args); err != nil {
		return err
	}
	d, err := newDeps(*root)
	if err != nil {
		return err
	}
	path, err := newVerdictService(d).Dashboard(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Tableau de bord régénéré : %s\n", path)
	return nil
}

// newVerdictService câble UC-005.
func newVerdictService(d *deps) *service.VerdictService {
	return service.NewVerdictService(d.store, d.store, d.store, d.store, d.specs, d.specs,
		d.index, d.toolchain, d.dashboard, specs.Digest, d.clock)
}
