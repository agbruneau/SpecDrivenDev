// Package ports déclare ce que les services attendent de l'extérieur : dépôt de matrices,
// classificateur d'échappement, runner de mesures, calcul d'empreintes, horloge, provenance.
// Les interfaces sont petites et définies côté consommateur (BEPG ch. 20, p. 514-517).
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// ErrNotFound signale l'absence d'une matrice, d'une campagne ou d'un fichier attendu. La
// sentinelle vit au niveau du port pour que le service puisse distinguer une absence légitime
// d'une lecture en erreur, ce qu'une sentinelle d'adaptateur ne lui permet pas (A-123).
var ErrNotFound = errors.New("introuvable")

// ErrLockHeld signale qu'une autre campagne tient results/.campaign-lock (BR-003-3).
var ErrLockHeld = errors.New("une campagne est déjà en cours")

// Clock rend l'heure courante ; injectée pour que les tests n'attendent jamais (BEPG p. 225-227).
type Clock interface {
	Now() time.Time
}

// Digester calcule l'empreinte du harnais de mesure (BR-003-1, C-005).
type Digester interface {
	HarnessDigest(ctx context.Context) (string, error)
}

// ProvenanceProbe capture la provenance d'une mesure (NFR-001).
type ProvenanceProbe interface {
	Capture(ctx context.Context) (models.Provenance, error)
}

// SubjectSource rend le contenu des fichiers d'un sujet, relatifs au répertoire de la Matrix.
type SubjectSource interface {
	RenderCell(cell models.Cell) (map[string]string, error)
	RenderProbe(probe models.Probe) (map[string]string, error)
	RenderModule(matrixID string) (map[string]string, error)
}

// MatrixRepository conserve les matrices générées. Une Matrix est immuable après écriture
// (BR-001-2) : le dépôt ne propose aucune mise à jour.
type MatrixRepository interface {
	Exists(ctx context.Context, matrixID string) (bool, error)
	Load(ctx context.Context, matrixID string) (models.Matrix, error)
	// WriteSources écrit les fichiers générés avant compilation (UC-001, étapes 4-5).
	WriteSources(ctx context.Context, matrixID string, files map[string]string) error
	// Finalize écrit matrix.json et rend la matrice utilisable (UC-001, étape 6).
	Finalize(ctx context.Context, matrix models.Matrix) error
	Remove(ctx context.Context, matrixID string) error
	Dir(matrixID string) string
	List(ctx context.Context) ([]string, error)
}

// CampaignLock indique si une campagne est en cours (précondition de UC-001 et UC-003).
type CampaignLock interface {
	LockHeld(ctx context.Context) (bool, error)
}

// CompileError porte le message du compilateur quand un paquet de sujet ne compile pas
// (UC-001 A3, UC-002 A2).
type CompileError struct {
	SubjectID string
	Output    string
}

// Error rend le message du compilateur.
func (e *CompileError) Error() string {
	return "compilation de " + e.SubjectID + " : " + e.Output
}

// Compiler compile un paquet de sujet et rend les lignes de diagnostic d'échappement (C-003).
type Compiler interface {
	// EscapeAnalysis compile le paquet du sujet avec le diagnostic d'échappement et rend les
	// lignes brutes du compilateur. Elle rend un *CompileError si le paquet ne compile pas.
	EscapeAnalysis(ctx context.Context, matrixDir, subjectID string) ([]string, error)
	// Build compile le paquet du sujet sans diagnostic, pour vérifier qu'il est compilable.
	Build(ctx context.Context, matrixDir, subjectID string) error
}

// EscapeClassifier attribue une catégorie à la sortie du compilateur pour un sujet (BR-002-1).
// sourcePath est le chemin du fichier source du sujet, relatif au répertoire de la Matrix.
type EscapeClassifier interface {
	Classify(ctx context.Context, matrixDir, subjectID, sourcePath string, reasons []string) (models.EscapeVerdict, error)
}

// RunOptions porte les drapeaux de mesure imposés par C-003.
type RunOptions struct {
	Count     int
	BenchTime string
	CPU       int
}

// BenchmarkRunner mesure un sujet dans un processus `go test` distinct (BR-003-4).
type BenchmarkRunner interface {
	Run(ctx context.Context, matrixDir, subjectID string, opts RunOptions) (models.Measurement, error)
}

// EscapeStore conserve les fichiers de verdicts d'échappement (UC-002).
type EscapeStore interface {
	WriteEscapeReport(ctx context.Context, report models.EscapeReport, capturedAt time.Time) (string, error)
	ListEscapeReports(ctx context.Context, matrixID string) ([]string, error)
	ReadEscapeReport(ctx context.Context, path string) (models.EscapeReport, error)
}

