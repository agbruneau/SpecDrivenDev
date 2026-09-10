package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// ErrCriteriaChanged signale que les critères ont changé depuis le démarrage de la campagne
// (UC-005, A1 ; BR-003-5).
var ErrCriteriaChanged = errors.New("critères de réfutation modifiés depuis la campagne")

// Evidence rassemble les résultats sur lesquels un critère est évalué. Aucun paramètre de
// décision n'y figure : le verdict ne dépend que du critère gelé (BR-005-1).
type Evidence struct {
	Campaign         models.Campaign
	Matrix           models.Matrix
	ComparisonSet    *models.ComparisonSet
	ComparisonPath   string
	Measurements     map[string]models.Measurement
	MeasurementPaths map[string]string
	EscapeReport     *models.EscapeReport
	EscapePath       string
}

// evaluation est le résultat de l'évaluation d'un critère.
type evaluation struct {
	Outcome   models.Outcome
	Rationale string
	Files     []string
}

// evaluator évalue le critère gelé d'une hypothèse sur les résultats fournis.
type evaluator func(Evidence) evaluation

// VerdictReportSummary est ce que UC-005 rend observable (étape 8).
type VerdictReportSummary struct {
	CampaignID    string
	Path          string
	DashboardPath string
	Report        models.VerdictReport
	Inconclusive  []models.Verdict
}

// VerdictService met en œuvre UC-005 Produire les verdicts.
type VerdictService struct {
	repo        ports.MatrixRepository
	store       ports.CampaignStore
	escapeStore ports.EscapeStore
	verdicts    ports.VerdictStore
	hypotheses  ports.HypothesisSource
	useCases    ports.UseCaseSource
	code        ports.CodeIndex
	tests       ports.TestReporter
	dashboard   ports.DashboardWriter
	digestOf    HypothesisDigester
	clock       ports.Clock
}

// NewVerdictService câble le service UC-005.
func NewVerdictService(repo ports.MatrixRepository, store ports.CampaignStore, escapeStore ports.EscapeStore,
	verdicts ports.VerdictStore, hypotheses ports.HypothesisSource, useCases ports.UseCaseSource,
	code ports.CodeIndex, tests ports.TestReporter, dashboard ports.DashboardWriter,
	digestOf HypothesisDigester, clock ports.Clock) *VerdictService {
	return &VerdictService{repo: repo, store: store, escapeStore: escapeStore, verdicts: verdicts,
		hypotheses: hypotheses, useCases: useCases, code: code, tests: tests, dashboard: dashboard,
		digestOf: digestOf, clock: clock}
}

// Produce exécute UC-005 sur une campagne et, s'il y a lieu, un fichier de verdicts d'échappement.
//
// UC-005 Produire les verdicts — étapes 1 à 8.
func (s *VerdictService) Produce(ctx context.Context, campaignID, escapePath string) (VerdictReportSummary, error) {
	campaign, err := s.store.LoadCampaign(ctx, campaignID)
	if err != nil {
		return VerdictReportSummary{}, err
	}
	if campaign.Status != models.CampaignCompleted {
		return VerdictReportSummary{}, fmt.Errorf("%w : la campagne %s est au statut %s", ErrPrecondition, campaignID, campaign.Status)
	}

	// Étape 2 et A1 : l'empreinte des critères courants doit être celle de la campagne.
	catalogue, err := s.hypotheses.Load(ctx)
	if err != nil {
		return VerdictReportSummary{}, err
	}
	digest, err := s.digestOf(catalogue, campaign.HypothesisIDs)
	if err != nil {
		return VerdictReportSummary{}, err
	}
	if digest != campaign.HypothesesDigest {
		return VerdictReportSummary{}, fmt.Errorf("%w : campagne %s figée sur %s, critères courants %s ; créer une nouvelle H-### plutôt que de modifier un critère",
			ErrCriteriaChanged, campaignID, campaign.HypothesesDigest, digest)
	}

	evidence, err := s.gather(ctx, campaign, escapePath)
	if err != nil {
		return VerdictReportSummary{}, err
	}

	// Étapes 3 à 5 : évaluation, hypothèse par hypothèse.
	report := models.VerdictReport{CampaignID: campaignID, ProducedAt: s.clock.Now()}
	var inconclusive []models.Verdict
	for _, id := range campaign.HypothesisIDs {
		verdict := s.evaluate(id, evidence)
		if err := verdict.Validate(); err != nil {
			return VerdictReportSummary{}, err
		}
		report.Verdicts = append(report.Verdicts, verdict)
		if verdict.Outcome == models.OutcomeInconclusive {
			inconclusive = append(inconclusive, verdict)
		}
	}

	// Étape 6 : écriture du fichier de verdicts.
	path, err := s.verdicts.WriteVerdictReport(ctx, report)
	if err != nil {
		return VerdictReportSummary{}, err
	}

	// Étape 7 : régénération du tableau de bord (FR-007, BR-005-3).
	dashboardPath, err := s.Dashboard(ctx)
	if err != nil {
		return VerdictReportSummary{}, err
	}
	return VerdictReportSummary{CampaignID: campaignID, Path: path, DashboardPath: dashboardPath,
		Report: report, Inconclusive: inconclusive}, nil
}

