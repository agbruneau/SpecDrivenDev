package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// Doublures en mémoire : le service ne parle qu'aux ports, aucun test n'a besoin du disque ni de
// la chaîne d'outils (BEPG p. 373-375).

// errNotFound est la sentinelle du port : gather doit pouvoir distinguer une absence légitime
// d'une lecture en erreur, et un fake qui inventerait la sienne masquerait ce contrôle (A-123).
var errNotFound = ports.ErrNotFound

type fakeClock struct{ instant time.Time }

func (c *fakeClock) Now() time.Time { return c.instant }

func (c *fakeClock) advance(d time.Duration) { c.instant = c.instant.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{instant: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}
}

type fakeDigester struct {
	digest string
	err    error
}

func (d *fakeDigester) HarnessDigest(context.Context) (string, error) { return d.digest, d.err }

type fakeProvenance struct {
	provenance models.Provenance
	err        error
}

func (p *fakeProvenance) Capture(context.Context) (models.Provenance, error) {
	return p.provenance, p.err
}

func newProvenance() *fakeProvenance {
	return &fakeProvenance{provenance: models.Provenance{
		GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "cpu",
		CapturedAt: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
	}}
}

type fakeSource struct{ err error }

func (s *fakeSource) RenderCell(cell models.Cell) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return map[string]string{cell.SourceFile: "package subject // " + cell.ID()}, nil
}

func (s *fakeSource) RenderProbe(probe models.Probe) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return map[string]string{probe.SourceFile: "package subject // " + probe.ID()}, nil
}

func (s *fakeSource) RenderModule(matrixID string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return map[string]string{"go.mod": "module escapebench.local/matrix/" + matrixID}, nil
}

type fakeCompiler struct {
	failing map[string]string
	lines   map[string][]string
	err     error
	built   []string
}

func (c *fakeCompiler) Build(_ context.Context, _, subjectID string) error {
	c.built = append(c.built, subjectID)
	if c.err != nil {
		return c.err
	}
	if output, ok := c.failing[subjectID]; ok {
		return &ports.CompileError{SubjectID: subjectID, Output: output}
	}
	return nil
}

func (c *fakeCompiler) EscapeAnalysis(_ context.Context, _, subjectID string) ([]string, error) {
	if c.err != nil {
		return nil, c.err
	}
	if output, ok := c.failing[subjectID]; ok {
		return nil, &ports.CompileError{SubjectID: subjectID, Output: output}
	}
	return c.lines[subjectID], nil
}

// fakeClassifier attribue la catégorie programmée par identifiant de sujet.
type fakeClassifier struct {
	categories map[string]models.EscapeCategory
	err        error
}

func (c *fakeClassifier) Classify(_ context.Context, _, subjectID, _ string, reasons []string) (models.EscapeVerdict, error) {
	if c.err != nil {
		return models.EscapeVerdict{}, c.err
	}
	category, ok := c.categories[subjectID]
	if !ok || category == models.CategoryNone {
		return models.EscapeVerdict{CellID: subjectID, Status: models.EscapeStatusOK, Category: models.CategoryNone}, nil
	}
	return models.EscapeVerdict{
		CellID: subjectID, Escapes: true, Category: category,
		CompilerReason: strings.Join(reasons, " "), Status: models.EscapeStatusOK,
	}, nil
}

type fakeLock struct {
	held bool
	err  error
}

func (l *fakeLock) LockHeld(context.Context) (bool, error) { return l.held, l.err }

// memoryStore tient lieu de dépôt de matrices, de verdicts d'échappement, de campagnes et de
// verdicts d'hypothèses.
type memoryStore struct {
	matrices     map[string]models.Matrix
	sources      map[string]map[string]string
	escapes      map[string][]models.EscapeReport
	escapePaths  map[string][]string
	campaigns    map[string]models.Campaign
	measurements map[string][]models.Measurement
	comparisons  map[string][]models.ComparisonSet
	verdicts     []models.VerdictReport
	locked       bool
	lockedBy     string
	writeErr     error
	// comparisonErr simule une lecture en erreur du fichier de comparaison, distincte de son
	// absence : le service doit les traiter différemment (A-123).
	comparisonErr   error
	partialWriteErr error
	stamp           int
}

