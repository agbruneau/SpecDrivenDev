// Package store met en œuvre les dépôts sur système de fichiers : matrices, verdicts
// d'échappement, campagnes et mesures, comparaisons, verdicts d'hypothèses. L'écriture est
// strictement additive (NFR-004, BR-001-2, BR-002-3, BR-003-3, BR-004-3) ; les deux seules
// exceptions, nommées par BR-003-3, sont la transition de statut de campaign.json et le verrou
// de campagne.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// ErrImmutable signale une tentative de réécriture d'un fichier de résultats.
var ErrImmutable = errors.New("fichier de résultats immuable")

// ErrNotFound signale l'absence d'une matrice, d'une campagne ou d'un fichier attendu. C'est la
// sentinelle du port : le service doit pouvoir distinguer une absence légitime d'une lecture en
// erreur, et il n'importe pas cet adaptateur (A-123).
var ErrNotFound = ports.ErrNotFound

// ErrLockHeld signale qu'une autre campagne tient le verrou (BR-003-3).
var ErrLockHeld = ports.ErrLockHeld

// LockName est le nom du verrou de campagne, relatif à results/.
const LockName = ".campaign-lock"

// Store est le dépôt racine ; root contient matrices/ et results/.
type Store struct {
	root string
}

// New construit un Store enraciné sur le répertoire du projet.
func New(root string) *Store { return &Store{root: root} }

// Root rend la racine du dépôt.
func (s *Store) Root() string { return s.root }

// matricesDir rend le chemin absolu de matrices/.
func (s *Store) matricesDir() string { return filepath.Join(s.root, "matrices") }

// resultsDir rend le chemin absolu de results/.
func (s *Store) resultsDir() string { return filepath.Join(s.root, "results") }

// Dir rend le répertoire d'une Matrix.
func (s *Store) Dir(matrixID string) string { return filepath.Join(s.matricesDir(), matrixID) }

// Exists indique si une Matrix complète (avec son matrix.json) existe déjà (UC-001, A2).
func (s *Store) Exists(_ context.Context, matrixID string) (bool, error) {
	_, err := os.Stat(filepath.Join(s.Dir(matrixID), "matrix.json"))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("accès à la matrice %s : %w", matrixID, err)
	}
}

// Load lit une Matrix.
func (s *Store) Load(_ context.Context, matrixID string) (models.Matrix, error) {
	var dto matrixDTO
	if err := readJSON(filepath.Join(s.Dir(matrixID), "matrix.json"), &dto); err != nil {
		return models.Matrix{}, fmt.Errorf("lecture de la matrice %s : %w", matrixID, err)
	}
	return dto.toModel(), nil
}

// List rend les identifiants des matrices présentes, triés.
func (s *Store) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.matricesDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture de matrices/ : %w", err)
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.matricesDir(), entry.Name(), "matrix.json")); err == nil {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// WriteSources écrit les fichiers générés d'une matrice, avant compilation (UC-001, étapes 4-5).
// Une matrice déjà finalisée est immuable : le dépôt refuse d'écrire par-dessus (BR-001-2).
func (s *Store) WriteSources(_ context.Context, matrixID string, files map[string]string) error {
	dir := s.Dir(matrixID)
	if present, err := exists(filepath.Join(dir, "matrix.json")); err != nil {
		return err
	} else if present {
		return fmt.Errorf("%w : matrices/%s existe déjà (BR-001-2)", ErrImmutable, matrixID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("création de matrices/%s : %w", matrixID, err)
	}
	for _, rel := range sortedKeys(files) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("création de %s : %w", rel, err)
		}
		if err := writeFileAtomic(full, []byte(files[rel])); err != nil {
			return err
		}
	}
	return nil
}

