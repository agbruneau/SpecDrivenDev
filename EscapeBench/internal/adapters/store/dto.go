package store

import (
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// Les DTO portent les noms d'attributs du modèle d'entités et les tags d'encodage. Les entités de
// internal/models restent sans tag (C-004, CLAUDE.md) : la conversion se fait ici.

type provenanceDTO struct {
	GoVersion           string    `json:"goVersion"`
	GOOS                string    `json:"goos"`
	GOARCH              string    `json:"goarch"`
	CPUModel            string    `json:"cpuModel"`
	L1DataCacheBytes    int64     `json:"l1DataCacheBytes,omitempty"`
	LastLevelCacheBytes int64     `json:"lastLevelCacheBytes,omitempty"`
	PageSizeBytes       int64     `json:"pageSizeBytes,omitempty"`
	GOMAXPROCS          int       `json:"gomaxprocs,omitempty"`
	CapturedAt          time.Time `json:"capturedAt"`
}

func toProvenanceDTO(p models.Provenance) provenanceDTO {
	return provenanceDTO{
		GoVersion: p.GoVersion, GOOS: p.GOOS, GOARCH: p.GOARCH, CPUModel: p.CPUModel,
		L1DataCacheBytes: p.L1DataCacheBytes, LastLevelCacheBytes: p.LastLevelCacheBytes,
		PageSizeBytes: p.PageSizeBytes, GOMAXPROCS: p.GOMAXPROCS, CapturedAt: p.CapturedAt.UTC(),
	}
}

func (d provenanceDTO) toModel() models.Provenance {
	return models.Provenance{
		GoVersion: d.GoVersion, GOOS: d.GOOS, GOARCH: d.GOARCH, CPUModel: d.CPUModel,
		L1DataCacheBytes: d.L1DataCacheBytes, LastLevelCacheBytes: d.LastLevelCacheBytes,
		PageSizeBytes: d.PageSizeBytes, GOMAXPROCS: d.GOMAXPROCS, CapturedAt: d.CapturedAt,
	}
}

type typeSpecDTO struct {
	Name            string `json:"name"`
	SizeBytes       int    `json:"sizeBytes"`
	WordCount       int    `json:"wordCount"`
	HasPointerField bool   `json:"hasPointerField"`
	Layout          string `json:"layout,omitempty"`
}

type cellDTO struct {
	ID          string      `json:"id"`
	TypeSpec    typeSpecDTO `json:"typeSpec"`
	Profile     string      `json:"lifetimeProfile"`
	PassingMode string      `json:"passingMode"`
	Repeat      int         `json:"repeat,omitempty"`
	Payload     int         `json:"payload,omitempty"`
	Replicate   int         `json:"replicate,omitempty"`
	SourceFile  string      `json:"sourceFile"`
}

func toCellDTO(c models.Cell) cellDTO {
	dto := cellDTO{
		ID: c.ID(),
		TypeSpec: typeSpecDTO{
			Name:            c.TypeSpec.Name,
			SizeBytes:       c.TypeSpec.SizeBytes,
			WordCount:       c.TypeSpec.WordCount(),
			HasPointerField: c.TypeSpec.HasPointerField,
			Layout:          string(c.TypeSpec.Layout),
		},
		Profile:     string(c.Profile),
		PassingMode: string(c.PassingMode),
		SourceFile:  c.SourceFile,
	}
	if c.Repeat > 1 {
		dto.Repeat = c.Repeat
	}
	if c.Payload > 1 {
		dto.Payload = c.Payload
	}
	if c.Replicate > 1 {
		dto.Replicate = c.Replicate
	}
	return dto
}

func (d cellDTO) toModel() models.Cell {
	// Un fichier antérieur à C-008 ne porte pas de disposition : c'est celle d'origine.
	layout := models.Layout(d.TypeSpec.Layout)
	if layout == "" {
		layout = models.LayoutArrayFill
	}
	repeat := d.Repeat
	if repeat < 1 {
		repeat = 1
	}
	payload := d.Payload
	if payload < 1 {
		payload = 1
	}
	// Un réplicat perdu à la lecture rendrait deux cellules homonymes et Matrix.Validate
	// refuserait la matrice rechargée pour identifiants en double (C-009).
	replicate := d.Replicate
	if replicate < 1 {
		replicate = 1
	}
	return models.Cell{
		TypeSpec: models.TypeSpec{
			Name:            d.TypeSpec.Name,
			SizeBytes:       d.TypeSpec.SizeBytes,
			HasPointerField: d.TypeSpec.HasPointerField,
			Layout:          layout,
		},
		Profile:     models.LifetimeProfile(d.Profile),
		PassingMode: models.PassingMode(d.PassingMode),
		Repeat:      repeat,
		Payload:     payload,
		Replicate:   replicate,
		SourceFile:  d.SourceFile,
	}
}

type probeDTO struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Parameter  int    `json:"parameter"`
	SourceFile string `json:"sourceFile"`
}

