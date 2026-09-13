// Package models contient les entités du banc (docs/entity-model.md) : TypeSpec, LifetimeProfile,
// Cell, Probe, Matrix, EscapeVerdict, Provenance, Campaign, Measurement, Comparison, ComparisonSet,
// Hypothesis, Verdict. Bibliothèque standard uniquement ; aucun tag d'encodage (C-004, CLAUDE.md) —
// les DTO et l'encodage JSON vivent dans internal/adapters/store.
package models

import (
	"errors"
	"fmt"
	"math"
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
	// Profils ajoutés par C-008. STORED_IN_SLICE et STORED_IN_STRUCT reprennent la forme de
	// STORED_IN_MAP pour les deux autres conteneurs que le livre nomme (H-009).
	// RETURNED_ALLOCATING retourne une valeur accompagnée d'une charge allouée, de sorte que le
	// bras valeur alloue déjà et qu'un doublement soit calculable (H-010).
	ProfileStoredInSlice  LifetimeProfile = "STORED_IN_SLICE"
	ProfileStoredInStruct LifetimeProfile = "STORED_IN_STRUCT"
	ProfileReturnedAlloc  LifetimeProfile = "RETURNED_ALLOCATING"
)

// LifetimeProfiles énumère les profils connus, dans l'ordre canonique.
func LifetimeProfiles() []LifetimeProfile {
	return []LifetimeProfile{ProfileLocal, ProfileReturned, ProfileCapturedByClosure,
		ProfileSentOnChannel, ProfileStoredInMap, ProfileStoredInSlice, ProfileStoredInStruct,
		ProfileReturnedAlloc}
}

// ReferenceProfiles énumère les cinq profils de la matrice de référence (BR-001-4). Les profils
// ajoutés par C-008 n'en font pas partie : la matrice de référence garde ses 220 Cell.
func ReferenceProfiles() []LifetimeProfile {
	return []LifetimeProfile{ProfileLocal, ProfileReturned, ProfileCapturedByClosure, ProfileSentOnChannel, ProfileStoredInMap}
}

// Layout est la disposition des champs d'un TypeSpec (C-008). Elle décide si le type est
// assignable aux registres de la convention d'appel de Go, ce qui change le coût du passage par
// valeur indépendamment de la taille.
type Layout string

const (
	// LayoutArrayFill est la disposition d'origine : `Tag uint64` suivi de `Fill [n-1]uint64`.
	// Un tableau de longueur supérieure à 1 n'étant pas assignable aux registres, tout type de
	// 24 octets ou plus est passé en mémoire.
	LayoutArrayFill Layout = "ARRAY_FILL"
	// LayoutNamedFields déclare `sizeBytes / 8` champs `uint64` un à un, sans tableau ; la
	// variante à champ pointeur remplace le dernier par un `*uint64`.
	LayoutNamedFields Layout = "NAMED_FIELDS"
	// LayoutNamedFieldsSham est le témoin nul : même déclaration que NAMED_FIELDS, mais les deux
	// cellules de la paire exécutent le corps du mode VALUE. Son delta vrai est nul par
	// construction, ce qui mesure l'écart entre deux binaires.
	LayoutNamedFieldsSham Layout = "NAMED_FIELDS_SHAM"
)

// Layouts énumère les dispositions connues, dans l'ordre canonique.
func Layouts() []Layout { return []Layout{LayoutArrayFill, LayoutNamedFields, LayoutNamedFieldsSham} }

// Valid indique si la disposition appartient à la liste du modèle d'entités.
func (l Layout) Valid() bool {
	for _, known := range Layouts() {
		if l == known {
			return true
		}
	}
	return false
}