// Finalize écrit matrix.json ; la matrice devient immuable (UC-001, étape 6 ; BR-001-2).
func (s *Store) Finalize(_ context.Context, matrix models.Matrix) error {
	if err := matrix.Validate(); err != nil {
		return err
	}
	manifest := filepath.Join(s.Dir(matrix.ID), "matrix.json")
	if present, err := exists(manifest); err != nil {
		return err
	} else if present {
		return fmt.Errorf("%w : matrices/%s existe déjà (BR-001-2)", ErrImmutable, matrix.ID)
	}
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		return fmt.Errorf("création de matrices/%s : %w", matrix.ID, err)
	}
	return writeJSONExclusive(manifest, toMatrixDTO(matrix))
}

// Remove supprime le répertoire d'une matrice incomplète (UC-001, A3 : aucun répertoire partiel
// ne subsiste). Elle ne s'applique jamais à une matrice déjà écrite avec son matrix.json.
func (s *Store) Remove(ctx context.Context, matrixID string) error {
	complete, err := s.Exists(ctx, matrixID)
	if err != nil {
		return err
	}
	if complete {
		return fmt.Errorf("%w : matrices/%s est complète (BR-001-2)", ErrImmutable, matrixID)
	}
	if err := os.RemoveAll(s.Dir(matrixID)); err != nil {
		return fmt.Errorf("suppression de matrices/%s : %w", matrixID, err)
	}
	return nil
}

// escapeDir rend le répertoire des verdicts d'échappement d'une Matrix.
func (s *Store) escapeDir(matrixID string) string {
	return filepath.Join(s.resultsDir(), "escape", matrixID)
}

// WriteEscapeReport écrit un fichier de verdicts horodaté (UC-002, étape 6 ; BR-002-3).
func (s *Store) WriteEscapeReport(_ context.Context, report models.EscapeReport, capturedAt time.Time) (string, error) {
	if err := report.Provenance.Validate(); err != nil {
		return "", err
	}
	dir := s.escapeDir(report.MatrixID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("création de results/escape/%s : %w", report.MatrixID, err)
	}
	path := filepath.Join(dir, Stamp(capturedAt)+".json")
	if present, err := exists(path); err != nil {
		return "", err
	} else if present {
		// A-052 : l'horodatage est à la seconde. Deux écritures rapprochées tombent sur le même
		// nom, ce qui n'est pas une réécriture interdite mais une collision de nom.
		return "", fmt.Errorf("%w : %s existe déjà (BR-002-3) ; l'horodatage est à la seconde, réessayer à la suivante",
			ErrImmutable, path)
	}
	if err := writeJSONExclusive(path, toEscapeReportDTO(report)); err != nil {
		return "", err
	}
	return s.rel(path), nil
}

// ListEscapeReports rend les chemins des fichiers de verdicts d'une Matrix, du plus ancien au plus
// récent.
func (s *Store) ListEscapeReports(_ context.Context, matrixID string) ([]string, error) {
	entries, err := os.ReadDir(s.escapeDir(matrixID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture de results/escape/%s : %w", matrixID, err)
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			paths = append(paths, s.rel(filepath.Join(s.escapeDir(matrixID), entry.Name())))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// ReadEscapeReport lit un fichier de verdicts désigné par un chemin relatif à la racine.
func (s *Store) ReadEscapeReport(_ context.Context, path string) (models.EscapeReport, error) {
	var dto escapeReportDTO
	if err := readJSON(s.abs(path), &dto); err != nil {
		return models.EscapeReport{}, fmt.Errorf("lecture de %s : %w", path, err)
	}
	return dto.toModel(), nil
}

// campaignDir rend le répertoire d'une campagne.
func (s *Store) campaignDir(campaignID string) string {
	return filepath.Join(s.resultsDir(), "campaigns", campaignID)
}

// CampaignPath rend le chemin, relatif à la racine, du campaign.json d'une campagne.
func (s *Store) CampaignPath(campaignID string) string {
	return s.rel(filepath.Join(s.campaignDir(campaignID), "campaign.json"))
}

// MeasurementPath rend le chemin, relatif à la racine, de la mesure d'un sujet.
func (s *Store) MeasurementPath(campaignID, subjectID string) string {
	return s.rel(filepath.Join(s.campaignDir(campaignID), "measurements", models.SubjectDir(subjectID)+".json"))
}

// NextCampaignID rend un identifiant nouveau de la forme C-<date>-<n> pour le jour donné.
func (s *Store) NextCampaignID(_ context.Context, day time.Time) (string, error) {
	prefix := "C-" + day.UTC().Format("2006-01-02") + "-"
	entries, err := os.ReadDir(filepath.Join(s.resultsDir(), "campaigns"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("lecture de results/campaigns : %w", err)
	}
	next := 1
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), prefix)); err == nil && n >= next {
			next = n + 1
		}
	}
	return prefix + strconv.Itoa(next), nil
}

