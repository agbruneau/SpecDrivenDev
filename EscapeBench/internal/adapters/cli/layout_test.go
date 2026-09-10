package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

func TestParseParametersDimensionsDeC008(t *testing.T) {
	t.Parallel()
	params, err := ParseParameters("sizes=24;layouts=NAMED_FIELDS,NAMED_FIELDS_SHAM;repeats=1,2,4,16;probes=POINTER_CHASE:16384")
	if err != nil {
		t.Fatalf("ParseParameters : %v", err)
	}
	if !reflect.DeepEqual(params.Layouts, []models.Layout{models.LayoutNamedFields, models.LayoutNamedFieldsSham}) {
		t.Fatalf("dispositions = %v", params.Layouts)
	}
	if !reflect.DeepEqual(params.Repeats, []int{1, 2, 4, 16}) {
		t.Fatalf("répétitions = %v", params.Repeats)
	}
	if len(params.Probes) != 1 || params.Probes[0].Kind != models.ProbePointerChase {
		t.Fatalf("sondes = %v", params.Probes)
	}
}

func TestParseParametersDimensionsInvalides(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"répétition illisible": "sizes=24;repeats=quatre",
		"clé inconnue citée":   "sizes=24;dispositions=NAMED_FIELDS",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseParameters(spec)
			if err == nil || !errors.Is(err, ErrUsage) {
				t.Fatalf("erreur = %v, ErrUsage attendue", err)
			}
		})
	}
	// Le message d'usage doit nommer les clés reconnues, y compris celles de C-008.
	_, err := ParseParameters("sizes=24;inconnue=1")
	if err == nil || !strings.Contains(err.Error(), "layouts") || !strings.Contains(err.Error(), "repeats") {
		t.Fatalf("le message doit citer les nouvelles clés : %v", err)
	}
	// Une disposition inconnue est rejetée à la validation, avec son nom.
	params, err := ParseParameters("sizes=24;layouts=PACKED")
	if err != nil {
		t.Fatalf("ParseParameters : %v", err)
	}
	if err := params.Validate(); err == nil || !strings.Contains(err.Error(), "PACKED") {
		t.Fatalf("Validate = %v", err)
	}
}

func TestRenderProvenanceAvecTaillesMemoire(t *testing.T) {
	t.Parallel()
	p := models.Provenance{
		GoVersion: "go1.27.0", GOOS: "windows", GOARCH: "amd64", CPUModel: "Core Ultra",
		L1DataCacheBytes: 49152, LastLevelCacheBytes: 36 << 20, PageSizeBytes: 4096, GOMAXPROCS: 24,
		CapturedAt: time.Date(2026, 9, 10, 16, 0, 0, 0, time.UTC),
	}
	got := RenderProvenance(p)
	for _, needle := range []string{"L1d 48 Ko", "dernier niveau 36 Mo", "page 4 Ko", "GOMAXPROCS 24"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("%q absent de %q", needle, got)
		}
	}
	// Une provenance antérieure à C-008 ne doit rien afficher de ces champs.
	ancienne := models.Provenance{GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "cpu", CapturedAt: p.CapturedAt}
	if strings.Contains(RenderProvenance(ancienne), "L1d") {
		t.Fatalf("aucune taille ne doit être inventée : %q", RenderProvenance(ancienne))
	}
}

func TestHumanBytes(t *testing.T) {
	t.Parallel()
	cases := map[int64]string{
		4096:        "4 Ko",
		49152:       "48 Ko",
		36 << 20:    "36 Mo",
		1000:        "1000 o",
		1<<20 + 512: "1049088 o",
	}
	for value, want := range cases {
		if got := humanBytes(value); got != want {
			t.Fatalf("humanBytes(%d) = %q, %q attendu", value, got, want)
		}
	}
}
