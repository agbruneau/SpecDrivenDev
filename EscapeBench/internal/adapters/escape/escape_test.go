package escape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

// fixture écrit un source de sujet dans un répertoire de matrice temporaire et rend son chemin
// relatif ainsi que le numéro de la ligne portant le marqueur `//ESCAPE`.
func fixture(t *testing.T, source string) (matrixDir, sourcePath string, markedLine int) {
	t.Helper()
	matrixDir = t.TempDir()
	sourcePath = "subjects/s/subject.go"
	full := filepath.Join(matrixDir, filepath.FromSlash(sourcePath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := os.WriteFile(full, []byte(source), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	for i, line := range strings.Split(source, "\n") {
		if strings.Contains(line, "//ESCAPE") {
			markedLine = i + 1
		}
	}
	if markedLine == 0 {
		t.Fatal("le fixture doit porter un marqueur //ESCAPE")
	}
	return matrixDir, sourcePath, markedLine
}

func TestIsEscape(t *testing.T) {
	t.Parallel()
	// « does not escape » contient le mot « escape » mais affirme le contraire : le confondre
	// avec un échappement ferait classer toutes les cellules comme échappantes.
	// Mutation : retirer le filtre « does not escape » ⇒ échec attendu.
	cases := map[string]bool{
		"moved to heap: t":                   true,
		"&t escapes to heap":                 true,
		"func literal escapes to heap":       true,
		"make([]elem, 1024) escapes to heap": true,
		"t does not escape":                  false,
		"func literal does not escape":       false,
		"make(map[int]T, 1) does not escape": false,
		"can inline newValue":                false,
		"inlining call to sum":               false,
	}
	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()
			if got := IsEscape(message); got != want {
				t.Fatalf("IsEscape(%q) = %v, attendu %v", message, got, want)
			}
		})
	}
}

func TestClassifySansEchappement(t *testing.T) {
	t.Parallel()
	matrixDir, sourcePath, _ := fixture(t, "package subject\n\nfunc Run(n int) uint64 { //ESCAPE\n\treturn uint64(n)\n}\n")
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{
		"subjects/s/subject.go:3:6: can inline Run",
		"subjects/s/subject.go:3:11: t does not escape",
		"ligne sans position",
	})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Escapes || verdict.Category != models.CategoryNone || verdict.CompilerReason != "" {
		t.Fatalf("verdict inattendu : %+v", verdict)
	}
	if err := verdict.Validate(); err != nil {
		t.Fatalf("le verdict produit doit être valide : %v", err)
	}
}