// CreateCampaign écrit le campaign.json initial d'une campagne (UC-003, étape 3).
func (s *Store) CreateCampaign(_ context.Context, campaign models.Campaign) error {
	if err := campaign.Validate(); err != nil {
		return err
	}
	path := filepath.Join(s.campaignDir(campaign.ID), "campaign.json")
	if present, err := exists(path); err != nil {
		return err
	} else if present {
		return fmt.Errorf("%w : la campagne %s existe déjà", ErrImmutable, campaign.ID)
	}
	if err := os.MkdirAll(filepath.Join(s.campaignDir(campaign.ID), "measurements"), 0o755); err != nil {
		return fmt.Errorf("création de results/campaigns/%s : %w", campaign.ID, err)
	}
	return writeJSONExclusive(path, toCampaignDTO(campaign))
}

// LoadCampaign lit une campagne.
func (s *Store) LoadCampaign(_ context.Context, campaignID string) (models.Campaign, error) {
	var dto campaignDTO
	path := filepath.Join(s.campaignDir(campaignID), "campaign.json")
	if err := readJSON(path, &dto); err != nil {
		return models.Campaign{}, fmt.Errorf("lecture de la campagne %s : %w", campaignID, err)
	}
	return dto.toModel(), nil
}

// SetCampaignStatus fait passer le statut d'une campagne. C'est l'une des deux seules écritures
// non additives autorisées par BR-003-3 ; aucun autre champ n'est modifié.
//
// Révision du 2026-09-12 (A-046) : BR-003-3 n'admet qu'une transition, `RUNNING` vers `COMPLETED`
// ou `ABORTED`. Rien ne l'appliquait : une campagne close pouvait être rouverte ou réécrite, ce
// qui aurait fait passer pour valide une campagne déjà abandonnée.
func (s *Store) SetCampaignStatus(ctx context.Context, campaignID string, status models.CampaignStatus, finishedAt time.Time, abortReason string) error {
	campaign, err := s.LoadCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	if campaign.Status != models.CampaignRunning {
		return fmt.Errorf("%w : la campagne %s est au statut %s ; BR-003-3 n'admet que RUNNING vers COMPLETED ou ABORTED",
			ErrImmutable, campaignID, campaign.Status)
	}
	if status != models.CampaignCompleted && status != models.CampaignAborted {
		return fmt.Errorf("%w : transition de %s vers %s refusée (BR-003-3)",
			ErrImmutable, campaign.Status, status)
	}
	campaign.Status = status
	campaign.FinishedAt = finishedAt
	campaign.AbortReason = abortReason
	return writeJSONAtomic(filepath.Join(s.campaignDir(campaignID), "campaign.json"), toCampaignDTO(campaign))
}

