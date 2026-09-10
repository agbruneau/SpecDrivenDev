package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

type escapeFixture struct {
	service    *EscapeService
	store      *memoryStore
	compiler   *fakeCompiler
	classifier *fakeClassifier
	provenance *fakeProvenance
	digester   *fakeDigester
	clock      *fakeClock
	matrix     models.Matrix
}

func newEscapeFixture(t *testing.T) *escapeFixture {
	t.Helper()
	f := &escapeFixture{
		store:      newStore(),
		compiler:   &fakeCompiler{failing: map[string]string{}, lines: map[string][]string{}},
		classifier: &fakeClassifier{categories: map[string]models.EscapeCategory{}},
		provenance: newProvenance(),
		digester:   &fakeDigester{digest: "harness-v1"},
		clock:      newClock(),
	}
	matrix, err := models.NewMatrix(smallParameters(), "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.matrix = matrix
	f.store.matrices[matrix.ID] = matrix
	f.service = NewEscapeService(f.store, f.compiler, f.classifier, f.store, f.provenance, f.digester, f.clock)
	return f
}

func TestUC002_MainFlow(t *testing.T) {
	t.Parallel()
	f := newEscapeFixture(t)
	f.classifier.categories[f.matrix.Cells[0].ID()] = models.CategoryReturnPointer
	f.classifier.categories[f.matrix.Cells[1].ID()] = models.CategoryOther
	f.compiler.lines[f.matrix.Cells[0].ID()] = []string{"subject.go:5:2: moved to heap: t"}

	summary, err := f.service.Classify(context.Background(), f.matrix.ID)
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	// Étape 6 : un fichier de verdicts avec la Provenance et un EscapeVerdict par Cell.
	if summary.Path == "" {
		t.Fatal("un fichier de verdicts doit être écrit")
	}
	report := f.store.escapes[f.matrix.ID][0]
	if len(report.Verdicts) != len(f.matrix.Cells) {
		t.Fatalf("%d verdicts pour %d cellules", len(report.Verdicts), len(f.matrix.Cells))
	}
	if err := report.Provenance.Validate(); err != nil {
		t.Fatalf("NFR-001 : %v", err)
	}
	// Les sondes ne sont jamais classées par UC-002.
	for _, verdict := range report.Verdicts {
		if _, _, isProbe := models.ParseProbeID(verdict.CellID); isProbe {
			t.Fatalf("une sonde a été classée : %s", verdict.CellID)
		}
	}
	// Étape 7 : décompte par catégorie et nombre de OTHER.
	if summary.Counts[models.CategoryReturnPointer] != 1 || summary.OtherCount != 1 {
		t.Fatalf("décompte = %v", summary.Counts)
	}
	if summary.Counts[models.CategoryNone] != len(f.matrix.Cells)-2 {
		t.Fatalf("décompte NONE = %d", summary.Counts[models.CategoryNone])
	}
	if summary.CompileErrors != 0 {
		t.Fatalf("aucune erreur de compilation attendue, %d obtenues", summary.CompileErrors)
	}
	if summary.Reproducibility != nil {
		t.Fatal("aucun rapport antérieur : aucune comparaison de reproductibilité")
	}
}

func TestUC002_A1_HarnaisModifie(t *testing.T) {
	t.Parallel()
	// A1 : le système refuse, affiche les deux empreintes et n'écrit aucun fichier.
	// Mutation : retirer la vérification d'empreinte ⇒ échec attendu.
	f := newEscapeFixture(t)
	f.digester.digest = "harness-v2"
	_, err := f.service.Classify(context.Background(), f.matrix.ID)
	if !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
	if len(f.store.escapes) != 0 {
		t.Fatal("A1 : results/ reste inchangé")
	}
}

func TestUC002_A2_CelluleNonCompilable(t *testing.T) {
	t.Parallel()
	// A2 : la cellule est consignée COMPILE_ERROR, la classification se poursuit.
	f := newEscapeFixture(t)
	fautive := f.matrix.Cells[0].ID()
	f.compiler.failing[fautive] = "subject.go:3:1: undefined: x"

	summary, err := f.service.Classify(context.Background(), f.matrix.ID)
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if summary.CompileErrors != 1 {
		t.Fatalf("%d cellules en COMPILE_ERROR, 1 attendue", summary.CompileErrors)
	}
	report := f.store.escapes[f.matrix.ID][0]
	if len(report.Verdicts) != len(f.matrix.Cells) {
		t.Fatalf("la classification doit se poursuivre : %d verdicts", len(report.Verdicts))
	}
	for _, verdict := range report.Verdicts {
		if verdict.CellID != fautive {
			continue
		}
		if verdict.Status != models.EscapeStatusCompileError || verdict.CompilerError == "" {
			t.Fatalf("verdict de la cellule fautive = %+v", verdict)
		}
	}
}

func TestUC002_A3_ReproductibiliteRespectee(t *testing.T) {
	t.Parallel()
	// A3 : un second passage sur la même toolchain compare les deux fichiers.
	f := newEscapeFixture(t)
	ctx := context.Background()
	f.classifier.categories[f.matrix.Cells[0].ID()] = models.CategoryReturnPointer
	if _, err := f.service.Classify(ctx, f.matrix.ID); err != nil {
		t.Fatalf("Classify : %v", err)
	}
	f.provenance.provenance.CapturedAt = f.provenance.provenance.CapturedAt.Add(time.Hour)
	summary, err := f.service.Classify(ctx, f.matrix.ID)
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if summary.Reproducibility == nil {
		t.Fatal("A3 : la comparaison doit avoir lieu")
	}
	if summary.Reproducibility.Violation {
		t.Fatalf("NFR-002 : aucun écart attendu, %v obtenus", summary.Reproducibility.Differing)
	}
	if len(f.store.escapes[f.matrix.ID]) != 2 {
		t.Fatal("BR-002-3 : un nouveau fichier horodaté est écrit, l'ancien reste")
	}
}

func TestUC002_A3_ViolationDeNFR002(t *testing.T) {
	t.Parallel()
	// NFR-002 : deux exécutions sur la même toolchain doivent donner des verdicts identiques.
	// Mutation : ne pas comparer les catégories ⇒ échec attendu.
	f := newEscapeFixture(t)
	ctx := context.Background()
	cell := f.matrix.Cells[0].ID()
	f.classifier.categories[cell] = models.CategoryReturnPointer
	if _, err := f.service.Classify(ctx, f.matrix.ID); err != nil {
		t.Fatalf("Classify : %v", err)
	}
	f.classifier.categories[cell] = models.CategoryChannelSend
	f.provenance.provenance.CapturedAt = f.provenance.provenance.CapturedAt.Add(time.Hour)
	summary, err := f.service.Classify(ctx, f.matrix.ID)
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if summary.Reproducibility == nil || !summary.Reproducibility.Violation {
		t.Fatal("la divergence doit être signalée comme violation de NFR-002")
	}
	if len(summary.Reproducibility.Differing) != 1 || summary.Reproducibility.Differing[0] != cell {
		t.Fatalf("cellules divergentes = %v", summary.Reproducibility.Differing)
	}
}

func TestUC002_A3_ToolchainDifferenteNestPasComparee(t *testing.T) {
	t.Parallel()
	f := newEscapeFixture(t)
	ctx := context.Background()
	if _, err := f.service.Classify(ctx, f.matrix.ID); err != nil {
		t.Fatalf("Classify : %v", err)
	}
	f.provenance.provenance.GoVersion = "go1.26.0"
	summary, err := f.service.Classify(ctx, f.matrix.ID)
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if summary.Reproducibility != nil {
		t.Fatal("NFR-002 ne porte que sur une même toolchain")
	}
}

func TestUC002_PreconditionsEtErreurs(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	t.Run("matrice absente", func(t *testing.T) {
		t.Parallel()
		f := newEscapeFixture(t)
		if _, err := f.service.Classify(context.Background(), "M-inconnue"); err == nil {
			t.Fatal("Classify aurait dû échouer")
		}
	})
	t.Run("matrice sans cellule", func(t *testing.T) {
		t.Parallel()
		f := newEscapeFixture(t)
		sansCell := f.matrix
		sansCell.ID = "M-sonly"
		sansCell.Cells = nil
		f.store.matrices[sansCell.ID] = sansCell
		_, err := f.service.Classify(context.Background(), sansCell.ID)
		if !errors.Is(err, ErrPrecondition) {
			t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
		}
	})
	for name, breakIt := range map[string]func(*escapeFixture){
		"empreinte illisible":  func(f *escapeFixture) { f.digester.err = boom },
		"provenance illisible": func(f *escapeFixture) { f.provenance.err = boom },
		"compilateur en panne": func(f *escapeFixture) { f.compiler.err = boom },
		"classificateur cassé": func(f *escapeFixture) { f.classifier.err = boom },
		"écriture impossible":  func(f *escapeFixture) { f.store.writeErr = boom },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newEscapeFixture(t)
			breakIt(f)
			if _, err := f.service.Classify(context.Background(), f.matrix.ID); err == nil {
				t.Fatal("Classify aurait dû échouer")
			}
		})
	}
}

