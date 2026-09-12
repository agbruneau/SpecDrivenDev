// Package service contient les cas d'utilisation (docs/use-cases) : un type par UC, sans I/O
// directe — le service ne parle qu'aux interfaces de ports (BEPG ch. 14, p. 367).
package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// ErrPrecondition signale une précondition de cas d'utilisation non satisfaite.
var ErrPrecondition = errors.New("précondition non satisfaite")

// ErrHarnessChanged signale que l'empreinte du harnais diffère de celle enregistrée (C-005).
var ErrHarnessChanged = errors.New("le harnais a changé")

// ErrNoSubjectMeasured signale une campagne abandonnée faute d'un seul sujet mesuré (UC-003,
// étape 8). Elle existe pour que les bords puissent la distinguer par errors.Is plutôt que par
// comparaison de texte.
var ErrNoSubjectMeasured = errors.New("aucun sujet mesuré")

// ErrSubjectsNotCompilable signale qu'au moins un sujet de la matrice ne compile pas (UC-001, A3).
var ErrSubjectsNotCompilable = errors.New("sujets non compilables")

// SubjectFailure associe un sujet au message qui le concerne.
type SubjectFailure struct {
	SubjectID string
	Message   string
}

// MatrixReport est ce que UC-001 rend observable (étape 7).
type MatrixReport struct {
	MatrixID       string
	TypeSpecCount  int
	CellCount      int
	ProbeCount     int
	HarnessDigest  string
	AlreadyExisted bool
	CompileErrors  []SubjectFailure
}

// MatrixGenerator met en œuvre UC-001 Générer la matrice de cellules.
type MatrixGenerator struct {
	repo     ports.MatrixRepository
	source   ports.SubjectSource
	compiler ports.Compiler
	digester ports.Digester
	lock     ports.CampaignLock
	clock    ports.Clock
}

// NewMatrixGenerator câble le service UC-001.
func NewMatrixGenerator(repo ports.MatrixRepository, source ports.SubjectSource, compiler ports.Compiler,
	digester ports.Digester, lock ports.CampaignLock, clock ports.Clock) *MatrixGenerator {
	return &MatrixGenerator{repo: repo, source: source, compiler: compiler, digester: digester, lock: lock, clock: clock}
}