func toProbeDTO(p models.Probe) probeDTO {
	return probeDTO{ID: p.ID(), Kind: string(p.Kind), Parameter: p.Parameter, SourceFile: p.SourceFile}
}

func (d probeDTO) toModel() models.Probe {
	return models.Probe{Kind: models.ProbeKind(d.Kind), Parameter: d.Parameter, SourceFile: d.SourceFile}
}

type probeSpecDTO struct {
	Kind      string `json:"kind"`
	Parameter int    `json:"parameter"`
}

type matrixParametersDTO struct {
	Sizes                []int          `json:"sizes"`
	PointerFieldVariants []bool         `json:"pointerFieldVariants"`
	Profiles             []string       `json:"lifetimeProfiles"`
	PassingModes         []string       `json:"passingModes"`
	Layouts              []string       `json:"layouts,omitempty"`
	Repeats              []int          `json:"repeats,omitempty"`
	Payloads             []int          `json:"payloads,omitempty"`
	Replicates           int            `json:"replicates,omitempty"`
	Probes               []probeSpecDTO `json:"probes"`
}

func toParametersDTO(p models.MatrixParameters) matrixParametersDTO {
	d := matrixParametersDTO{
		Sizes:                p.Sizes,
		PointerFieldVariants: p.PointerFieldVariants,
		Probes:               []probeSpecDTO{},
	}
	for _, profile := range p.Profiles {
		d.Profiles = append(d.Profiles, string(profile))
	}
	for _, mode := range p.PassingModes {
		d.PassingModes = append(d.PassingModes, string(mode))
	}
	for _, layout := range p.Layouts {
		d.Layouts = append(d.Layouts, string(layout))
	}
	d.Repeats = p.Repeats
	d.Payloads = p.Payloads
	if p.Replicates > 1 {
		d.Replicates = p.Replicates
	}
	for _, spec := range p.Probes {
		d.Probes = append(d.Probes, probeSpecDTO{Kind: string(spec.Kind), Parameter: spec.Parameter})
	}
	return d
}

func (d matrixParametersDTO) toModel() models.MatrixParameters {
	p := models.MatrixParameters{Sizes: d.Sizes, PointerFieldVariants: d.PointerFieldVariants}
	for _, profile := range d.Profiles {
		p.Profiles = append(p.Profiles, models.LifetimeProfile(profile))
	}
	for _, mode := range d.PassingModes {
		p.PassingModes = append(p.PassingModes, models.PassingMode(mode))
	}
	for _, layout := range d.Layouts {
		p.Layouts = append(p.Layouts, models.Layout(layout))
	}
	p.Repeats = d.Repeats
	p.Payloads = d.Payloads
	p.Replicates = d.Replicates
	if p.Replicates < 1 {
		p.Replicates = models.DefaultReplicates()
	}
	for _, spec := range d.Probes {
		p.Probes = append(p.Probes, models.ProbeSpec{Kind: models.ProbeKind(spec.Kind), Parameter: spec.Parameter})
	}
	return p
}

type matrixDTO struct {
	ID            string              `json:"id"`
	Parameters    matrixParametersDTO `json:"parameters"`
	HarnessDigest string              `json:"harnessDigest"`
	Cells         []cellDTO           `json:"cells"`
	Probes        []probeDTO          `json:"probes"`
	GeneratedAt   time.Time           `json:"generatedAt"`
}