// RegisterAssignable indique si la disposition permet le passage en registres pour toute taille.
func (l Layout) RegisterAssignable() bool {
	return l == LayoutNamedFields || l == LayoutNamedFieldsSham
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
	// ProbePointerChase est ajoutée par C-008 : anneau de nœuds d'une ligne de cache chaînés en
	// permutation, une itération de b.N valant un seul accès dont l'adresse a été lue à l'accès
	// précédent. C'est une mesure de latence, là où les deux parcours mesurent un débit (H-008).
	ProbePointerChase ProbeKind = "POINTER_CHASE"
)

// ProbeKinds énumère les genres connus, dans l'ordre canonique.
func ProbeKinds() []ProbeKind {
	return []ProbeKind{ProbeSequentialScan, ProbeScatteredScan, ProbeAppendPrealloc, ProbeAppendGrow, ProbePointerChase}
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
	Layout          Layout
}

// WordCount est l'attribut dérivé wordCount (mots machine de 64 bits).
func (t TypeSpec) WordCount() int { return t.SizeBytes / WordBytes }

// Validate applique les règles de validation du modèle d'entités.
func (t TypeSpec) Validate() error {
	if t.Name == "" {
		return invalid("TypeSpec.name est requis")
	}
	if !t.Layout.Valid() {
		return invalid("TypeSpec.layout %q inconnue (%s)", t.Layout, t.Name)
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

// TypeSpecName construit un nom de type Go déterministe et unique par (taille, champ pointeur,
// disposition). Le remplissage à quatre chiffres garde l'ordre lexicographique aligné sur l'ordre
// des tailles. La disposition d'origine ne porte aucun marqueur : les identifiants de cellules des
// campagnes antérieures à C-008 restent inchangés.
func TypeSpecName(sizeBytes int, hasPointerField bool, layout Layout) string {
	marker := ""
	switch layout {
	case LayoutNamedFields:
		marker = "Fields"
	case LayoutNamedFieldsSham:
		marker = "Sham"
	}
	if hasPointerField {
		return fmt.Sprintf("Size%04d%sPtr", sizeBytes, marker)
	}
	return fmt.Sprintf("Size%04d%sPlain", sizeBytes, marker)
}

// NewTypeSpec construit un TypeSpec validé.
func NewTypeSpec(sizeBytes int, hasPointerField bool, layout Layout) (TypeSpec, error) {
	t := TypeSpec{
		Name:            TypeSpecName(sizeBytes, hasPointerField, layout),
		SizeBytes:       sizeBytes,
		HasPointerField: hasPointerField,
		Layout:          layout,
	}
	return t, t.Validate()
}

// Cell est l'unité de mesure : un TypeSpec × un LifetimeProfile × un mode de passage, avec le
// nombre d'instances produites par opération pour les profils qui en dépendent (C-008).
type Cell struct {
	TypeSpec    TypeSpec
	Profile     LifetimeProfile
	PassingMode PassingMode
	Repeat      int
	Payload     int
	// Replicate est ajouté par C-009 : le rang de cette mesure parmi les réplicats indépendants
	// de la même paire. Deux réplicats sont deux sujets, donc deux paquets Go, deux binaires et
	// deux processus `go test` ; c'est cette indépendance qui fait du plancher de H-012 une
	// grandeur mesurée et non postulée.
	Replicate  int
	SourceFile string
}

// ProfileSegment rend le deuxième segment de l'identifiant : le code du profil, suffixé du nombre
// d'instances par opération puis du nombre de charges par instance quand l'un ou l'autre diffère de
// un. Une répétition et une charge de un ne laissent aucune trace : les identifiants antérieurs à
// C-008 sont inchangés.
func (c Cell) ProfileSegment() string {
	segment := string(c.Profile)
	if c.Repeat > 1 {
		segment = fmt.Sprintf("%s_R%d", segment, c.Repeat)
	}
	if c.Payload > 1 {
		segment = fmt.Sprintf("%s_K%d", segment, c.Payload)
	}
	if c.Replicate > 1 {
		segment = fmt.Sprintf("%s_X%d", segment, c.Replicate)
	}
	return segment
}

// ID rend l'identifiant immuable de la forme <TypeSpec.name>/<LifetimeProfile.code>/<passingMode>.
func (c Cell) ID() string {
	return c.TypeSpec.Name + "/" + c.ProfileSegment() + "/" + string(c.PassingMode)
}

// Repetitions rend le nombre d'instances produites par opération, au moins une.
func (c Cell) Repetitions() int {
	if c.Repeat < 1 {
		return 1
	}
	return c.Repeat
}

// Payloads rend le nombre de charges allouées par instance, au moins une. C'est le k dont dépend le
// rapport d'allocations que mesure H-010 : le bras valeur en alloue k, le bras pointeur k + 1.
func (c Cell) Payloads() int {
	if c.Payload < 1 {
		return 1
	}
	return c.Payload
}

// ReplicateIndex rend le rang du réplicat, au moins un.
func (c Cell) ReplicateIndex() int {
	if c.Replicate < 1 {
		return 1
	}
	return c.Replicate
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
	// A-260 écarté, vérification faite : la valeur zéro de ces trois dimensions est ramenée à un
	// par Repetitions, Payloads et ReplicateIndex, et Cell.ID() passe par ces accesseurs. Une
	// cellule à zéro est donc identique, identifiant compris, à la même cellule à un — ce n'est pas
	// une cellule invalide mais la valeur zéro d'un champ dont le défaut est un. L'exiger ≥ 1
	// rejetterait aussi toute Cell décodée d'un matrix.json antérieur à C-008, où ces champs sont
	// absents. Seule une valeur négative, qui ne se ramène à rien, reste refusée.
	if c.Repeat < 0 {
		return invalid("Cell.repeat doit être positif (%s)", c.ID())
	}
	if c.Payload < 0 {
		return invalid("Cell.payload doit être positif (%s)", c.ID())
	}
	if c.Replicate < 0 {
		return invalid("Cell.replicate doit être positif (%s)", c.ID())
	}
	// Le témoin nul n'a de sens que par paire complète : ses deux cellules exécutent le même
	// corps, celui du mode VALUE.
	if c.TypeSpec.Layout == LayoutNamedFieldsSham && c.Profile != ProfileLocal {
		return invalid("le témoin nul %s ne se mesure qu'en profil LOCAL (%s)", LayoutNamedFieldsSham, c.ID())
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
	// A-008 : le message du compilateur n'a de sens qu'en COMPILE_ERROR. Le laisser passer en OK
	// autorisait un verdict qui se contredit lui-même.
	if v.CompilerError != "" {
		return invalid("EscapeVerdict.compilerError doit être vide si OK (%s)", v.CellID)
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

// Provenance identifie la toolchain et la machine d'une mesure (NFR-001). Les trois tailles
// mémoire sont ajoutées par C-008 : sans elles, aucune hypothèse portant sur la hiérarchie de
// cache n'est interprétable. Elles valent zéro quand la détection échoue, et H-008 rend alors son
// verdict non concluant plutôt que de supposer une valeur.
type Provenance struct {
	GoVersion           string
	GOOS                string
	GOARCH              string
	CPUModel            string
	L1DataCacheBytes    int64
	LastLevelCacheBytes int64
	PageSizeBytes       int64
	GOMAXPROCS          int
	CapturedAt          time.Time
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
	// CriteriaDigests est l'empreinte du critère de chaque hypothèse gelée, prise à la création
	// (BR-003-5). L'empreinte d'ensemble dit qu'un critère a changé ; celles-ci disent lequel,
	// ce qu'exige le flux A1 de UC-005. Absente des campagnes antérieures à cette révision.
	CriteriaDigests map[string]string
	Count           int
	// BenchTime et CPU sont les paramètres de mesure de C-003 sous lesquels les Measurement de
	// cette Campaign ont été prises. Ils y sont consignés pour qu'une reprise (UC-003, A4) les
	// restitue au lieu de reprendre les drapeaux de la ligne de commande du moment.
	BenchTime   string
	CPU         int
	Status      CampaignStatus
	Provenance  Provenance
	StartedAt   time.Time
	FinishedAt  time.Time
	AbortReason string
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
	case len(c.HypothesisIDs) == 0:
		return invalid("Campaign.hypothesisIds est requis (BR-003-5, %s)", c.ID)
	case c.StartedAt.IsZero():
		return invalid("Campaign.startedAt est requis (%s)", c.ID)
	case c.Status != CampaignRunning && c.Status != CampaignCompleted && c.Status != CampaignAborted:
		return invalid("Campaign.status %q inconnu", c.Status)
	}
	// A-005 : le modèle d'entités déclare ces champs requis selon le statut ; rien ne l'appliquait.
	// Une campagne close sans horodatage de fin, ou abandonnée sans raison, passait la validation
	// et se retrouvait dans results/ sans que le chercheur puisse savoir quand ni pourquoi.
	if c.Status != CampaignRunning && c.FinishedAt.IsZero() {
		return invalid("Campaign.finishedAt est requis si %s (%s)", c.Status, c.ID)
	}
	if c.Status == CampaignAborted && c.AbortReason == "" {
		return invalid("Campaign.abortReason est requis si ABORTED (UC-003 A2, %s)", c.ID)
	}
	if c.Status == CampaignRunning && !c.FinishedAt.IsZero() {
		return invalid("Campaign.finishedAt ne peut être posé si RUNNING (%s)", c.ID)
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
	// QuietudeOccupancy est l'attestation de quiétude de C-010 : la fraction de la capacité de la
	// machine consommée pendant la fenêtre de mesure par tout ce qui n'est pas le sujet, une fois
	// retranché le travail de la campagne elle-même. Elle vaut de 0 à 1 et ne se lit que si
	// QuietudeMeasured est vrai. Le champ est facultatif : un fichier de résultats antérieur à
	// C-010 reste valide, et une campagne qui ne le porte pas rend H-013 non concluante plutôt
	// que fausse. Deux champs plutôt qu'un pointeur : une occupation nulle est une valeur
	// légitime, qu'une absence ne doit pas imiter.
	QuietudeOccupancy float64
	QuietudeMeasured  bool
}

// QuietudeThreshold est le seuil qu'annonce C-010 : au-delà, la machine faisait pendant la fenêtre
// de mesure assez de travail étranger pour que la latence non résidente de H-013 ne soit plus
// imputable à la seule hiérarchie mémoire.
//
// Sa valeur est fixée avant toute mesure de H-013, à partir de deux relevés faits sur la machine du
// catalogue le 2026-09-10 : au repos, sans campagne, l'occupation vaut de 4,2 à 7,5 pour cent ;
// un seul cœur occupé sur les vingt-quatre en ajoute 4,2. Un seuil de 12 pour cent laisse donc
// passer le bruit de fond d'un poste de travail ordinaire et refuse tout ce qui occupe un cœur
// entier de plus. C'est la classe de charge avec laquelle la contre-épreuve du même jour a produit
// ses fausses infirmations.
const QuietudeThreshold = 0.12

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
	// A-016 : une valeur négative, infinie ou NaN traverserait médianes, rapports et intervalle de
	// confiance jusqu'au verdict, sans qu'aucun évaluateur ne la remarque.
	for i, ns := range m.NsPerOp {
		if math.IsNaN(ns) || math.IsInf(ns, 0) || ns < 0 {
			return invalid("Measurement de %s : nsPerOp[%d] = %v, une durée finie et positive est attendue",
				m.SubjectID, i, ns)
		}
	}
	for i, bytes := range m.BytesPerOp {
		if bytes < 0 {
			return invalid("Measurement de %s : bytesPerOp[%d] = %d, une valeur positive est attendue",
				m.SubjectID, i, bytes)
		}
	}
	for i, allocs := range m.AllocsPerOp {
		if allocs < 0 {
			return invalid("Measurement de %s : allocsPerOp[%d] = %d, une valeur positive est attendue",
				m.SubjectID, i, allocs)
		}
	}
	// A-009 : C-010 définit quietudeOccupancy comme une fraction. Hors de [0, 1], la garde de
	// quiétude de H-013 compare au seuil une valeur qui n'est pas une fraction.
	if m.QuietudeMeasured {
		if math.IsNaN(m.QuietudeOccupancy) || m.QuietudeOccupancy < 0 || m.QuietudeOccupancy > 1 {
			return invalid("Measurement de %s : quietudeOccupancy = %v, une fraction de [0, 1] est attendue (C-010)",
				m.SubjectID, m.QuietudeOccupancy)
		}
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
	Layout          Layout
	Profile         LifetimeProfile
	DeltaNsPerOp    float64
	CILow           float64
	CIHigh          float64
	Significant     bool
	MedianValueNs   float64
	MedianPointerNs float64
}

// EffectiveLayout rend la disposition de la paire, ARRAY_FILL quand elle est absente. Une
// Comparison lue d'un fichier antérieur à C-008 n'en porte pas ; c'est alors la seule disposition
// qui existait. Toute clé de série passe par ici, de sorte qu'un fichier ancien et un fichier neuf
// tombent dans la même série plutôt que dans deux.
func (c Comparison) EffectiveLayout() Layout {
	if c.Layout == "" {
		return LayoutArrayFill
	}
	return c.Layout
}

// TippingKey identifie une série de points de bascule : un profil × une disposition × la présence
// d'un champ pointeur.
//
// Révision du 2026-09-10 : la disposition est entrée dans la clé. Sans elle, une campagne à
// plusieurs dispositions — toute campagne H-007 — versait ses Comparison ARRAY_FILL, NAMED_FIELDS
// et NAMED_FIELDS_SHAM dans une même série, et le témoin nul, dont l'écart est nul par
// construction, suffisait à interrompre le suffixe favorable au pointeur et à retourner le point
// de bascule.
type TippingKey struct {
	Profile         LifetimeProfile
	Layout          Layout
	HasPointerField bool
}

// TippingNotObserved est la valeur de tippingPoints quand aucune taille ne satisfait la condition.
const TippingNotObserved = -1

// SmallStructBytes est la borne « une à trois mots machine » de BEPG p. 253, en octets sur 64 bits.
// Elle vit dans le modèle parce que deux couches s'y adossent : les évaluateurs de H-001, H-002 et
// H-007, et la validation des paramètres de matrice, qui borne le témoin nul à ces tailles.
const SmallStructBytes = 24

// ExcludedPair consigne une paire écartée et sa raison (BR-004-1).
type ExcludedPair struct {
	ValueCellID   string
	PointerCellID string
	Reason        string
}

// ExcludedSeries consigne une série dont le point de bascule n'est pas calculé, et pourquoi
// (UC-004, A4).
type ExcludedSeries struct {
	Key    TippingKey
	Reason string
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
	// ExcludedSeries dit quelles séries n'ont pas de point de bascule et pourquoi. Sans elle, une
	// série à plusieurs paires par taille disparaissait du fichier sans qu'aucune ligne n'indique
	// ni sa valeur, ni « non observé », ni la raison (A-032).
	ExcludedSeries []ExcludedSeries
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
	case h.Statement == "":
		// A-006 : l'énoncé entre dans l'empreinte gelée des critères (BR-003-5). Un énoncé vide
		// produit une empreinte valide sur une hypothèse qui ne dit rien.
		return invalid("Hypothesis.statement est requis (%s)", h.ID)
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
	case v.CampaignID == "":
		// A-007 : un verdict sans campagne ne peut plus être rattaché aux mesures qui le fondent
		// (BR-005-2), et le tableau de bord affiche une ligne sans campagne.
		return invalid("Verdict.campaignId est requis (%s)", v.HypothesisID)
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