func TestClassifyParCause(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		source  string
		message string
		want    models.EscapeCategory
	}{
		{
			name: "retour de pointeur",
			source: `package subject

type T struct{ Tag uint64 }

func producePointer(i int) *T {
	t := T{Tag: uint64(i)} //ESCAPE
	return &t
}
`,
			message: "moved to heap: t",
			want:    models.CategoryReturnPointer,
		},
		{
			name: "capture par closure, par le littéral",
			source: `package subject

func Run(n int) uint64 {
	var s uint64
	for i := 0; i < n; i++ {
		t := uint64(i) //ESCAPE
		s += keepClosure(func() uint64 { return t })
	}
	return s
}

var kept func() uint64

func keepClosure(f func() uint64) uint64 { kept = f; return f() }
`,
			message: "func literal escapes to heap",
			want:    models.CategoryClosureCapture,
		},
		{
			name: "capture par closure, via un alias",
			source: `package subject

type T struct{ Tag uint64 }

func Run(n int) uint64 {
	var s uint64
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		p := &t
		s += keepClosure(func() uint64 { return p.Tag })
	}
	return s
}

var kept func() uint64

func keepClosure(f func() uint64) uint64 { kept = f; return f() }
`,
			message: "moved to heap: t",
			want:    models.CategoryClosureCapture,
		},
		{
			name: "envoi sur canal",
			source: `package subject

type T struct{ Tag uint64 }

func Run(n int) uint64 {
	ch := make(chan *T, 1)
	var s uint64
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		ch <- &t
		s += (<-ch).Tag
	}
	return s
}
`,
			message: "moved to heap: t",
			want:    models.CategoryChannelSend,
		},
		{
			name: "stockage dans une map",
			source: `package subject

type T struct{ Tag uint64 }

func Run(n int) uint64 {
	m := make(map[int]*T, 1)
	var s uint64
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		m[0] = &t
		s += m[0].Tag
	}
	return s
}
`,
			message: "moved to heap: t",
			want:    models.CategoryContainerStore,
		},
		{
			name: "stockage par append",
			source: `package subject

type T struct{ Tag uint64 }

func Run(n int) uint64 {
	var kept []*T
	var s uint64
	for i := 0; i < n; i++ {
		t := T{Tag: uint64(i)} //ESCAPE
		kept = append(kept, &t)
		s += kept[0].Tag
	}
	return s
}
`,
			message: "moved to heap: t",
			want:    models.CategoryContainerStore,
		},
		{
			name: "allocation retournée : le porteur nomme la cause",
			source: `package subject

type T struct{ Tag uint64 }

type payload struct{ A uint64 }

func produceValueAlloc(i int) (T, *payload) {
	t := T{Tag: uint64(i)}
	var p *payload
	for k := 0; k < 1; k++ {
		p = new(payload) //ESCAPE
		p.A = uint64(i + k)
	}
	return t, p
}
`,
			message: "new(payload) escapes to heap",
			want:    models.CategoryReturnPointer,
		},
		{
			name: "allocation envoyée sur un canal",
			source: `package subject

type payload struct{ A uint64 }

func Run(n int) uint64 {
	ch := make(chan *payload, 1)
	var s uint64
	for i := 0; i < n; i++ {
		p := new(payload) //ESCAPE
		ch <- p
		s += (<-ch).A
	}
	return s
}
`,
			message: "new(payload) escapes to heap",
			want:    models.CategoryChannelSend,
		},
		{
			name: "allocation stockée dans un conteneur",
			source: `package subject

type payload struct{ A uint64 }

func Run(n int) uint64 {
	m := make(map[int]*payload, 1)
	var s uint64
	for i := 0; i < n; i++ {
		p := new(payload) //ESCAPE
		m[0] = p
		s += m[0].A
	}
	return s
}
`,
			message: "new(payload) escapes to heap",
			want:    models.CategoryContainerStore,
		},
		{
			name: "allocation capturée par une closure",
			source: `package subject

type payload struct{ A uint64 }

var kept func() uint64

func keepClosure(f func() uint64) uint64 { kept = f; return f() }

func Run(n int) uint64 {
	var s uint64
	for i := 0; i < n; i++ {
		p := new(payload) //ESCAPE
		s += keepClosure(func() uint64 { return p.A })
	}
	return s
}
`,
			message: "new(payload) escapes to heap",
			want:    models.CategoryClosureCapture,
		},
		{
			name: "littéral composite adressé et retourné",
			source: `package subject

type payload struct{ A uint64 }

func produce(i int) *payload {
	p := &payload{A: uint64(i)} //ESCAPE
	return p
}
`,
			message: "&payload{...} escapes to heap",
			want:    models.CategoryReturnPointer,
		},
		{
			name: "cause hors des quatre du livre",
			source: `package subject

var global *uint64

func Run(n int) uint64 {
	t := uint64(n) //ESCAPE
	global = &t
	return t
}
`,
			message: "moved to heap: t",
			want:    models.CategoryOther,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			matrixDir, sourcePath, line := fixture(t, tc.source)
			raw := fmt.Sprintf("%s:%d:3: %s", sourcePath, line, tc.message)
			verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{raw})
			if err != nil {
				t.Fatalf("Classify : %v", err)
			}
			if !verdict.Escapes {
				t.Fatal("le verdict doit signaler un échappement")
			}
			if verdict.Category != tc.want {
				t.Fatalf("catégorie = %s, attendue %s", verdict.Category, tc.want)
			}
			if verdict.CompilerReason != raw {
				t.Fatalf("la ligne brute du compilateur doit être conservée telle quelle : %q", verdict.CompilerReason)
			}
			if err := verdict.Validate(); err != nil {
				t.Fatalf("le verdict produit doit être valide : %v", err)
			}
		})
	}
}

func TestClassifyPrivilegieLaCauseInformative(t *testing.T) {
	t.Parallel()
	// Le compilateur émet plusieurs lignes ; la catégorie retenue est la première qui n'est pas
	// OTHER, et la ligne brute conservée est celle qui l'a déterminée (BR-002-1).
	source := `package subject

var global *uint64

func Run(n int) uint64 {
	t := uint64(n) //ESCAPE
	global = &t
	return t
}

func keepClosure(f func() uint64) uint64 { return f() }
`
	matrixDir, sourcePath, line := fixture(t, source)
	other := fmt.Sprintf("%s:%d:3: moved to heap: t", sourcePath, line)
	closure := fmt.Sprintf("%s:%d:20: func literal escapes to heap", sourcePath, line+5)
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{other, closure})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryClosureCapture {
		t.Fatalf("catégorie = %s, attendue CLOSURE_CAPTURE", verdict.Category)
	}
	if verdict.CompilerReason != closure {
		t.Fatalf("raison = %q, attendue %q", verdict.CompilerReason, closure)
	}
}

