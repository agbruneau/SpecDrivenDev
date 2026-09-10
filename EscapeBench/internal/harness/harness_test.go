package harness

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer : %v", err)
	}
	return renderer
}

func TestDigestStableEtNonVide(t *testing.T) {
	t.Parallel()
	// BR-003-1 : l'empreinte du harnais doit être reproductible d'un appel à l'autre, sinon
	// aucune campagne ne pourrait être validée.
	first := newRenderer(t).Digest()
	second := newRenderer(t).Digest()
	if first == "" || first != second {
		t.Fatalf("empreinte instable : %q vs %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("empreinte SHA-256 attendue sur 64 caractères, %d obtenus", len(first))
	}
}

func TestRenderCellPourTousLesProfilsEtModes(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	for _, size := range []int{8, 16, 24, 4096} {
		for _, hasPointer := range []bool{false, true} {
			for _, profile := range models.LifetimeProfiles() {
				for _, mode := range models.PassingModes() {
					spec, err := models.NewTypeSpec(size, hasPointer, models.LayoutArrayFill)
					if err != nil {
						t.Fatalf("NewTypeSpec : %v", err)
					}
					cell := models.Cell{TypeSpec: spec, Profile: profile, PassingMode: mode}
					cell.SourceFile = models.SourcePath(cell.ID())
					t.Run(cell.ID(), func(t *testing.T) {
						files, err := renderer.RenderCell(cell)
						if err != nil {
							t.Fatalf("RenderCell : %v", err)
						}
						source := requireFile(t, files, models.SourcePath(cell.ID()))
						assertParses(t, source)
						assertContains(t, source, "func Run(n int) uint64")
						assertContains(t, source, "func Setup()")
						// Le contrôle de taille à la compilation doit toujours être émis.
						assertContains(t, source, "unsafe.Sizeof")
						if !strings.Contains(source, "for i := 0; i < n; i++") {
							t.Fatal("le corps mesuré doit boucler sur n")
						}
						testFile := requireFile(t, files, "subjects/"+models.SubjectDir(cell.ID())+"/subject_test.go")
						assertParses(t, testFile)
						assertContains(t, testFile, "b.ReportAllocs()")
						assertContains(t, testFile, "b.ResetTimer()")
					})
				}
			}
		}
	}
}

func TestRenderCellNEmetQueLesAidesUtilisees(t *testing.T) {
	t.Parallel()
	// Une fonction inutilisée serait analysée par -gcflags=-m et produirait des lignes
	// d'échappement parasites qui fausseraient UC-002 et H-006.
	// Mutation : émettre producePointer pour un profil LOCAL ⇒ échec attendu.
	renderer := newRenderer(t)
	spec, _ := models.NewTypeSpec(24, false, models.LayoutArrayFill)
	cases := []struct {
		profile models.LifetimeProfile
		mode    models.PassingMode
		present []string
		absent  []string
	}{
		{models.ProfileLocal, models.PassingValue, []string{"consumeValue"}, []string{"consumePointer", "producePointer", "keepClosure"}},
		{models.ProfileLocal, models.PassingPointer, []string{"consumePointer"}, []string{"consumeValue", "produceValue", "keepClosure"}},
		{models.ProfileReturned, models.PassingValue, []string{"produceValue"}, []string{"producePointer", "consumeValue", "keepClosure"}},
		{models.ProfileReturned, models.PassingPointer, []string{"producePointer"}, []string{"produceValue", "consumePointer", "keepClosure"}},
		{models.ProfileCapturedByClosure, models.PassingValue, []string{"keepClosure", "keptClosure"}, []string{"consumeValue", "producePointer"}},
		{models.ProfileSentOnChannel, models.PassingPointer, []string{"chan *"}, []string{"keepClosure", "producePointer"}},
		{models.ProfileStoredInMap, models.PassingValue, []string{"map[int]"}, []string{"keepClosure", "consumePointer"}},
	}
	for _, tc := range cases {
		cell := models.Cell{TypeSpec: spec, Profile: tc.profile, PassingMode: tc.mode}
		cell.SourceFile = models.SourcePath(cell.ID())
		t.Run(cell.ID(), func(t *testing.T) {
			t.Parallel()
			files, err := renderer.RenderCell(cell)
			if err != nil {
				t.Fatalf("RenderCell : %v", err)
			}
			source := requireFile(t, files, cell.SourceFile)
			for _, needle := range tc.present {
				assertContains(t, source, needle)
			}
			for _, needle := range tc.absent {
				if strings.Contains(source, needle) {
					t.Fatalf("%q ne devrait pas être émis pour %s", needle, cell.ID())
				}
			}
		})
	}
}

func TestRenderCellDisposeLeRemplissage(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	// Un type de 8 octets n'a qu'un mot : aucun accès à Fill ne doit être émis, sinon l'indice
	// constant -1 empêche la compilation.
	spec8, _ := models.NewTypeSpec(8, false, models.LayoutArrayFill)
	cell8 := models.Cell{TypeSpec: spec8, Profile: models.ProfileLocal, PassingMode: models.PassingValue}
	cell8.SourceFile = models.SourcePath(cell8.ID())
	files, err := renderer.RenderCell(cell8)
	if err != nil {
		t.Fatalf("RenderCell : %v", err)
	}
	source := requireFile(t, files, cell8.SourceFile)
	assertContains(t, source, "Fill [0]uint64")
	if strings.Contains(source, "t.Fill[") {
		t.Fatal("un type d'un seul mot ne doit pas indexer Fill")
	}
	spec24, _ := models.NewTypeSpec(24, false, models.LayoutArrayFill)
	cell24 := models.Cell{TypeSpec: spec24, Profile: models.ProfileLocal, PassingMode: models.PassingValue}
	cell24.SourceFile = models.SourcePath(cell24.ID())
	files, err = renderer.RenderCell(cell24)
	if err != nil {
		t.Fatalf("RenderCell : %v", err)
	}
	source = requireFile(t, files, cell24.SourceFile)
	assertContains(t, source, "Fill [2]uint64")
	assertContains(t, source, "t.Fill[1]")
}

func TestRenderProbe(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	cases := []struct {
		probe   models.Probe
		present []string
	}{
		{models.Probe{Kind: models.ProbeSequentialScan, Parameter: 65536}, []string{"for j := range data", "elemCount"}},
		{models.Probe{Kind: models.ProbeScatteredScan, Parameter: 65536}, []string{"for j := range ptrs", "Fisher-Yates"}},
		{models.Probe{Kind: models.ProbeAppendPrealloc, Parameter: 1000}, []string{"make([]int, 0, count)"}},
		{models.Probe{Kind: models.ProbeAppendGrow, Parameter: 1000}, []string{"var buf []int"}},
	}
	for _, tc := range cases {
		probe := tc.probe
		probe.SourceFile = models.SourcePath(probe.ID())
		t.Run(probe.ID(), func(t *testing.T) {
			t.Parallel()
			files, err := renderer.RenderProbe(probe)
			if err != nil {
				t.Fatalf("RenderProbe : %v", err)
			}
			source := requireFile(t, files, probe.SourceFile)
			assertParses(t, source)
			for _, needle := range tc.present {
				assertContains(t, source, needle)
			}
			assertParses(t, requireFile(t, files, "subjects/"+models.SubjectDir(probe.ID())+"/subject_test.go"))
		})
	}
}

func TestRenderRefuseLesEntreesInvalides(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	if _, err := renderer.RenderCell(models.Cell{}); err == nil {
		t.Fatal("une Cell invalide doit être refusée")
	}
	if _, err := renderer.RenderProbe(models.Probe{}); err == nil {
		t.Fatal("une Probe invalide doit être refusée")
	}
	// Un jeu de travail qui n'est pas un multiple de la taille d'un élément produirait un
	// parcours dont le nombre d'éléments ne correspond pas au paramètre annoncé.
	probe := models.Probe{Kind: models.ProbeSequentialScan, Parameter: 100, SourceFile: "x"}
	if _, err := renderer.RenderProbe(probe); err == nil {
		t.Fatal("un jeu de travail non multiple de 64 octets doit être refusé")
	}
}

func TestRenderModule(t *testing.T) {
	t.Parallel()
	files, err := newRenderer(t).RenderModule("M-abc123")
	if err != nil {
		t.Fatalf("RenderModule : %v", err)
	}
	content := requireFile(t, files, "go.mod")
	// C-007 : chaque matrice est un module imbriqué, invisible de `go ./...` du module principal.
	assertContains(t, content, "module escapebench.local/matrix/M-abc123")
	assertContains(t, content, "go 1.25")
}

func requireFile(t *testing.T, files map[string]string, path string) string {
	t.Helper()
	content, ok := files[path]
	if !ok {
		keys := make([]string, 0, len(files))
		for key := range files {
			keys = append(keys, key)
		}
		t.Fatalf("fichier %s absent ; présents : %v", path, keys)
	}
	return content
}

func assertParses(t *testing.T, source string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "subject.go", source, parser.SkipObjectResolution); err != nil {
		t.Fatalf("le source généré n'est pas du Go valide : %v\n%s", err, source)
	}
}

func assertContains(t *testing.T, source, needle string) {
	t.Helper()
	if !strings.Contains(source, needle) {
		t.Fatalf("%q absent du source généré :\n%s", needle, source)
	}
}