// WriteMeasurement écrit la mesure d'un sujet dès qu'elle est complète (UC-003, étape 6).
func (s *Store) WriteMeasurement(_ context.Context, m models.Measurement) error {
	// A-054 : sans ces contrôles, une Measurement sans identifiant écrivait sous
	// `results/campaigns//measurements/.json`, et une campagne inexistante se voyait créer un
	// répertoire de mesures sans campaign.json.
	if err := s.requireCampaign(m.CampaignID, "Measurement"); err != nil {
		return err
	}
	if m.SubjectID == "" {
		return fmt.Errorf("Measurement.subjectId est requis")
	}
	path := s.abs(s.MeasurementPath(m.CampaignID, m.SubjectID))
	if present, err := exists(path); err != nil {
		return err
	} else if present {
		return fmt.Errorf("%w : %s existe déjà (BR-003-3)", ErrImmutable, s.rel(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("création du répertoire des mesures : %w", err)
	}
	return writeJSONExclusive(path, toMeasurementDTO(m))
}

// LoadMeasurements lit toutes les mesures d'une campagne, triées par identifiant de sujet.
func (s *Store) LoadMeasurements(_ context.Context, campaignID string) ([]models.Measurement, error) {
	dir := filepath.Join(s.campaignDir(campaignID), "measurements")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture des mesures de %s : %w", campaignID, err)
	}
	var out []models.Measurement
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var dto measurementDTO
		if err := readJSON(filepath.Join(dir, entry.Name()), &dto); err != nil {
			return nil, fmt.Errorf("lecture de %s : %w", entry.Name(), err)
		}
		out = append(out, dto.toModel())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubjectID < out[j].SubjectID })
	return out, nil
}

