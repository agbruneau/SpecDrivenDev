package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUC003_CommandeCtxvet(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"ctxvet", "../../lab/corpus"}, &out, &errOut); code != 1 {
		t.Fatalf("code %d, 1 attendu (diagnostics) ; %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "io_without_context_leak.go") || !strings.Contains(out.String(), "net.Dial ignore le contexte") {
		t.Fatalf("sortie sans le diagnostic attendu :\n%s", out.String())
	}
	if code := run(context.Background(), []string{"ctxvet", "../../internal/results"}, &out, &errOut); code != 0 {
		t.Fatalf("paquet sain : code %d", code)
	}
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"inconnue"}, {"verdict"}, {"ctxvet"}} {
		var out, errOut bytes.Buffer
		if code := run(context.Background(), args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("%v : code %d, sortie %q", args, code, errOut.String())
		}
	}
}

// copyFile copie src vers dst en créant les répertoires.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestUC002_CommandeVerdict rejoue la sous-commande verdict sur une copie de la campagne de
// référence, dans une racine jetable : le results/ du dépôt n'est que lu.
func TestUC002_CommandeVerdict(t *testing.T) {
	root := t.TempDir()
	const id = "R-2026-09-13-2"
	copyFile(t, filepath.Join("..", "..", "docs", "requirements.md"), filepath.Join(root, "docs", "requirements.md"))
	copyFile(t, filepath.Join("..", "..", "results", "runs", id+".json"), filepath.Join(root, "results", "runs", id+".json"))

	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"verdict", "-root", root, "-run", id}, &out, &errOut); code != 0 {
		t.Fatalf("code %d ; %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "H-001") || !strings.Contains(out.String(), "verdicts écrits dans") {
		t.Fatalf("sortie inattendue :\n%s", out.String())
	}
	if code := run(context.Background(), []string{"verdict", "-root", root, "-run", id}, &out, &errOut); code != 1 {
		t.Fatalf("second verdict du même jour : code %d, 1 attendu (écriture exclusive)", code)
	}
	if code := run(context.Background(), []string{"verdict", "-root", root, "-run", "R-absente"}, &out, &errOut); code != 1 {
		t.Fatalf("campagne absente : code %d", code)
	}

	bad := t.TempDir()
	copyFile(t, filepath.Join(root, "results", "runs", id+".json"), filepath.Join(bad, "results", "runs", id+".json"))
	if code := run(context.Background(), []string{"verdict", "-root", bad, "-run", id}, &out, &errOut); code != 1 {
		t.Fatalf("spécification absente : code %d", code)
	}
	if err := os.MkdirAll(filepath.Join(bad, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "docs", "requirements.md"), []byte("rien"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run(context.Background(), []string{"verdict", "-root", bad, "-run", id}, &out, &errOut); code != 1 {
		t.Fatalf("spécification sans hypothèses : code %d", code)
	}
}

func TestUC001_CommandeRunRefusee(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"run", "-root", t.TempDir(), "-reps", "4"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "NFR-002") {
		t.Fatalf("code %d, sortie %q", code, errOut.String())
	}
	if code := run(context.Background(), []string{"run", "-reps", "x"}, &out, &errOut); code != 2 {
		t.Fatalf("drapeau invalide : code %d", code)
	}
}
