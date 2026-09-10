// Package models contient les entités du banc (docs/entity-model.md) : TypeSpec, LifetimeProfile,
// Cell, Probe, Matrix, EscapeVerdict, Provenance, Campaign, Measurement, Comparison, ComparisonSet,
// Hypothesis, Verdict. Bibliothèque standard uniquement ; aucun tag d'encodage (C-004, CLAUDE.md) —
// les DTO et l'encodage JSON vivent dans internal/adapters/store.
package models

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// ErrValidation enveloppe toute violation d'une règle du modèle d'entités.
var ErrValidation = errors.New("modèle invalide")

// invalid construit une erreur de validation traçable au champ fautif.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w : %s", ErrValidation, fmt.Sprintf(format, args...))
}

// PassingMode est le mode de passage d'une Cell (entity-model : Cell.passingMode).
type PassingMode string

const (
	PassingValue   PassingMode = "VALUE"
	PassingPointer PassingMode = "POINTER"
)

// PassingModes énumère les valeurs valides, dans l'ordre canonique.
func PassingModes() []PassingMode { return []PassingMode{PassingValue, PassingPointer} }

// Valid indique si le mode appartient à la liste du modèle d'entités.
func (m PassingMode) Valid() bool { return m == PassingValue || m == PassingPointer }

// LifetimeProfile est le profil de durée de vie d'une valeur dans le harnais.
type LifetimeProfile string

const (
	ProfileLocal             LifetimeProfile = "LOCAL"
	ProfileReturned          LifetimeProfile = "RETURNED"
	ProfileCapturedByClosure LifetimeProfile = "CAPTURED_BY_CLOSURE"
	ProfileSentOnChannel     LifetimeProfile = "SENT_ON_CHANNEL"
	ProfileStoredInMap       LifetimeProfile = "STORED_IN_MAP"
)

// LifetimeProfiles énumère les cinq profils, dans l'ordre canonique.
func LifetimeProfiles() []LifetimeProfile {
	return []LifetimeProfile{ProfileLocal, ProfileReturned, ProfileCapturedByClosure, ProfileSentOnChannel, ProfileStoredInMap}
}

// Valid indique si le profil appartient à la liste du modèle d'entités.
func (p LifetimeProfile) Valid() bool {
	for _, known := range LifetimeProfiles() {
		if p == known {
			return true
		}
	}
	return false
}

// ProbeKind est le genre d'une sonde (entity-model : Probe.kind).
type ProbeKind string

const (
	ProbeSequentialScan ProbeKind = "SEQUENTIAL_SCAN"
	ProbeScatteredScan  ProbeKind = "SCATTERED_SCAN"
	ProbeAppendPrealloc ProbeKind = "APPEND_PREALLOC"
	ProbeAppendGrow     ProbeKind = "APPEND_GROW"
)

// ProbeKinds énumère les quatre genres, dans l'ordre canonique.
func ProbeKinds() []ProbeKind {
	return []ProbeKind{ProbeSequentialScan, ProbeScatteredScan, ProbeAppendPrealloc, ProbeAppendGrow}
}

// Valid indique si le genre appartient à la liste du modèle d'entités.
func (k ProbeKind) Valid() bool {
	for _, known := range ProbeKinds() {
		if k == known {
			return true
		}
	}
	return false
}

// Bornes de taille des TypeSpec (entity-model : TypeSpec.sizeBytes).
const (
	MinSizeBytes = 8
	MaxSizeBytes = 4096
	WordBytes    = 8
)

// TypeSpec décrit un type struct généré.
type TypeSpec struct {
	Name            string
	SizeBytes       int
	HasPointerField bool
}

// WordCount est l'attribut dérivé wordCount (mots machine de 64 bits).
func (t TypeSpec) WordCount() int { return t.SizeBytes / WordBytes }

// Validate applique les règles de validation du modèle d'entités.
func (t TypeSpec) Validate() error {
	if t.Name == "" {
		return invalid("TypeSpec.name est requis")
	}
	return ValidateSize(t.SizeBytes)
}

