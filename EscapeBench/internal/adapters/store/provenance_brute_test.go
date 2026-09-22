package store

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// UC-003, étape 3 et BR-003-3 (D-60) : la Provenance étendue survit à l'écriture, à la relecture
// et à la transition de statut, seule réécriture admise de campaign.json.
// Mutation : oublier coreTypes dans toModel ⇒ échec attendu.
func TestUC003_Etape3_ProvenanceEtendueConservee(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	p := provenance()
	p.OSVersion = "Windows 10.0.26220.1 (25H2)"
	p.PowerPlan = "Utilisation normale (381b4222-f694-41f0-9685-ff5bb260df2e)"
	p.CPUAffinity = "non épinglé"
	p.CoreTypes = models.CoreTypes{Performance: 8, Efficiency: 16}
	c := models.Campaign{ID: "C-2026-09-22-1", MatrixID: "M-1", HarnessDigest: "h", HypothesesDigest: "d",
		HypothesisIDs: []string{"H-001"}, Count: 20, Status: models.CampaignRunning,
		Provenance: p, StartedAt: p.CapturedAt}
	if err := s.CreateCampaign(ctx, c); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	if err := s.SetCampaignStatus(ctx, c.ID, models.CampaignCompleted, p.CapturedAt.Add(time.Hour), ""); err != nil {
		t.Fatalf("SetCampaignStatus : %v", err)
	}
	relue, err := s.LoadCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("LoadCampaign : %v", err)
	}
	if !reflect.DeepEqual(relue.Provenance, p) {
		t.Fatalf("provenance relue = %+v, %+v attendue", relue.Provenance, p)
	}
}

// Un fichier antérieur au 2026-09-22 ne porte aucun des nouveaux champs : il se relit, et une
// provenance sans eux ne les invente pas à l'écriture (D-60).
// Mutation : sérialiser coreTypes même vide ⇒ échec attendu.
func TestUC003_ChampsFacultatifsAbsentsDesFichiersAnterieurs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	seedCampaign(t, s, "C-1")
	content, err := os.ReadFile(filepath.Join(s.Root(), filepath.FromSlash(s.CampaignPath("C-1"))))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	for _, champ := range []string{"osVersion", "powerPlan", "cpuAffinity", "coreTypes"} {
		if strings.Contains(string(content), champ) {
			t.Fatalf("%s ne doit pas apparaître quand il est vide", champ)
		}
	}
	if err := s.WriteMeasurement(ctx, models.Measurement{CampaignID: "C-1", SubjectID: "s",
		Status: models.MeasurementFailed, FailureReason: "boom"}); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
	content, err = os.ReadFile(filepath.Join(s.Root(), filepath.FromSlash(s.MeasurementPath("C-1", "s"))))
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	for _, champ := range []string{"iterations", "rawOutputFile"} {
		if strings.Contains(string(content), champ) {
			t.Fatalf("%s ne doit pas apparaître quand il est vide", champ)
		}
	}
}

// UC-003, étape 6 et BR-003-3 (D-60) : la sortie brute est créée sous raw/ avant la Measurement
// qui en porte le chemin ; elle n'entre pas dans le JSON ; une sortie brute orpheline, laissée par
// une tentative interrompue avant l'écriture de la Measurement, n'est jamais réécrite.
// Mutation : écrire la sortie brute par writeFileAtomic ⇒ l'orpheline est écrasée, échec attendu.
func TestUC003_BR3_SortieBruteCreeeJamaisReecrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	seedCampaign(t, s, "C-1")
	m := models.Measurement{CampaignID: "C-1", SubjectID: "T/LOCAL/VALUE", Status: models.MeasurementComplete,
		NsPerOp: []float64{1}, BytesPerOp: []int64{0}, AllocsPerOp: []int64{0}, Iterations: []int64{1000},
		RawOutput: "BenchmarkSubject\t1000\t1 ns/op\t0 B/op\t0 allocs/op\n"}
	if err := s.WriteMeasurement(ctx, m); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
	relues, err := s.LoadMeasurements(ctx, "C-1")
	if err != nil || len(relues) != 1 {
		t.Fatalf("LoadMeasurements = %d, %v", len(relues), err)
	}
	relue := relues[0]
	if relue.RawOutputFile != "results/campaigns/C-1/raw/T_LOCAL_VALUE.txt" {
		t.Fatalf("rawOutputFile = %q", relue.RawOutputFile)
	}
	if !reflect.DeepEqual(relue.Iterations, []int64{1000}) {
		t.Fatalf("iterations = %v", relue.Iterations)
	}
	if relue.RawOutput != "" {
		t.Fatal("la sortie brute ne doit pas être recopiée dans le JSON de la Measurement")
	}
	brute, err := os.ReadFile(filepath.Join(s.Root(), filepath.FromSlash(relue.RawOutputFile)))
	if err != nil || string(brute) != m.RawOutput {
		t.Fatalf("sortie brute = %q, %v", brute, err)
	}

	// Une orpheline : sortie brute présente, Measurement absente. La reprise prend le nom suivant.
	orpheline := filepath.Join(s.Root(), "results", "campaigns", "C-1", "raw", "U_LOCAL_VALUE.txt")
	if err := os.WriteFile(orpheline, []byte("tentative interrompue"), 0o644); err != nil {
		t.Fatalf("préparation : %v", err)
	}
	m.SubjectID, m.RawOutput = "U/LOCAL/VALUE", "reprise"
	if err := s.WriteMeasurement(ctx, m); err != nil {
		t.Fatalf("WriteMeasurement après orpheline : %v", err)
	}
	if content, _ := os.ReadFile(orpheline); string(content) != "tentative interrompue" {
		t.Fatalf("l'orpheline a été réécrite : %q", content)
	}
	relues, _ = s.LoadMeasurements(ctx, "C-1")
	if got := relues[1].RawOutputFile; got != "results/campaigns/C-1/raw/U_LOCAL_VALUE-1.txt" {
		t.Fatalf("rawOutputFile de la reprise = %q", got)
	}

	// Une Measurement déjà écrite est refusée sans créer de sortie brute de plus.
	if err := s.WriteMeasurement(ctx, m); err == nil {
		t.Fatal("une Measurement ne se réécrit pas (BR-003-3)")
	}
	entries, _ := os.ReadDir(filepath.Join(s.Root(), "results", "campaigns", "C-1", "raw"))
	if len(entries) != 3 {
		t.Fatalf("%d fichiers sous raw/, 3 attendus", len(entries))
	}
}
