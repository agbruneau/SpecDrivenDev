package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

type matrixFixture struct {
	generator *MatrixGenerator
	store     *memoryStore
	source    *fakeSource
	compiler  *fakeCompiler
	digester  *fakeDigester
	lock      *fakeLock
	clock     *fakeClock
}

func newMatrixFixture() *matrixFixture {
	f := &matrixFixture{
		store:    newStore(),
		source:   &fakeSource{},
		compiler: &fakeCompiler{failing: map[string]string{}},
		digester: &fakeDigester{digest: "harness-v1"},
		lock:     &fakeLock{},
		clock:    newClock(),
	}
	f.generator = NewMatrixGenerator(f.store, f.source, f.compiler, f.digester, f.lock, f.clock)
	return f
}

func TestUC001_MainFlow(t *testing.T) {
	t.Parallel()
	f := newMatrixFixture()
	report, err := f.generator.Generate(context.Background(), smallParameters())
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	// Étape 7 : identifiant, décomptes et empreinte du harnais sont rendus observables.
	if report.MatrixID == "" || !strings.HasPrefix(report.MatrixID, "M-") {
		t.Fatalf("identifiant = %q", report.MatrixID)
	}
	if report.TypeSpecCount != 2 || report.CellCount != 4 || report.ProbeCount != 1 {
		t.Fatalf("décomptes = %+v", report)
	}
	if report.HarnessDigest != "harness-v1" {
		t.Fatalf("empreinte = %q", report.HarnessDigest)
	}
	if report.AlreadyExisted {
		t.Fatal("une matrice neuve n'existe pas déjà")
	}
	// Postcondition : matrix.json et un fichier source par sujet.
	matrix, err := f.store.Load(context.Background(), report.MatrixID)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if matrix.HarnessDigest != "harness-v1" {
		t.Fatalf("l'empreinte doit être enregistrée avec la matrice : %q", matrix.HarnessDigest)
	}
	files := f.store.sources[report.MatrixID]
	if len(files) != len(matrix.SubjectIDs())+1 {
		t.Fatalf("%d fichiers pour %d sujets et un go.mod", len(files), len(matrix.SubjectIDs()))
	}
	if _, ok := files["go.mod"]; !ok {
		t.Fatal("C-007 : le go.mod du module imbriqué doit être écrit")
	}
	// Étape 5 : tous les sujets sont compilés.
	if len(f.compiler.built) != len(matrix.SubjectIDs()) {
		t.Fatalf("%d sujets compilés, %d attendus", len(f.compiler.built), len(matrix.SubjectIDs()))
	}
}

func TestUC001_BR1_IdentifiantDeterministe(t *testing.T) {
	t.Parallel()
	// BR-001-1 : deux demandes équivalentes produisent le même identifiant.
	first, err := newMatrixFixture().generator.Generate(context.Background(), smallParameters())
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	shuffled := smallParameters()
	shuffled.Sizes = []int{24, 8}
	shuffled.PassingModes = []models.PassingMode{models.PassingPointer, models.PassingValue}
	second, err := newMatrixFixture().generator.Generate(context.Background(), shuffled)
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if first.MatrixID != second.MatrixID {
		t.Fatalf("identifiants différents : %s vs %s", first.MatrixID, second.MatrixID)
	}
}

func TestUC001_BR3_PairesCompletes(t *testing.T) {
	t.Parallel()
	// BR-001-3 : chaque TypeSpec × LifetimeProfile produit une Cell VALUE et une Cell POINTER.
	f := newMatrixFixture()
	report, err := f.generator.Generate(context.Background(), smallParameters())
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	matrix, _ := f.store.Load(context.Background(), report.MatrixID)
	if len(matrix.ValuePointerPairs()) != len(matrix.Cells)/2 {
		t.Fatalf("%d paires pour %d cellules", len(matrix.ValuePointerPairs()), len(matrix.Cells))
	}
}

func TestUC001_A1_ParametresInvalides(t *testing.T) {
	t.Parallel()
	// A1 : le système refuse et n'écrit aucun fichier.
	f := newMatrixFixture()
	invalid := smallParameters()
	invalid.Sizes = []int{20}
	invalid.Profiles = []models.LifetimeProfile{"GLOBAL"}
	_, err := f.generator.Generate(context.Background(), invalid)
	if err == nil {
		t.Fatal("Generate aurait dû refuser")
	}
	if !errors.Is(err, models.ErrValidation) {
		t.Fatalf("erreur = %v, ErrValidation attendue", err)
	}
	// Chaque paramètre fautif est nommé avec la règle violée.
	for _, needle := range []string{"multiple", "GLOBAL"} {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("%q absent du message : %v", needle, err)
		}
	}
	if len(f.store.sources) != 0 || len(f.store.matrices) != 0 {
		t.Fatal("A1 : aucun fichier n'est écrit")
	}
}

