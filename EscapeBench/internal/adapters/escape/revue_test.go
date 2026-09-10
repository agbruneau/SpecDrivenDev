package escape

import (
	"context"
	"fmt"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// Non-régression de la revue du 2026-09-10, constat « StarExpr est du code mort ».
//
// isContainerTarget acceptait `*x = ...` alors qu'aucun gabarit du harnais n'en produit. La branche
// était inatteignable et n'élargissait que la surface de faux positifs.
//
// Ce test fixe en même temps la limite connue du classificateur, pour qu'une évolution du gabarit
// ne la découvre pas en silence. Écrire l'adresse d'une locale dans le champ d'une struct puis
// retourner cette struct est, au sens du livre, un retour de pointeur ; le classificateur y voit un
// stockage dans un conteneur, parce que ses alias ne traversent pas les champs : il sait que
// `p := &t` fait de `p` un porteur de l'adresse de `t`, pas que `s.P = &t` fait de `s` un porteur.
//
// Aucun verdict n'en dépend aujourd'hui : le seul site du harnais qui écrive une adresse dans un
// champ hors conteneur est `t.P = &anchor`, où `anchor` est une variable de paquet que le
// compilateur ne déplace jamais sur le tas, donc aucune ligne d'échappement ne le désigne. Rendre
// `anchor` locale suffirait à faire mentir la catégorie, et H-006 comme H-009 lisent la catégorie.
func TestClassifyLimiteConnueChampDeStructRetourne(t *testing.T) {
	t.Parallel()
	source := `package subject

type T struct{ P *uint64 }

func produce(i int) T {
	var anchor uint64 //ESCAPE
	anchor = uint64(i)
	var t T
	t.P = &anchor
	return t
}
`
	matrixDir, sourcePath, line := fixture(t, source)
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{
		fmt.Sprintf("%s:%d:6: moved to heap: anchor", sourcePath, line),
	})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	// Comportement actuel, consigné et non souhaité. Le jour où les alias traverseront les champs,
	// remplacer cette attente par models.CategoryReturnPointer.
	if verdict.Category != models.CategoryContainerStore {
		t.Fatalf("catégorie = %s ; la limite documentée donne CONTAINER_STORE. Si le classificateur "+
			"suit désormais les alias à travers les champs, mettre à jour ce test et le commentaire "+
			"de isContainerTarget : %+v", verdict.Category, verdict)
	}
}

// Le stockage dans un conteneur reste reconnu par ses deux formes réelles : l'élément indexé et le
// champ d'une struct qui n'est pas retournée.
func TestClassifyStockageDansConteneurToujoursReconnu(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"élément indexé": `package subject

type T struct{ Tag uint64 }

func Run(n int) uint64 {
	var s uint64
	c := make([]*T, 1)
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		c[0] = &t
		s += c[0].Tag
	}
	return s
}
`,
		"champ de struct": `package subject

type T struct{ Tag uint64 }

type holder struct{ P *T }

func Run(n int) uint64 {
	var s uint64
	c := new(holder)
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		c.P = &t
		s += c.P.Tag
	}
	return s
}
`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			matrixDir, sourcePath, line := fixture(t, source)
			verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{
				fmt.Sprintf("%s:%d:3: moved to heap: t", sourcePath, line),
			})
			if err != nil {
				t.Fatalf("Classify : %v", err)
			}
			if verdict.Category != models.CategoryContainerStore {
				t.Fatalf("catégorie = %s, CONTAINER_STORE attendue : %+v", verdict.Category, verdict)
			}
		})
	}
}
