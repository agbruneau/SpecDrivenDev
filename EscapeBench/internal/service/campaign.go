package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// CampaignOptions porte la demande de campagne (UC-003, étape 1).
type CampaignOptions struct {
	MatrixID      string
	Count         int
	HypothesisIDs []string
	BenchTime     string
	CPU           int
	Resume        string
}

// CampaignReport est ce que UC-003 rend observable (étapes 4 et 8).
type CampaignReport struct {
	CampaignID       string
	Status           models.CampaignStatus
	Provenance       models.Provenance
	HarnessDigest    string
	HypothesesDigest string
	HypothesisIDs    []string
	CellCount        int
	ProbeCount       int
	Measured         int
	Failed           int
	Resumed          bool
	Duration         time.Duration
	AbortReason      string
}

// HypothesisDigester calcule l'empreinte des critères d'un sous-ensemble d'hypothèses (BR-003-5).
type HypothesisDigester func(hypotheses []models.Hypothesis, ids []string) (string, error)

// CampaignService met en œuvre UC-003 Exécuter une campagne de mesure.
type CampaignService struct {
	repo        ports.MatrixRepository
	store       ports.CampaignStore
	escapeStore ports.EscapeStore
	runner      ports.BenchmarkRunner
	digester    ports.Digester
	provenance  ports.ProvenanceProbe
	hypotheses  ports.HypothesisSource
	digestOf    HypothesisDigester
	clock       ports.Clock
}

// NewCampaignService câble le service UC-003.
func NewCampaignService(repo ports.MatrixRepository, store ports.CampaignStore, escapeStore ports.EscapeStore,
	runner ports.BenchmarkRunner, digester ports.Digester, provenance ports.ProvenanceProbe,
	hypotheses ports.HypothesisSource, digestOf HypothesisDigester, clock ports.Clock) *CampaignService {
	return &CampaignService{repo: repo, store: store, escapeStore: escapeStore, runner: runner,
		digester: digester, provenance: provenance, hypotheses: hypotheses, digestOf: digestOf, clock: clock}
}

// Run exécute UC-003.
//
// UC-003 Exécuter une campagne de mesure — étapes 1 à 8.
func (s *CampaignService) Run(ctx context.Context, opts CampaignOptions) (CampaignReport, error) {
	if opts.Resume != "" {
		return s.resume(ctx, opts)
	}
	return s.start(ctx, opts)
}

// start couvre le scénario principal et A1.
func (s *CampaignService) start(ctx context.Context, opts CampaignOptions) (CampaignReport, error) {
	// A1 : nombre de répétitions insuffisant — aucune Campaign n'est créée.
	if opts.Count < models.MinCount {
		return CampaignReport{}, fmt.Errorf("%w : %d répétitions demandées, minimum %d (NFR-003)",
			ErrPrecondition, opts.Count, models.MinCount)
	}
	matrix, err := s.repo.Load(ctx, opts.MatrixID)
	if err != nil {
		return CampaignReport{}, err
	}
	if len(matrix.Cells) == 0 && len(matrix.Probes) == 0 {
		return CampaignReport{}, fmt.Errorf("%w : la matrice %s n'a ni Cell ni Probe", ErrPrecondition, opts.MatrixID)
	}

	provenance, err := s.provenance.Capture(ctx)
	if err != nil {
		return CampaignReport{}, err
	}

	// Précondition : les verdicts d'échappement existent pour la toolchain courante, dès lors que
	// la matrice contient au moins une Cell (les Probe ne sont jamais classées par UC-002).
	if len(matrix.Cells) > 0 {
		if err := s.requireEscapeVerdicts(ctx, matrix.ID, provenance); err != nil {
			return CampaignReport{}, err
		}
	}

	// Étape 2 : l'empreinte du harnais est celle de la Matrix.
	digest, err := s.digester.HarnessDigest(ctx)
	if err != nil {
		return CampaignReport{}, err
	}
	if digest != matrix.HarnessDigest {
		return CampaignReport{}, fmt.Errorf("%w : matrice %s enregistrée avec %s, harnais courant %s (C-005)",
			ErrHarnessChanged, matrix.ID, matrix.HarnessDigest, digest)
	}

	// Étape 3 : création de la Campaign avec l'empreinte des critères (BR-003-5).
	hypothesisIDs, hypothesesDigest, err := s.freezeCriteria(ctx, opts.HypothesisIDs)
	if err != nil {
		return CampaignReport{}, err
	}
	// C-009 : H-007 indexe ses Comparison par taille et n'en retient qu'une, arbitrairement. Sur
	// une matrice à réplicats elle jugerait donc un réplicat tiré au hasard de l'ordre du fichier,
	// sans erreur ni trace. La campagne est refusée plutôt que de laisser sortir ce verdict ; c'est
	// H-012 qui lit une série répliquée.
	if matrix.Parameters.Replicates > models.DefaultReplicates() {
		for _, id := range hypothesisIDs {
			if id == "H-007" {
				return CampaignReport{}, fmt.Errorf(
					"%w : la matrice %s porte %d réplicats et H-007 n'en lit qu'un, arbitrairement ; retirer H-007 de --hypotheses ou employer une matrice sans réplicat (C-009)",
					ErrPrecondition, matrix.ID, matrix.Parameters.Replicates)
			}
		}
	}

	startedAt := s.clock.Now()
	campaignID, err := s.store.NextCampaignID(ctx, startedAt)
	if err != nil {
		return CampaignReport{}, err
	}
	campaign := models.Campaign{
		ID: campaignID, MatrixID: matrix.ID, HarnessDigest: digest,
		HypothesesDigest: hypothesesDigest, HypothesisIDs: hypothesisIDs, Count: opts.Count,
		Status: models.CampaignRunning, Provenance: provenance, StartedAt: startedAt,
	}
	if err := s.store.CreateCampaign(ctx, campaign); err != nil {
		return CampaignReport{}, err
	}
	return s.measure(ctx, campaign, matrix, opts, nil)
}