// ValidateSize applique la règle « multiple de 8 entre 8 et 4096 ».
func ValidateSize(size int) error {
	if size < MinSizeBytes || size > MaxSizeBytes {
		return invalid("taille %d hors de [%d, %d]", size, MinSizeBytes, MaxSizeBytes)
	}
	if size%WordBytes != 0 {
		return invalid("taille %d n'est pas un multiple de %d", size, WordBytes)
	}
	return nil
}

// TypeSpecName construit un nom de type Go déterministe et unique par (taille, champ pointeur).
// Le remplissage à quatre chiffres garde l'ordre lexicographique aligné sur l'ordre des tailles.
func TypeSpecName(sizeBytes int, hasPointerField bool) string {
	if hasPointerField {
		return fmt.Sprintf("Size%04dPtr", sizeBytes)
	}
	return fmt.Sprintf("Size%04dPlain", sizeBytes)
}

// NewTypeSpec construit un TypeSpec validé.
func NewTypeSpec(sizeBytes int, hasPointerField bool) (TypeSpec, error) {
	t := TypeSpec{Name: TypeSpecName(sizeBytes, hasPointerField), SizeBytes: sizeBytes, HasPointerField: hasPointerField}
	return t, t.Validate()
}

// Cell est l'unité de mesure : un TypeSpec × un LifetimeProfile × un mode de passage.
type Cell struct {
	TypeSpec    TypeSpec
	Profile     LifetimeProfile
	PassingMode PassingMode
	SourceFile  string
}

// ID rend l'identifiant immuable de la forme <TypeSpec.name>/<LifetimeProfile.code>/<passingMode>.
func (c Cell) ID() string {
	return c.TypeSpec.Name + "/" + string(c.Profile) + "/" + string(c.PassingMode)
}

// Validate applique les règles de validation du modèle d'entités.
func (c Cell) Validate() error {
	if err := c.TypeSpec.Validate(); err != nil {
		return err
	}
	if !c.Profile.Valid() {
		return invalid("LifetimeProfile %q inconnu", c.Profile)
	}
	if !c.PassingMode.Valid() {
		return invalid("passingMode %q inconnu", c.PassingMode)
	}
	if c.SourceFile == "" {
		return invalid("Cell.sourceFile est requis (%s)", c.ID())
	}
	return nil
}

// Probe est une sonde de mesure indépendante des TypeSpec (FR-006, H-004, H-005).
type Probe struct {
	Kind       ProbeKind
	Parameter  int
	SourceFile string
}

// ID rend l'identifiant immuable de la forme probe/<kind>/<parameter>. Le deuxième segment est un
// genre de Probe, jamais un code de LifetimeProfile : les identifiants restent disjoints de ceux
// des Cell (entity-model, Probe.id).
func (p Probe) ID() string { return fmt.Sprintf("probe/%s/%d", p.Kind, p.Parameter) }

// Validate applique les règles de validation du modèle d'entités.
func (p Probe) Validate() error {
	if !p.Kind.Valid() {
		return invalid("ProbeKind %q inconnu", p.Kind)
	}
	if p.Parameter <= 0 {
		return invalid("Probe.parameter doit être > 0 (%s)", p.Kind)
	}
	if p.SourceFile == "" {
		return invalid("Probe.sourceFile est requis (%s)", p.ID())
	}
	return nil
}

// Catégories d'échappement (entity-model : EscapeVerdict.category).
type EscapeCategory string

const (
	CategoryReturnPointer  EscapeCategory = "RETURN_POINTER"
	CategoryClosureCapture EscapeCategory = "CLOSURE_CAPTURE"
	CategoryChannelSend    EscapeCategory = "CHANNEL_SEND"
	CategoryContainerStore EscapeCategory = "CONTAINER_STORE"
	CategoryOther          EscapeCategory = "OTHER"
	CategoryNone           EscapeCategory = "NONE"
)

// EscapeCategories énumère les catégories, dans l'ordre canonique.
func EscapeCategories() []EscapeCategory {
	return []EscapeCategory{CategoryReturnPointer, CategoryClosureCapture, CategoryChannelSend,
		CategoryContainerStore, CategoryOther, CategoryNone}
}