// WriteComparisonSet écrit un fichier de comparaison horodaté (UC-004, étape 6 ; BR-004-3).
func (s *Store) WriteComparisonSet(_ context.Context, set models.ComparisonSet) (string, error) {
	if err := s.requireCampaign(set.CampaignID, "ComparisonSet"); err != nil {
		return "", err
	}
	path := filepath.Join(s.campaignDir(set.CampaignID), "comparison-"+Stamp(set.ComputedAt)+".json")
	if present, err := exists(path); err != nil {
		return "", err
	} else if present {
		return "", fmt.Errorf("%w : %s existe déjà (BR-004-3) ; l'horodatage est à la seconde, réessayer à la suivante",
			ErrImmutable, s.rel(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("création du répertoire de la campagne : %w", err)
	}
	if err := writeJSONExclusive(path, toComparisonSetDTO(set)); err != nil {
		return "", err
	}
	return s.rel(path), nil
}

// LatestComparisonSet rend le fichier de comparaison le plus récent d'une campagne.
func (s *Store) LatestComparisonSet(_ context.Context, campaignID string) (models.ComparisonSet, string, error) {
	entries, err := os.ReadDir(s.campaignDir(campaignID))
	if errors.Is(err, os.ErrNotExist) {
		return models.ComparisonSet{}, "", fmt.Errorf("%w : campagne %s", ErrNotFound, campaignID)
	}
	if err != nil {
		// A-124 : toute erreur de lecture devenait « introuvable », donc un verdict non concluant
		// « exécuter compare » là où le fichier existe mais n'est pas lisible.
		return models.ComparisonSet{}, "", fmt.Errorf("lecture du répertoire de la campagne %s : %w", campaignID, err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "comparison-") && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return models.ComparisonSet{}, "", fmt.Errorf("%w : aucun fichier de comparaison pour %s", ErrNotFound, campaignID)
	}
	sort.Strings(names)
	path := filepath.Join(s.campaignDir(campaignID), names[len(names)-1])
	var dto comparisonSetDTO
	if err := readJSON(path, &dto); err != nil {
		return models.ComparisonSet{}, "", fmt.Errorf("lecture de %s : %w", path, err)
	}
	return dto.toModel(), s.rel(path), nil
}

// AcquireLock pose results/.campaign-lock (BR-003-3, seconde exception).
//
// Un verrou déjà présent qui porte exactement l'identifiant demandé est repris plutôt que refusé.
// C'est le cas normal de la reprise (UC-003, A4), dont le déclencheur est précisément un processus
// mort : son defer de libération n'a pas tourné, et le verrou orphelin interdisait la reprise de
// la campagne qu'il protège, sans qu'aucune sous-commande sache le retirer (A-044). Un
// campaignID vide ne reprend rien : il sert au démarrage, avant que l'identifiant soit dérivé.
func (s *Store) AcquireLock(_ context.Context, campaignID string) error {
	if err := os.MkdirAll(s.resultsDir(), 0o755); err != nil {
		return fmt.Errorf("création de results/ : %w", err)
	}
	path := filepath.Join(s.resultsDir(), LockName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		holder, readErr := s.lockHolder()
		switch {
		case readErr != nil:
			return fmt.Errorf("%w (results/%s), contenu illisible : %w", ErrLockHeld, LockName, readErr)
		case campaignID != "" && holder == campaignID:
			return nil
		case holder == "":
			return fmt.Errorf("%w (results/%s), sans identifiant de campagne", ErrLockHeld, LockName)
		default:
			return fmt.Errorf("%w : results/%s est tenu par %s", ErrLockHeld, LockName, holder)
		}
	}
	if err != nil {
		return fmt.Errorf("pose du verrou de campagne : %w", err)
	}
	if err := writeLock(file, campaignID); err != nil {
		// Un verrou vide subsisterait et bloquerait UC-001 comme UC-003 sans que rien ne dise
		// pourquoi (A-126).
		_ = os.Remove(path)
		return err
	}
	return nil
}

// AdoptLock inscrit un identifiant dans le verrou déjà tenu par ce processus. Le verrou est posé
// avant que l'identifiant de la campagne soit dérivé (UC-003, étape 3) ; c'est cette écriture qui
// le rend reconnaissable par une reprise ultérieure.
func (s *Store) AdoptLock(_ context.Context, campaignID string) error {
	path := filepath.Join(s.resultsDir(), LockName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("nommage du verrou de campagne : %w", err)
	}
	return writeLock(file, campaignID)
}

// writeLock écrit l'identifiant dans le verrou et ferme le fichier.
func writeLock(file *os.File, campaignID string) error {
	_, err := file.WriteString(campaignID + "\n")
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("écriture du verrou de campagne : %w", err)
	}
	return nil
}

// lockHolder rend l'identifiant de campagne inscrit dans le verrou, vide s'il n'en porte pas.
func (s *Store) lockHolder() (string, error) {
	content, err := os.ReadFile(filepath.Join(s.resultsDir(), LockName))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// ReleaseLock retire results/.campaign-lock.
func (s *Store) ReleaseLock(_ context.Context) error {
	err := os.Remove(filepath.Join(s.resultsDir(), LockName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retrait du verrou de campagne : %w", err)
	}
	return nil
}

// LockHeld indique si une campagne est en cours.
func (s *Store) LockHeld(_ context.Context) (bool, error) {
	_, err := os.Stat(filepath.Join(s.resultsDir(), LockName))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("lecture du verrou de campagne : %w", err)
	}
}

// WriteVerdictReport écrit un fichier de verdicts horodaté (UC-005, étape 6).
func (s *Store) WriteVerdictReport(_ context.Context, report models.VerdictReport) (string, error) {
	dir := filepath.Join(s.resultsDir(), "verdicts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("création de results/verdicts : %w", err)
	}
	path := filepath.Join(dir, report.CampaignID+"-"+Stamp(report.ProducedAt)+".json")
	if present, err := exists(path); err != nil {
		return "", err
	} else if present {
		return "", fmt.Errorf("%w : %s existe déjà (NFR-004) ; l'horodatage est à la seconde, réessayer à la suivante",
			ErrImmutable, s.rel(path))
	}
	if err := writeJSONExclusive(path, toVerdictReportDTO(report)); err != nil {
		return "", err
	}
	return s.rel(path), nil
}

// requireCampaign exige un identifiant de campagne non vide, désignant une campagne existante.
func (s *Store) requireCampaign(campaignID, what string) error {
	if campaignID == "" {
		return fmt.Errorf("%s.campaignId est requis", what)
	}
	present, err := exists(filepath.Join(s.campaignDir(campaignID), "campaign.json"))
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("%w : campagne %s", ErrNotFound, campaignID)
	}
	return nil
}

// VerdictReports rend tous les rapports de verdicts, du plus ancien au plus récent (UC-005).
func (s *Store) VerdictReports(_ context.Context) ([]models.VerdictReport, error) {
	dir := filepath.Join(s.resultsDir(), "verdicts")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture de results/verdicts : %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	// A-048 : trier sur le seul horodatage laissait l'ordre de deux rapports de la même seconde
	// à sort.Slice, qui n'est pas stable. Le tableau de bord retient le dernier verdict rendu sur
	// une hypothèse : deux rapports de même seconde pouvaient donner deux tableaux différents.
	sort.Slice(names, func(i, j int) bool {
		if a, b := stampOf(names[i]), stampOf(names[j]); a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	out := make([]models.VerdictReport, 0, len(names))
	for _, name := range names {
		var dto verdictReportDTO
		if err := readJSON(filepath.Join(dir, name), &dto); err != nil {
			return nil, fmt.Errorf("lecture de %s : %w", name, err)
		}
		out = append(out, dto.toModel())
	}
	return out, nil
}

// LatestVerdictReport rend le fichier de verdicts le plus récent, toutes campagnes confondues.
func (s *Store) LatestVerdictReport(_ context.Context) (models.VerdictReport, string, error) {
	dir := filepath.Join(s.resultsDir(), "verdicts")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return models.VerdictReport{}, "", fmt.Errorf("%w : aucun verdict", ErrNotFound)
	}
	if err != nil {
		return models.VerdictReport{}, "", fmt.Errorf("lecture de results/verdicts : %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return models.VerdictReport{}, "", fmt.Errorf("%w : aucun verdict", ErrNotFound)
	}
	// A-048 : trier sur le seul horodatage laissait l'ordre de deux rapports de la même seconde
	// à sort.Slice, qui n'est pas stable. Le tableau de bord retient le dernier verdict rendu sur
	// une hypothèse : deux rapports de même seconde pouvaient donner deux tableaux différents.
	sort.Slice(names, func(i, j int) bool {
		if a, b := stampOf(names[i]), stampOf(names[j]); a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	path := filepath.Join(dir, names[len(names)-1])
	var dto verdictReportDTO
	if err := readJSON(path, &dto); err != nil {
		return models.VerdictReport{}, "", fmt.Errorf("lecture de %s : %w", path, err)
	}
	return dto.toModel(), s.rel(path), nil
}

// stampOf extrait l'horodatage d'un nom de fichier de verdicts `<campaignId>-<stamp>.json`.
func stampOf(name string) string {
	base := strings.TrimSuffix(name, ".json")
	if idx := strings.LastIndex(base, "-"); idx >= 0 {
		return base[idx+1:]
	}
	return base
}

// Stamp rend l'horodatage utilisé dans les noms de fichiers de résultats.
func Stamp(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

// rel rend un chemin relatif à la racine, en séparateurs POSIX, pour la traçabilité (BR-005-2).
func (s *Store) rel(path string) string {
	relative, err := filepath.Rel(s.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

// abs rend le chemin absolu d'un chemin relatif à la racine.
func (s *Store) abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.root, filepath.FromSlash(path))
}

// sortedKeys rend les clés d'une table, triées.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sortedTippingKeys rend les clés de points de bascule dans un ordre stable.
func sortedTippingKeys(m map[models.TippingKey]int) []models.TippingKey {
	keys := make([]models.TippingKey, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Profile != keys[j].Profile {
			return keys[i].Profile < keys[j].Profile
		}
		if keys[i].Layout != keys[j].Layout {
			return keys[i].Layout < keys[j].Layout
		}
		return !keys[i].HasPointerField && keys[j].HasPointerField
	})
	return keys
}

// readJSON lit et décode un fichier JSON.
func readJSON(path string, target any) error {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w : %s", ErrNotFound, path)
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(content, target)
}

// exists indique si un chemin existe.
//
// Révision du 2026-09-12 (A-053) : les gardes d'immutabilité faisaient `if _, err := os.Stat(p);
// err == nil`, traitant donc toute erreur d'accès autre que « absent » — permission refusée,
// chemin dont un segment n'est pas un répertoire, erreur d'entrée-sortie — comme une absence, et
// laissant l'écriture se poursuivre. L'erreur est désormais rendue.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("accès à %s : %w", path, err)
	}
}

// writeJSONExclusive encode et écrit un fichier qui ne doit jamais en écraser un autre.
func writeJSONExclusive(path string, value any) error {
	content, err := encodeJSON(path, value)
	if err != nil {
		return err
	}
	return writeFileExclusive(path, content)
}

// writeJSONAtomic encode et écrit un fichier JSON de façon atomique.
func writeJSONAtomic(path string, value any) error {
	content, err := encodeJSON(path, value)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, content)
}

// encodeJSON encode une valeur, terminée par un saut de ligne.
func encodeJSON(path string, value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encodage de %s : %w", path, err)
	}
	return append(content, '\n'), nil
}

// writeFileAtomic écrit par fichier temporaire puis renommage : aucun fichier partiel ne subsiste
// si l'écriture échoue (postconditions d'échec de UC-001 à UC-005).
//
// os.Rename écrase une cible existante, sur Linux comme sur Windows. Cette fonction est donc
// réservée aux deux écritures que BR-003-3 autorise à remplacer un fichier : la transition de
// statut de campaign.json et le verrou de campagne. Toute création immuable passe par
// writeFileExclusive.
func writeFileAtomic(path string, content []byte) error {
	tmpName, err := writeTemp(path, content)
	if err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renommage vers %s : %w", path, err)
	}
	return nil
}