func TestClassifySourceIllisible(t *testing.T) {
	t.Parallel()
	// Sans arbre syntaxique, la cause reste inconnue : c'est une observation OTHER, pas une
	// erreur — H-006 compte cette cellule comme hors des quatre causes (BR-002-2).
	matrixDir := t.TempDir()
	raw := "subjects/s/subject.go:5:3: moved to heap: t"
	verdict, err := New().Classify(context.Background(), matrixDir, "c", "subjects/s/subject.go", []string{raw})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryOther || !verdict.Escapes || verdict.CompilerReason != raw {
		t.Fatalf("verdict inattendu : %+v", verdict)
	}

	// Un fichier syntaxiquement invalide donne le même résultat.
	full := filepath.Join(matrixDir, "subjects", "s", "subject.go")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := os.WriteFile(full, []byte("package subject\nfunc ("), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	verdict, err = New().Classify(context.Background(), matrixDir, "c", "subjects/s/subject.go", []string{raw})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryOther {
		t.Fatalf("catégorie = %s, attendue OTHER", verdict.Category)
	}
}

func TestClassifyExpressionNonNommee(t *testing.T) {
	t.Parallel()
	// `make(...) escapes to heap` ne nomme pas une variable. Depuis A-072 le classificateur
	// remonte à la variable porteuse (`data`), mais son usage ne relève d'aucune des quatre
	// causes du livre — affectation à une variable de paquet — donc la cellule reste OTHER.
	source := `package subject

var data []int

func Setup() {
	data = make([]int, 1024) //ESCAPE
}
`
	matrixDir, sourcePath, line := fixture(t, source)
	raw := fmt.Sprintf("%s:%d:10: make([]int, 1024) escapes to heap", sourcePath, line)
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{raw})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryOther {
		t.Fatalf("catégorie = %s, attendue OTHER", verdict.Category)
	}
}

func TestClassifyAllocationSansPorteur(t *testing.T) {
	t.Parallel()
	// A-072 : sans variable porteuse à la ligne du diagnostic, la cause n'est pas décidable et la
	// cellule reste OTHER. C'est la garde qui empêche le correctif de deviner une cause.
	source := `package subject

type payload struct{ A uint64 }

func consume(p *payload) uint64 { return p.A }

func Run(n int) uint64 {
	return consume(new(payload)) //ESCAPE
}
`
	matrixDir, sourcePath, line := fixture(t, source)
	raw := fmt.Sprintf("%s:%d:17: new(payload) escapes to heap", sourcePath, line)
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{raw})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryOther {
		t.Fatalf("catégorie = %s, attendue OTHER", verdict.Category)
	}
}

func TestIsAllocationMessage(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"new(payload) escapes to heap":       true,
		"make([]int, 4) escapes to heap":     true,
		"&payload{...} escapes to heap":      true,
		"&t escapes to heap":                 false,
		"moved to heap: t":                   false,
		"func literal escapes to heap":       false,
		"new(payload) does not escape":       false,
		"appel(new(payload)) escapes à côté": false,
	}
	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()
			if got := isAllocationMessage(message); got != want {
				t.Fatalf("isAllocationMessage(%q) = %v, attendu %v", message, got, want)
			}
		})
	}
}

func TestClassifyHorsFonction(t *testing.T) {
	t.Parallel()
	// Une position qui ne tombe dans aucun corps de fonction ne permet aucune conclusion.
	source := `package subject

var global *uint64 //ESCAPE

func Run() {}
`
	matrixDir, sourcePath, line := fixture(t, source)
	raw := fmt.Sprintf("%s:%d:5: moved to heap: global", sourcePath, line)
	verdict, err := New().Classify(context.Background(), matrixDir, "c", sourcePath, []string{raw})
	if err != nil {
		t.Fatalf("Classify : %v", err)
	}
	if verdict.Category != models.CategoryOther {
		t.Fatalf("catégorie = %s, attendue OTHER", verdict.Category)
	}
}

func TestEscapedIdentifier(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"moved to heap: t":               "t",
		"&value escapes to heap":         "value",
		"value escapes to heap":          "value",
		"make([]int, 4) escapes to heap": "",
		"func literal escapes to heap":   "",
		"quelque chose d'autre":          "",
		"moved to heap: ":                "",
	}
	for message, want := range cases {
		t.Run(message, func(t *testing.T) {
			t.Parallel()
			if got := escapedIdentifier(message); got != want {
				t.Fatalf("escapedIdentifier(%q) = %q, attendu %q", message, got, want)
			}
		})
	}
}

func TestIsIdentifier(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{"t": true, "_x": true, "a1": true, "": false, "1a": false, "a-b": false, "a.b": false}
	for value, want := range cases {
		if got := isIdentifier(value); got != want {
			t.Fatalf("isIdentifier(%q) = %v, attendu %v", value, got, want)
		}
	}
}