func TestUC002_VerdictInvalideRefuse(t *testing.T) {
	t.Parallel()
	// Un classificateur qui produirait un verdict incohérent (échappe sans catégorie) doit
	// arrêter le cas d'utilisation plutôt qu'écrire un fichier invalide (BR-002-1).
	f := newEscapeFixture(t)
	f.classifier.categories[f.matrix.Cells[0].ID()] = "INTERFACE"
	if _, err := f.service.Classify(context.Background(), f.matrix.ID); err == nil {
		t.Fatal("Classify aurait dû refuser un verdict invalide")
	}
	if len(f.store.escapes) != 0 {
		t.Fatal("aucun fichier ne doit être écrit")
	}
}

func TestDifferingVerdicts(t *testing.T) {
	t.Parallel()
	previous := models.EscapeReport{Verdicts: []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryReturnPointer, Status: models.EscapeStatusOK},
		{CellID: "b", Category: models.CategoryNone, Status: models.EscapeStatusOK},
	}}
	current := models.EscapeReport{Verdicts: []models.EscapeVerdict{
		{CellID: "a", Escapes: true, Category: models.CategoryChannelSend, Status: models.EscapeStatusOK},
		{CellID: "b", Category: models.CategoryNone, Status: models.EscapeStatusOK},
		{CellID: "c", Category: models.CategoryNone, Status: models.EscapeStatusOK},
	}}
	differing := DifferingVerdicts(previous, current)
	if len(differing) != 2 || differing[0] != "a" || differing[1] != "c" {
		t.Fatalf("divergences = %v", differing)
	}
	if len(DifferingVerdicts(previous, previous)) != 0 {
		t.Fatal("deux rapports identiques ne divergent pas")
	}
}