func newStore() *memoryStore {
	return &memoryStore{
		matrices:     map[string]models.Matrix{},
		sources:      map[string]map[string]string{},
		escapes:      map[string][]models.EscapeReport{},
		escapePaths:  map[string][]string{},
		campaigns:    map[string]models.Campaign{},
		measurements: map[string][]models.Measurement{},
		comparisons:  map[string][]models.ComparisonSet{},
	}
}

func (s *memoryStore) nextStamp() string {
	s.stamp++
	return fmt.Sprintf("%04d", s.stamp)
}

func (s *memoryStore) Exists(_ context.Context, matrixID string) (bool, error) {
	_, ok := s.matrices[matrixID]
	return ok, nil
}

func (s *memoryStore) Load(_ context.Context, matrixID string) (models.Matrix, error) {
	matrix, ok := s.matrices[matrixID]
	if !ok {
		return models.Matrix{}, fmt.Errorf("%w : matrice %s", errNotFound, matrixID)
	}
	return matrix, nil
}

func (s *memoryStore) WriteSources(_ context.Context, matrixID string, files map[string]string) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.sources[matrixID] = files
	// partialWriteErr simule une écriture interrompue à mi-parcours : les fichiers déjà écrits
	// subsistent, ce que A-250 laissait en place sans nettoyage.
	if s.partialWriteErr != nil {
		return s.partialWriteErr
	}
	return nil
}

func (s *memoryStore) Finalize(_ context.Context, matrix models.Matrix) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.matrices[matrix.ID] = matrix
	return nil
}

func (s *memoryStore) Remove(_ context.Context, matrixID string) error {
	delete(s.sources, matrixID)
	return nil
}

func (s *memoryStore) Dir(matrixID string) string { return "matrices/" + matrixID }

func (s *memoryStore) List(context.Context) ([]string, error) {
	ids := make([]string, 0, len(s.matrices))
	for id := range s.matrices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *memoryStore) WriteEscapeReport(_ context.Context, report models.EscapeReport, _ time.Time) (string, error) {
	if s.writeErr != nil {
		return "", s.writeErr
	}
	path := "results/escape/" + report.MatrixID + "/" + s.nextStamp() + ".json"
	s.escapes[report.MatrixID] = append(s.escapes[report.MatrixID], report)
	s.escapePaths[report.MatrixID] = append(s.escapePaths[report.MatrixID], path)
	return path, nil
}

func (s *memoryStore) ListEscapeReports(_ context.Context, matrixID string) ([]string, error) {
	return s.escapePaths[matrixID], nil
}

func (s *memoryStore) ReadEscapeReport(_ context.Context, path string) (models.EscapeReport, error) {
	for matrixID, paths := range s.escapePaths {
		for i, candidate := range paths {
			if candidate == path {
				return s.escapes[matrixID][i], nil
			}
		}
	}
	return models.EscapeReport{}, fmt.Errorf("%w : %s", errNotFound, path)
}

func (s *memoryStore) CreateCampaign(_ context.Context, campaign models.Campaign) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	if err := campaign.Validate(); err != nil {
		return err
	}
	s.campaigns[campaign.ID] = campaign
	return nil
}

func (s *memoryStore) LoadCampaign(_ context.Context, campaignID string) (models.Campaign, error) {
	campaign, ok := s.campaigns[campaignID]
	if !ok {
		return models.Campaign{}, fmt.Errorf("%w : campagne %s", errNotFound, campaignID)
	}
	return campaign, nil
}

func (s *memoryStore) SetCampaignStatus(_ context.Context, campaignID string, status models.CampaignStatus,
	finishedAt time.Time, abortReason string) error {
	campaign, ok := s.campaigns[campaignID]
	if !ok {
		return fmt.Errorf("%w : campagne %s", errNotFound, campaignID)
	}
	campaign.Status = status
	campaign.FinishedAt = finishedAt
	campaign.AbortReason = abortReason
	s.campaigns[campaignID] = campaign
	return nil
}