func toMatrixDTO(m models.Matrix) matrixDTO {
	d := matrixDTO{
		ID:            m.ID,
		Parameters:    toParametersDTO(m.Parameters),
		HarnessDigest: m.HarnessDigest,
		Cells:         make([]cellDTO, 0, len(m.Cells)),
		Probes:        make([]probeDTO, 0, len(m.Probes)),
		GeneratedAt:   m.GeneratedAt.UTC(),
	}
	for _, cell := range m.Cells {
		d.Cells = append(d.Cells, toCellDTO(cell))
	}
	for _, probe := range m.Probes {
		d.Probes = append(d.Probes, toProbeDTO(probe))
	}
	return d
}

func (d matrixDTO) toModel() models.Matrix {
	m := models.Matrix{
		ID:            d.ID,
		Parameters:    d.Parameters.toModel(),
		HarnessDigest: d.HarnessDigest,
		GeneratedAt:   d.GeneratedAt,
	}
	for _, cell := range d.Cells {
		m.Cells = append(m.Cells, cell.toModel())
	}
	for _, probe := range d.Probes {
		m.Probes = append(m.Probes, probe.toModel())
	}
	return m
}

type escapeVerdictDTO struct {
	CellID         string `json:"cellId"`
	Escapes        bool   `json:"escapes"`
	CompilerReason string `json:"compilerReason"`
	Category       string `json:"category"`
	Status         string `json:"status"`
	CompilerError  string `json:"compilerError,omitempty"`
}

type escapeReportDTO struct {
	MatrixID   string             `json:"matrixId"`
	Provenance provenanceDTO      `json:"provenance"`
	Verdicts   []escapeVerdictDTO `json:"verdicts"`
}

func toEscapeReportDTO(r models.EscapeReport) escapeReportDTO {
	d := escapeReportDTO{MatrixID: r.MatrixID, Provenance: toProvenanceDTO(r.Provenance), Verdicts: make([]escapeVerdictDTO, 0, len(r.Verdicts))}
	for _, v := range r.Verdicts {
		d.Verdicts = append(d.Verdicts, escapeVerdictDTO{
			CellID: v.CellID, Escapes: v.Escapes, CompilerReason: v.CompilerReason,
			Category: string(v.Category), Status: string(v.Status), CompilerError: v.CompilerError,
		})
	}
	return d
}

func (d escapeReportDTO) toModel() models.EscapeReport {
	r := models.EscapeReport{MatrixID: d.MatrixID, Provenance: d.Provenance.toModel()}
	for _, v := range d.Verdicts {
		r.Verdicts = append(r.Verdicts, models.EscapeVerdict{
			CellID: v.CellID, Escapes: v.Escapes, CompilerReason: v.CompilerReason,
			Category: models.EscapeCategory(v.Category), Status: models.EscapeStatus(v.Status), CompilerError: v.CompilerError,
		})
	}
	return r
}

type campaignDTO struct {
	ID               string        `json:"id"`
	MatrixID         string        `json:"matrixId"`
	HarnessDigest    string        `json:"harnessDigest"`
	HypothesesDigest string        `json:"hypothesesDigest"`
	HypothesisIDs    []string      `json:"hypothesisIds"`
	Count            int           `json:"count"`
	Status           string        `json:"status"`
	Provenance       provenanceDTO `json:"provenance"`
	StartedAt        time.Time     `json:"startedAt"`
	FinishedAt       *time.Time    `json:"finishedAt,omitempty"`
	AbortReason      string        `json:"abortReason,omitempty"`
}

func toCampaignDTO(c models.Campaign) campaignDTO {
	d := campaignDTO{
		ID: c.ID, MatrixID: c.MatrixID, HarnessDigest: c.HarnessDigest,
		HypothesesDigest: c.HypothesesDigest, HypothesisIDs: c.HypothesisIDs, Count: c.Count,
		Status: string(c.Status), Provenance: toProvenanceDTO(c.Provenance),
		StartedAt: c.StartedAt.UTC(), AbortReason: c.AbortReason,
	}
	if !c.FinishedAt.IsZero() {
		finished := c.FinishedAt.UTC()
		d.FinishedAt = &finished
	}
	return d
}

func (d campaignDTO) toModel() models.Campaign {
	c := models.Campaign{
		ID: d.ID, MatrixID: d.MatrixID, HarnessDigest: d.HarnessDigest,
		HypothesesDigest: d.HypothesesDigest, HypothesisIDs: d.HypothesisIDs, Count: d.Count,
		Status: models.CampaignStatus(d.Status), Provenance: d.Provenance.toModel(),
		StartedAt: d.StartedAt, AbortReason: d.AbortReason,
	}
	if d.FinishedAt != nil {
		c.FinishedAt = *d.FinishedAt
	}
	return c
}

