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

func TestUC001_Detecteurs(t *testing.T) {
	ds := Detectors()
	if len(ds) != 8 || ds[0] != DetectorBare || ds[5] != DetectorProgram || ds[7] != DetectorCtxvet {
		t.Fatalf("Detectors = %v : six dynamiques puis VET et CTXVET attendus (C-006)", ds)
	}
}

func TestNFR004_EcritureJSONEtRelecture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "R-2026-09-22-1.json")
	want := Run{ID: "R-2026-09-22-1", Provenance: Provenance{GoVersion: "go1.27.0", CPU: "x", NumCPU: 2, OSVersion: "Windows 10.0.26220"}, Reps: 5}
	if err := WriteJSONExclusive(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRun(path)
	if err != nil || got.ID != want.ID || got.Provenance != want.Provenance {
		t.Fatalf("LoadRun = %+v, %v", got, err)
	}
	if err := WriteJSONExclusive(path, want); err == nil {
		t.Fatal("une campagne existante a été réécrite")
	}
	if err := WriteJSONExclusive(filepath.Join(t.TempDir(), "x.json"), make(chan int)); err == nil {
		t.Fatal("une valeur non encodable doit être une erreur")
	}
	if _, err := LoadRun(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("campagne absente : erreur attendue")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRun(bad); err == nil {
		t.Fatal("JSON invalide : erreur attendue")
	}
}

// TestNFR001_CampagnesArchiveesLisibles vérifie que les campagnes archivées, antérieures à
// osVersion (modèle d'entités, révision du 2026-09-22), se relisent sans lui, et que les campagnes
// postérieures le portent. Lecture seule.
func TestNFR001_CampagnesArchiveesLisibles(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "results", "runs", "R-*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("aucune campagne archivée : %v", err)
	}
	for _, p := range paths {
		r, err := LoadRun(p)
		if err != nil {
			t.Fatal(err)
		}
		recent := filepath.Base(p) >= "R-2026-09-22" // osVersion est écrit depuis D-19
		if r.Provenance.GoVersion == "" || r.Provenance.CPU == "" || len(r.Observations) == 0 || (r.Provenance.OSVersion != "") != recent {
			t.Errorf("%s : provenance %+v, %d observations", filepath.Base(p), r.Provenance, len(r.Observations))
		}
	}
}
