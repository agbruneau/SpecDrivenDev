package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

func TestMatrixRoundTripAvecDispositionEtRepetition(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	params := models.MatrixParameters{
		Sizes:                []int{24},
		PointerFieldVariants: []bool{true},
		Profiles:             []models.LifetimeProfile{models.ProfileReturnedAlloc},
		PassingModes:         models.PassingModes(),
		Layouts:              []models.Layout{models.LayoutNamedFields},
		Repeats:              []int{1, 4},
	}
	matrix, err := models.NewMatrix(params, "digest", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	if err := s.Finalize(ctx, matrix); err != nil {
		t.Fatalf("Finalize : %v", err)
	}
	relue, err := s.Load(ctx, matrix.ID)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if len(relue.Cells) != len(matrix.Cells) {
		t.Fatalf("%d cellules relues, %d attendues", len(relue.Cells), len(matrix.Cells))
	}
	for i, cell := range relue.Cells {
		if cell.ID() != matrix.Cells[i].ID() {
			t.Fatalf("identifiant altéré : %s contre %s", cell.ID(), matrix.Cells[i].ID())
		}
		if cell.TypeSpec.Layout != models.LayoutNamedFields {
			t.Fatalf("disposition altérée pour %s : %q", cell.ID(), cell.TypeSpec.Layout)
		}
		if cell.Repetitions() != matrix.Cells[i].Repetitions() {
			t.Fatalf("répétition altérée pour %s", cell.ID())
		}
	}
	if err := relue.Validate(); err != nil {
		t.Fatalf("la matrice relue doit rester valide : %v", err)
	}
	// Les deux dimensions se relisent dans les paramètres.
	if len(relue.Parameters.Layouts) != 1 || relue.Parameters.Layouts[0] != models.LayoutNamedFields {
		t.Fatalf("dispositions relues = %v", relue.Parameters.Layouts)
	}
	if len(relue.Parameters.Repeats) != 2 {
		t.Fatalf("répétitions relues = %v", relue.Parameters.Repeats)
	}
	// L'identifiant recalculé depuis les paramètres relus doit être le même (BR-001-1).
	if relue.Parameters.MatrixID() != matrix.ID {
		t.Fatalf("identifiant recalculé = %s, %s attendu", relue.Parameters.MatrixID(), matrix.ID)
	}
}

func TestMatrixAnterieureAC008RelueSansDisposition(t *testing.T) {
	t.Parallel()
	// Un matrix.json écrit avant C-008 ne porte pas de disposition : la relire doit donner la
	// disposition d'origine, faute de quoi les campagnes archivées deviendraient illisibles.
	// Mutation : laisser la disposition vide à la relecture ⇒ échec attendu, Validate refuse.
	ctx := context.Background()
	s := newStore(t)
	ancien := map[string]any{
		"id": "M-ancienne",
		"parameters": map[string]any{
			"sizes": []int{24}, "pointerFieldVariants": []bool{false},
			"lifetimeProfiles": []string{"LOCAL"}, "passingModes": []string{"VALUE", "POINTER"},
			"probes": []any{},
		},
		"harnessDigest": "digest",
		"cells": []any{
			map[string]any{
				"id": "Size0024Plain/LOCAL/VALUE",
				"typeSpec": map[string]any{
					"name": "Size0024Plain", "sizeBytes": 24, "wordCount": 3, "hasPointerField": false,
				},
				"lifetimeProfile": "LOCAL", "passingMode": "VALUE",
				"sourceFile": "subjects/Size0024Plain_LOCAL_VALUE/subject.go",
			},
		},
		"probes":      []any{},
		"generatedAt": "2026-09-10T12:00:00Z",
	}
	dir := s.Dir("M-ancienne")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	content, err := json.Marshal(ancien)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "matrix.json"), content, 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	relue, err := s.Load(ctx, "M-ancienne")
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if relue.Cells[0].TypeSpec.Layout != models.LayoutArrayFill {
		t.Fatalf("disposition relue = %q, %s attendue", relue.Cells[0].TypeSpec.Layout, models.LayoutArrayFill)
	}
	if relue.Cells[0].Repetitions() != 1 {
		t.Fatalf("répétition relue = %d", relue.Cells[0].Repetitions())
	}
	if err := relue.Validate(); err != nil {
		t.Fatalf("une matrice antérieure à C-008 doit rester valide : %v", err)
	}
}

