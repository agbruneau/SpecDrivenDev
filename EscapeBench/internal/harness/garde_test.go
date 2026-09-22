package harness

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// Lot 9 de l'audit (A-073, A-081, A-246, D-63). Ces tests type-vérifient les sources rendues avec
// go/types, sous les tailles du compilateur gc sur amd64 et arm64 (C-006) : une garde
// `unsafe.Sizeof` ne se prouve qu'en compilant, et parser.ParseFile n'évalue aucune constante.

// typeCheck type-vérifie un source de sujet. subject.go n'importe que « unsafe », que l'importeur
// résout sans données d'export.
func typeCheck(source, arch string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "subject.go", source, 0)
	if err != nil {
		return err
	}
	config := types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", arch)}
	_, err = config.Check("subject", fset, []*ast.File{file}, nil)
	return err
}

var archs = []string{"amd64", "arm64"}

// TestUC001_A073_GardeDeTailleBilaterale exige que chaque cellule rendue compile, et qu'une taille
// annoncée différente de la taille réelle — plus grande comme plus petite — l'en empêche.
// Mutation : retirer la seconde constante de cell.go.tmpl ⇒ le cas « type trop grand » compile.
func TestUC001_A073_GardeDeTailleBilaterale(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	for _, layout := range models.Layouts() {
		for _, size := range []int{8, 16, 24, 128, 1024} {
			for _, hasPointer := range []bool{false, true} {
				spec, err := models.NewTypeSpec(size, hasPointer, layout)
				if err != nil {
					t.Fatalf("NewTypeSpec : %v", err)
				}
				cell := models.Cell{TypeSpec: spec, Profile: models.ProfileLocal, PassingMode: models.PassingValue}
				cell.SourceFile = models.SourcePath(cell.ID())
				t.Run(cell.ID(), func(t *testing.T) {
					data := newCellData(cell)
					for _, arch := range archs {
						source, err := renderer.render("cell.go.tmpl", data)
						if err != nil {
							t.Fatalf("rendu : %v", err)
						}
						if err := typeCheck(source, arch); err != nil {
							t.Fatalf("%s : la cellule ne compile pas : %v\n%s", arch, err, source)
						}
						for _, delta := range []int{-8, 8} {
							lying := data
							lying.SizeBytes += delta
							source, err := renderer.render("cell.go.tmpl", lying)
							if err != nil {
								t.Fatalf("rendu : %v", err)
							}
							if typeCheck(source, arch) == nil {
								t.Fatalf("%s : une taille annoncée de %d pour un type de %d octets compile encore",
									arch, lying.SizeBytes, size)
							}
						}
					}
				})
			}
		}
	}
}

// TestUC001_A081_SondesSurLaLigneDeCache exige que les sondes qui parcourent la mémoire compilent
// sur les deux lignes de cache admises, et que leurs nœuds en occupent exactement une.
// Mutation : figer nodeBytes à 64 dans probe.go.tmpl ⇒ échec à 128.
func TestUC001_A081_SondesSurLaLigneDeCache(t *testing.T) {
	t.Parallel()
	renderer := newRenderer(t)
	for _, line := range []int{64, 128} {
		for _, kind := range []models.ProbeKind{models.ProbePointerChase, models.ProbeSequentialScan, models.ProbeScatteredScan} {
			probe := models.Probe{Kind: kind, Parameter: 16384}
			probe.SourceFile = models.SourcePath(probe.ID())
			files, err := renderer.RenderProbe(probe, line)
			if err != nil {
				t.Fatalf("%s, ligne %d : %v", kind, line, err)
			}
			source := requireFile(t, files, probe.SourceFile)
			for _, arch := range archs {
				if err := typeCheck(source, arch); err != nil {
					t.Fatalf("%s, ligne %d, %s : %v\n%s", kind, line, arch, err, source)
				}
			}
			if !strings.Contains(source, "Bytes = "+strconv.Itoa(line)) {
				t.Fatalf("%s : la ligne de %d octets n'est pas celle du gabarit :\n%s", kind, line, source)
			}
		}
	}
	// Un jeu de travail qui n'est pas un multiple de la ligne, ou une ligne inconnue, est refusé.
	probe := models.Probe{Kind: models.ProbePointerChase, Parameter: 64 * 3, SourceFile: "x"}
	if _, err := renderer.RenderProbe(probe, 128); err == nil {
		t.Fatal("192 octets ne font pas un nombre entier de lignes de 128")
	}
	if _, err := renderer.RenderProbe(models.Probe{Kind: models.ProbePointerChase, Parameter: 16384, SourceFile: "x"}, 32); err == nil {
		t.Fatal("une ligne de 32 octets doit être refusée")
	}
}

// TestUC001_A246_EmpreinteCouvreHarnessGo exige que harness.go entre dans l'empreinte : un
// changement de la dérivation des champs change le code généré et doit changer l'empreinte.
// Mutation : retirer harnessSource de computeDigest ⇒ échec.
func TestUC001_A246_EmpreinteCouvreHarnessGo(t *testing.T) {
	t.Parallel()
	content, err := templatesFS.ReadFile(harnessSource)
	if err != nil || !strings.Contains(string(content), "func layoutOf(") {
		t.Fatalf("harness.go n'est pas embarqué : %v", err)
	}
	digest := newRenderer(t).Digest()
	templatesOnly, err := digestFiles([]string{
		"templates/cell.go.tmpl", "templates/go.mod.tmpl", "templates/probe.go.tmpl", "templates/subject_test.go.tmpl",
	})
	if err != nil {
		t.Fatalf("empreinte des seuls gabarits : %v", err)
	}
	if digest == templatesOnly {
		t.Fatal("l'empreinte ne couvre que les gabarits : harness.go n'y entre pas")
	}
	// L'empreinte publiée avant le lot 9 ne peut pas subsister.
	if digest == "551ce66be89b2dda720d9fd5ea7a3fb840977f09aed9bbbdf2ed5d112796257b" {
		t.Fatal("l'empreinte est restée celle de la série 551ce66b…")
	}
}