func (s *memoryStore) NextCampaignID(_ context.Context, day time.Time) (string, error) {
	return fmt.Sprintf("C-%s-%d", day.UTC().Format("2006-01-02"), len(s.campaigns)+1), nil
}

func (s *memoryStore) WriteMeasurement(_ context.Context, m models.Measurement) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	for _, existing := range s.measurements[m.CampaignID] {
		if existing.SubjectID == m.SubjectID {
			return fmt.Errorf("mesure déjà écrite pour %s", m.SubjectID)
		}
	}
	s.measurements[m.CampaignID] = append(s.measurements[m.CampaignID], m)
	return nil
}

func (s *memoryStore) LoadMeasurements(_ context.Context, campaignID string) ([]models.Measurement, error) {
	out := append([]models.Measurement(nil), s.measurements[campaignID]...)
	sort.Slice(out, func(i, j int) bool { return out[i].SubjectID < out[j].SubjectID })
	return out, nil
}

func (s *memoryStore) WriteComparisonSet(_ context.Context, set models.ComparisonSet) (string, error) {
	if s.writeErr != nil {
		return "", s.writeErr
	}
	s.comparisons[set.CampaignID] = append(s.comparisons[set.CampaignID], set)
	return "results/campaigns/" + set.CampaignID + "/comparison-" + s.nextStamp() + ".json", nil
}

func (s *memoryStore) LatestComparisonSet(_ context.Context, campaignID string) (models.ComparisonSet, string, error) {
	if s.comparisonErr != nil {
		return models.ComparisonSet{}, "", s.comparisonErr
	}
	sets := s.comparisons[campaignID]
	if len(sets) == 0 {
		return models.ComparisonSet{}, "", fmt.Errorf("%w : comparaison de %s", errNotFound, campaignID)
	}
	return sets[len(sets)-1], "results/campaigns/" + campaignID + "/comparison-latest.json", nil
}

func (s *memoryStore) MeasurementPath(campaignID, subjectID string) string {
	return "results/campaigns/" + campaignID + "/measurements/" + models.SubjectDir(subjectID) + ".json"
}

func (s *memoryStore) CampaignPath(campaignID string) string {
	return "results/campaigns/" + campaignID + "/campaign.json"
}

// AcquireLock reproduit la reprise du verrou du store réel : un verrou qui porte l'identifiant
// demandé est repris, comme l'exige A4 (A-044).
func (s *memoryStore) AcquireLock(_ context.Context, campaignID string) error {
	if s.locked {
		if campaignID != "" && s.lockedBy == campaignID {
			return nil
		}
		return fmt.Errorf("%w : tenu par %s", ports.ErrLockHeld, s.lockedBy)
	}
	s.locked = true
	s.lockedBy = campaignID
	return nil
}

func (s *memoryStore) AdoptLock(_ context.Context, campaignID string) error {
	if !s.locked {
		return fmt.Errorf("nommage d'un verrou non tenu")
	}
	s.lockedBy = campaignID
	return nil
}

func (s *memoryStore) ReleaseLock(context.Context) error {
	s.locked = false
	s.lockedBy = ""
	return nil
}

func (s *memoryStore) LockHeld(context.Context) (bool, error) { return s.locked, nil }

func (s *memoryStore) WriteVerdictReport(_ context.Context, report models.VerdictReport) (string, error) {
	if s.writeErr != nil {
		return "", s.writeErr
	}
	s.verdicts = append(s.verdicts, report)
	return "results/verdicts/" + report.CampaignID + "-" + s.nextStamp() + ".json", nil
}

// VerdictReports rend tous les rapports, du plus ancien au plus récent (C-009 : le tableau de bord
// agrège les campagnes, deux hypothèses ne pouvant pas cohabiter dans l'une d'elles).
func (s *memoryStore) VerdictReports(context.Context) ([]models.VerdictReport, error) {
	return append([]models.VerdictReport(nil), s.verdicts...), nil
}

type fakeRunner struct {
	measurements map[string]models.Measurement
	err          error
	onRun        func(subjectID string)
	opts         ports.RunOptions
	order        []string
}

