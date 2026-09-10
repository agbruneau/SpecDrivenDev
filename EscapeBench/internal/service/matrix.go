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
		return MatrixReport{}, err
	}

	// Étape 5 : compilation de l'ensemble des cellules et des sondes. A3 supprime le répertoire.
	if failures := g.compileAll(ctx, matrix); len(failures) > 0 {
		if err := g.repo.Remove(ctx, matrix.ID); err != nil {
			return MatrixReport{}, err
		}
		return MatrixReport{MatrixID: matrix.ID, CompileErrors: failures},
			fmt.Errorf("%d sujet(s) ne compilent pas ; matrices/%s supprimée", len(failures), matrix.ID)
	}

	// Étape 6 : écriture de matrix.json avec l'empreinte du harnais.
	if err := g.repo.Finalize(ctx, matrix); err != nil {
		return MatrixReport{}, err
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
func (g *MatrixGenerator) compileAll(ctx context.Context, matrix models.Matrix) []SubjectFailure {
	dir := g.repo.Dir(matrix.ID)
	var failures []SubjectFailure
	for _, subjectID := range matrix.SubjectIDs() {
		if err := g.compiler.Build(ctx, dir, subjectID); err != nil {
			failures = append(failures, SubjectFailure{SubjectID: subjectID, Message: err.Error()})
		}
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].SubjectID < failures[j].SubjectID })
	return failures
}

// merge recopie src dans dst.
func merge(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}
