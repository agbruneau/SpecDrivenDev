// Package dashboard régénère docs/dashboard.md à partir des verdicts, du statut des cas
// d'utilisation et du résultat des tests (FR-007, BR-005-3). Le fichier n'est jamais édité à la
// main : ce paquet est le seul à l'écrire.
package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agbruneau/escapebench/internal/ports"
)

// Writer écrit docs/dashboard.md.
type Writer struct {
	path string
	root string
}

// NewWriter construit un rédacteur enraciné sur le répertoire du projet.
func NewWriter(root string) *Writer {
	return &Writer{path: filepath.Join(root, "docs", "dashboard.md"), root: root}
}

// Path rend le chemin, relatif à la racine, du tableau de bord.
func (w *Writer) Path() string { return "docs/dashboard.md" }

// Write régénère le tableau de bord et rend son chemin relatif.
func (w *Writer) Write(_ context.Context, data ports.DashboardData) (string, error) {
	content := Render(data)
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return "", fmt.Errorf("création de docs/ : %w", err)
	}
	if err := os.WriteFile(w.path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("écriture de docs/dashboard.md : %w", err)
	}
	return w.Path(), nil
}

// mark rend la marque de colonne d'un booléen.
func mark(ok bool) string {
	if ok {
		return "✔"
	}
	return "✕"
}

// dash rend un tiret cadratin quand la valeur est vide.
func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// Render produit le contenu Markdown du tableau de bord.
func Render(data ports.DashboardData) string {
	var b strings.Builder
	b.WriteString("# Tableau de bord — EscapeBench\n\n")
	fmt.Fprintf(&b, "Généré par `escapebench dashboard` et `escapebench verdict` (UC-005, FR-007) ; ne pas éditer à la main (BR-005-3). Régénéré le %s.\n\n",
		data.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))

	b.WriteString("## Cas d'utilisation\n\n")
	b.WriteString("| Use case | Linked FR | UC status | Code | Unit | Integration | Regression | Integrity |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, uc := range data.UseCases {
		fmt.Fprintf(&b, "| %s %s | %s | %s | %s | %s | %s | %s | %s |\n",
			uc.ID, uc.Title, dash(strings.Join(uc.LinkedFR, ", ")), dash(uc.Status),
			mark(uc.Code), mark(uc.Unit), dash(uc.Integration), mark(uc.Regression), uc.Integrity)
	}
	b.WriteString("\nStatuts (SDD, p. 140, colonne `Review` écrite `Reviewed` dans les fichiers de UC) : ")
	b.WriteString("Draft → Reviewed → Approved → Implemented → Verified → Deployed (= campagne exécutée et rapport publié). ")
	b.WriteString("Le passage `Reviewed` → `Approved` est une décision humaine consignée dans le fichier du UC.\n\n")

	b.WriteString("## Hypothèses\n\n")
	b.WriteString("| Hypothèse | Source BEPG | UC liés | Critère gelé (au statut `Approved`) | Campagne | Verdict |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, h := range data.Hypotheses {
		frozen := "Non"
		if h.Frozen {
			frozen = "Oui"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			h.ID, dash(h.SourcePages), dash(strings.Join(h.UseCases, ", ")), frozen, dash(h.CampaignID), dash(h.Outcome))
	}

	b.WriteString("\n## Synthèse\n\n")
	b.WriteString("| Indicateur | Total | Complet | En cours | Non démarré | Couverture |\n")
	b.WriteString("|---|---|---|---|---|---|\n")

	frTotal, frDone := functionalRequirements(data)
	writeSummaryRow(&b, "Exigences fonctionnelles", frTotal, frDone, frTotal-frDone, 0)

	ucTotal := len(data.UseCases)
	ucDone, ucStarted := 0, 0
	for _, uc := range data.UseCases {
		switch {
		case uc.Integrity == "Strong":
			ucDone++
		case uc.Code:
			ucStarted++
		}
	}
	writeSummaryRow(&b, "Cas d'utilisation", ucTotal, ucDone, ucStarted, ucTotal-ucDone-ucStarted)

	hTotal := len(data.Hypotheses)
	hDone := 0
	for _, h := range data.Hypotheses {
		if h.Outcome != "" {
			hDone++
		}
	}
	writeSummaryRow(&b, "Hypothèses avec verdict", hTotal, hDone, 0, hTotal-hDone)
	return b.String()
}

// functionalRequirements compte les FR distincts et ceux dont tous les UC porteurs sont Strong.
func functionalRequirements(data ports.DashboardData) (total, done int) {
	strong := make(map[string]bool)
	weak := make(map[string]bool)
	for _, uc := range data.UseCases {
		for _, fr := range uc.LinkedFR {
			if uc.Integrity == "Strong" {
				strong[fr] = true
			} else {
				weak[fr] = true
			}
		}
	}
	all := make(map[string]bool, len(strong)+len(weak))
	for fr := range strong {
		all[fr] = true
	}
	for fr := range weak {
		all[fr] = true
	}
	for fr := range all {
		if strong[fr] && !weak[fr] {
			done++
		}
	}
	return len(all), done
}

// writeSummaryRow écrit une ligne de synthèse avec sa couverture en pourcentage.
func writeSummaryRow(b *strings.Builder, label string, total, done, started, notStarted int) {
	coverage := 0
	if total > 0 {
		coverage = done * 100 / total
	}
	fmt.Fprintf(b, "| %s | %d | %d | %d | %d | %d %% |\n", label, total, done, started, notStarted, coverage)
}