// writeFileExclusive crée un fichier qui ne doit jamais en écraser un autre.
//
// Révision du 2026-09-12 (A-043) : toutes les gardes d'immutabilité faisaient un os.Stat puis,
// séparément, un os.Rename qui remplace la cible sans rien dire. L'immutabilité que NFR-004,
// BR-001-2, BR-002-3, BR-003-3 et BR-004-3 exigent n'était donc garantie que pour un seul
// processus : deux runners concurrents pouvaient écraser un fichier de résultats. Le lien dur
// échoue avec os.ErrExist quand la cible existe, ce qui ferme la fenêtre au niveau du système de
// fichiers plutôt qu'au niveau du contrôle préalable.
func writeFileExclusive(path string, content []byte) error {
	tmpName, err := writeTemp(path, content)
	if err != nil {
		return err
	}
	defer os.Remove(tmpName)
	linkErr := os.Link(tmpName, path)
	if linkErr == nil {
		return nil
	}
	if errors.Is(linkErr, os.ErrExist) {
		return fmt.Errorf("%w : %s existe déjà", ErrImmutable, path)
	}
	// Repli pour un système de fichiers sans lien dur (FAT, exFAT, certains montages réseau) : la
	// garde redevient contrôle-puis-renommage, donc valable pour un seul processus.
	present, statErr := exists(path)
	if statErr != nil {
		return statErr
	}
	if present {
		return fmt.Errorf("%w : %s existe déjà", ErrImmutable, path)
	}
	if renameErr := os.Rename(tmpName, path); renameErr != nil {
		return fmt.Errorf("écriture exclusive de %s : %w", path, errors.Join(linkErr, renameErr))
	}
	return nil
}

// writeTemp écrit le contenu dans un fichier temporaire voisin de la cible et rend son nom.
//
// Révision du 2026-09-12 (A-050) : sans synchronisation avant le renommage, une coupure de courant
// pouvait laisser à sa place un fichier de résultats de longueur nulle — les métadonnées du
// renommage atteignant le disque avant les données.
func writeTemp(path string, content []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return "", fmt.Errorf("création du fichier temporaire pour %s : %w", path, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("écriture de %s : %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("synchronisation de %s : %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("fermeture de %s : %w", path, err)
	}
	return tmpName, nil
}