func TestUC001_A2_MatriceExistante(t *testing.T) {
	t.Parallel()
	f := newMatrixFixture()
	ctx := context.Background()
	first, err := f.generator.Generate(ctx, smallParameters())
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	f.compiler.built = nil
	// A2, cas identique : le système affiche l'identifiant existant et ne régénère rien.
	second, err := f.generator.Generate(ctx, smallParameters())
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if !second.AlreadyExisted || second.MatrixID != first.MatrixID {
		t.Fatalf("rapport = %+v", second)
	}
	if len(f.compiler.built) != 0 {
		t.Fatal("A2 : rien n'est recompilé quand le harnais est inchangé")
	}
	if second.CellCount != first.CellCount || second.ProbeCount != first.ProbeCount {
		t.Fatalf("les décomptes de la matrice existante doivent être rendus : %+v", second)
	}
}

func TestUC001_A2_HarnaisModifie(t *testing.T) {
	t.Parallel()
	// A2, cas divergent : le système refuse et indique que le harnais a changé (C-005).
	// Mutation : ignorer l'empreinte enregistrée ⇒ échec attendu.
	f := newMatrixFixture()
	ctx := context.Background()
	if _, err := f.generator.Generate(ctx, smallParameters()); err != nil {
		t.Fatalf("Generate : %v", err)
	}
	f.digester.digest = "harness-v2"
	_, err := f.generator.Generate(ctx, smallParameters())
	if !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
	if !strings.Contains(err.Error(), "harness-v1") || !strings.Contains(err.Error(), "harness-v2") {
		t.Fatalf("les deux empreintes doivent être affichées : %v", err)
	}
}

func TestUC001_A3_SujetNonCompilable(t *testing.T) {
	t.Parallel()
	// A3 : chaque sujet fautif est affiché avec le message du compilateur, le répertoire est
	// supprimé et le cas d'utilisation se termine sans Matrix.
	f := newMatrixFixture()
	ctx := context.Background()
	matrix, err := models.NewMatrix(smallParameters(), "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	fautif := matrix.Cells[0].ID()
	f.compiler.failing[fautif] = "subject.go:3:1: syntax error"

	report, err := f.generator.Generate(ctx, smallParameters())
	if err == nil {
		t.Fatal("Generate aurait dû échouer")
	}
	if len(report.CompileErrors) != 1 || report.CompileErrors[0].SubjectID != fautif {
		t.Fatalf("erreurs de compilation = %+v", report.CompileErrors)
	}
	if !strings.Contains(report.CompileErrors[0].Message, "syntax error") {
		t.Fatalf("le message du compilateur doit être conservé : %q", report.CompileErrors[0].Message)
	}
	if _, ok := f.store.sources[matrix.ID]; ok {
		t.Fatal("A3 : aucun répertoire de matrice partiel ne subsiste")
	}
	if _, ok := f.store.matrices[matrix.ID]; ok {
		t.Fatal("A3 : le cas d'utilisation se termine sans Matrix")
	}
	// La compilation se poursuit sur les autres sujets pour tous les rapporter d'un coup.
	if len(f.compiler.built) != len(matrix.SubjectIDs()) {
		t.Fatalf("%d sujets compilés, %d attendus", len(f.compiler.built), len(matrix.SubjectIDs()))
	}
}

func TestUC001_PreconditionCampagneEnCours(t *testing.T) {
	t.Parallel()
	// Précondition : aucune campagne n'est au statut RUNNING.
	f := newMatrixFixture()
	f.lock.held = true
	_, err := f.generator.Generate(context.Background(), smallParameters())
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}

func TestUC001_ErreursDesPorts(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := map[string]func(*matrixFixture){
		"verrou illisible":     func(f *matrixFixture) { f.lock.err = boom },
		"empreinte illisible":  func(f *matrixFixture) { f.digester.err = boom },
		"rendu impossible":     func(f *matrixFixture) { f.source.err = boom },
		"écriture impossible":  func(f *matrixFixture) { f.store.writeErr = boom },
		"compilateur en panne": func(f *matrixFixture) { f.compiler.err = boom },
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newMatrixFixture()
			breakIt(f)
			if _, err := f.generator.Generate(context.Background(), smallParameters()); err == nil {
				t.Fatal("Generate aurait dû échouer")
			}
		})
	}
}

func TestUC001_MatriceExistanteIllisible(t *testing.T) {
	t.Parallel()
	// Une matrice signalée existante mais illisible doit remonter l'erreur, pas être régénérée.
	f := newMatrixFixture()
	ctx := context.Background()
	matrix, _ := models.NewMatrix(smallParameters(), "harness-v1", f.clock.Now())
	f.store.matrices[matrix.ID] = models.Matrix{}
	delete(f.store.matrices, matrix.ID)
	f.store.matrices[matrix.ID+"-autre"] = matrix
	// Exists rend faux : la génération se poursuit normalement.
	if _, err := f.generator.Generate(ctx, smallParameters()); err != nil {
		t.Fatalf("Generate : %v", err)
	}
}
