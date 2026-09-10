// Package harness contient le code de mesure commun à toutes les cellules (C-005) : les gabarits
// de la boucle de benchmark, des profils de durée de vie et des fonctions puits. Son empreinte
// SHA-256 est enregistrée avec chaque Matrix et chaque Campaign (BR-003-1) ; il est figé pendant
// une campagne (hook guard-paths, results/.campaign-lock).
//
// Les gabarits sont embarqués dans le binaire : l'empreinte porte donc exactement le code de
// mesure qui a produit les sujets, indépendamment du répertoire d'exécution.
package harness

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"go/format"
	"io/fs"
	"sort"
	"strings"
	"text/template"

	"github.com/agbruneau/escapebench/internal/models"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Renderer rend les fichiers source d'un sujet à partir des gabarits embarqués.
type Renderer struct {
	templates *template.Template
	digest    string
}

// NewRenderer analyse les gabarits et calcule l'empreinte du harnais.
func NewRenderer() (*Renderer, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("analyse des gabarits du harnais : %w", err)
	}
	digest, err := computeDigest()
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: tmpl, digest: digest}, nil
}

// Digest rend l'empreinte SHA-256 du harnais (BR-003-1).
func (r *Renderer) Digest() string { return r.digest }

// computeDigest calcule l'empreinte des gabarits : noms triés, taille et contenu de chacun.
func computeDigest() (string, error) {
	entries, err := fs.Glob(templatesFS, "templates/*.tmpl")
	if err != nil {
		return "", fmt.Errorf("lecture des gabarits du harnais : %w", err)
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, name := range entries {
		content, err := templatesFS.ReadFile(name)
		if err != nil {
			return "", fmt.Errorf("lecture du gabarit %s : %w", name, err)
		}
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(content))
		h.Write(content)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// cellData porte les valeurs du gabarit d'une Cell.
type cellData struct {
	SubjectID       string
	TypeName        string
	SizeBytes       int
	Layout          models.Layout
	NamedFields     bool
	Fields          []string
	FillLen         int
	HasPointerField bool
	FirstWord       string
	HasLastWord     bool
	LastWord        string
	Init            []string
	Profile         models.LifetimeProfile
	PassingMode     models.PassingMode
	Pointer         bool
	Repeat          int
	Body            string

	NeedConsumeValue        bool
	NeedConsumePointer      bool
	NeedProduceValue        bool
	NeedProducePointer      bool
	NeedProduceValueAlloc   bool
	NeedProducePointerAlloc bool
	NeedPayload             bool
	NeedClosure             bool
	NeedHolder              bool
}

// pointerWord est l'expression qui lit le mot occupé par le champ pointeur. Elle ne déréférence
// pas : lire la cible ferait un travail différent de la lecture d'un champ entier.
const pointerWord = "uint64(uintptr(unsafe.Pointer(t.P)))"

// newCellData dérive les valeurs du gabarit d'une Cell. Seules les fonctions utilisées par le
// profil sont émises : une fonction inutilisée produirait des lignes d'échappement parasites dans
// la sortie de `-gcflags=-m` et fausserait la classification de UC-002.
func newCellData(cell models.Cell) cellData {
	words := cell.TypeSpec.WordCount()
	namedFields := cell.TypeSpec.Layout.RegisterAssignable()
	// Le témoin nul exécute le corps du mode valeur des deux côtés de la paire : son delta vrai
	// est nul par construction, ce qui mesure l'écart entre deux binaires (C-008).
	pointer := cell.PassingMode == models.PassingPointer && cell.TypeSpec.Layout != models.LayoutNamedFieldsSham
	effectiveMode := models.PassingValue
	if pointer {
		effectiveMode = models.PassingPointer
	}

	d := cellData{
		SubjectID:       cell.ID(),
		TypeName:        cell.TypeSpec.Name,
		SizeBytes:       cell.TypeSpec.SizeBytes,
		Layout:          cell.TypeSpec.Layout,
		NamedFields:     namedFields,
		FillLen:         words - 1,
		HasPointerField: cell.TypeSpec.HasPointerField,
		Profile:         cell.Profile,
		PassingMode:     cell.PassingMode,
		Pointer:         pointer,
		Repeat:          cell.Repetitions(),
		Body:            string(cell.Profile) + "_" + string(effectiveMode),
	}
	d.Fields, d.FirstWord, d.LastWord, d.HasLastWord, d.Init = layoutOf(namedFields, cell.TypeSpec.HasPointerField, words)

	switch cell.Profile {
	case models.ProfileLocal:
		d.NeedConsumeValue = !pointer
		d.NeedConsumePointer = pointer
	case models.ProfileReturned:
		d.NeedProduceValue = !pointer
		d.NeedProducePointer = pointer
	case models.ProfileReturnedAlloc:
		d.NeedPayload = true
		d.NeedProduceValueAlloc = !pointer
		d.NeedProducePointerAlloc = pointer
	case models.ProfileCapturedByClosure:
		d.NeedClosure = true
	case models.ProfileStoredInStruct:
		d.NeedHolder = true
	}
	return d
}

// layoutOf dérive la déclaration des champs, les expressions de lecture du premier et du dernier
// mot, et les affectations du constructeur, pour une disposition et une taille données.
func layoutOf(namedFields, hasPointerField bool, words int) (fields []string, first, last string, hasLast bool, init []string) {
	if !namedFields {
		// Disposition d'origine : un mot de tête, puis un tableau de remplissage.
		if hasPointerField {
			first = pointerWord
			init = append(init, "t.P = &anchor")
		} else {
			first = "t.Tag"
			init = append(init, "t.Tag = uint64(i)")
		}
		if words > 1 {
			hasLast = true
			last = fmt.Sprintf("t.Fill[%d]", words-2)
			init = append(init, fmt.Sprintf("t.Fill[%d] = uint64(i)", words-2))
		}
		return nil, first, last, hasLast, init
	}

	// Disposition à champs nommés : un champ par mot, le dernier devenant un pointeur lorsque la
	// variante en demande un. Aucun tableau, donc le type reste assignable aux registres.
	plainCount := words
	if hasPointerField {
		plainCount = words - 1
	}
	for i := 0; i < plainCount; i++ {
		fields = append(fields, fmt.Sprintf("F%d uint64", i))
	}
	if hasPointerField {
		fields = append(fields, "P *uint64")
	}

	switch {
	case plainCount == 0:
		// Un seul mot, occupé par le pointeur.
		first = pointerWord
	default:
		first = "t.F0"
		init = append(init, "t.F0 = uint64(i)")
	}
	switch {
	case hasPointerField:
		init = append(init, "t.P = &anchor")
		if plainCount > 0 {
			hasLast = true
			last = pointerWord
		}
	case plainCount > 1:
		hasLast = true
		last = fmt.Sprintf("t.F%d", plainCount-1)
		init = append(init, fmt.Sprintf("t.F%d = uint64(i)", plainCount-1))
	}
	return fields, first, last, hasLast, init
}

// RenderCell rend les fichiers du paquet d'une Cell, indexés par chemin relatif au répertoire de
// la Matrix.
func (r *Renderer) RenderCell(cell models.Cell) (map[string]string, error) {
	if err := cell.Validate(); err != nil {
		return nil, err
	}
	data := newCellData(cell)
	source, err := r.render("cell.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendu de la cellule %s : %w", cell.ID(), err)
	}
	test, err := r.render("subject_test.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendu du test de %s : %w", cell.ID(), err)
	}
	dir := "subjects/" + models.SubjectDir(cell.ID())
	return map[string]string{
		dir + "/subject.go":      source,
		dir + "/subject_test.go": test,
	}, nil
}