// evaluate applique le critère gelé d'une hypothèse (étapes 4 et 5).
func (s *VerdictService) evaluate(id string, evidence Evidence) models.Verdict {
	verdict := models.Verdict{HypothesisID: id, CampaignID: evidence.Campaign.ID}
	evaluate, ok := evaluators[id]
	if !ok {
		verdict.Outcome = models.OutcomeInconclusive
		verdict.Rationale = "aucun évaluateur mécanique n'est implémenté pour " + id +
			" ; le critère doit rester évaluable sur des champs de Comparison, Measurement ou EscapeVerdict"
		verdict.ResultFiles = []string{evidence.CampaignPath()}
		return verdict
	}
	result := evaluate(evidence)
	verdict.Outcome = result.Outcome
	verdict.Rationale = result.Rationale
	verdict.ResultFiles = result.Files
	if len(verdict.ResultFiles) == 0 {
		verdict.ResultFiles = []string{evidence.CampaignPath()}
	}
	return verdict
}

// CampaignPath rend le chemin du campaign.json, cité par défaut dans un rationale (BR-005-2).
func (e Evidence) CampaignPath() string {
	return "results/campaigns/" + e.Campaign.ID + "/campaign.json"
}

// gather rassemble comparaisons, mesures et verdicts d'échappement disponibles.
func (s *VerdictService) gather(ctx context.Context, campaign models.Campaign, escapePath string) (Evidence, error) {
	evidence := Evidence{
		Campaign:         campaign,
		Measurements:     map[string]models.Measurement{},
		MeasurementPaths: map[string]string{},
	}
	matrix, err := s.repo.Load(ctx, campaign.MatrixID)
	if err != nil {
		return Evidence{}, err
	}
	evidence.Matrix = matrix

	measurements, err := s.store.LoadMeasurements(ctx, campaign.ID)
	if err != nil {
		return Evidence{}, err
	}
	for _, m := range measurements {
		evidence.Measurements[m.SubjectID] = m
		evidence.MeasurementPaths[m.SubjectID] = s.store.MeasurementPath(campaign.ID, m.SubjectID)
	}

	set, path, err := s.store.LatestComparisonSet(ctx, campaign.ID)
	if err == nil {
		evidence.ComparisonSet = &set
		evidence.ComparisonPath = path
	}

	if escapePath == "" {
		paths, err := s.escapeStore.ListEscapeReports(ctx, campaign.MatrixID)
		if err != nil {
			return Evidence{}, err
		}
		for i := len(paths) - 1; i >= 0; i-- {
			report, err := s.escapeStore.ReadEscapeReport(ctx, paths[i])
			if err != nil {
				return Evidence{}, err
			}
			if report.Provenance.SameToolchain(campaign.Provenance) {
				evidence.EscapeReport = &report
				evidence.EscapePath = paths[i]
				break
			}
		}
	} else {
		report, err := s.escapeStore.ReadEscapeReport(ctx, escapePath)
		if err != nil {
			return Evidence{}, err
		}
		evidence.EscapeReport = &report
		evidence.EscapePath = escapePath
	}
	return evidence, nil
}

