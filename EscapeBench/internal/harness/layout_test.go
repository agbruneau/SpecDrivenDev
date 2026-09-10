package harness

import (
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// cellOf construit une cellule prête à rendre.
func cellOf(t *testing.T, size int, hasPointer bool, layout models.Layout, profile models.LifetimeProfile, mode models.PassingMode, repeat int) models.Cell {
	t.Helper()
	spec, err := models.NewTypeSpec(size, hasPointer, layout)
	if err != nil {
		t.Fatalf("NewTypeSpec : %v", err)
	}
	cell := models.Cell{TypeSpec: spec, Profile: profile, PassingMode: mode, Repeat: repeat}
	cell.SourceFile = models.SourcePath(cell.ID())
	return cell
}

func TestRenderChampsNommesSansTableau(t *testing.T) {
	t.Parallel()
	// C-008 : la disposition à champs nommés ne doit contenir aucun tableau, faute de quoi le
	// type sortirait des registres et mesurerait la même chose que la disposition d'origine.
	// Mutation : émettre un Fill dans la branche NAMED_FIELDS ⇒ échec attendu.
	renderer := newRenderer(t)
	cases := []struct {
		size    int
		pointer bool
		fields  []string
		absent  []string
	}{
		{8, false, []string{"F0 uint64"}, []string{"Fill", "F1"}},
		{8, true, []string{"P *uint64"}, []string{"Fill", "F0"}},
		{16, false, []string{"F0 uint64", "F1 uint64"}, []string{"Fill"}},
		{16, true, []string{"F0 uint64", "P *uint64"}, []string{"Fill", "F1 uint64"}},
		{24, false, []string{"F0 uint64", "F1 uint64", "F2 uint64"}, []string{"Fill"}},
		{80, false, []string{"F0 uint64", "F9 uint64"}, []string{"Fill"}},
	}
	for _, tc := range cases {
		cell := cellOf(t, tc.size, tc.pointer, models.LayoutNamedFields, models.ProfileLocal, models.PassingValue, 1)
		t.Run(cell.ID(), func(t *testing.T) {
			t.Parallel()
			files, err := renderer.RenderCell(cell)
			if err != nil {
				t.Fatalf("RenderCell : %v", err)
			}
			source := requireFile(t, files, cell.SourceFile)
			assertParses(t, source)
			// gofmt aligne les déclarations de champs : la comparaison ignore les espaces
			// multiples plutôt que d'imposer une mise en forme au générateur.
			collapsed := strings.Join(strings.Fields(source), " ")
			for _, needle := range tc.fields {
				if !strings.Contains(collapsed, needle) {
					t.Fatalf("%q absent du source de %s :\n%s", needle, cell.ID(), source)
				}
			}
			for _, needle := range tc.absent {
				if strings.Contains(collapsed, needle) {
					t.Fatalf("%q ne devrait pas figurer dans %s :\n%s", needle, cell.ID(), source)
				}
			}
			// Le contrôle de taille à la compilation reste émis pour chaque disposition.
			assertContains(t, source, "unsafe.Sizeof")
			// L'ancre suit le type du champ pointeur.
			if tc.pointer {
				assertContains(t, source, "var anchor uint64")
			}
		})
	}
}

func TestRenderTemoinNulExecuteLeCorpsValeur(t *testing.T) {
	t.Parallel()
	// Le témoin nul mesure l'écart entre deux binaires : ses deux cellules doivent exécuter le
	// même travail, celui du mode valeur, sinon son delta ne serait pas nul par construction.
	// Mutation : laisser le mode POINTER émettre consumePointer ⇒ échec attendu.
	renderer := newRenderer(t)
	for _, mode := range models.PassingModes() {
		cell := cellOf(t, 24, false, models.LayoutNamedFieldsSham, models.ProfileLocal, mode, 1)
		files, err := renderer.RenderCell(cell)
		if err != nil {
			t.Fatalf("RenderCell : %v", err)
		}
		source := requireFile(t, files, cell.SourceFile)
		assertParses(t, source)
		assertContains(t, source, "consumeValue")
		if strings.Contains(source, "consumePointer") {
			t.Fatalf("%s ne doit pas passer par pointeur :\n%s", cell.ID(), source)
		}
	}
}

func TestRenderProfilsConteneurs(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	cases := []struct {
		profile models.LifetimeProfile
		mode    models.PassingMode
		present []string
	}{
		{models.ProfileStoredInSlice, models.PassingPointer, []string{"make([]*", "c[0] = &t"}},
		{models.ProfileStoredInSlice, models.PassingValue, []string{"make([]Size", "c[0] = t"}},
		{models.ProfileStoredInStruct, models.PassingPointer, []string{"type holder struct", "new(holder)", "c.P = &t"}},
		{models.ProfileStoredInStruct, models.PassingValue, []string{"type holder struct", "c.V = t"}},
	}
	for _, tc := range cases {
		cell := cellOf(t, 24, false, models.LayoutArrayFill, tc.profile, tc.mode, 1)
		t.Run(cell.ID(), func(t *testing.T) {
			t.Parallel()
			files, err := renderer.RenderCell(cell)
			if err != nil {
				t.Fatalf("RenderCell : %v", err)
			}
			source := requireFile(t, files, cell.SourceFile)
			assertParses(t, source)
			for _, needle := range tc.present {
				assertContains(t, source, needle)
			}
		})
	}
}

func TestRenderProfilQuiAlloueDejaCoteValeur(t *testing.T) {
	t.Parallel()
	// H-010 exige une base d'allocation non nulle des deux côtés, qui varie avec le nombre
	// d'instances produites par opération.
	renderer := newRenderer(t)
	for _, repeat := range []int{1, 4} {
		for _, mode := range models.PassingModes() {
			cell := cellOf(t, 24, true, models.LayoutArrayFill, models.ProfileReturnedAlloc, mode, repeat)
			files, err := renderer.RenderCell(cell)
			if err != nil {
				t.Fatalf("RenderCell : %v", err)
			}
			source := requireFile(t, files, cell.SourceFile)
			assertParses(t, source)
			// La charge allouée accompagne chaque instance, des deux côtés de la paire.
			assertContains(t, source, "type payload struct")
			assertContains(t, source, "new(payload)")
			if repeat > 1 {
				assertContains(t, source, "for r := 0; r < 4; r++")
			}
			if mode == models.PassingPointer {
				assertContains(t, source, "producePointerAlloc")
			} else {
				assertContains(t, source, "produceValueAlloc")
			}
		}
	}
}

func TestRenderSondeAChaineDependante(t *testing.T) {
	t.Parallel()
	// La sonde de H-008 doit émettre des chargements dépendants : l'adresse suivante est lue
	// dans le nœud courant. Mutation : parcourir une tranche d'indices ⇒ échec attendu, la
	// sonde mesurerait un débit comme celle de H-004.
	renderer := newRenderer(t)
	probe := models.Probe{Kind: models.ProbePointerChase, Parameter: 16384}
	probe.SourceFile = models.SourcePath(probe.ID())
	files, err := renderer.RenderProbe(probe)
	if err != nil {
		t.Fatalf("RenderProbe : %v", err)
	}
	source := requireFile(t, files, probe.SourceFile)
	assertParses(t, source)
	for _, needle := range []string{"next *node", "p = p.next", "nodeCount", "Fisher-Yates", "unsafe.Sizeof(node{})"} {
		assertContains(t, source, needle)
	}
	// Aucun parcours indexé : ce serait un accès indépendant.
	if strings.Contains(source, "for j := range") {
		t.Fatalf("la chaîne dépendante ne parcourt aucune tranche :\n%s", source)
	}
}

func TestRenderSondeRefuseUnJeuDeTravailTropPetit(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	for _, kind := range []models.ProbeKind{models.ProbePointerChase, models.ProbeSequentialScan} {
		probe := models.Probe{Kind: kind, Parameter: 64, SourceFile: "x"}
		if _, err := renderer.RenderProbe(probe); err == nil {
			t.Fatalf("un jeu de travail d'un seul nœud doit être refusé pour %s", kind)
		}
	}
}

func TestDigestChangeAvecLesGabarits(t *testing.T) {
	t.Parallel()
	// C-008 le dit : toute modification des gabarits change l'empreinte du harnais, donc
	// invalide matrices et campagnes antérieures. L'empreinte doit donc être celle des gabarits
	// courants, et non une constante figée.
	digest := newRenderer(t).Digest()
	if len(digest) != 64 {
		t.Fatalf("empreinte = %q", digest)
	}
	if digest == "022b7a58027e1bbcaa6f4c2fdb0e51bb1f02f05b6cbabc71cdd71ee53b1f3b4f" {
		t.Fatal("l'empreinte ne peut pas être restée celle d'avant C-008 : les gabarits ont changé")
	}
}