// BookCategories rend les quatre causes listées par BEPG p. 238-242 ; H-006 est infirmée dès qu'un
// échappement observé n'appartient à aucune d'elles.
func BookCategories() []EscapeCategory {
	return []EscapeCategory{CategoryReturnPointer, CategoryClosureCapture, CategoryChannelSend, CategoryContainerStore}
}

// InBook indique si la catégorie figure parmi les quatre causes du livre.
func (c EscapeCategory) InBook() bool {
	for _, known := range BookCategories() {
		if c == known {
			return true
		}
	}
	return false
}

// Valid indique si la catégorie appartient à la liste du modèle d'entités.
func (c EscapeCategory) Valid() bool {
	for _, known := range EscapeCategories() {
		if c == known {
			return true
		}
	}
	return false
}

// EscapeStatus est le statut d'un EscapeVerdict.
type EscapeStatus string

const (
	EscapeStatusOK           EscapeStatus = "OK"
	EscapeStatusCompileError EscapeStatus = "COMPILE_ERROR"
)

// EscapeVerdict est le résultat de la classification d'une cellule par le compilateur.
type EscapeVerdict struct {
	CellID         string
	Escapes        bool
	CompilerReason string
	Category       EscapeCategory
	Status         EscapeStatus
	CompilerError  string
}

// Validate applique les règles de validation du modèle d'entités.
func (v EscapeVerdict) Validate() error {
	if v.CellID == "" {
		return invalid("EscapeVerdict.cellId est requis")
	}
	if v.Status == EscapeStatusCompileError {
		if v.CompilerError == "" {
			return invalid("EscapeVerdict.compilerError est requis si COMPILE_ERROR (%s)", v.CellID)
		}
		return nil
	}
	if v.Status != EscapeStatusOK {
		return invalid("EscapeVerdict.status %q inconnu (%s)", v.Status, v.CellID)
	}
	if !v.Category.Valid() {
		return invalid("EscapeVerdict.category %q inconnue (%s)", v.Category, v.CellID)
	}
	// BR-002-1 : un verdict qui échappe porte exactement une catégorie, jamais NONE.
	if v.Escapes && v.Category == CategoryNone {
		return invalid("EscapeVerdict qui échappe ne peut être NONE (%s)", v.CellID)
	}
	if !v.Escapes && v.Category != CategoryNone {
		return invalid("EscapeVerdict qui n'échappe pas doit être NONE (%s)", v.CellID)
	}
	if !v.Escapes && v.CompilerReason != "" {
		return invalid("EscapeVerdict.compilerReason doit être vide si escapes est faux (%s)", v.CellID)
	}
	return nil
}

// EscapeReport est le contenu d'un fichier de verdicts d'échappement (UC-002, étape 6).
type EscapeReport struct {
	MatrixID   string
	Provenance Provenance
	Verdicts   []EscapeVerdict
}

// CountByCategory rend le décompte par catégorie (UC-002, étape 7).
func (r EscapeReport) CountByCategory() map[EscapeCategory]int {
	counts := make(map[EscapeCategory]int, len(EscapeCategories()))
	for _, v := range r.Verdicts {
		if v.Status == EscapeStatusOK {
			counts[v.Category]++
		}
	}
	return counts
}

// CountCompileErrors rend le nombre de cellules en COMPILE_ERROR (UC-002, A2).
func (r EscapeReport) CountCompileErrors() int {
	n := 0
	for _, v := range r.Verdicts {
		if v.Status == EscapeStatusCompileError {
			n++
		}
	}
	return n
}

// Provenance identifie la toolchain et la machine d'une mesure (NFR-001).
type Provenance struct {
	GoVersion  string
	GOOS       string
	GOARCH     string
	CPUModel   string
	CapturedAt time.Time
}

// Validate applique NFR-001 : un résultat sans provenance complète est invalide.
func (p Provenance) Validate() error {
	switch {
	case p.GoVersion == "":
		return invalid("Provenance.goVersion est requis (NFR-001)")
	case p.GOOS == "":
		return invalid("Provenance.goos est requis (NFR-001)")
	case p.GOARCH == "":
		return invalid("Provenance.goarch est requis (NFR-001)")
	case p.CPUModel == "":
		return invalid("Provenance.cpuModel est requis (NFR-001)")
	case p.CapturedAt.IsZero():
		return invalid("Provenance.capturedAt est requis (NFR-001)")
	}
	return nil
}