// Dashboard régénère docs/dashboard.md sans produire de nouveau verdict (UC-005, étape 7).
func (s *VerdictService) Dashboard(ctx context.Context) (string, error) {
	useCases, err := s.useCases.LoadUseCases(ctx)
	if err != nil {
		return "", err
	}
	catalogue, err := s.hypotheses.Load(ctx)
	if err != nil {
		return "", err
	}
	regression, err := s.tests.RunAll(ctx)
	if err != nil {
		return "", err
	}

	data := ports.DashboardData{GeneratedAt: s.clock.Now()}
	for _, useCase := range useCases {
		hasCode, hasIntegration, err := s.code.References(ctx, useCase.ID)
		if err != nil {
			return "", err
		}
		outcome, err := s.tests.RunUseCaseTests(ctx, useCase.ID)
		if err != nil {
			return "", err
		}
		row := ports.DashboardUseCase{
			ID: useCase.ID, Title: useCase.Title, LinkedFR: useCase.LinkedFR, Status: useCase.Status,
			Code:        hasCode,
			Unit:        outcome.Selected > 0 && outcome.Passed,
			Integration: "—",
			Regression:  regression.Passed,
		}
		if hasIntegration {
			row.Integration = "✔"
		}
		row.Integrity = integrity(row)
		data.UseCases = append(data.UseCases, row)
	}

	frozen := frozenHypotheses(useCases, catalogue)
	// Tous les rapports sont lus, du plus ancien au plus récent, et le dernier verdict rendu sur
	// une hypothèse l'emporte. Ne lire que le dernier rapport effacerait du tableau les hypothèses
	// que la dernière campagne n'a pas gelées, et deux d'entre elles ne peuvent pas cohabiter dans
	// une même campagne (C-009).
	byHypothesis := map[string]models.Verdict{}
	reports, err := s.verdicts.VerdictReports(ctx)
	if err != nil {
		return "", err
	}
	for _, report := range reports {
		for _, verdict := range report.Verdicts {
			byHypothesis[verdict.HypothesisID] = verdict
		}
	}
	for _, hypothesis := range catalogue {
		row := ports.DashboardHypothesis{
			ID: hypothesis.ID, SourcePages: hypothesis.SourcePages,
			UseCases: hypothesis.UseCases, Frozen: frozen[hypothesis.ID],
		}
		if verdict, ok := byHypothesis[hypothesis.ID]; ok {
			row.CampaignID = verdict.CampaignID
			row.Outcome = string(verdict.Outcome)
		}
		data.Hypotheses = append(data.Hypotheses, row)
	}
	return s.dashboard.Write(ctx, data)
}

// integrity rend la colonne Integrity d'une ligne du tableau de bord.
func integrity(row ports.DashboardUseCase) string {
	switch {
	case !row.Code:
		return "Weak"
	case row.Unit && row.Regression:
		return "Strong"
	default:
		return "Partial"
	}
}

// frozenHypotheses rend les hypothèses dont le critère est gelé : celles dont au moins un cas
// d'utilisation porteur est au statut Approved ou au-delà (guide, §4).
func frozenHypotheses(useCases []ports.UseCaseStatus, catalogue []models.Hypothesis) map[string]bool {
	approved := map[string]bool{}
	for _, useCase := range useCases {
		switch useCase.Status {
		case "Approved", "Implemented", "Verified", "Deployed":
			approved[useCase.ID] = true
		}
	}
	frozen := map[string]bool{}
	for _, hypothesis := range catalogue {
		for _, id := range hypothesis.UseCases {
			if approved[id] {
				frozen[hypothesis.ID] = true
				break
			}
		}
	}
	return frozen
}

// join rend une énumération lisible dans un rationale.
func join(values []string) string { return strings.Join(values, ", ") }

// sortedSubjectIDs rend les identifiants d'une table de mesures, triés.
func sortedSubjectIDs(m map[string]models.Measurement) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
