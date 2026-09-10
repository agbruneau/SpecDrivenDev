package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/escapebench/internal/adapters/cli"
)

// projectDocs copie docs/ du dépôt vers une racine temporaire : le binaire y écrira matrices/,
// results/ et dashboard.md sans toucher au dépôt.
func projectDocs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "docs")
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, "docs", relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatalf("copie de docs/ : %v", err)
	}
	return root
}

func TestRunSousCommandeInconnue(t *testing.T) {
	t.Parallel()
	err := run(context.Background(), "compile", nil)
	if !errors.Is(err, cli.ErrUsage) {
		t.Fatalf("erreur = %v, ErrUsage attendue", err)
	}
}

func TestRunOptionsManquantes(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"escape sans matrice":        {"escape"},
		"compare sans campagne":      {"compare"},
		"verdict sans campagne":      {"verdict"},
		"campaign sans matrice":      {"campaign"},
		"matrix sans mode":           {"matrix"},
		"matrix avec les deux modes": {"matrix", "--reference", "--params", "sizes=8"},
		"matrix aux paramètres faux": {"matrix", "--params", "sizes=huit"},
		"drapeau inconnu":            {"escape", "--inconnu"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := run(context.Background(), args[0], args[1:])
			if !errors.Is(err, cli.ErrUsage) {
				t.Fatalf("erreur = %v, ErrUsage attendue", err)
			}
		})
	}
}

func TestRunRacineIntrouvable(t *testing.T) {
	t.Parallel()
	// Sans docs/requirements.md dans les répertoires parents, aucune sous-commande ne démarre.
	hors := t.TempDir()
	for _, args := range [][]string{
		{"matrix", "--reference", "--root", hors},
		{"escape", "--matrix", "M-1", "--root", hors},
		{"campaign", "--matrix", "M-1", "--root", hors},
		{"compare", "--campaign", "C-1", "--root", hors},
		{"verdict", "--campaign", "C-1", "--root", hors},
		{"dashboard", "--root", hors},
	} {
		if err := run(context.Background(), args[0], args[1:]); err == nil {
			t.Fatalf("%v aurait dû échouer hors d'un projet EscapeBench", args)
		}
	}
}

func TestRunChaineCompleteUC001aUC005(t *testing.T) {
	// Chaîne complète sur une matrice d'un seul sujet : génération, classification, campagne,
	// comparaison, verdicts, tableau de bord. Elle utilise la vraie chaîne d'outils Go.
	root := projectDocs(t)
	ctx := context.Background()
	// Une paire VALUE/POINTER d'une seule taille : le minimum pour que UC-004 ait une paire.
	params := "sizes=8;pointer=false;profiles=LOCAL;modes=VALUE,POINTER"

	if err := run(ctx, "matrix", []string{"--root", root, "--params", params}); err != nil {
		t.Fatalf("matrix : %v", err)
	}
	matrixID := onlyMatrixID(t, root)

	// UC-001 A2 : une seconde génération ne régénère rien.
	if err := run(ctx, "matrix", []string{"--root", root, "--params", params}); err != nil {
		t.Fatalf("matrix (seconde fois) : %v", err)
	}

	if err := run(ctx, "escape", []string{"--root", root, "--matrix", matrixID}); err != nil {
		t.Fatalf("escape : %v", err)
	}
	assertExists(t, filepath.Join(root, "results", "escape", matrixID))

	if err := run(ctx, "campaign", []string{"--root", root, "--matrix", matrixID,
		"--count", "20", "--benchtime", "1ms", "--hypotheses", "H-001,H-002"}); err != nil {
		t.Fatalf("campaign : %v", err)
	}
	campaignID := onlyCampaignID(t, root)

	if err := run(ctx, "compare", []string{"--root", root, "--campaign", campaignID}); err != nil {
		t.Fatalf("compare : %v", err)
	}
	if err := run(ctx, "verdict", []string{"--root", root, "--campaign", campaignID}); err != nil {
		t.Fatalf("verdict : %v", err)
	}
	assertExists(t, filepath.Join(root, "results", "verdicts"))

	dashboard := filepath.Join(root, "docs", "dashboard.md")
	assertExists(t, dashboard)
	content, err := os.ReadFile(dashboard)
	if err != nil {
		t.Fatalf("lecture du tableau de bord : %v", err)
	}
	for _, needle := range []string{"# Tableau de bord — EscapeBench", "H-001", "UC-001"} {
		if !strings.Contains(string(content), needle) {
			t.Fatalf("%q absent du tableau de bord", needle)
		}
	}

	// UC-005 étape 7 : la régénération seule fonctionne aussi.
	if err := run(ctx, "dashboard", []string{"--root", root}); err != nil {
		t.Fatalf("dashboard : %v", err)
	}
}

