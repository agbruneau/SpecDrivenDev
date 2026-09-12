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
	// CampaignFile est le chemin du campaign.json, rendu par le port. A-040 : Evidence le
	// recomposait à la main, dupliquant la disposition de results/ dans le service.
	CampaignFile string
	EscapeReport *models.EscapeReport
	EscapePath   string
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
		return VerdictReportSummary{}, fmt.Errorf("%w : campagne %s figée sur %s, critères courants %s%s ; créer une nouvelle H-### plutôt que de modifier un critère",
			ErrCriteriaChanged, campaignID, campaign.HypothesesDigest, digest,
			s.describeChangedCriteria(campaign, catalogue))
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

	// A-034 : les collectes faillibles de l'étape 7 — tests, index de code, statuts des cas
	// d'utilisation — sont faites avant l'écriture de l'étape 6. Un échec de l'une d'elles laissait
	// sinon un fichier de verdicts écrit derrière une erreur, contre la postcondition d'échec de
	// UC-005 (« results/ et docs/dashboard.md sont inchangés »), et le chercheur qui relançait
	// obtenait un second fichier de verdicts pour la même campagne.
	collected, err := s.collectDashboard(ctx)
	if err != nil {
		return VerdictReportSummary{}, err
	}

	// Étape 6 : écriture du fichier de verdicts.
	path, err := s.verdicts.WriteVerdictReport(ctx, report)
	if err != nil {
		return VerdictReportSummary{}, err
	}

	// Étape 7 : régénération du tableau de bord (FR-007, BR-005-3). Il reste ici une fenêtre
	// irréductible : le tableau lit le rapport tout juste écrit, donc après l'étape 6. Le fichier
	// de verdicts n'est pas supprimé si elle échoue — il est immuable (NFR-004) —, mais son chemin
	// est rendu avec l'erreur pour que le chercheur sache ce qui existe déjà.
	dashboardPath, err := s.writeDashboard(ctx, collected)
	if err != nil {
		return VerdictReportSummary{CampaignID: campaignID, Path: path, Report: report,
			Inconclusive: inconclusive}, err
	}
	return VerdictReportSummary{CampaignID: campaignID, Path: path, DashboardPath: dashboardPath,
		Report: report, Inconclusive: inconclusive}, nil
}

// describeChangedCriteria nomme les hypothèses dont le critère a changé depuis la campagne. La
// campagne conserve l'empreinte de chaque critère gelé ; celles qui ne concordent plus sont
// exactement celles à rouvrir sous une nouvelle H-### (UC-005, A1).
//
// Les campagnes antérieures à cette révision ne portent pas ces empreintes : le message reste
// alors celui de l'empreinte d'ensemble, sans nommer personne.
func (s *VerdictService) describeChangedCriteria(campaign models.Campaign, catalogue []models.Hypothesis) string {
	if len(campaign.CriteriaDigests) == 0 {
		return ""
	}
	var changed, missing []string
	for _, id := range campaign.HypothesisIDs {
		recorded, ok := campaign.CriteriaDigests[id]
		if !ok {
			continue
		}
		current, err := s.digestOf(catalogue, []string{id})
		if err != nil {
			missing = append(missing, id)
			continue
		}
		if current != recorded {
			changed = append(changed, id)
		}
	}
	sort.Strings(changed)
	sort.Strings(missing)
	switch {
	case len(changed) > 0 && len(missing) > 0:
		return fmt.Sprintf(" ; critères modifiés : %s ; hypothèses absentes du catalogue : %s",
			join(changed), join(missing))
	case len(changed) > 0:
		return " ; critères modifiés : " + join(changed)
	case len(missing) > 0:
		return " ; hypothèses absentes du catalogue : " + join(missing)
	}
	return ""
}

