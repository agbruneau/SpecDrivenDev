//go:build integration_test

// Tests d'intégration de UC-002 : ils rendent les sujets avec le harnais réel, les compilent avec
// la chaîne d'outils réelle (`go build -gcflags=-m`) et classent la sortie véritable du
// compilateur. C'est le seul niveau où le couple « motifs de classification / messages du
// compilateur » est réellement éprouvé : la suite unitaire fabrique ces messages à la main et ne
// verrait pas un changement de formulation du compilateur.
package escape_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/adapters/escape"
	"github.com/agbruneau/escapebench/internal/adapters/gotool"
	"github.com/agbruneau/escapebench/internal/harness"
	"github.com/agbruneau/escapebench/internal/models"
)

// renderMatrix écrit dans un répertoire temporaire le module et les sources des cellules données,
// exactement comme UC-001 le ferait sous matrices/.
func renderMatrix(t *testing.T, cells []models.Cell) string {
	t.Helper()
	renderer, err := harness.NewRenderer()
	if err != nil {
		t.Fatalf("harnais : %v", err)
	}
	dir := t.TempDir()
	files, err := renderer.RenderModule("M-integration")
	if err != nil {
		t.Fatalf("rendu du module : %v", err)
	}
	for _, cell := range cells {
		cellFiles, err := renderer.RenderCell(cell)
		if err != nil {
			t.Fatalf("rendu de la cellule %s : %v", cell.ID(), err)
		}
		for rel, content := range cellFiles {
			files[rel] = content
		}
	}
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("création de %s : %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("écriture de %s : %v", rel, err)
		}
	}
	return dir
}

// TestUC002_ClassificationSurCompilateurReel compile chaque cellule et exige que la catégorie
// rendue soit celle que le profil de durée de vie impose.
//
// La cellule RETURNED_ALLOCATING en mode VALUE est la raison d'être de ce test (A-072) : le
// compilateur y nomme l'expression d'allocation (`new(payload) escapes to heap`) et non la
// variable. Classée OTHER, elle faisait compter 120 cellules « hors des quatre causes du livre »
// et réfuter H-006 à tort. La cause est le retour du pointeur, la première des quatre.
func TestUC002_ClassificationSurCompilateurReel(t *testing.T) {
	params := models.MatrixParameters{
		Sizes:                []int{24},
		PointerFieldVariants: []bool{false},
		Profiles: []models.LifetimeProfile{
			models.ProfileLocal, models.ProfileReturned, models.ProfileCapturedByClosure,
			models.ProfileSentOnChannel, models.ProfileStoredInMap, models.ProfileStoredInSlice,
			models.ProfileStoredInStruct, models.ProfileReturnedAlloc,
		},
		PassingModes: models.PassingModes(),
		Layouts:      models.DefaultLayouts(),
		Repeats:      models.DefaultRepeats(),
		Payloads:     models.DefaultPayloads(),
		Replicates:   models.DefaultReplicates(),
		Probes:       []models.ProbeSpec{{Kind: models.ProbeAppendGrow, Parameter: 1000}},
	}
	matrix, err := models.NewMatrix(params, "integration", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("matrice : %v", err)
	}
	dir := renderMatrix(t, matrix.Cells)

	ctx := context.Background()
	compiler := gotool.New(nil, "")
	classifier := escape.New()

	// Ce que BR-002-2 impose pour chaque profil échappant : la catégorie nomme la cause, jamais
	// OTHER, qui est réservé aux raisons hors des quatre causes du livre.
	expected := map[models.LifetimeProfile]models.EscapeCategory{
		models.ProfileReturned:          models.CategoryReturnPointer,
		models.ProfileCapturedByClosure: models.CategoryClosureCapture,
		models.ProfileSentOnChannel:     models.CategoryChannelSend,
		models.ProfileStoredInMap:       models.CategoryContainerStore,
		models.ProfileStoredInSlice:     models.CategoryContainerStore,
		models.ProfileStoredInStruct:    models.CategoryContainerStore,
		models.ProfileReturnedAlloc:     models.CategoryReturnPointer,
	}

	for _, cell := range matrix.Cells {
		want, judged := expected[cell.Profile]
		if !judged {
			continue
		}
		t.Run(cell.ID(), func(t *testing.T) {
			lines, err := compiler.EscapeAnalysis(ctx, dir, cell.ID())
			if err != nil {
				t.Fatalf("compilation : %v", err)
			}
			verdict, err := classifier.Classify(ctx, dir, cell.ID(), cell.SourceFile, lines)
			if err != nil {
				t.Fatalf("classification : %v", err)
			}
			// En mode VALUE, passer une copie ne fait rien échapper : seul le profil
			// RETURNED_ALLOCATING échappe dans les deux modes, par la charge qu'il alloue.
			mustEscape := cell.PassingMode == models.PassingPointer ||
				cell.Profile == models.ProfileReturnedAlloc
			if !verdict.Escapes {
				if mustEscape {
					t.Fatalf("le profil %s en mode %s doit faire échapper la valeur ; lignes : %v",
						cell.Profile, cell.PassingMode, lines)
				}
				return
			}
			if verdict.Category != want {
				t.Fatalf("catégorie = %s, attendue %s (raison retenue : %s)",
					verdict.Category, want, verdict.CompilerReason)
			}
		})
	}
}
