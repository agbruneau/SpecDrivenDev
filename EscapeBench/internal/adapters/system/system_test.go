package system

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClock(t *testing.T) {
	t.Parallel()
	if (Clock{}).Now().Location() != time.UTC {
		t.Fatal("l'horloge système rend l'heure en UTC")
	}
	instant := time.Date(2026, 9, 10, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))
	if got := (FixedClock{Instant: instant}).Now(); !got.Equal(instant) || got.Location() != time.UTC {
		t.Fatalf("FixedClock.Now = %v", got)
	}
}

func TestCaptureProvenance(t *testing.T) {
	t.Parallel()
	instant := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	prober := NewProber(FixedClock{Instant: instant}, func() string { return "  Un CPU  " })
	provenance, err := prober.Capture(context.Background())
	if err != nil {
		t.Fatalf("Capture : %v", err)
	}
	if provenance.CPUModel != "Un CPU" {
		t.Fatalf("cpuModel = %q", provenance.CPUModel)
	}
	if !provenance.CapturedAt.Equal(instant) {
		t.Fatalf("capturedAt = %v", provenance.CapturedAt)
	}
	// NFR-001 : aucun champ n'est laissé vide.
	// Mutation : rendre CPUModel vide sans repli ⇒ échec attendu.
	if err := provenance.Validate(); err != nil {
		t.Fatalf("provenance incomplète : %v", err)
	}
	if !strings.HasPrefix(provenance.GoVersion, "go") {
		t.Fatalf("goVersion = %q", provenance.GoVersion)
	}
}

func TestCaptureAvecCPUInconnu(t *testing.T) {
	t.Parallel()
	prober := NewProber(nil, func() string { return "   " })
	provenance, err := prober.Capture(context.Background())
	if err != nil {
		t.Fatalf("Capture : %v", err)
	}
	if provenance.CPUModel != "inconnu" {
		t.Fatalf("cpuModel = %q, « inconnu » attendu", provenance.CPUModel)
	}
	if err := provenance.Validate(); err != nil {
		t.Fatalf("NFR-001 : %v", err)
	}
}

func TestDetectCPUModel(t *testing.T) {
	t.Parallel()
	// La détection réelle ne rend jamais la chaîne vide, quelle que soit la plateforme.
	if strings.TrimSpace(DetectCPUModel()) == "" {
		t.Fatal("DetectCPUModel ne doit jamais rendre une chaîne vide")
	}
	prober := NewProber(nil, nil)
	if _, err := prober.Capture(context.Background()); err != nil {
		t.Fatalf("Capture avec détection réelle : %v", err)
	}
}

func TestCPUModelFromCPUInfo(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"processor\t: 0\nmodel name\t: AMD Ryzen 9\nflags\t: fpu\n": "AMD Ryzen 9",
		"Processor\t: ARMv8\nHardware\t: BCM2835\n":                 "BCM2835",
		"processor\t: 0\n":              "",
		"aucune ligne avec deux-points": "",
	}
	for content, want := range cases {
		if got := cpuModelFromCPUInfo(content); got != want {
			t.Fatalf("cpuModelFromCPUInfo(%q) = %q, attendu %q", content, got, want)
		}
	}
}