func TestRunCampagneRefuseeSansVerdictsDEchappement(t *testing.T) {
	root := projectDocs(t)
	ctx := context.Background()
	params := "sizes=8;pointer=false;profiles=LOCAL;modes=VALUE,POINTER"
	if err := run(ctx, "matrix", []string{"--root", root, "--params", params}); err != nil {
		t.Fatalf("matrix : %v", err)
	}
	// La précondition de UC-003 n'est pas satisfaite : UC-002 n'a pas été exécuté.
	err := run(ctx, "campaign", []string{"--root", root, "--matrix", onlyMatrixID(t, root), "--count", "20", "--benchtime", "1ms"})
	if err == nil {
		t.Fatal("campaign aurait dû être refusée")
	}
	if !strings.Contains(err.Error(), "escapebench escape") {
		t.Fatalf("le message doit indiquer la sous-commande manquante : %v", err)
	}
}

func TestRunCompareCampagneInconnue(t *testing.T) {
	root := projectDocs(t)
	if err := run(context.Background(), "compare", []string{"--root", root, "--campaign", "C-inconnue"}); err == nil {
		t.Fatal("compare aurait dû échouer")
	}
	if err := run(context.Background(), "verdict", []string{"--root", root, "--campaign", "C-inconnue"}); err == nil {
		t.Fatal("verdict aurait dû échouer")
	}
	if err := run(context.Background(), "escape", []string{"--root", root, "--matrix", "M-inconnue"}); err == nil {
		t.Fatal("escape aurait dû échouer")
	}
}

func TestHarnessDigestStable(t *testing.T) {
	t.Parallel()
	d, err := newDeps(projectDocs(t))
	if err != nil {
		t.Fatalf("newDeps : %v", err)
	}
	first, err := d.HarnessDigest(context.Background())
	if err != nil {
		t.Fatalf("HarnessDigest : %v", err)
	}
	second, _ := d.HarnessDigest(context.Background())
	if first == "" || first != second {
		t.Fatalf("empreinte instable : %q vs %q", first, second)
	}
}

func TestNewDepsSansRacineExplicite(t *testing.T) {
	t.Parallel()
	// Sans --root, la racine est cherchée depuis le répertoire courant : les tests de ce paquet
	// s'exécutent dans cmd/escapebench, la remontée doit trouver le dépôt.
	d, err := newDeps("")
	if err != nil {
		t.Fatalf("newDeps : %v", err)
	}
	if _, err := os.Stat(filepath.Join(d.root, "docs", "requirements.md")); err != nil {
		t.Fatalf("racine = %q : %v", d.root, err)
	}
}

func onlyMatrixID(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "matrices"))
	if err != nil {
		t.Fatalf("lecture de matrices/ : %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("%d matrices, 1 attendue", len(entries))
	}
	return entries[0].Name()
}

func onlyCampaignID(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "results", "campaigns"))
	if err != nil {
		t.Fatalf("lecture de results/campaigns : %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("%d campagnes, 1 attendue", len(entries))
	}
	return entries[0].Name()
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s devrait exister : %v", path, err)
	}
}