// SameToolchain compare les seuls champs qui identifient la toolchain (UC-002, A3) : capturedAt et
// cpuModel ne sont pas comparés.
func (p Provenance) SameToolchain(other Provenance) bool {
	return p.GoVersion == other.GoVersion && p.GOOS == other.GOOS && p.GOARCH == other.GOARCH
}

// CampaignStatus est le statut d'une Campaign.
type CampaignStatus string

const (
	CampaignRunning   CampaignStatus = "RUNNING"
	CampaignCompleted CampaignStatus = "COMPLETED"
	CampaignAborted   CampaignStatus = "ABORTED"
)

// MinCount est le nombre minimal de répétitions par sujet (NFR-003).
const MinCount = 20

// Campaign est une exécution de mesure sur une Matrix.
type Campaign struct {
	ID               string
	MatrixID         string
	HarnessDigest    string
	HypothesesDigest string
	HypothesisIDs    []string
	Count            int
	Status           CampaignStatus
	Provenance       Provenance
	StartedAt        time.Time
	FinishedAt       time.Time
	AbortReason      string
}

// Validate applique les règles de validation du modèle d'entités.
func (c Campaign) Validate() error {
	switch {
	case c.ID == "":
		return invalid("Campaign.id est requis")
	case c.MatrixID == "":
		return invalid("Campaign.matrixId est requis")
	case c.HarnessDigest == "":
		return invalid("Campaign.harnessDigest est requis (BR-003-1)")
	case c.HypothesesDigest == "":
		return invalid("Campaign.hypothesesDigest est requis (BR-003-5)")
	case c.Count < MinCount:
		return invalid("Campaign.count %d < %d (NFR-003)", c.Count, MinCount)
	case c.Status != CampaignRunning && c.Status != CampaignCompleted && c.Status != CampaignAborted:
		return invalid("Campaign.status %q inconnu", c.Status)
	}
	return c.Provenance.Validate()
}

// MeasurementStatus est le statut d'une Measurement.
type MeasurementStatus string

const (
	MeasurementComplete MeasurementStatus = "COMPLETE"
	MeasurementFailed   MeasurementStatus = "FAILED"
)

// Measurement porte les `count` répétitions d'un sujet (Cell ou Probe) d'une Campaign.
type Measurement struct {
	CampaignID    string
	SubjectID     string
	NsPerOp       []float64
	BytesPerOp    []int64
	AllocsPerOp   []int64
	Status        MeasurementStatus
	FailureReason string
}

// Validate applique les règles de validation du modèle d'entités pour un `count` donné.
func (m Measurement) Validate(count int) error {
	if m.CampaignID == "" || m.SubjectID == "" {
		return invalid("Measurement.campaignId et subjectId sont requis")
	}
	if m.Status == MeasurementFailed {
		if m.FailureReason == "" {
			return invalid("Measurement.failureReason est requis si FAILED (%s)", m.SubjectID)
		}
		if len(m.NsPerOp) != 0 || len(m.BytesPerOp) != 0 || len(m.AllocsPerOp) != 0 {
			return invalid("les listes sont vides si FAILED (%s)", m.SubjectID)
		}
		return nil
	}
	if m.Status != MeasurementComplete {
		return invalid("Measurement.status %q inconnu (%s)", m.Status, m.SubjectID)
	}
	if len(m.NsPerOp) != count || len(m.BytesPerOp) != count || len(m.AllocsPerOp) != count {
		return invalid("Measurement de %s : %d/%d/%d valeurs, %d attendues (NFR-003)",
			m.SubjectID, len(m.NsPerOp), len(m.BytesPerOp), len(m.AllocsPerOp), count)
	}
	return nil
}

// MedianNs rend la médiane de nsPerOp ; 0 si la mesure est vide.
func (m Measurement) MedianNs() float64 { return MedianFloat(m.NsPerOp) }

// MedianBytes rend la médiane de bytesPerOp ; 0 si la mesure est vide.
func (m Measurement) MedianBytes() float64 { return MedianInt(m.BytesPerOp) }

