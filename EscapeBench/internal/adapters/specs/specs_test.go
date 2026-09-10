package specs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/models"
)

const requirementsFixture = `# Catalogue d'exigences — EscapeBench

## Exigences fonctionnelles

| ID | Titre | Récit utilisateur |
|---|---|---|
| FR-001 | Générer la matrice | ... |

## Hypothèses à éprouver

| ID | Source (BEPG) | Énoncé réfutable | Critère de réfutation | UC liés |
|---|---|---|---|---|
| H-001 | p. 245 | énoncé un | critère un | UC-003, UC-004 |
| H-002 | p. 253 | énoncé deux | critère deux | UC-003 |
`

func writeDocs(t *testing.T, requirements string, useCases map[string]string) string {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "use-cases"), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if requirements != "" {
		if err := os.WriteFile(filepath.Join(docs, "requirements.md"), []byte(requirements), 0o644); err != nil {
			t.Fatalf("écriture : %v", err)
		}
	}
	for name, content := range useCases {
		if err := os.WriteFile(filepath.Join(docs, "use-cases", name), []byte(content), 0o644); err != nil {
			t.Fatalf("écriture : %v", err)
		}
	}
	return root
}

func TestLoadHypotheses(t *testing.T) {
	t.Parallel()
	root := writeDocs(t, requirementsFixture, nil)
	hypotheses, err := NewReader(root).Load(context.Background())
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if len(hypotheses) != 2 {
		t.Fatalf("%d hypothèses, 2 attendues", len(hypotheses))
	}
	if hypotheses[0].ID != "H-001" || hypotheses[0].SourcePages != "p. 245" {
		t.Fatalf("hypothèse = %+v", hypotheses[0])
	}
	if hypotheses[0].RefutationCriterion != "critère un" {
		t.Fatalf("critère = %q", hypotheses[0].RefutationCriterion)
	}
	if len(hypotheses[0].UseCases) != 2 || hypotheses[0].UseCases[1] != "UC-004" {
		t.Fatalf("UC liés = %v", hypotheses[0].UseCases)
	}
	// La ligne FR-001 du même fichier ne doit pas être prise pour une hypothèse.
	for _, h := range hypotheses {
		if strings.HasPrefix(h.ID, "FR-") {
			t.Fatalf("ligne FR lue comme hypothèse : %+v", h)
		}
	}
}

func TestLoadHypothesesErreurs(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"fichier absent":   "",
		"sans hypothèse":   "# Catalogue\n\n| FR-001 | x | y |\n",
		"ligne incomplète": "| H-001 | p. 1 | énoncé |\n",
		"critère manquant": "| H-001 | p. 1 | énoncé |  | UC-003 |\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := writeDocs(t, content, nil)
			if _, err := NewReader(root).Load(context.Background()); err == nil {
				t.Fatal("Load aurait dû échouer")
			}
		})
	}
}

func TestDigestGeleLesCriteres(t *testing.T) {
	t.Parallel()
	// BR-003-5 : l'empreinte porte l'énoncé et le critère, et ne dépend pas de l'ordre des
	// identifiants demandés. Mutation : retirer le critère du calcul ⇒ échec attendu.
	catalogue := []models.Hypothesis{
		{ID: "H-001", Statement: "s1", RefutationCriterion: "c1"},
		{ID: "H-002", Statement: "s2", RefutationCriterion: "c2"},
	}
	first, err := Digest(catalogue, []string{"H-001", "H-002"})
	if err != nil {
		t.Fatalf("Digest : %v", err)
	}
	second, err := Digest(catalogue, []string{"H-002", "H-001"})
	if err != nil {
		t.Fatalf("Digest : %v", err)
	}
	if first != second {
		t.Fatal("l'ordre des identifiants ne doit pas changer l'empreinte")
	}
	partial, err := Digest(catalogue, []string{"H-001"})
	if err != nil {
		t.Fatalf("Digest : %v", err)
	}
	if partial == first {
		t.Fatal("une sélection différente doit produire une empreinte différente")
	}
	modified := []models.Hypothesis{
		{ID: "H-001", Statement: "s1", RefutationCriterion: "c1 modifié"},
		{ID: "H-002", Statement: "s2", RefutationCriterion: "c2"},
	}
	changed, err := Digest(modified, []string{"H-001", "H-002"})
	if err != nil {
		t.Fatalf("Digest : %v", err)
	}
	if changed == first {
		t.Fatal("un critère modifié doit changer l'empreinte")
	}
	if _, err := Digest(catalogue, []string{"H-999"}); err == nil {
		t.Fatal("une hypothèse absente du catalogue doit être une erreur")
	}
}

func TestChangedCriteria(t *testing.T) {
	t.Parallel()
	recorded := []models.Hypothesis{
		{ID: "H-001", Statement: "s1", RefutationCriterion: "c1"},
		{ID: "H-002", Statement: "s2", RefutationCriterion: "c2"},
	}
	current := []models.Hypothesis{
		{ID: "H-001", Statement: "s1", RefutationCriterion: "c1 modifié"},
		{ID: "H-002", Statement: "s2 modifié", RefutationCriterion: "c2"},
		{ID: "H-003", Statement: "s3", RefutationCriterion: "c3"},
	}
	changed := ChangedCriteria(recorded, current)
	if len(changed) != 2 || changed[0] != "H-001" || changed[1] != "H-002" {
		t.Fatalf("changements = %v", changed)
	}
	if len(ChangedCriteria(recorded, recorded)) != 0 {
		t.Fatal("aucun changement attendu")
	}
}

