package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/ports"
)

func sampleData() ports.DashboardData {
	return ports.DashboardData{
		GeneratedAt: time.Date(2026, 9, 10, 14, 5, 0, 0, time.UTC),
		UseCases: []ports.DashboardUseCase{
			{ID: "UC-001", Title: "Générer la matrice", LinkedFR: []string{"FR-001"}, Status: "Implemented",
				Code: true, Unit: true, Integration: "✔", Regression: true, Integrity: "Strong"},
			{ID: "UC-002", Title: "Classer l'échappement", LinkedFR: []string{"FR-002"}, Status: "Approved",
				Code: true, Unit: false, Integration: "—", Regression: true, Integrity: "Partial"},
			{ID: "UC-003", Title: "Exécuter une campagne", LinkedFR: []string{"FR-003", "FR-006"}, Status: "Reviewed",
				Code: false, Unit: false, Integration: "—", Regression: true, Integrity: "Weak"},
		},
		Hypotheses: []ports.DashboardHypothesis{
			{ID: "H-001", SourcePages: "p. 245", UseCases: []string{"UC-003"}, Frozen: true, CampaignID: "C-1", Outcome: "REFUTED"},
			{ID: "H-002", SourcePages: "p. 253", UseCases: nil, Frozen: false},
		},
	}
}

func TestRenderContenu(t *testing.T) {
	t.Parallel()
	content := Render(sampleData())
	for _, needle := range []string{
		"# Tableau de bord — EscapeBench",
		"2026-09-10 14:05 UTC",
		"| UC-001 Générer la matrice | FR-001 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |",
		"| UC-003 Exécuter une campagne | FR-003, FR-006 | Reviewed | ✕ | ✕ | — | ✔ | Weak |",
		"| H-001 | p. 245 | UC-003 | Oui | C-1 | REFUTED |",
		"| H-002 | p. 253 | — | Non | — | — |",
		"ne pas éditer à la main (BR-005-3)",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%q absent du tableau de bord :\n%s", needle, content)
		}
	}
}

func TestRenderSynthese(t *testing.T) {
	t.Parallel()
	content := Render(sampleData())
	// FR-001 n'est porté que par UC-001, qui est Strong : l'exigence est complète.
	// FR-002, FR-003 et FR-006 ne le sont pas. Mutation : compter un FR porté par un UC
	// non-Strong comme complet ⇒ échec attendu.
	if !strings.Contains(content, "| Exigences fonctionnelles | 4 | 1 | 3 | 0 | 25 % |") {
		t.Fatalf("synthèse des exigences inattendue :\n%s", content)
	}
	if !strings.Contains(content, "| Cas d'utilisation | 3 | 1 | 1 | 1 | 33 % |") {
		t.Fatalf("synthèse des cas d'utilisation inattendue :\n%s", content)
	}
	if !strings.Contains(content, "| Hypothèses avec verdict | 2 | 1 | 0 | 1 | 50 % |") {
		t.Fatalf("synthèse des hypothèses inattendue :\n%s", content)
	}
}

func TestRenderSansDonnees(t *testing.T) {
	t.Parallel()
	content := Render(ports.DashboardData{GeneratedAt: time.Unix(0, 0)})
	if !strings.Contains(content, "| Exigences fonctionnelles | 0 | 0 | 0 | 0 | 0 % |") {
		t.Fatalf("un tableau vide doit rester lisible :\n%s", content)
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writer := NewWriter(root)
	path, err := writer.Write(context.Background(), sampleData())
	if err != nil {
		t.Fatalf("Write : %v", err)
	}
	if path != "docs/dashboard.md" || writer.Path() != path {
		t.Fatalf("chemin = %q", path)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "dashboard.md"))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if !strings.HasPrefix(string(content), "# Tableau de bord") {
		t.Fatalf("contenu inattendu :\n%s", content)
	}
	// BR-005-3 : la régénération remplace le fichier, elle ne l'ajoute pas.
	if _, err := writer.Write(context.Background(), sampleData()); err != nil {
		t.Fatalf("Write : %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(root, "docs", "dashboard.md"))
	if len(second) != len(content) {
		t.Fatal("la régénération doit produire le même fichier, non l'allonger")
	}
}

func TestMarkEtDash(t *testing.T) {
	t.Parallel()
	if mark(true) != "✔" || mark(false) != "✕" {
		t.Fatal("marques inattendues")
	}
	if dash("") != "—" || dash("  ") != "—" || dash("x") != "x" {
		t.Fatal("tiret cadratin inattendu")
	}
}
