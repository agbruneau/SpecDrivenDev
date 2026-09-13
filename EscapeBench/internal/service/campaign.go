package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
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
func (s *CampaignService) start(ctx context.Context, opts CampaignOptions) (report CampaignReport, err error) {
	// A1 : nombre de répétitions insuffisant — aucune Campaign n'est créée.
	if opts.Count < models.MinCount {
		return CampaignReport{}, fmt.Errorf("%w : %d répétitions demandées, minimum %d (NFR-003)",
			ErrPrecondition, opts.Count, models.MinCount)
	}
	// A-156 : le service garde la même règle que la ligne de commande. Un appelant qui ne passe
	// pas par le CLI ne doit pas pouvoir lancer une campagne dont chaque sujet échouera.
	if err := validateBenchTime(opts.BenchTime); err != nil {
		return CampaignReport{}, err
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
	hypothesisIDs, hypothesesDigest, criteriaDigests, err := s.freezeCriteria(ctx, opts.HypothesisIDs)
	if err != nil {
		return CampaignReport{}, err
	}
	// C-009 : H-007 indexe ses Comparison par taille et n'en retient qu'une, arbitrairement. Sur
	// une matrice à réplicats elle jugerait donc un réplicat tiré au hasard de l'ordre du fichier,
	// sans erreur ni trace. La campagne est refusée plutôt que de laisser sortir ce verdict ; c'est
	// H-012 qui lit une série répliquée.
	if matrix.Parameters.Replicates > models.DefaultReplicates() && slices.Contains(hypothesisIDs, "H-007") {
		return CampaignReport{}, fmt.Errorf(
			"%w : la matrice %s porte %d réplicats et H-007 n'en lit qu'un, arbitrairement ; retirer H-007 de --hypotheses ou employer une matrice sans réplicat (C-009)",
			ErrPrecondition, matrix.ID, matrix.Parameters.Replicates)
	}

	// Le verrou est posé avant la dérivation de l'identifiant, et non à l'entrée de la boucle de
	// mesure : sinon deux lancements simultanés obtiennent le même identifiant de NextCampaignID,
	// écrivent tous deux campaign.json, et le perdant laisse une campagne RUNNING orpheline
	// derrière lui après l'échec du verrou (A-263).
	if err := s.store.AcquireLock(ctx, ""); err != nil {
		return CampaignReport{}, err
	}
	defer s.releaseLock(ctx, &err)

	startedAt := s.clock.Now()
	campaignID, err := s.store.NextCampaignID(ctx, startedAt)
	if err != nil {
		return CampaignReport{}, err
	}
	campaign := models.Campaign{
		ID: campaignID, MatrixID: matrix.ID, HarnessDigest: digest,
		HypothesesDigest: hypothesesDigest, HypothesisIDs: hypothesisIDs,
		CriteriaDigests: criteriaDigests, Count: opts.Count,
		BenchTime: opts.BenchTime, CPU: opts.CPU,
		Status: models.CampaignRunning, Provenance: provenance, StartedAt: startedAt,
	}
	if err := s.store.CreateCampaign(ctx, campaign); err != nil {
		return CampaignReport{}, err
	}
	// Le verrou porte maintenant un identifiant : c'est lui que la reprise (A4) reconnaîtra.
	if err := s.store.AdoptLock(ctx, campaign.ID); err != nil {
		return CampaignReport{}, err
	}
	return s.measure(ctx, campaign, matrix, opts, nil)
}

// resume couvre A4 : reprise après interruption.
func (s *CampaignService) resume(ctx context.Context, opts CampaignOptions) (report CampaignReport, err error) {
	// A4, étape 5 : la reprise porte sur la Campaign désignée, avec ses paramètres gelés. Accepter
	// en silence une autre matrice, un autre nombre de répétitions ou d'autres hypothèses ferait
	// croire au chercheur qu'il étend ou redirige la campagne (A-030).
	if opts.MatrixID != "" || opts.Count != 0 || len(opts.HypothesisIDs) > 0 {
		return CampaignReport{}, fmt.Errorf(
			"%w : --resume reprend la campagne %s avec la matrice, le nombre de répétitions et les hypothèses gelés à sa création ; retirer --matrix, --count et --hypotheses",
			ErrPrecondition, opts.Resume)
	}
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
	// A4, étape 2 : la toolchain de la reprise est celle de la Campaign. Measurement ne porte pas
	// de provenance propre — c'est Campaign.provenance qui atteste NFR-001 pour toutes ses
	// mesures —, donc une reprise sur une autre toolchain rendrait cette attestation fausse sans
	// erreur ni trace (A-021).
	current, err := s.provenance.Capture(ctx)
	if err != nil {
		return CampaignReport{}, err
	}
	if !campaign.Provenance.SameToolchain(current) || campaign.Provenance.CPUModel != current.CPUModel {
		return CampaignReport{}, fmt.Errorf(
			"%w : campagne %s mesurée sous %s %s/%s sur %q, reprise demandée sous %s %s/%s sur %q ; une reprise ne mélange pas deux toolchains (NFR-001, BR-003-2)",
			ErrPrecondition, campaign.ID,
			campaign.Provenance.GoVersion, campaign.Provenance.GOOS, campaign.Provenance.GOARCH, campaign.Provenance.CPUModel,
			current.GoVersion, current.GOOS, current.GOARCH, current.CPUModel)
	}
	existing, err := s.store.LoadMeasurements(ctx, campaign.ID)
	if err != nil {
		return CampaignReport{}, err
	}
	// A4, étape 4 : toute Measurement écrite est faite, quel que soit son statut. Ne retenir que
	// les COMPLETE faisait remesurer un sujet consigné FAILED par A3, dont l'écriture est ensuite
	// refusée par l'immutabilité de BR-003-3 : la reprise butait indéfiniment sur ce sujet (A-020).
	done := make(map[string]models.MeasurementStatus, len(existing))
	for _, m := range existing {
		done[m.SubjectID] = m.Status
	}
	// A4, étape 5 : les paramètres de mesure sont ceux de la Campaign, pas ceux de la ligne de
	// commande du moment (A-265).
	opts.Count = campaign.Count
	opts.BenchTime = campaign.BenchTime
	opts.CPU = campaign.CPU

	// A4, étape 3 : la reprise reprend le verrou qui porte son propre identifiant. C'est le cas
	// normal, le déclencheur d'A4 étant précisément un processus mort dont le defer de libération
	// n'a pas tourné (A-044).
	if err := s.store.AcquireLock(ctx, campaign.ID); err != nil {
		return CampaignReport{}, err
	}
	defer s.releaseLock(ctx, &err)

	return s.measure(ctx, campaign, matrix, opts, done)
}

// measure couvre les étapes 5 à 8, A2 et A3.
func (s *CampaignService) measure(ctx context.Context, campaign models.Campaign, matrix models.Matrix,
	opts CampaignOptions, alreadyDone map[string]models.MeasurementStatus) (CampaignReport, error) {
	report := CampaignReport{
		CampaignID: campaign.ID, Status: campaign.Status, Provenance: campaign.Provenance,
		HarnessDigest: campaign.HarnessDigest, HypothesesDigest: campaign.HypothesesDigest,
		HypothesisIDs: campaign.HypothesisIDs, CellCount: len(matrix.Cells),
		ProbeCount: len(matrix.Probes), Resumed: alreadyDone != nil,
	}

	runOpts := ports.RunOptions{Count: opts.Count, BenchTime: opts.BenchTime, CPU: opts.CPU}
	for _, subjectID := range matrix.SubjectIDs() {
		// A5 : une interruption arrête la boucle. Sans cette garde, le sujet courant était
		// consigné FAILED, tous les suivants échouaient en chaîne en quelques millisecondes, et la
		// campagne se clôturait COMPLETED : A4 devenait inatteignable, puisque la reprise exige
		// RUNNING (A-261). La Campaign reste RUNNING, donc reprenable.
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("campagne %s interrompue, reprenable par --resume %s : %w",
				campaign.ID, campaign.ID, err)
		}
		if status, ok := alreadyDone[subjectID]; ok {
			if status == models.MeasurementComplete {
				report.Measured++
			} else {
				report.Failed++
			}
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
		cause := fmt.Errorf("%w : campagne %s invalide pour tout verdict", ErrHarnessChanged, campaign.ID)
		return s.abort(ctx, report, campaign.ID, finishedAt, reason, cause)
	}

	// Étape 8 : la campagne est COMPLETED si au moins un sujet a été mesuré.
	if report.Measured == 0 {
		reason := "aucun sujet mesuré"
		cause := fmt.Errorf("%w : campagne %s, %s", ErrNoSubjectMeasured, campaign.ID, reason)
		return s.abort(ctx, report, campaign.ID, finishedAt, reason, cause)
	}
	if err := s.store.SetCampaignStatus(ctx, campaign.ID, models.CampaignCompleted, finishedAt, ""); err != nil {
		return report, err
	}
	report.Status = models.CampaignCompleted
	return report, nil
}

// releaseLock retire le verrou et joint son échec à l'erreur rendue.
//
// Révision du 2026-09-12 (A-145) : l'échec était avalé. Un verrou orphelin bloque ensuite UC-001
// comme UC-003, et le chercheur n'avait aucune trace de l'instant où il est apparu — le hook
// guard-paths interdisant par ailleurs à un agent de le retirer.
func (s *CampaignService) releaseLock(ctx context.Context, err *error) {
	if releaseErr := s.store.ReleaseLock(ctx); releaseErr != nil {
		*err = errors.Join(*err, fmt.Errorf("libération du verrou de campagne : %w", releaseErr))
	}
}

// abort consigne l'abandon d'une campagne et rend l'erreur qui l'a motivé. Quand la consignation
// échoue elle-même, les deux erreurs sont jointes : l'erreur d'origine — la seule qui dise
// pourquoi la campagne est invalide — était perdue au profit de l'erreur d'écriture (A-024).
func (s *CampaignService) abort(ctx context.Context, report CampaignReport, campaignID string,
	finishedAt time.Time, reason string, cause error) (CampaignReport, error) {
	report.Status = models.CampaignAborted
	report.AbortReason = reason
	if err := s.store.SetCampaignStatus(ctx, campaignID, models.CampaignAborted, finishedAt, reason); err != nil {
		return report, errors.Join(cause, fmt.Errorf("consignation de l'abandon de %s : %w", campaignID, err))
	}
	return report, cause
}

// validateBenchTime contrôle la durée de mesure de C-003. La valeur vide est admise : elle laisse
// `go test` appliquer son propre défaut.
func validateBenchTime(value string) error {
	if value == "" {
		return nil
	}
	if count, found := strings.CutSuffix(value, "x"); found {
		n, err := strconv.Atoi(count)
		if err != nil || n <= 0 {
			return fmt.Errorf("%w : benchtime %q : la forme <n>x attend un nombre d'itérations positif (C-003)",
				ErrPrecondition, value)
		}
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return fmt.Errorf("%w : benchtime %q n'est ni une durée Go positive ni un nombre d'itérations (C-003)",
			ErrPrecondition, value)
	}
	return nil
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
func (s *CampaignService) freezeCriteria(ctx context.Context, requested []string) ([]string, string, map[string]string, error) {
	catalogue, err := s.hypotheses.Load(ctx)
	if err != nil {
		return nil, "", nil, err
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
		return nil, "", nil, err
	}
	// A-185 : l'empreinte d'ensemble dit qu'un critère a changé, jamais lequel. Celles-ci le
	// disent, ce qu'exige le flux A1 de UC-005 — le chercheur doit savoir quelle hypothèse
	// rouvrir, et le catalogue au moment de la campagne n'est conservé nulle part.
	perHypothesis := make(map[string]string, len(ids))
	for _, id := range ids {
		single, err := s.digestOf(catalogue, []string{id})
		if err != nil {
			return nil, "", nil, err
		}
		perHypothesis[id] = single
	}
	return ids, digest, perHypothesis, nil
}