// MedianAllocs rend la médiane de allocsPerOp ; 0 si la mesure est vide.
func (m Measurement) MedianAllocs() float64 { return MedianInt(m.AllocsPerOp) }

// MedianFloat rend la médiane d'un échantillon sans le modifier.
func MedianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// MedianInt rend la médiane d'un échantillon entier sans le modifier.
func MedianInt(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	asFloat := make([]float64, len(values))
	for i, v := range values {
		asFloat[i] = float64(v)
	}
	return MedianFloat(asFloat)
}

// Comparison est la comparaison d'une paire (VALUE, POINTER) d'une Campaign (UC-004).
// Les attributs de la paire (taille, champ pointeur, profil) sont dénormalisés pour que le fichier
// de comparaison suffise à évaluer H-001 et H-002 sans relire la Matrix (BR-005-2).
type Comparison struct {
	CampaignID      string
	ValueCellID     string
	PointerCellID   string
	SizeBytes       int
	HasPointerField bool
	Profile         LifetimeProfile
	DeltaNsPerOp    float64
	CILow           float64
	CIHigh          float64
	Significant     bool
	MedianValueNs   float64
	MedianPointerNs float64
}

// TippingKey identifie une série de points de bascule : un profil × la présence d'un champ pointeur.
type TippingKey struct {
	Profile         LifetimeProfile
	HasPointerField bool
}

// TippingNotObserved est la valeur de tippingPoints quand aucune taille ne satisfait la condition.
const TippingNotObserved = -1

// ExcludedPair consigne une paire écartée et sa raison (BR-004-1).
type ExcludedPair struct {
	ValueCellID   string
	PointerCellID string
	Reason        string
}

// ComparisonSet est le contenu d'un fichier de comparaison d'une Campaign (UC-004).
type ComparisonSet struct {
	CampaignID    string
	MatrixID      string
	ComputedAt    time.Time
	Method        string
	Comparisons   []Comparison
	TippingPoints map[TippingKey]int
	ExcludedPairs []ExcludedPair
}

// Hypothesis est une hypothèse à éprouver, lue dans docs/requirements.md.
type Hypothesis struct {
	ID                  string
	SourcePages         string
	Statement           string
	RefutationCriterion string
	UseCases            []string
}

// Validate applique les règles de validation du modèle d'entités.
func (h Hypothesis) Validate() error {
	switch {
	case h.ID == "":
		return invalid("Hypothesis.id est requis")
	case h.SourcePages == "":
		return invalid("Hypothesis.sourcePages est requis (%s)", h.ID)
	case h.RefutationCriterion == "":
		return invalid("Hypothesis.refutationCriterion est requis (%s)", h.ID)
	}
	return nil
}

// Outcome est le verdict porté sur une hypothèse.
type Outcome string

const (
	OutcomeConfirmed    Outcome = "CONFIRMED"
	OutcomeRefuted      Outcome = "REFUTED"
	OutcomeInconclusive Outcome = "INCONCLUSIVE"
)

// Verdict est le résultat de l'évaluation d'un critère de réfutation gelé (UC-005).
type Verdict struct {
	HypothesisID string
	CampaignID   string
	Outcome      Outcome
	Rationale    string
	ResultFiles  []string
}

// Validate applique les règles de validation du modèle d'entités.
func (v Verdict) Validate() error {
	switch {
	case v.HypothesisID == "":
		return invalid("Verdict.hypothesisId est requis")
	case v.Outcome != OutcomeConfirmed && v.Outcome != OutcomeRefuted && v.Outcome != OutcomeInconclusive:
		return invalid("Verdict.outcome %q inconnu (%s)", v.Outcome, v.HypothesisID)
	case v.Rationale == "":
		return invalid("Verdict.rationale est requis (BR-005-2, %s)", v.HypothesisID)
	case len(v.ResultFiles) == 0:
		return invalid("Verdict.resultFiles est requis (BR-005-2, %s)", v.HypothesisID)
	}
	return nil
}

// VerdictReport est le contenu d'un fichier de verdicts (UC-005, étape 6).
type VerdictReport struct {
	CampaignID string
	ProducedAt time.Time
	Verdicts   []Verdict
}