const useCaseFixture = `# Use Case: Générer la matrice

## Overview

**Use Case ID:** UC-001
**Use Case Name:** Générer la matrice de cellules
**Primary Actor:** Chercheur
**Status:** Approved

**Linked Requirements:** FR-001, NFR-002, C-001
**Entities:** Matrix

## Preconditions
- La toolchain Go est installée.
`

func TestLoadUseCases(t *testing.T) {
	t.Parallel()
	root := writeDocs(t, requirementsFixture, map[string]string{
		"UC-001-generer-matrice.md": useCaseFixture,
		"UC-002-classer.md": strings.Replace(strings.Replace(useCaseFixture,
			"UC-001", "UC-002", 1), "Approved", "Reviewed", 1),
		"note.txt": "ignoré",
	})
	useCases, err := NewReader(root).LoadUseCases(context.Background())
	if err != nil {
		t.Fatalf("LoadUseCases : %v", err)
	}
	if len(useCases) != 2 {
		t.Fatalf("%d cas d'utilisation, 2 attendus", len(useCases))
	}
	if useCases[0].ID != "UC-001" || useCases[0].Status != "Approved" {
		t.Fatalf("cas d'utilisation = %+v", useCases[0])
	}
	if useCases[0].Title != "Générer la matrice de cellules" {
		t.Fatalf("titre = %q", useCases[0].Title)
	}
	// NFR-002 figure dans la même ligne : il ne doit pas être lu comme FR-002.
	// Mutation : retirer la limite de mot du motif FR ⇒ échec attendu.
	if len(useCases[0].LinkedFR) != 1 || useCases[0].LinkedFR[0] != "FR-001" {
		t.Fatalf("FR liés = %v", useCases[0].LinkedFR)
	}
	if useCases[1].ID != "UC-002" || useCases[1].Status != "Reviewed" {
		t.Fatalf("cas d'utilisation = %+v", useCases[1])
	}
}

func TestLoadUseCasesErreurs(t *testing.T) {
	t.Parallel()
	vide := writeDocs(t, requirementsFixture, nil)
	if _, err := NewReader(vide).LoadUseCases(context.Background()); err == nil {
		t.Fatal("un dossier sans cas d'utilisation doit être une erreur")
	}
	sansID := writeDocs(t, requirementsFixture, map[string]string{
		"UC-009-sans-id.md": "# Use Case\n\n**Status:** Draft\n",
	})
	if _, err := NewReader(sansID).LoadUseCases(context.Background()); err == nil {
		t.Fatal("un cas d'utilisation sans Use Case ID doit être une erreur")
	}
}

func TestIndexReferences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir : %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("écriture : %v", err)
		}
	}
	write("internal/service/matrix.go", "package service\n\n// UC-001 Générer la matrice — étapes 1 à 7.\nfunc Generate() {}\n")
	write("internal/service/matrix_test.go", "package service\n\nfunc TestUC001_MainFlow() {}\n")
	write("internal/service/escape_integration_test.go", "//go:build integration_test\n\npackage service\n\nfunc TestUC002_MainFlow() {}\n")
	write("cmd/escapebench/main.go", "package main\n\n// UC-005 verdicts\nfunc main() {}\n")
	write("internal/service/notes.txt", "UC-004 ne compte pas : ce n'est pas du Go")

	index := NewIndex(root)
	for _, tc := range []struct {
		id          string
		code        bool
		integration bool
	}{
		{"UC-001", true, false},
		{"UC-002", true, true},
		{"UC-005", true, false},
		{"UC-004", false, false},
	} {
		code, integration, err := index.References(context.Background(), tc.id)
		if err != nil {
			t.Fatalf("References(%s) : %v", tc.id, err)
		}
		if code != tc.code || integration != tc.integration {
			t.Fatalf("References(%s) = %v, %v ; attendu %v, %v", tc.id, code, integration, tc.code, tc.integration)
		}
	}
}

func TestIndexSansSources(t *testing.T) {
	t.Parallel()
	code, integration, err := NewIndex(t.TempDir()).References(context.Background(), "UC-001")
	if err != nil {
		t.Fatalf("References : %v", err)
	}
	if code || integration {
		t.Fatal("un dépôt sans sources ne référence aucun cas d'utilisation")
	}
}

func TestSplitRowEtSplitList(t *testing.T) {
	t.Parallel()
	cells := splitRow("|  a | b |  c  |")
	if len(cells) != 3 || cells[0] != "a" || cells[2] != "c" {
		t.Fatalf("splitRow = %q", cells)
	}
	if got := splitList("UC-003, UC-004 ,"); len(got) != 2 || got[1] != "UC-004" {
		t.Fatalf("splitList = %q", got)
	}
	if got := splitList("—"); got != nil {
		t.Fatalf("splitList = %q, le tiret cadratin signifie « aucun »", got)
	}
}