func TestProvenanceEtComparaisonPortentLesAjoutsDeC008(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	prov := provenance()
	prov.L1DataCacheBytes = 49152
	prov.LastLevelCacheBytes = 36 << 20
	prov.PageSizeBytes = 4096
	prov.GOMAXPROCS = 24

	report := models.EscapeReport{MatrixID: "M-1", Provenance: prov,
		Verdicts: []models.EscapeVerdict{{CellID: "a", Category: models.CategoryNone, Status: models.EscapeStatusOK}}}
	path, err := s.WriteEscapeReport(ctx, report, prov.CapturedAt)
	if err != nil {
		t.Fatalf("WriteEscapeReport : %v", err)
	}
	relu, err := s.ReadEscapeReport(ctx, path)
	if err != nil {
		t.Fatalf("ReadEscapeReport : %v", err)
	}
	if relu.Provenance.L1DataCacheBytes != 49152 || relu.Provenance.LastLevelCacheBytes != 36<<20 {
		t.Fatalf("tailles de cache altérées : %+v", relu.Provenance)
	}
	if relu.Provenance.PageSizeBytes != 4096 || relu.Provenance.GOMAXPROCS != 24 {
		t.Fatalf("provenance altérée : %+v", relu.Provenance)
	}

	set := models.ComparisonSet{
		CampaignID: "C-1", MatrixID: "M-1", ComputedAt: prov.CapturedAt, Method: "bootstrap",
		Comparisons: []models.Comparison{{
			CampaignID: "C-1", ValueCellID: "v", PointerCellID: "p", SizeBytes: 24,
			Layout: models.LayoutNamedFieldsSham, Profile: models.ProfileLocal,
		}},
		TippingPoints: map[models.TippingKey]int{},
	}
	if _, err := s.WriteComparisonSet(ctx, set); err != nil {
		t.Fatalf("WriteComparisonSet : %v", err)
	}
	relue, _, err := s.LatestComparisonSet(ctx, "C-1")
	if err != nil {
		t.Fatalf("LatestComparisonSet : %v", err)
	}
	if relue.Comparisons[0].Layout != models.LayoutNamedFieldsSham {
		t.Fatalf("disposition altérée : %q", relue.Comparisons[0].Layout)
	}
	// Une comparaison antérieure à C-008 se relit avec la disposition d'origine.
	var dto comparisonSetDTO
	dto.Comparisons = []comparisonDTO{{ValueCellID: "v", PointerCellID: "p"}}
	if got := dto.toModel().Comparisons[0].Layout; got != models.LayoutArrayFill {
		t.Fatalf("disposition par défaut = %q", got)
	}
}

func TestCellDTOOmetLaRepetitionParDefaut(t *testing.T) {
	t.Parallel()
	// Une répétition de un ne doit pas apparaître dans le fichier : les matrices antérieures à
	// C-008 restent octet pour octet comparables.
	spec, _ := models.NewTypeSpec(24, false, models.LayoutArrayFill)
	cell := models.Cell{TypeSpec: spec, Profile: models.ProfileLocal, PassingMode: models.PassingValue, Repeat: 1, SourceFile: "x"}
	content, err := json.Marshal(toCellDTO(cell))
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	if strings.Contains(string(content), "repeat") {
		t.Fatalf("le champ repeat ne doit pas être écrit pour une répétition de un : %s", content)
	}
	cell.Repeat = 4
	content, _ = json.Marshal(toCellDTO(cell))
	if !strings.Contains(string(content), `"repeat":4`) {
		t.Fatalf("le champ repeat doit être écrit au-delà de un : %s", content)
	}
}
