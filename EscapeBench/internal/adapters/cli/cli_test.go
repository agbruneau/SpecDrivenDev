package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/service"
)

func TestParseParametersComplet(t *testing.T) {
	t.Parallel()
	params, err := ParseParameters("sizes=8,24;pointer=false;profiles=LOCAL,RETURNED;modes=VALUE;probes=SEQUENTIAL_SCAN:65536,APPEND_GROW:1000")
	if err != nil {
		t.Fatalf("ParseParameters : %v", err)
	}
	if !reflect.DeepEqual(params.Sizes, []int{8, 24}) {
		t.Fatalf("tailles = %v", params.Sizes)
	}
	if !reflect.DeepEqual(params.PointerFieldVariants, []bool{false}) {
		t.Fatalf("variantes = %v", params.PointerFieldVariants)
	}
	if !reflect.DeepEqual(params.Profiles, []models.LifetimeProfile{models.ProfileLocal, models.ProfileReturned}) {
		t.Fatalf("profils = %v", params.Profiles)
	}
	if !reflect.DeepEqual(params.PassingModes, []models.PassingMode{models.PassingValue}) {
		t.Fatalf("modes = %v", params.PassingModes)
	}
	if len(params.Probes) != 2 || params.Probes[0].Parameter != 65536 {
		t.Fatalf("sondes = %v", params.Probes)
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("les paramètres analysés doivent être valides : %v", err)
	}
}

func TestParseParametersDefauts(t *testing.T) {
	t.Parallel()
	// Les clés absentes reprennent la matrice de référence : les deux variantes de champ
	// pointeur, ses cinq profils et les deux modes. Prendre les huit profils du modèle ferait
	// produire à une spécification antérieure à C-008 une matrice plus grosse que celle de
	// référence, sur laquelle les critères gelés H-001 à H-006 seraient réévalués.
	params, err := ParseParameters("sizes=8")
	if err != nil {
		t.Fatalf("ParseParameters : %v", err)
	}
	if len(params.Profiles) != len(models.ReferenceProfiles()) || len(params.PassingModes) != 2 || len(params.PointerFieldVariants) != 2 {
		t.Fatalf("valeurs par défaut inattendues : %+v", params)
	}
	if len(params.Probes) != 0 {
		t.Fatalf("aucune sonde par défaut, obtenu %v", params.Probes)
	}
	// Disposition et répétition restent vides : la normalisation leur applique la valeur d'avant
	// C-008, ce qui garde l'identifiant des matrices antérieures.
	normalized := params.Normalize()
	if len(normalized.Layouts) != 1 || normalized.Layouts[0] != models.LayoutArrayFill {
		t.Fatalf("dispositions par défaut = %v", normalized.Layouts)
	}
	if len(normalized.Repeats) != 1 || normalized.Repeats[0] != 1 {
		t.Fatalf("répétitions par défaut = %v", normalized.Repeats)
	}
	if _, err := ParseParameters("sizes=8; ;"); err != nil {
		t.Fatalf("les clauses vides sont ignorées : %v", err)
	}
}

func TestParseParametersErreurs(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"sans sizes":           "profiles=LOCAL",
		"clause sans égal":     "sizes",
		"clé inconnue":         "sizes=8;couleur=rouge",
		"taille illisible":     "sizes=huit",
		"taille vide":          "sizes=",
		"pointeur illisible":   "sizes=8;pointer=peut-être",
		"pointeur vide":        "sizes=8;pointer=",
		"sonde sans paramètre": "sizes=8;probes=APPEND_GROW",
		"paramètre illisible":  "sizes=8;probes=APPEND_GROW:mille",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseParameters(spec)
			if err == nil {
				t.Fatal("ParseParameters aurait dû refuser")
			}
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("erreur = %v, ErrUsage attendue", err)
			}
		})
	}
}

func TestParseHypotheses(t *testing.T) {
	t.Parallel()
	if got := ParseHypotheses(" H-001, H-002 ,"); !reflect.DeepEqual(got, []string{"H-001", "H-002"}) {
		t.Fatalf("ParseHypotheses = %v", got)
	}
	if got := ParseHypotheses(""); got != nil {
		t.Fatalf("ParseHypotheses = %v", got)
	}
}

func TestFindRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "requirements.md"), []byte("# x"), 0o644); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	found, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("FindRoot : %v", err)
	}
	if filepath.Clean(found) != filepath.Clean(root) {
		t.Fatalf("FindRoot = %q, attendu %q", found, root)
	}
	if _, err := FindRoot(t.TempDir()); err == nil {
		t.Fatal("FindRoot doit échouer hors d'un projet EscapeBench")
	}
}