// dashboardCollection porte ce que l'étape 7 lit avant d'écrire quoi que ce soit.
type dashboardCollection struct {
	useCases   []ports.UseCaseStatus
	catalogue  []models.Hypothesis
	regression ports.TestOutcome
	rows       []ports.DashboardUseCase
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
// C'est le port qui le donne : la disposition de results/ appartient à l'adaptateur.
func (e Evidence) CampaignPath() string { return e.CampaignFile }

// gather rassemble comparaisons, mesures et verdicts d'échappement disponibles.
func (s *VerdictService) gather(ctx context.Context, campaign models.Campaign, escapePath string) (Evidence, error) {
	evidence := Evidence{
		Campaign:         campaign,
		CampaignFile:     s.store.CampaignPath(campaign.ID),
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

	// A-123 : l'erreur n'était ni inspectée ni propagée. Un fichier de comparaison tronqué ou
	// illisible devenait « aucun fichier de comparaison ; exécuter compare », donc un verdict non
	// concluant écrit dans results/ et repris au tableau de bord, là où le fichier existe. Seule
	// l'absence est une absence.
	set, path, err := s.store.LatestComparisonSet(ctx, campaign.ID)
	switch {
	case err == nil:
		evidence.ComparisonSet = &set
		evidence.ComparisonPath = path
	case errors.Is(err, ports.ErrNotFound):
		// Absence légitime : les évaluateurs rendront non concluant en le disant.
	default:
		return Evidence{}, err
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
		// A-035 : un fichier désigné par --escape n'était rattaché à rien. H-006 et H-009 rendaient
		// leur verdict sur les cellules d'une autre matrice sans erreur ni trace. La toolchain,
		// elle, n'est pas contrôlée : désigner un fichier produit sous une autre est justement ce
		// que --escape permet, et le rapport de verdicts cite le fichier employé.
		if report.MatrixID != campaign.MatrixID {
			return Evidence{}, fmt.Errorf(
				"%w : %s porte les verdicts d'échappement de la matrice %s, la campagne %s mesure %s",
				ErrPrecondition, escapePath, report.MatrixID, campaign.ID, campaign.MatrixID)
		}
		evidence.EscapeReport = &report
		evidence.EscapePath = escapePath
	}
	return evidence, nil
}

// Dashboard régénère docs/dashboard.md sans produire de nouveau verdict (UC-005, étape 7).
func (s *VerdictService) Dashboard(ctx context.Context) (string, error) {
	collected, err := s.collectDashboard(ctx)
	if err != nil {
		return "", err
	}
	return s.writeDashboard(ctx, collected)
}

// collectDashboard lit tout ce dont l'étape 7 a besoin, sans rien écrire.
func (s *VerdictService) collectDashboard(ctx context.Context) (dashboardCollection, error) {
	useCases, err := s.useCases.LoadUseCases(ctx)
	if err != nil {
		return dashboardCollection{}, err
	}
	// A-090 : un statut hors de la liste admise dégelait silencieusement les hypothèses portées
	// par le cas d'utilisation, frozenHypotheses ne reconnaissant que les statuts nommés.
	for _, useCase := range useCases {
		if !validUseCaseStatus(useCase.Status) {
			return dashboardCollection{}, fmt.Errorf(
				"%w : statut %q de %s inconnu ; valeurs admises : %s",
				ErrPrecondition, useCase.Status, useCase.ID, join(useCaseStatuses()))
		}
	}
	catalogue, err := s.hypotheses.Load(ctx)
	if err != nil {
		return dashboardCollection{}, err
	}
	regression, err := s.tests.RunAll(ctx)
	if err != nil {
		return dashboardCollection{}, err
	}
	collected := dashboardCollection{useCases: useCases, catalogue: catalogue, regression: regression}
	for _, useCase := range useCases {
		hasCode, hasIntegration, err := s.code.References(ctx, useCase.ID)
		if err != nil {
			return dashboardCollection{}, err
		}
		outcome, err := s.tests.RunUseCaseTests(ctx, useCase.ID)
		if err != nil {
			return dashboardCollection{}, err
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
		collected.rows = append(collected.rows, row)
	}
	return collected, nil
}

// writeDashboard compose et écrit le tableau de bord à partir de ce qui a été collecté.
func (s *VerdictService) writeDashboard(ctx context.Context, collected dashboardCollection) (string, error) {
	useCases, catalogue := collected.useCases, collected.catalogue
	data := ports.DashboardData{GeneratedAt: s.clock.Now(), UseCases: collected.rows}

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

// useCaseStatuses énumère les statuts qu'un cas d'utilisation peut porter, dans l'ordre de la
// chaîne de maturité du guide (§4).
func useCaseStatuses() []string {
	return []string{"Draft", "Reviewed", "Approved", "Implemented", "Verified", "Deployed"}
}

// validUseCaseStatus indique si un statut est de la liste admise.
//
// Révision du 2026-09-12 (A-090) : rien ne le contrôlait. Une faute de frappe dans l'en-tête d'un
// cas d'utilisation — « Aproved » — dégelait silencieusement les hypothèses qu'il porte, puisque
// frozenHypotheses ne reconnaît que les statuts nommés. Le tableau de bord les affichait alors
// comme non gelées, et rien ne signalait l'erreur.
func validUseCaseStatus(status string) bool {
	for _, admitted := range useCaseStatuses() {
		if status == admitted {
			return true
		}
	}
	return false
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
