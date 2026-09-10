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
)

// ErrImmutable signale une tentative de réécriture d'un fichier de résultats.
var ErrImmutable = errors.New("fichier de résultats immuable")

// ErrNotFound signale l'absence d'une matrice, d'une campagne ou d'un fichier attendu.
var ErrNotFound = errors.New("introuvable")

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
	if _, err := os.Stat(filepath.Join(dir, "matrix.json")); err == nil {
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
	if _, err := os.Stat(manifest); err == nil {
		return fmt.Errorf("%w : matrices/%s existe déjà (BR-001-2)", ErrImmutable, matrix.ID)
	}
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		return fmt.Errorf("création de matrices/%s : %w", matrix.ID, err)
	}
	return writeJSONAtomic(manifest, toMatrixDTO(matrix))
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
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%w : %s existe déjà (BR-002-3)", ErrImmutable, path)
	}
	if err := writeJSONAtomic(path, toEscapeReportDTO(report)); err != nil {
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
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w : la campagne %s existe déjà", ErrImmutable, campaign.ID)
	}
	if err := os.MkdirAll(filepath.Join(s.campaignDir(campaign.ID), "measurements"), 0o755); err != nil {
		return fmt.Errorf("création de results/campaigns/%s : %w", campaign.ID, err)
	}
	return writeJSONAtomic(path, toCampaignDTO(campaign))
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
func (s *Store) SetCampaignStatus(ctx context.Context, campaignID string, status models.CampaignStatus, finishedAt time.Time, abortReason string) error {
	campaign, err := s.LoadCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	campaign.Status = status
	campaign.FinishedAt = finishedAt
	campaign.AbortReason = abortReason
	return writeJSONAtomic(filepath.Join(s.campaignDir(campaignID), "campaign.json"), toCampaignDTO(campaign))
}

// WriteMeasurement écrit la mesure d'un sujet dès qu'elle est complète (UC-003, étape 6).
func (s *Store) WriteMeasurement(_ context.Context, m models.Measurement) error {
	path := s.abs(s.MeasurementPath(m.CampaignID, m.SubjectID))
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w : %s existe déjà (BR-003-3)", ErrImmutable, s.rel(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("création du répertoire des mesures : %w", err)
	}
	return writeJSONAtomic(path, toMeasurementDTO(m))
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
	path := filepath.Join(s.campaignDir(set.CampaignID), "comparison-"+Stamp(set.ComputedAt)+".json")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%w : %s existe déjà (BR-004-3)", ErrImmutable, s.rel(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("création du répertoire de la campagne : %w", err)
	}
	if err := writeJSONAtomic(path, toComparisonSetDTO(set)); err != nil {
		return "", err
	}
	return s.rel(path), nil
}

// LatestComparisonSet rend le fichier de comparaison le plus récent d'une campagne.
func (s *Store) LatestComparisonSet(_ context.Context, campaignID string) (models.ComparisonSet, string, error) {
	entries, err := os.ReadDir(s.campaignDir(campaignID))
	if err != nil {
		return models.ComparisonSet{}, "", fmt.Errorf("%w : campagne %s", ErrNotFound, campaignID)
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
func (s *Store) AcquireLock(_ context.Context, campaignID string) error {
	if err := os.MkdirAll(s.resultsDir(), 0o755); err != nil {
		return fmt.Errorf("création de results/ : %w", err)
	}
	path := filepath.Join(s.resultsDir(), LockName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("une campagne est déjà en cours (results/%s)", LockName)
	}
	if err != nil {
		return fmt.Errorf("pose du verrou de campagne : %w", err)
	}
	defer file.Close()
	_, err = file.WriteString(campaignID + "\n")
	return err
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
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%w : %s existe déjà (NFR-004)", ErrImmutable, s.rel(path))
	}
	if err := writeJSONAtomic(path, toVerdictReportDTO(report)); err != nil {
		return "", err
	}
	return s.rel(path), nil
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
	sort.Slice(names, func(i, j int) bool { return stampOf(names[i]) < stampOf(names[j]) })
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

// writeJSONAtomic encode et écrit un fichier JSON de façon atomique.
func writeJSONAtomic(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage de %s : %w", path, err)
	}
	return writeFileAtomic(path, append(content, '\n'))
}

// writeFileAtomic écrit par fichier temporaire puis renommage : aucun fichier partiel ne subsiste
// si l'écriture échoue (postconditions d'échec de UC-001 à UC-005).
func writeFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("création du fichier temporaire pour %s : %w", path, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("écriture de %s : %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("fermeture de %s : %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renommage vers %s : %w", path, err)
	}
	return nil
}
