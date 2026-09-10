package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// ReproducibilityCheck est le résultat de la comparaison avec un fichier de verdicts antérieur
// portant la même toolchain (UC-002, A3 ; NFR-002).
type ReproducibilityCheck struct {
	ComparedTo string
	Differing  []string
	Violation  bool
}

// EscapeReportSummary est ce que UC-002 rend observable (étape 7).
type EscapeReportSummary struct {
	MatrixID        string
	Provenance      models.Provenance
	Path            string
	Counts          map[models.EscapeCategory]int
	OtherCount      int
	CompileErrors   int
	Reproducibility *ReproducibilityCheck
}

// EscapeService met en œuvre UC-002 Classer l'échappement.
type EscapeService struct {
	repo       ports.MatrixRepository
	compiler   ports.Compiler
	classifier ports.EscapeClassifier
	store      ports.EscapeStore
	provenance ports.ProvenanceProbe
	digester   ports.Digester
	clock      ports.Clock
}

// NewEscapeService câble le service UC-002.
func NewEscapeService(repo ports.MatrixRepository, compiler ports.Compiler, classifier ports.EscapeClassifier,
	store ports.EscapeStore, provenance ports.ProvenanceProbe, digester ports.Digester, clock ports.Clock) *EscapeService {
	return &EscapeService{repo: repo, compiler: compiler, classifier: classifier, store: store,
		provenance: provenance, digester: digester, clock: clock}
}

// Classify exécute UC-002 sur une matrice identifiée.
//
// UC-002 Classer l'échappement — étapes 1 à 7.
func (s *EscapeService) Classify(ctx context.Context, matrixID string) (EscapeReportSummary, error) {
	// Précondition : la Matrix existe avec au moins une Cell.
	matrix, err := s.repo.Load(ctx, matrixID)
	if err != nil {
		return EscapeReportSummary{}, err
	}
	if len(matrix.Cells) == 0 {
		return EscapeReportSummary{}, fmt.Errorf("%w : la matrice %s ne contient aucune Cell", ErrPrecondition, matrixID)
	}

	// Étape 2 : vérification de l'empreinte du harnais. A1 refuse sans rien écrire.
	digest, err := s.digester.HarnessDigest(ctx)
	if err != nil {
		return EscapeReportSummary{}, err
	}
	if digest != matrix.HarnessDigest {
		return EscapeReportSummary{}, fmt.Errorf("%w : matrice %s enregistrée avec %s, harnais courant %s (C-005)",
			ErrHarnessChanged, matrixID, matrix.HarnessDigest, digest)
	}
	provenance, err := s.provenance.Capture(ctx)
	if err != nil {
		return EscapeReportSummary{}, err
	}

	// Étapes 3 à 5 : compilation avec diagnostic, extraction et catégorisation, cellule par cellule.
	report := models.EscapeReport{MatrixID: matrix.ID, Provenance: provenance}
	dir := s.repo.Dir(matrix.ID)
	for _, cell := range matrix.Cells {
		verdict, err := s.classifyCell(ctx, dir, cell)
		if err != nil {
			return EscapeReportSummary{}, err
		}
		if err := verdict.Validate(); err != nil {
			return EscapeReportSummary{}, err
		}
		report.Verdicts = append(report.Verdicts, verdict)
	}

	// A3 : un fichier existe déjà pour la même toolchain — comparer avant d'écrire le nouveau.
	check, err := s.checkReproducibility(ctx, matrix.ID, report)
	if err != nil {
		return EscapeReportSummary{}, err
	}

	// Étape 6 : écriture d'un nouveau fichier horodaté (BR-002-3, NFR-004).
	path, err := s.store.WriteEscapeReport(ctx, report, provenance.CapturedAt)
	if err != nil {
		return EscapeReportSummary{}, err
	}

	// Étape 7.
	counts := report.CountByCategory()
	return EscapeReportSummary{
		MatrixID: matrix.ID, Provenance: provenance, Path: path, Counts: counts,
		OtherCount: counts[models.CategoryOther], CompileErrors: report.CountCompileErrors(),
		Reproducibility: check,
	}, nil
}

// classifyCell compile une cellule et attribue sa catégorie. A2 : une cellule qui ne compile pas
// est consignée au statut COMPILE_ERROR et la classification se poursuit.
func (s *EscapeService) classifyCell(ctx context.Context, dir string, cell models.Cell) (models.EscapeVerdict, error) {
	lines, err := s.compiler.EscapeAnalysis(ctx, dir, cell.ID())
	if err != nil {
		var compileErr *ports.CompileError
		if errors.As(err, &compileErr) {
			return models.EscapeVerdict{
				CellID: cell.ID(), Status: models.EscapeStatusCompileError, CompilerError: compileErr.Output,
			}, nil
		}
		return models.EscapeVerdict{}, err
	}
	return s.classifier.Classify(ctx, dir, cell.ID(), cell.SourceFile, lines)
}

// checkReproducibility compare le nouveau rapport au plus récent portant la même toolchain
// (UC-002, A3). Un écart est une violation de NFR-002.
func (s *EscapeService) checkReproducibility(ctx context.Context, matrixID string, report models.EscapeReport) (*ReproducibilityCheck, error) {
	paths, err := s.store.ListEscapeReports(ctx, matrixID)
	if err != nil {
		return nil, err
	}
	for i := len(paths) - 1; i >= 0; i-- {
		previous, err := s.store.ReadEscapeReport(ctx, paths[i])
		if err != nil {
			return nil, err
		}
		if !previous.Provenance.SameToolchain(report.Provenance) {
			continue
		}
		differing := DifferingVerdicts(previous, report)
		return &ReproducibilityCheck{ComparedTo: paths[i], Differing: differing, Violation: len(differing) > 0}, nil
	}
	return nil, nil
}

// DifferingVerdicts rend les identifiants de cellules dont le verdict diffère entre deux rapports.
func DifferingVerdicts(a, b models.EscapeReport) []string {
	index := make(map[string]models.EscapeVerdict, len(a.Verdicts))
	for _, verdict := range a.Verdicts {
		index[verdict.CellID] = verdict
	}
	var differing []string
	for _, verdict := range b.Verdicts {
		previous, ok := index[verdict.CellID]
		if !ok {
			differing = append(differing, verdict.CellID)
			continue
		}
		if previous.Escapes != verdict.Escapes || previous.Category != verdict.Category || previous.Status != verdict.Status {
			differing = append(differing, verdict.CellID)
		}
	}
	sort.Strings(differing)
	return differing
}