func TestRenderProvenance(t *testing.T) {
	t.Parallel()
	p := models.Provenance{GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "Ryzen",
		CapturedAt: time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC)}
	got := RenderProvenance(p)
	for _, needle := range []string{"go1.25.0", "linux/amd64", "Ryzen", "2026-09-10 12:30:00 UTC"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de %q", needle, got)
		}
	}
}

func TestRenderMatrix(t *testing.T) {
	t.Parallel()
	got := RenderMatrix(service.MatrixReport{MatrixID: "M-1", TypeSpecCount: 4, CellCount: 16, ProbeCount: 2, HarnessDigest: "abc"})
	for _, needle := range []string{"Matrice : M-1", "TypeSpec : 4", "Cell : 16", "Probe : 2", "abc"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de %q", needle, got)
		}
	}
	existing := RenderMatrix(service.MatrixReport{MatrixID: "M-1", AlreadyExisted: true})
	if !strings.Contains(existing, "rien de régénéré") {
		t.Fatalf("A2 doit être visible : %q", existing)
	}
	failed := RenderMatrix(service.MatrixReport{MatrixID: "M-1", CompileErrors: []service.SubjectFailure{
		{SubjectID: "s", Message: "\n  erreur de compilation\nligne suivante"},
	}})
	if !strings.Contains(failed, "Sujets non compilables : 1") || !strings.Contains(failed, "erreur de compilation") {
		t.Fatalf("A3 doit être visible : %q", failed)
	}
}

func TestRenderEscape(t *testing.T) {
	t.Parallel()
	summary := service.EscapeReportSummary{
		MatrixID: "M-1", Path: "results/escape/M-1/x.json",
		Counts:     map[models.EscapeCategory]int{models.CategoryReturnPointer: 2, models.CategoryOther: 1},
		OtherCount: 1, CompileErrors: 3,
		Reproducibility: &service.ReproducibilityCheck{ComparedTo: "results/escape/M-1/y.json", Differing: []string{"a", "b"}, Violation: true},
	}
	got := RenderEscape(summary)
	for _, needle := range []string{"RETURN_POINTER   2", "OTHER            1", "COMPILE_ERROR : 3", "VIOLATION de NFR-002", "a, b"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de :\n%s", needle, got)
		}
	}
	sans := RenderEscape(service.EscapeReportSummary{MatrixID: "M-1", Counts: map[models.EscapeCategory]int{}})
	if strings.Contains(sans, "Reproductibilité") {
		t.Fatalf("sans comparaison antérieure, aucune ligne de reproductibilité : %q", sans)
	}
}

func TestRenderCampaign(t *testing.T) {
	t.Parallel()
	got := RenderCampaign(service.CampaignReport{
		CampaignID: "C-1", Status: models.CampaignCompleted, HarnessDigest: "h", HypothesesDigest: "d",
		HypothesisIDs: []string{"H-001", "H-002"}, CellCount: 8, ProbeCount: 2, Measured: 9, Failed: 1,
		Resumed: true, Duration: 90 * time.Second,
	})
	for _, needle := range []string{"C-1 — statut : COMPLETED", "H-001, H-002", "Cell : 8 · Probe : 2", "9 / 1", "Reprise", "1m30s"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de :\n%s", needle, got)
		}
	}
	aborted := RenderCampaign(service.CampaignReport{CampaignID: "C-2", Status: models.CampaignAborted, AbortReason: "harnais modifié"})
	if !strings.Contains(aborted, "Abandon : harnais modifié") {
		t.Fatalf("A2 doit être visible : %q", aborted)
	}
}

func TestRenderComparison(t *testing.T) {
	t.Parallel()
	report := service.ComparisonReport{
		CampaignID: "C-1", Path: "results/campaigns/C-1/comparison-x.json",
		Set: models.ComparisonSet{
			Method: "bootstrap",
			Comparisons: []models.Comparison{
				{SizeBytes: 24, Profile: models.ProfileLocal, DeltaNsPerOp: -1, CILow: -2, CIHigh: -0.5, Significant: true},
				{SizeBytes: 8, Profile: models.ProfileLocal, DeltaNsPerOp: 1, CILow: 0.5, CIHigh: 2, Significant: true},
				{SizeBytes: 8, Profile: models.ProfileLocal, HasPointerField: true, DeltaNsPerOp: 0, CILow: -1, CIHigh: 1},
			},
			TippingPoints: map[models.TippingKey]int{
				{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill, HasPointerField: false}: 24,
				{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill, HasPointerField: true}:  models.TippingNotObserved,
			},
			ExcludedPairs: []models.ExcludedPair{{ValueCellID: "v", PointerCellID: "p", Reason: "FAILED"}},
		},
		ExcludedCount: 1,
	}
	got := RenderComparison(report)
	for _, needle := range []string{"LOCAL, ARRAY_FILL, sans champ pointeur", "LOCAL, ARRAY_FILL, avec champ pointeur",
		"Point de bascule : 24 octets", "Point de bascule : non observé", "Paires exclues : 1", "v / p : FAILED"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de :\n%s", needle, got)
		}
	}
	// Les tailles sont présentées en ordre croissant.
	if strings.Index(got, "  8 ") > strings.Index(got, "  24 ") {
		t.Fatalf("les tailles doivent être triées :\n%s", got)
	}
}