// Generate exécute UC-001. Elle rend un MatrixReport et, en cas d'échec, laisse `matrices/`
// inchangé : aucun répertoire de matrice partiel ne subsiste (postcondition d'échec).
//
// UC-001 Générer la matrice de cellules — étapes 1 à 7.
func (g *MatrixGenerator) Generate(ctx context.Context, params models.MatrixParameters) (MatrixReport, error) {
	// Préconditions : aucune campagne au statut RUNNING.
	held, err := g.lock.LockHeld(ctx)
	if err != nil {
		return MatrixReport{}, err
	}
	if held {
		return MatrixReport{}, fmt.Errorf("%w : une campagne est en cours (results/.campaign-lock)", ErrPrecondition)
	}

	// Étape 2 : validation des paramètres. A1 rapporte chaque paramètre fautif sans rien écrire.
	normalized := params.Normalize()
	if err := normalized.Validate(); err != nil {
		return MatrixReport{}, err
	}

	digest, err := g.digester.HarnessDigest(ctx)
	if err != nil {
		return MatrixReport{}, err
	}

	// Étape 3 : identifiant déterministe (BR-001-1).
	matrixID := normalized.MatrixID()

	// A2 : la matrice existe déjà.
	exists, err := g.repo.Exists(ctx, matrixID)
	if err != nil {
		return MatrixReport{}, err
	}
	if exists {
		existing, err := g.repo.Load(ctx, matrixID)
		if err != nil {
			return MatrixReport{}, err
		}
		if existing.HarnessDigest != digest {
			return MatrixReport{}, fmt.Errorf("%w : matrice %s générée avec %s, harnais courant %s (C-005)",
				ErrHarnessChanged, matrixID, existing.HarnessDigest, digest)
		}
		return MatrixReport{
			MatrixID: existing.ID, TypeSpecCount: existing.TypeSpecCount(),
			CellCount: len(existing.Cells), ProbeCount: len(existing.Probes),
			HarnessDigest: existing.HarnessDigest, AlreadyExisted: true,
		}, nil
	}

	// Étape 4 : génération des TypeSpec, Cell, Probe et de leurs fichiers source.
	matrix, err := models.NewMatrix(normalized, digest, g.clock.Now())
	if err != nil {
		return MatrixReport{}, err
	}
	files, err := g.renderAll(matrix)
	if err != nil {
		return MatrixReport{}, err
	}
	if err := g.repo.WriteSources(ctx, matrix.ID, files); err != nil {
		// A-250 : sans ce nettoyage, un répertoire partiel subsistait. La génération suivante de la
		// même matrice y écrivait par-dessus des sources dont on ne sait pas si elles sont
		// complètes, et l'identifiant étant déterministe, c'est le cas normal d'une reprise.
		return MatrixReport{}, errors.Join(err, g.removePartial(ctx, matrix.ID))
	}

	// Étape 5 : compilation de l'ensemble des cellules et des sondes. A3 supprime le répertoire.
	failures, err := g.compileAll(ctx, matrix)
	if err != nil {
		return MatrixReport{}, errors.Join(err, g.removePartial(ctx, matrix.ID))
	}
	if len(failures) > 0 {
		report := MatrixReport{MatrixID: matrix.ID, CompileErrors: failures}
		cause := fmt.Errorf("%w : %d sujet(s) de matrices/%s ne compilent pas",
			ErrSubjectsNotCompilable, len(failures), matrix.ID)
		if err := g.repo.Remove(ctx, matrix.ID); err != nil {
			// L'échec du nettoyage ne doit pas effacer la cause : sans elle le chercheur ne sait
			// pas quels sujets ne compilent pas, et il ignore qu'un répertoire partiel subsiste,
			// que la garde d'immutabilité opposera à la prochaine génération (A-026).
			return report, errors.Join(cause, fmt.Errorf(
				"matrices/%s n'a pas pu être supprimée et subsiste incomplète : %w", matrix.ID, err))
		}
		return report, fmt.Errorf("%w ; matrices/%s supprimée", cause, matrix.ID)
	}

	// Étape 6 : écriture de matrix.json avec l'empreinte du harnais.
	if err := g.repo.Finalize(ctx, matrix); err != nil {
		return MatrixReport{}, errors.Join(err, g.removePartial(ctx, matrix.ID))
	}

	// Étape 7.
	return MatrixReport{
		MatrixID: matrix.ID, TypeSpecCount: matrix.TypeSpecCount(),
		CellCount: len(matrix.Cells), ProbeCount: len(matrix.Probes), HarnessDigest: matrix.HarnessDigest,
	}, nil
}

// renderAll rend tous les fichiers de la matrice : go.mod du module imbriqué (C-007) et un paquet
// par sujet.
func (g *MatrixGenerator) renderAll(matrix models.Matrix) (map[string]string, error) {
	files := map[string]string{}
	module, err := g.source.RenderModule(matrix.ID)
	if err != nil {
		return nil, err
	}
	merge(files, module)
	for _, cell := range matrix.Cells {
		rendered, err := g.source.RenderCell(cell)
		if err != nil {
			return nil, err
		}
		merge(files, rendered)
	}
	for _, probe := range matrix.Probes {
		rendered, err := g.source.RenderProbe(probe)
		if err != nil {
			return nil, err
		}
		merge(files, rendered)
	}
	return files, nil
}

// compileAll compile chaque sujet et rend les échecs, triés par identifiant.
func (g *MatrixGenerator) compileAll(ctx context.Context, matrix models.Matrix) ([]SubjectFailure, error) {
	dir := g.repo.Dir(matrix.ID)
	var failures []SubjectFailure
	for _, subjectID := range matrix.SubjectIDs() {
		err := g.compiler.Build(ctx, dir, subjectID)
		if err == nil {
			continue
		}
		// A-249 : seul un refus du compilateur est un sujet non compilable (A3). Une toolchain
		// absente, un disque plein ou une annulation étaient consignés comme si le code généré
		// était fautif, et la matrice était supprimée pour une panne qui ne la concerne pas.
		var compileErr *ports.CompileError
		if !errors.As(err, &compileErr) {
			return nil, err
		}
		failures = append(failures, SubjectFailure{SubjectID: subjectID, Message: err.Error()})
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].SubjectID < failures[j].SubjectID })
	return failures, nil
}

// removePartial retire un répertoire de matrice laissé incomplet par un échec.
func (g *MatrixGenerator) removePartial(ctx context.Context, matrixID string) error {
	if err := g.repo.Remove(ctx, matrixID); err != nil {
		return fmt.Errorf("matrices/%s subsiste incomplète : %w", matrixID, err)
	}
	return nil
}

// merge recopie src dans dst.
func merge(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}