type measurementDTO struct {
	CampaignID    string    `json:"campaignId"`
	SubjectID     string    `json:"subjectId"`
	NsPerOp       []float64 `json:"nsPerOp"`
	BytesPerOp    []int64   `json:"bytesPerOp"`
	AllocsPerOp   []int64   `json:"allocsPerOp"`
	Status        string    `json:"status"`
	FailureReason string    `json:"failureReason,omitempty"`
}

func toMeasurementDTO(m models.Measurement) measurementDTO {
	return measurementDTO{
		CampaignID: m.CampaignID, SubjectID: m.SubjectID, NsPerOp: m.NsPerOp,
		BytesPerOp: m.BytesPerOp, AllocsPerOp: m.AllocsPerOp,
		Status: string(m.Status), FailureReason: m.FailureReason,
	}
}

func (d measurementDTO) toModel() models.Measurement {
	return models.Measurement{
		CampaignID: d.CampaignID, SubjectID: d.SubjectID, NsPerOp: d.NsPerOp,
		BytesPerOp: d.BytesPerOp, AllocsPerOp: d.AllocsPerOp,
		Status: models.MeasurementStatus(d.Status), FailureReason: d.FailureReason,
	}
}

type comparisonDTO struct {
	CampaignID      string  `json:"campaignId"`
	ValueCellID     string  `json:"valueCellId"`
	PointerCellID   string  `json:"pointerCellId"`
	SizeBytes       int     `json:"sizeBytes"`
	HasPointerField bool    `json:"hasPointerField"`
	Layout          string  `json:"layout,omitempty"`
	Profile         string  `json:"lifetimeProfile"`
	DeltaNsPerOp    float64 `json:"deltaNsPerOp"`
	CILow           float64 `json:"ciLow"`
	CIHigh          float64 `json:"ciHigh"`
	Significant     bool    `json:"significant"`
	MedianValueNs   float64 `json:"medianValueNsPerOp"`
	MedianPointerNs float64 `json:"medianPointerNsPerOp"`
}

type tippingPointDTO struct {
	Profile         string `json:"lifetimeProfile"`
	Layout          string `json:"layout,omitempty"`
	HasPointerField bool   `json:"hasPointerField"`
	SizeBytes       int    `json:"sizeBytes"`
	Observed        bool   `json:"observed"`
}

type excludedPairDTO struct {
	ValueCellID   string `json:"valueCellId"`
	PointerCellID string `json:"pointerCellId"`
	Reason        string `json:"reason"`
}

type comparisonSetDTO struct {
	CampaignID    string            `json:"campaignId"`
	MatrixID      string            `json:"matrixId"`
	ComputedAt    time.Time         `json:"computedAt"`
	Method        string            `json:"method"`
	Comparisons   []comparisonDTO   `json:"comparisons"`
	TippingPoints []tippingPointDTO `json:"tippingPoints"`
	ExcludedPairs []excludedPairDTO `json:"excludedPairs"`
}

func toComparisonSetDTO(s models.ComparisonSet) comparisonSetDTO {
	d := comparisonSetDTO{
		CampaignID: s.CampaignID, MatrixID: s.MatrixID, ComputedAt: s.ComputedAt.UTC(), Method: s.Method,
		Comparisons:   make([]comparisonDTO, 0, len(s.Comparisons)),
		TippingPoints: make([]tippingPointDTO, 0, len(s.TippingPoints)),
		ExcludedPairs: make([]excludedPairDTO, 0, len(s.ExcludedPairs)),
	}
	for _, c := range s.Comparisons {
		d.Comparisons = append(d.Comparisons, comparisonDTO{
			CampaignID: c.CampaignID, ValueCellID: c.ValueCellID, PointerCellID: c.PointerCellID,
			SizeBytes: c.SizeBytes, HasPointerField: c.HasPointerField, Layout: string(c.Layout), Profile: string(c.Profile),
			DeltaNsPerOp: c.DeltaNsPerOp, CILow: c.CILow, CIHigh: c.CIHigh, Significant: c.Significant,
			MedianValueNs: c.MedianValueNs, MedianPointerNs: c.MedianPointerNs,
		})
	}
	for _, key := range sortedTippingKeys(s.TippingPoints) {
		size := s.TippingPoints[key]
		layout := ""
		if key.Layout != "" && key.Layout != models.LayoutArrayFill {
			layout = string(key.Layout)
		}
		d.TippingPoints = append(d.TippingPoints, tippingPointDTO{
			Profile: string(key.Profile), Layout: layout, HasPointerField: key.HasPointerField,
			SizeBytes: size, Observed: size != models.TippingNotObserved,
		})
	}
	for _, e := range s.ExcludedPairs {
		d.ExcludedPairs = append(d.ExcludedPairs, excludedPairDTO{ValueCellID: e.ValueCellID, PointerCellID: e.PointerCellID, Reason: e.Reason})
	}
	return d
}