func TestRenderVerdicts(t *testing.T) {
	t.Parallel()
	summary := service.VerdictReportSummary{
		CampaignID: "C-1", Path: "results/verdicts/x.json", DashboardPath: "docs/dashboard.md",
		Report: models.VerdictReport{Verdicts: []models.Verdict{
			{HypothesisID: "H-001", Outcome: models.OutcomeRefuted, Rationale: "raison"},
			{HypothesisID: "H-004", Outcome: models.OutcomeInconclusive, Rationale: "données manquantes"},
		}},
		Inconclusive: []models.Verdict{{HypothesisID: "H-004", Rationale: "données manquantes"}},
	}
	got := RenderVerdicts(summary)
	for _, needle := range []string{"| H-001 | REFUTED | raison |", "H-004 : données manquantes", "docs/dashboard.md"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de :\n%s", needle, got)
		}
	}
	aucune := RenderVerdicts(service.VerdictReportSummary{CampaignID: "C-1"})
	if !strings.Contains(aucune, "INCONCLUSIVE : aucune") {
		t.Fatalf("l'absence d'hypothèse non concluante doit être dite : %q", aucune)
	}
}

func TestFirstLineEtTruncateList(t *testing.T) {
	t.Parallel()
	if got := firstLine("\n\n  première\nseconde"); got != "première" {
		t.Fatalf("firstLine = %q", got)
	}
	if got := firstLine("   "); got != "   " {
		t.Fatalf("firstLine = %q", got)
	}
	if got := truncateList([]string{"a", "b"}, 5); len(got) != 2 {
		t.Fatalf("truncateList = %v", got)
	}
	if got := truncateList([]string{"a", "b", "c"}, 2); len(got) != 3 || got[2] != "…" {
		t.Fatalf("truncateList = %v", got)
	}
}

// TestUC003_BenchTimeValide verrouille A-156 : la durée de mesure n'était validée nulle part. Une
// faute de frappe était transmise telle quelle à `go test`, qui refusait chaque sujet : la
// campagne entière se consignait en FAILED, sans qu'aucun contrôle n'ait eu lieu en amont.
func TestUC003_BenchTimeValide(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"250ms": true, "1s": true, "1m30s": true, "100x": true,
		"":     false, // le drapeau est requis par C-003
		"250":  false, // une durée Go porte son unité
		"1min": false, // faute de frappe pour 1m
		"-1s":  false,
		"0s":   false,
		"0x":   false,
		"abcx": false,
	}
	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			err := ValidateBenchTime(value)
			if valid && err != nil {
				t.Fatalf("ValidateBenchTime(%q) = %v, acceptée attendue", value, err)
			}
			if !valid {
				if err == nil {
					t.Fatalf("ValidateBenchTime(%q) acceptée, refus attendu", value)
				}
				if !errors.Is(err, ErrUsage) {
					t.Fatalf("erreur = %v, ErrUsage attendue", err)
				}
			}
		})
	}
}

// TestUC001_A1_CleRepetee verrouille A-086 : une clé répétée écrasait la précédente sans rien
// dire, produisant une matrice que le chercheur n'a pas demandée sous un identifiant qu'il
// n'attend pas.
func TestUC001_A1_CleRepetee(t *testing.T) {
	t.Parallel()
	_, err := ParseParameters("sizes=8;sizes=16")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("erreur = %v, ErrUsage attendue", err)
	}
}

// TestUC003_BR5_HypothesesDedoublonnees verrouille A-087 : l'empreinte des critères est calculée
// sur la liste, doublons compris ; `H-001,H-001` produisait donc une empreinte différente de
// `H-001` et deux verdicts pour la même hypothèse dans un même rapport.
func TestUC003_BR5_HypothesesDedoublonnees(t *testing.T) {
	t.Parallel()
	got := ParseHypotheses("H-001, H-002 ,H-001")
	want := []string{"H-001", "H-002"}
	if len(got) != len(want) {
		t.Fatalf("ParseHypotheses = %v, %v attendu", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseHypotheses = %v, %v attendu", got, want)
		}
	}
}
