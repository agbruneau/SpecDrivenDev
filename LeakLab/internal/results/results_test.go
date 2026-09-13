package results

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNFR004_EcritureExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "R-1.json")
	if err := WriteExclusive(path, []byte("premier")); err != nil {
		t.Fatalf("première écriture : %v", err)
	}
	if err := WriteExclusive(path, []byte("second")); err == nil {
		t.Fatal("un fichier existant a été écrasé")
	}
	if got, _ := os.ReadFile(path); string(got) != "premier" {
		t.Fatalf("contenu = %q, « premier » attendu", got)
	}
}

func TestUC001_NextRunID(t *testing.T) {
	dir := t.TempDir()
	day := time.Date(2026, 9, 13, 23, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{"répertoire vide", nil, "R-2026-09-13-1"},
		{"suite du jour", []string{"R-2026-09-13-1.json", "R-2026-09-13-9.json", "R-2026-09-13-10.json"}, "R-2026-09-13-11"},
		{"autres jours ignorés", []string{"R-2026-09-12-40.json"}, "R-2026-09-13-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := filepath.Join(dir, tt.name)
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(sub, f), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := NextRunID(sub, day)
			if err != nil || got != tt.want {
				t.Fatalf("NextRunID = %q, %v ; %q attendu", got, err, tt.want)
			}
		})
	}
	if got, err := NextRunID(filepath.Join(dir, "absent"), day); err != nil || got != "R-2026-09-13-1" {
		t.Fatalf("répertoire absent : %q, %v", got, err)
	}
}