// resume couvre A4 : reprise après interruption.
func (s *CampaignService) resume(ctx context.Context, opts CampaignOptions) (CampaignReport, error) {
	campaign, err := s.store.LoadCampaign(ctx, opts.Resume)
	if err != nil {
		return CampaignReport{}, err
	}
	if campaign.Status != models.CampaignRunning {
		return CampaignReport{}, fmt.Errorf("%w : la campagne %s est au statut %s, la reprise exige RUNNING",
			ErrPrecondition, campaign.ID, campaign.Status)
	}
	matrix, err := s.repo.Load(ctx, campaign.MatrixID)
	if err != nil {
		return CampaignReport{}, err
	}
	// A4, étape 1 : l'empreinte du harnais est vérifiée contre celle de la Campaign.
	digest, err := s.digester.HarnessDigest(ctx)
	if err != nil {
		return CampaignReport{}, err
	}
	if digest != campaign.HarnessDigest {
		return CampaignReport{}, fmt.Errorf("%w : campagne %s démarrée avec %s, harnais courant %s (BR-003-1)",
			ErrHarnessChanged, campaign.ID, campaign.HarnessDigest, digest)
	}
	existing, err := s.store.LoadMeasurements(ctx, campaign.ID)
	if err != nil {
		return CampaignReport{}, err
	}
	done := make(map[string]bool, len(existing))
	for _, m := range existing {
		if m.Status == models.MeasurementComplete {
			done[m.SubjectID] = true
		}
	}
	opts.Count = campaign.Count
	return s.measure(ctx, campaign, matrix, opts, done)
}

