// Package specs lit les artefacts de docs/ qui font autorité : le catalogue d'hypothèses de
// docs/requirements.md (BR-003-5) et le statut déclaré dans chaque fichier de cas d'utilisation
// (FR-007). Il ne les modifie jamais.
package specs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

// Reader lit les artefacts de docs/.
type Reader struct {
	docsDir string
}

// NewReader construit un lecteur enraciné sur le répertoire docs/ du projet.
func NewReader(root string) *Reader { return &Reader{docsDir: filepath.Join(root, "docs")} }

// hypothesisRowRe capture une ligne du tableau des hypothèses de docs/requirements.md.
var hypothesisRowRe = regexp.MustCompile(`^\|\s*(H-\d{3})\s*\|`)

// Load lit les hypothèses, leur énoncé, leur critère de réfutation et leurs cas d'utilisation.
func (r *Reader) Load(_ context.Context) ([]models.Hypothesis, error) {
	path := filepath.Join(r.docsDir, "requirements.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de docs/requirements.md : %w", err)
	}
	var out []models.Hypothesis
	for _, line := range strings.Split(string(content), "\n") {
		if !hypothesisRowRe.MatchString(strings.TrimSpace(line)) {
			continue
		}
		cells := splitRow(line)
		if len(cells) < 5 {
			return nil, fmt.Errorf("ligne d'hypothèse incomplète dans docs/requirements.md : %q", strings.TrimSpace(line))
		}
		h := models.Hypothesis{
			ID:                  cells[0],
			SourcePages:         cells[1],
			Statement:           cells[2],
			RefutationCriterion: cells[3],
			UseCases:            splitList(cells[4]),
		}
		if err := h.Validate(); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune hypothèse trouvée dans %s", path)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// splitRow découpe une ligne de tableau Markdown en cellules, sans les barres extérieures.
func splitRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// splitList découpe une énumération séparée par des virgules.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" && trimmed != "—" && trimmed != "-" {
			out = append(out, trimmed)
		}
	}
	return out
}

// Digest calcule l'empreinte des énoncés et critères des hypothèses désignées (BR-003-5).
// L'ordre des identifiants n'influe pas sur le résultat.
func Digest(hypotheses []models.Hypothesis, ids []string) (string, error) {
	byID := make(map[string]models.Hypothesis, len(hypotheses))
	for _, h := range hypotheses {
		byID[h.ID] = h
	}
	selected := append([]string(nil), ids...)
	sort.Strings(selected)
	h := sha256.New()
	for _, id := range selected {
		hypothesis, ok := byID[id]
		if !ok {
			return "", fmt.Errorf("hypothèse %s absente de docs/requirements.md", id)
		}
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00", hypothesis.ID, hypothesis.Statement, hypothesis.RefutationCriterion)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChangedCriteria rend les identifiants dont l'énoncé ou le critère diffère entre deux catalogues
// (UC-005, A1 : afficher les hypothèses dont le critère a changé).
func ChangedCriteria(recorded, current []models.Hypothesis) []string {
	byID := make(map[string]models.Hypothesis, len(recorded))
	for _, h := range recorded {
		byID[h.ID] = h
	}
	var changed []string
	for _, h := range current {
		previous, ok := byID[h.ID]
		if !ok {
			continue
		}
		if previous.Statement != h.Statement || previous.RefutationCriterion != h.RefutationCriterion {
			changed = append(changed, h.ID)
		}
	}
	sort.Strings(changed)
	return changed
}

var (
	useCaseIDRe    = regexp.MustCompile(`\*\*Use Case ID:\*\*\s*(UC-\d{3})`)
	useCaseNameRe  = regexp.MustCompile(`\*\*Use Case Name:\*\*\s*(.+)`)
	useCaseStatRe  = regexp.MustCompile(`\*\*Status:\*\*\s*(\w+)`)
	useCaseLinksRe = regexp.MustCompile(`\*\*Linked Requirements:\*\*\s*(.+)`)
	// La limite de mot est indispensable : sans elle, `FR-\d{3}` capture aussi le suffixe de
	// `NFR-002` et le tableau de bord attribue au cas d'utilisation des exigences inexistantes.
	frRe        = regexp.MustCompile(`\bFR-\d{3}\b`)
	ucCommentRe = regexp.MustCompile(`\bUC-\d{3}\b`)
	ucTestRe    = regexp.MustCompile(`TestUC(\d{3})_`)
)

// LoadUseCases lit le statut déclaré dans chaque fichier docs/use-cases/UC-###-*.md (FR-007).
func (r *Reader) LoadUseCases(_ context.Context) ([]ports.UseCaseStatus, error) {
	paths, err := filepath.Glob(filepath.Join(r.docsDir, "use-cases", "UC-*.md"))
	if err != nil {
		return nil, fmt.Errorf("lecture de docs/use-cases : %w", err)
	}
	sort.Strings(paths)
	var out []ports.UseCaseStatus
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("lecture de %s : %w", filepath.Base(path), err)
		}
		text := string(content)
		status := ports.UseCaseStatus{}
		if match := useCaseIDRe.FindStringSubmatch(text); match != nil {
			status.ID = match[1]
		}
		if match := useCaseNameRe.FindStringSubmatch(text); match != nil {
			status.Title = strings.TrimSpace(match[1])
		}
		if match := useCaseStatRe.FindStringSubmatch(text); match != nil {
			status.Status = match[1]
		}
		if match := useCaseLinksRe.FindStringSubmatch(text); match != nil {
			status.LinkedFR = frRe.FindAllString(match[1], -1)
		}
		if status.ID == "" {
			return nil, fmt.Errorf("%s ne déclare pas de Use Case ID", filepath.Base(path))
		}
		out = append(out, status)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun cas d'utilisation dans docs/use-cases")
	}
	return out, nil
}
