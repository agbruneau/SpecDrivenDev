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
	SubjectID          string
	TypeName           string
	SizeBytes          int
	FillLen            int
	HasFill            bool
	LastFill           int
	HasPointerField    bool
	Profile            models.LifetimeProfile
	PassingMode        models.PassingMode
	Body               string
	NeedConsumeValue   bool
	NeedConsumePointer bool
	NeedProduceValue   bool
	NeedProducePointer bool
	NeedClosure        bool
}

// newCellData dérive les valeurs du gabarit d'une Cell. Seules les fonctions utilisées par le
// profil sont émises : une fonction inutilisée produirait des lignes d'échappement parasites dans
// la sortie de `-gcflags=-m` et fausserait la classification de UC-002.
func newCellData(cell models.Cell) cellData {
	words := cell.TypeSpec.WordCount()
	d := cellData{
		SubjectID:       cell.ID(),
		TypeName:        cell.TypeSpec.Name,
		SizeBytes:       cell.TypeSpec.SizeBytes,
		FillLen:         words - 1,
		HasFill:         words > 1,
		LastFill:        words - 2,
		HasPointerField: cell.TypeSpec.HasPointerField,
		Profile:         cell.Profile,
		PassingMode:     cell.PassingMode,
		Body:            string(cell.Profile) + "_" + string(cell.PassingMode),
	}
	pointer := cell.PassingMode == models.PassingPointer
	switch cell.Profile {
	case models.ProfileLocal:
		d.NeedConsumeValue = !pointer
		d.NeedConsumePointer = pointer
	case models.ProfileReturned:
		d.NeedProduceValue = !pointer
		d.NeedProducePointer = pointer
	case models.ProfileCapturedByClosure:
		d.NeedClosure = true
	}
	return d
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
		IsPrealloc:  probe.Kind == models.ProbeAppendPrealloc,
	}
	if data.IsScan && probe.Parameter%64 != 0 {
		return nil, fmt.Errorf("%w : le jeu de travail d'une sonde %s doit être un multiple de 64 octets (%d)",
			models.ErrValidation, probe.Kind, probe.Parameter)
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