// measure couvre les étapes 5 à 8, A2 et A3.
func (s *CampaignService) measure(ctx context.Context, campaign models.Campaign, matrix models.Matrix,
	opts CampaignOptions, alreadyDone map[string]bool) (CampaignReport, error) {
	report := CampaignReport{
		CampaignID: campaign.ID, Provenance: campaign.Provenance, HarnessDigest: campaign.HarnessDigest,
		HypothesesDigest: campaign.HypothesesDigest, HypothesisIDs: campaign.HypothesisIDs,
		CellCount: len(matrix.Cells), ProbeCount: len(matrix.Probes), Resumed: alreadyDone != nil,
	}
	if err := s.store.AcquireLock(ctx, campaign.ID); err != nil {
		return report, err
	}
	defer func() { _ = s.store.ReleaseLock(ctx) }()

	runOpts := ports.RunOptions{Count: opts.Count, BenchTime: opts.BenchTime, CPU: opts.CPU}
	for _, subjectID := range matrix.SubjectIDs() {
		if alreadyDone[subjectID] {
			report.Measured++
			continue
		}
		measurement, err := s.runner.Run(ctx, s.repo.Dir(matrix.ID), subjectID, runOpts)
		if err != nil {
			return report, err
		}
		measurement.CampaignID = campaign.ID
		measurement.SubjectID = subjectID
		if err := measurement.Validate(opts.Count); err != nil {
			// Une mesure incohérente est consignée comme échec plutôt que perdue (A3).
			measurement = models.Measurement{CampaignID: campaign.ID, SubjectID: subjectID,
				Status: models.MeasurementFailed, FailureReason: err.Error()}
		}
		// Étape 6 : écriture dès que la mesure est complète.
		if err := s.store.WriteMeasurement(ctx, measurement); err != nil {
			return report, err
		}
		if measurement.Status == models.MeasurementComplete {
			report.Measured++
		} else {
			report.Failed++
		}
	}

	// Étape 7 : l'empreinte du harnais est inchangée. A2 abandonne la campagne.
	finishedAt := s.clock.Now()
	report.Duration = finishedAt.Sub(campaign.StartedAt)
	digest, err := s.digester.HarnessDigest(ctx)
	if err != nil {
		return report, err
	}
	if digest != campaign.HarnessDigest {
		reason := fmt.Sprintf("harnais %s au démarrage, %s à la fin (BR-003-1, C-005)", campaign.HarnessDigest, digest)
		if err := s.store.SetCampaignStatus(ctx, campaign.ID, models.CampaignAborted, finishedAt, reason); err != nil {
			return report, err
		}
		report.Status = models.CampaignAborted
		report.AbortReason = reason
		return report, fmt.Errorf("%w : campagne %s invalide pour tout verdict", ErrHarnessChanged, campaign.ID)
	}

	// Étape 8 : la campagne est COMPLETED si au moins un sujet a été mesuré.
	if report.Measured == 0 {
		reason := "aucun sujet mesuré"
		if err := s.store.SetCampaignStatus(ctx, campaign.ID, models.CampaignAborted, finishedAt, reason); err != nil {
			return report, err
		}
		report.Status = models.CampaignAborted
		report.AbortReason = reason
		return report, fmt.Errorf("campagne %s : %s", campaign.ID, reason)
	}
	if err := s.store.SetCampaignStatus(ctx, campaign.ID, models.CampaignCompleted, finishedAt, ""); err != nil {
		return report, err
	}
	report.Status = models.CampaignCompleted
	return report, nil
}

// requireEscapeVerdicts vérifie que UC-002 a été exécuté pour la toolchain courante.
func (s *CampaignService) requireEscapeVerdicts(ctx context.Context, matrixID string, provenance models.Provenance) error {
	paths, err := s.escapeStore.ListEscapeReports(ctx, matrixID)
	if err != nil {
		return err
	}
	for _, path := range paths {
		report, err := s.escapeStore.ReadEscapeReport(ctx, path)
		if err != nil {
			return err
		}
		if report.Provenance.SameToolchain(provenance) {
			return nil
		}
	}
	return fmt.Errorf("%w : aucun verdict d'échappement de %s pour %s %s/%s ; exécuter `escapebench escape --matrix %s`",
		ErrPrecondition, matrixID, provenance.GoVersion, provenance.GOOS, provenance.GOARCH, matrixID)
}

// freezeCriteria rend les hypothèses retenues et l'empreinte de leurs critères (BR-003-5).
// Sans sélection explicite, toutes les hypothèses du catalogue sont retenues.
func (s *CampaignService) freezeCriteria(ctx context.Context, requested []string) ([]string, string, error) {
	catalogue, err := s.hypotheses.Load(ctx)
	if err != nil {
		return nil, "", err
	}
	ids := requested
	if len(ids) == 0 {
		for _, h := range catalogue {
			ids = append(ids, h.ID)
		}
	}
	ids = append([]string(nil), ids...)
	sort.Strings(ids)
	digest, err := s.digestOf(catalogue, ids)
	if err != nil {
		return nil, "", err
	}
	return ids, digest, nil
}