func (d comparisonSetDTO) toModel() models.ComparisonSet {
	s := models.ComparisonSet{
		CampaignID: d.CampaignID, MatrixID: d.MatrixID, ComputedAt: d.ComputedAt, Method: d.Method,
		TippingPoints: make(map[models.TippingKey]int, len(d.TippingPoints)),
	}
	for _, c := range d.Comparisons {
		layout := models.Layout(c.Layout)
		if layout == "" {
			layout = models.LayoutArrayFill
		}
		s.Comparisons = append(s.Comparisons, models.Comparison{
			CampaignID: c.CampaignID, ValueCellID: c.ValueCellID, PointerCellID: c.PointerCellID,
			SizeBytes: c.SizeBytes, HasPointerField: c.HasPointerField, Layout: layout, Profile: models.LifetimeProfile(c.Profile),
			DeltaNsPerOp: c.DeltaNsPerOp, CILow: c.CILow, CIHigh: c.CIHigh, Significant: c.Significant,
			MedianValueNs: c.MedianValueNs, MedianPointerNs: c.MedianPointerNs,
		})
	}
	for _, t := range d.TippingPoints {
		layout := models.Layout(t.Layout)
		if layout == "" {
			layout = models.LayoutArrayFill
		}
		key := models.TippingKey{Profile: models.LifetimeProfile(t.Profile), Layout: layout, HasPointerField: t.HasPointerField}
		if t.Observed {
			s.TippingPoints[key] = t.SizeBytes
		} else {
			s.TippingPoints[key] = models.TippingNotObserved
		}
	}
	for _, e := range d.ExcludedPairs {
		s.ExcludedPairs = append(s.ExcludedPairs, models.ExcludedPair{ValueCellID: e.ValueCellID, PointerCellID: e.PointerCellID, Reason: e.Reason})
	}
	return s
}

type verdictDTO struct {
	HypothesisID string   `json:"hypothesisId"`
	CampaignID   string   `json:"campaignId"`
	Outcome      string   `json:"outcome"`
	Rationale    string   `json:"rationale"`
	ResultFiles  []string `json:"resultFiles"`
}

type verdictReportDTO struct {
	CampaignID string       `json:"campaignId"`
	ProducedAt time.Time    `json:"producedAt"`
	Verdicts   []verdictDTO `json:"verdicts"`
}

func toVerdictReportDTO(r models.VerdictReport) verdictReportDTO {
	d := verdictReportDTO{CampaignID: r.CampaignID, ProducedAt: r.ProducedAt.UTC(), Verdicts: make([]verdictDTO, 0, len(r.Verdicts))}
	for _, v := range r.Verdicts {
		d.Verdicts = append(d.Verdicts, verdictDTO{
			HypothesisID: v.HypothesisID, CampaignID: v.CampaignID, Outcome: string(v.Outcome),
			Rationale: v.Rationale, ResultFiles: v.ResultFiles,
		})
	}
	return d
}

func (d verdictReportDTO) toModel() models.VerdictReport {
	r := models.VerdictReport{CampaignID: d.CampaignID, ProducedAt: d.ProducedAt}
	for _, v := range d.Verdicts {
		r.Verdicts = append(r.Verdicts, models.Verdict{
			HypothesisID: v.HypothesisID, CampaignID: v.CampaignID, Outcome: models.Outcome(v.Outcome),
			Rationale: v.Rationale, ResultFiles: v.ResultFiles,
		})
	}
	return r
}