// probeData porte les valeurs du gabarit d'une Probe.
type probeData struct {
	SubjectID   string
	Kind        models.ProbeKind
	Parameter   int
	IsScan      bool
	IsScattered bool
	IsChase     bool
	IsPrealloc  bool
}

// RenderProbe rend les fichiers du paquet d'une Probe.
func (r *Renderer) RenderProbe(probe models.Probe) (map[string]string, error) {
	if err := probe.Validate(); err != nil {
		return nil, err
	}
	data := probeData{
		SubjectID:   probe.ID(),
		Kind:        probe.Kind,
		Parameter:   probe.Parameter,
		IsScan:      probe.Kind == models.ProbeSequentialScan || probe.Kind == models.ProbeScatteredScan,
		IsScattered: probe.Kind == models.ProbeScatteredScan,
		IsChase:     probe.Kind == models.ProbePointerChase,
		IsPrealloc:  probe.Kind == models.ProbeAppendPrealloc,
	}
	if data.IsScan || data.IsChase {
		if probe.Parameter%64 != 0 {
			return nil, fmt.Errorf("%w : le jeu de travail d'une sonde %s doit être un multiple de 64 octets (%d)",
				models.ErrValidation, probe.Kind, probe.Parameter)
		}
		if probe.Parameter < 128 {
			return nil, fmt.Errorf("%w : le jeu de travail d'une sonde %s doit compter au moins deux nœuds (%d octets)",
				models.ErrValidation, probe.Kind, probe.Parameter)
		}
	}
	source, err := r.render("probe.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendu de la sonde %s : %w", probe.ID(), err)
	}
	test, err := r.render("subject_test.go.tmpl", struct{ SubjectID string }{probe.ID()})
	if err != nil {
		return nil, fmt.Errorf("rendu du test de %s : %w", probe.ID(), err)
	}
	dir := "subjects/" + models.SubjectDir(probe.ID())
	return map[string]string{
		dir + "/subject.go":      source,
		dir + "/subject_test.go": test,
	}, nil
}

// RenderModule rend le go.mod du module imbriqué de la Matrix (C-007).
func (r *Renderer) RenderModule(matrixID string) (map[string]string, error) {
	content, err := r.renderRaw("go.mod.tmpl", struct{ MatrixID string }{matrixID})
	if err != nil {
		return nil, fmt.Errorf("rendu du go.mod de %s : %w", matrixID, err)
	}
	return map[string]string{"go.mod": content}, nil
}

// render exécute un gabarit puis met le résultat au format Go canonique.
func (r *Renderer) render(name string, data any) (string, error) {
	raw, err := r.renderRaw(name, data)
	if err != nil {
		return "", err
	}
	formatted, err := format.Source([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("le gabarit %s produit du Go invalide : %w\n%s", name, err, raw)
	}
	return string(formatted), nil
}

// renderRaw exécute un gabarit sans mise en forme.
func (r *Renderer) renderRaw(name string, data any) (string, error) {
	var b strings.Builder
	if err := r.templates.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