func (r *fakeRunner) Run(_ context.Context, _, subjectID string, opts ports.RunOptions) (models.Measurement, error) {
	r.opts = opts
	r.order = append(r.order, subjectID)
	if r.onRun != nil {
		r.onRun(subjectID)
	}
	if r.err != nil {
		return models.Measurement{}, r.err
	}
	if m, ok := r.measurements[subjectID]; ok {
		return m, nil
	}
	return completeMeasurementOf(subjectID, opts.Count, 1, 8, 1), nil
}

// completeMeasurementOf construit une mesure complète aux valeurs constantes.
func completeMeasurementOf(subjectID string, count int, ns float64, bytes, allocs int64) models.Measurement {
	m := models.Measurement{SubjectID: subjectID, Status: models.MeasurementComplete}
	for i := 0; i < count; i++ {
		m.NsPerOp = append(m.NsPerOp, ns)
		m.BytesPerOp = append(m.BytesPerOp, bytes)
		m.AllocsPerOp = append(m.AllocsPerOp, allocs)
	}
	return m
}

type fakeHypotheses struct {
	hypotheses []models.Hypothesis
	err        error
}

func (h *fakeHypotheses) Load(context.Context) ([]models.Hypothesis, error) {
	return h.hypotheses, h.err
}

func catalogue() []models.Hypothesis {
	var out []models.Hypothesis
	for _, id := range []string{"H-001", "H-002", "H-003", "H-004", "H-005", "H-006"} {
		out = append(out, models.Hypothesis{ID: id, SourcePages: "p. 1", Statement: "énoncé " + id,
			RefutationCriterion: "critère " + id, UseCases: []string{"UC-003"}})
	}
	return out
}

// digestOf est l'empreinte des critères utilisée par les tests : elle dépend des énoncés et des
// critères, comme celle de production.
func digestOf(hypotheses []models.Hypothesis, ids []string) (string, error) {
	byID := map[string]models.Hypothesis{}
	for _, h := range hypotheses {
		byID[h.ID] = h
	}
	selected := append([]string(nil), ids...)
	sort.Strings(selected)
	var b strings.Builder
	for _, id := range selected {
		h, ok := byID[id]
		if !ok {
			return "", fmt.Errorf("hypothèse %s absente", id)
		}
		fmt.Fprintf(&b, "%s|%s|%s;", h.ID, h.Statement, h.RefutationCriterion)
	}
	return b.String(), nil
}

type fakeUseCases struct {
	useCases []ports.UseCaseStatus
	err      error
}

func (u *fakeUseCases) LoadUseCases(context.Context) ([]ports.UseCaseStatus, error) {
	return u.useCases, u.err
}

type fakeCodeIndex struct {
	code        map[string]bool
	integration map[string]bool
	err         error
}

func (i *fakeCodeIndex) References(_ context.Context, useCaseID string) (bool, bool, error) {
	return i.code[useCaseID], i.integration[useCaseID], i.err
}

type fakeTests struct {
	perUseCase map[string]ports.TestOutcome
	all        ports.TestOutcome
	err        error
}

func (r *fakeTests) RunUseCaseTests(_ context.Context, useCaseID string) (ports.TestOutcome, error) {
	return r.perUseCase[useCaseID], r.err
}

func (r *fakeTests) RunAll(context.Context) (ports.TestOutcome, error) { return r.all, r.err }

type fakeDashboard struct {
	data ports.DashboardData
	err  error
}

func (d *fakeDashboard) Write(_ context.Context, data ports.DashboardData) (string, error) {
	if d.err != nil {
		return "", d.err
	}
	d.data = data
	return "docs/dashboard.md", nil
}

// smallParameters rend une matrice minuscule : deux tailles, un profil, les deux modes, une sonde.
func smallParameters() models.MatrixParameters {
	return models.MatrixParameters{
		Sizes:                []int{8, 24},
		PointerFieldVariants: []bool{false},
		Profiles:             []models.LifetimeProfile{models.ProfileLocal},
		PassingModes:         models.PassingModes(),
		Probes:               []models.ProbeSpec{{Kind: models.ProbeAppendGrow, Parameter: 1000}},
	}
}