// CampaignStore conserve les campagnes, leurs mesures et leurs comparaisons (UC-003, UC-004).
// L'écriture est strictement additive, sauf la transition de statut de campaign.json et le
// verrou de campagne, nommés par BR-003-3.
type CampaignStore interface {
	CreateCampaign(ctx context.Context, campaign models.Campaign) error
	LoadCampaign(ctx context.Context, campaignID string) (models.Campaign, error)
	SetCampaignStatus(ctx context.Context, campaignID string, status models.CampaignStatus, finishedAt time.Time, abortReason string) error
	NextCampaignID(ctx context.Context, day time.Time) (string, error)
	WriteMeasurement(ctx context.Context, m models.Measurement) error
	LoadMeasurements(ctx context.Context, campaignID string) ([]models.Measurement, error)
	WriteComparisonSet(ctx context.Context, set models.ComparisonSet) (string, error)
	LatestComparisonSet(ctx context.Context, campaignID string) (models.ComparisonSet, string, error)
	MeasurementPath(campaignID, subjectID string) string
	CampaignPath(campaignID string) string
	// AcquireLock pose results/.campaign-lock. Un verrou orphelin qui porte l'identifiant demandé
	// est repris : c'est le cas normal d'une reprise (UC-003, A4), dont le déclencheur est un
	// processus mort dont la libération n'a pas tourné. Un campaignID vide ne reprend rien.
	AcquireLock(ctx context.Context, campaignID string) error
	// AdoptLock inscrit un identifiant dans le verrou déjà tenu. Le verrou est posé avant que
	// l'identifiant de la campagne soit dérivé (UC-003, étape 3) : il est nommé ensuite.
	AdoptLock(ctx context.Context, campaignID string) error
	ReleaseLock(ctx context.Context) error
	LockHeld(ctx context.Context) (bool, error)
}

// VerdictStore conserve les fichiers de verdicts par hypothèse (UC-005).
type VerdictStore interface {
	WriteVerdictReport(ctx context.Context, report models.VerdictReport) (string, error)
	// VerdictReports rend tous les rapports, du plus ancien au plus récent. Le tableau de bord en
	// a besoin : une campagne ne couvre que les hypothèses qu'elle a gelées, et deux hypothèses du
	// catalogue ne peuvent pas cohabiter dans une même campagne (C-009). Ne lire que le dernier
	// rapport effacerait donc du tableau les verdicts que les campagnes précédentes ont rendus.
	VerdictReports(ctx context.Context) ([]models.VerdictReport, error)
}

// HypothesisSource lit les hypothèses et leurs critères dans docs/requirements.md (BR-003-5).
type HypothesisSource interface {
	Load(ctx context.Context) ([]models.Hypothesis, error)
}

// UseCaseSource lit le statut déclaré dans chaque fichier de cas d'utilisation (FR-007).
type UseCaseSource interface {
	LoadUseCases(ctx context.Context) ([]UseCaseStatus, error)
}

// CodeIndex indique si du code et des tests d'intégration référencent un cas d'utilisation
// (colonnes Code et Integration du tableau de bord).
type CodeIndex interface {
	References(ctx context.Context, useCaseID string) (hasCode bool, hasIntegration bool, err error)
}

// UseCaseStatus est l'état d'un cas d'utilisation tel que déclaré dans son fichier. La présence
// de code et de tests d'intégration ne s'y trouve pas : elle est constatée dans les sources par
// CodeIndex, jamais déclarée dans la spécification.
type UseCaseStatus struct {
	ID       string
	Title    string
	Status   string
	LinkedFR []string
}

// TestReporter exécute les tests nommés d'après un cas d'utilisation (FR-007, étape 7).
type TestReporter interface {
	RunUseCaseTests(ctx context.Context, useCaseID string) (TestOutcome, error)
	RunAll(ctx context.Context) (TestOutcome, error)
}

// TestOutcome résume l'exécution d'une sélection de tests.
type TestOutcome struct {
	Selected int
	Passed   bool
	Output   string
}

// DashboardWriter régénère docs/dashboard.md (BR-005-3).
type DashboardWriter interface {
	Write(ctx context.Context, data DashboardData) (string, error)
}

// DashboardData porte tout ce qui alimente le tableau de bord.
type DashboardData struct {
	GeneratedAt time.Time
	UseCases    []DashboardUseCase
	Hypotheses  []DashboardHypothesis
}

// DashboardUseCase est une ligne du kanban des cas d'utilisation.
type DashboardUseCase struct {
	ID          string
	Title       string
	LinkedFR    []string
	Status      string
	Code        bool
	Unit        bool
	Integration string
	Regression  bool
	Integrity   string
}

// DashboardHypothesis est une ligne du tableau des hypothèses.
type DashboardHypothesis struct {
	ID          string
	SourcePages string
	UseCases    []string
	Frozen      bool
	CampaignID  string
	Outcome     string
}
