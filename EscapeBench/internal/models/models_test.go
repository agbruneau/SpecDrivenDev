package models

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestEnumsValid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		valid bool
		got   bool
	}{
		{"PassingValue", true, PassingValue.Valid()},
		{"PassingPointer", true, PassingPointer.Valid()},
		{"passingInconnu", false, PassingMode("BOTH").Valid()},
		{"ProfileLocal", true, ProfileLocal.Valid()},
		{"ProfileStoredInMap", true, ProfileStoredInMap.Valid()},
		{"profilInconnu", false, LifetimeProfile("GLOBAL").Valid()},
		{"ProbeScatteredScan", true, ProbeScatteredScan.Valid()},
		{"probeInconnue", false, ProbeKind("RANDOM").Valid()},
		{"CategoryOther", true, CategoryOther.Valid()},
		{"catégorieInconnue", false, EscapeCategory("INTERFACE").Valid()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.valid {
				t.Fatalf("Valid() = %v, attendu %v", tc.got, tc.valid)
			}
		})
	}
	// BR-001-4 : la matrice de référence garde ses cinq profils, indépendamment des profils que
	// C-008 ajoute au catalogue. Mutation : rendre LifetimeProfiles() dans ReferenceParameters
	// ⇒ échec attendu, la matrice passerait de 220 à 352 cellules.
	if len(ReferenceProfiles()) != 5 {
		t.Fatalf("cinq profils de référence attendus, %d obtenus", len(ReferenceProfiles()))
	}
	if len(LifetimeProfiles()) != 8 {
		t.Fatalf("huit profils connus attendus, %d obtenus", len(LifetimeProfiles()))
	}
	for _, profile := range ReferenceProfiles() {
		if !profile.Valid() {
			t.Fatalf("%s doit rester un profil connu", profile)
		}
	}
	if len(ProbeKinds()) != 5 {
		t.Fatalf("cinq genres de sonde attendus, %d obtenus", len(ProbeKinds()))
	}
	if len(Layouts()) != 3 {
		t.Fatalf("trois dispositions attendues, %d obtenues", len(Layouts()))
	}
	for _, layout := range Layouts() {
		if !layout.Valid() {
			t.Fatalf("%s doit être une disposition connue", layout)
		}
	}
	if Layout("PACKED").Valid() {
		t.Fatal("une disposition inconnue ne doit pas être valide")
	}
	// Seules les dispositions à champs nommés restent assignables aux registres à toute taille.
	if LayoutArrayFill.RegisterAssignable() {
		t.Fatal("un tableau de remplissage sort le type des registres")
	}
	if !LayoutNamedFields.RegisterAssignable() || !LayoutNamedFieldsSham.RegisterAssignable() {
		t.Fatal("les dispositions à champs nommés restent assignables aux registres")
	}
}

func TestEscapeCategoryInBook(t *testing.T) {
	t.Parallel()
	// H-006 porte sur l'exhaustivité des quatre causes du livre : OTHER en est exclue par
	// construction, sinon l'hypothèse ne pourrait jamais être infirmée.
	// Mutation : ajouter CategoryOther à BookCategories ⇒ échec attendu.
	if len(BookCategories()) != 4 {
		t.Fatalf("quatre causes attendues, %d obtenues", len(BookCategories()))
	}
	for _, category := range BookCategories() {
		if !category.InBook() {
			t.Fatalf("%s devrait être dans le livre", category)
		}
	}
	for _, category := range []EscapeCategory{CategoryOther, CategoryNone} {
		if category.InBook() {
			t.Fatalf("%s ne devrait pas être dans le livre", category)
		}
	}
}

func TestValidateSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"borne basse", 8, false},
		{"borne haute", 4096, false},
		{"multiple", 24, false},
		{"trop petit", 4, true},
		{"trop grand", 4104, true},
		{"non multiple de 8", 20, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSize(tc.size)
			if tc.wantErr != (err != nil) {
				t.Fatalf("ValidateSize(%d) = %v, erreur attendue : %v", tc.size, err, tc.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Fatalf("erreur non enveloppée dans ErrValidation : %v", err)
			}
		})
	}
}

func TestTypeSpec(t *testing.T) {
	t.Parallel()
	spec, err := NewTypeSpec(24, false, LayoutArrayFill)
	if err != nil {
		t.Fatalf("NewTypeSpec : %v", err)
	}
	if spec.Name != "Size0024Plain" {
		t.Fatalf("nom = %q", spec.Name)
	}
	if spec.WordCount() != 3 {
		t.Fatalf("wordCount = %d, attendu 3", spec.WordCount())
	}
	withPointer, err := NewTypeSpec(24, true, LayoutArrayFill)
	if err != nil {
		t.Fatalf("NewTypeSpec : %v", err)
	}
	if withPointer.Name == spec.Name {
		t.Fatal("les deux variantes doivent avoir des noms distincts")
	}
	if _, err := NewTypeSpec(20, false, LayoutArrayFill); err == nil {
		t.Fatal("une taille non multiple de 8 doit être refusée")
	}
	if err := (TypeSpec{SizeBytes: 24, Layout: LayoutArrayFill}).Validate(); err == nil {
		t.Fatal("un TypeSpec sans nom doit être refusé")
	}
}

func TestCellAndProbeIdentifiers(t *testing.T) {
	t.Parallel()
	spec, _ := NewTypeSpec(64, true, LayoutArrayFill)
	cell := Cell{TypeSpec: spec, Profile: ProfileSentOnChannel, PassingMode: PassingPointer, SourceFile: "subjects/x/subject.go"}
	if got, want := cell.ID(), "Size0064Ptr/SENT_ON_CHANNEL/POINTER"; got != want {
		t.Fatalf("Cell.ID() = %q, attendu %q", got, want)
	}
	if err := cell.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	probe := Probe{Kind: ProbeAppendGrow, Parameter: 100000, SourceFile: "subjects/y/subject.go"}
	if got, want := probe.ID(), "probe/APPEND_GROW/100000"; got != want {
		t.Fatalf("Probe.ID() = %q, attendu %q", got, want)
	}
	// Le deuxième segment d'un identifiant de Probe n'est jamais un code de LifetimeProfile.
	if LifetimeProfile(strings.Split(probe.ID(), "/")[1]).Valid() {
		t.Fatal("les identifiants de Probe et de Cell doivent rester disjoints")
	}
}

func TestCellValidateRejects(t *testing.T) {
	t.Parallel()
	spec, _ := NewTypeSpec(8, false, LayoutArrayFill)
	cases := map[string]Cell{
		"profil inconnu": {TypeSpec: spec, Profile: "GLOBAL", PassingMode: PassingValue, SourceFile: "x"},
		"mode inconnu":   {TypeSpec: spec, Profile: ProfileLocal, PassingMode: "BOTH", SourceFile: "x"},
		"sans source":    {TypeSpec: spec, Profile: ProfileLocal, PassingMode: PassingValue},
	}
	for name, cell := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := cell.Validate(); err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
		})
	}
}

func TestProbeValidateRejects(t *testing.T) {
	t.Parallel()
	cases := map[string]Probe{
		"genre inconnu":    {Kind: "RANDOM", Parameter: 1, SourceFile: "x"},
		"paramètre nul":    {Kind: ProbeAppendGrow, Parameter: 0, SourceFile: "x"},
		"source manquante": {Kind: ProbeAppendGrow, Parameter: 1},
	}
	for name, probe := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := probe.Validate(); err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
		})
	}
}

func TestEscapeVerdictValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		verdict EscapeVerdict
		wantErr bool
	}{
		{"échappe avec catégorie", EscapeVerdict{CellID: "c", Escapes: true, Category: CategoryReturnPointer, CompilerReason: "r", Status: EscapeStatusOK}, false},
		{"n'échappe pas", EscapeVerdict{CellID: "c", Category: CategoryNone, Status: EscapeStatusOK}, false},
		{"compile error", EscapeVerdict{CellID: "c", Status: EscapeStatusCompileError, CompilerError: "boom"}, false},
		{"sans cellId", EscapeVerdict{Status: EscapeStatusOK, Category: CategoryNone}, true},
		{"compile error sans message", EscapeVerdict{CellID: "c", Status: EscapeStatusCompileError}, true},
		{"statut inconnu", EscapeVerdict{CellID: "c", Status: "MAYBE"}, true},
		{"catégorie inconnue", EscapeVerdict{CellID: "c", Status: EscapeStatusOK, Category: "INTERFACE"}, true},
		{"échappe en NONE", EscapeVerdict{CellID: "c", Escapes: true, Category: CategoryNone, Status: EscapeStatusOK}, true},
		{"n'échappe pas mais catégorisé", EscapeVerdict{CellID: "c", Category: CategoryChannelSend, Status: EscapeStatusOK}, true},
		{"n'échappe pas mais motivé", EscapeVerdict{CellID: "c", Category: CategoryNone, CompilerReason: "r", Status: EscapeStatusOK}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.verdict.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate = %v, erreur attendue : %v", err, tc.wantErr)
			}
		})
	}
}

func TestEscapeReportCounts(t *testing.T) {
	t.Parallel()
	report := EscapeReport{Verdicts: []EscapeVerdict{
		{CellID: "a", Escapes: true, Category: CategoryReturnPointer, Status: EscapeStatusOK},
		{CellID: "b", Escapes: true, Category: CategoryOther, Status: EscapeStatusOK},
		{CellID: "c", Category: CategoryNone, Status: EscapeStatusOK},
		{CellID: "d", Status: EscapeStatusCompileError, CompilerError: "boom"},
	}}
	counts := report.CountByCategory()
	if counts[CategoryReturnPointer] != 1 || counts[CategoryOther] != 1 || counts[CategoryNone] != 1 {
		t.Fatalf("décompte inattendu : %v", counts)
	}
	if report.CountCompileErrors() != 1 {
		t.Fatalf("CountCompileErrors = %d", report.CountCompileErrors())
	}
}

func TestProvenance(t *testing.T) {
	t.Parallel()
	complete := Provenance{GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "x", CapturedAt: time.Unix(1, 0)}
	if err := complete.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	for name, mutate := range map[string]func(*Provenance){
		"goVersion":  func(p *Provenance) { p.GoVersion = "" },
		"goos":       func(p *Provenance) { p.GOOS = "" },
		"goarch":     func(p *Provenance) { p.GOARCH = "" },
		"cpuModel":   func(p *Provenance) { p.CPUModel = "" },
		"capturedAt": func(p *Provenance) { p.CapturedAt = time.Time{} },
	} {
		t.Run("sans "+name, func(t *testing.T) {
			t.Parallel()
			incomplete := complete
			mutate(&incomplete)
			if err := incomplete.Validate(); err == nil {
				t.Fatal("NFR-001 : une provenance incomplète doit être refusée")
			}
		})
	}
	// UC-002 A3 : seuls goVersion, goos et goarch identifient la toolchain.
	// Mutation : comparer aussi capturedAt ⇒ échec attendu.
	other := complete
	other.CapturedAt = time.Unix(999, 0)
	other.CPUModel = "autre"
	if !complete.SameToolchain(other) {
		t.Fatal("capturedAt et cpuModel ne participent pas à l'identité de la toolchain")
	}
	other.GOARCH = "arm64"
	if complete.SameToolchain(other) {
		t.Fatal("une architecture différente est une autre toolchain")
	}
}

func TestCampaignValidate(t *testing.T) {
	t.Parallel()
	provenance := Provenance{GoVersion: "g", GOOS: "o", GOARCH: "a", CPUModel: "c", CapturedAt: time.Unix(1, 0)}
	base := Campaign{ID: "C-1", MatrixID: "M-1", HarnessDigest: "h", HypothesesDigest: "d",
		HypothesisIDs: []string{"H-001"}, Count: MinCount, Status: CampaignRunning,
		Provenance: provenance, StartedAt: time.Unix(1, 0)}
	if err := base.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	for name, mutate := range map[string]func(*Campaign){
		"sans id":               func(c *Campaign) { c.ID = "" },
		"sans matrixId":         func(c *Campaign) { c.MatrixID = "" },
		"sans harnessDigest":    func(c *Campaign) { c.HarnessDigest = "" },
		"sans hypothesesDigest": func(c *Campaign) { c.HypothesesDigest = "" },
		"count insuffisant":     func(c *Campaign) { c.Count = MinCount - 1 },
		"statut inconnu":        func(c *Campaign) { c.Status = "PAUSED" },
		"provenance vide":       func(c *Campaign) { c.Provenance = Provenance{} },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			invalid := base
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
		})
	}
}

func TestMeasurementValidateAndMedians(t *testing.T) {
	t.Parallel()
	complete := Measurement{
		CampaignID: "C-1", SubjectID: "s", Status: MeasurementComplete,
		NsPerOp: []float64{3, 1, 2}, BytesPerOp: []int64{10, 30, 20}, AllocsPerOp: []int64{1, 1, 1},
	}
	if err := complete.Validate(3); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	if err := complete.Validate(4); err == nil {
		t.Fatal("NFR-003 : le nombre de valeurs doit être exactement count")
	}
	if got := complete.MedianNs(); got != 2 {
		t.Fatalf("MedianNs = %v, attendu 2", got)
	}
	if got := complete.MedianBytes(); got != 20 {
		t.Fatalf("MedianBytes = %v, attendu 20", got)
	}
	if got := complete.MedianAllocs(); got != 1 {
		t.Fatalf("MedianAllocs = %v, attendu 1", got)
	}
	failed := Measurement{CampaignID: "C-1", SubjectID: "s", Status: MeasurementFailed, FailureReason: "boom"}
	if err := failed.Validate(3); err != nil {
		t.Fatalf("une mesure FAILED sans valeurs est valide : %v", err)
	}
	if err := (Measurement{CampaignID: "C", SubjectID: "s", Status: MeasurementFailed}).Validate(3); err == nil {
		t.Fatal("une mesure FAILED sans raison doit être refusée")
	}
	withValues := failed
	withValues.NsPerOp = []float64{1}
	if err := withValues.Validate(3); err == nil {
		t.Fatal("une mesure FAILED ne porte aucune valeur")
	}
	if err := (Measurement{Status: MeasurementComplete}).Validate(0); err == nil {
		t.Fatal("campaignId et subjectId sont requis")
	}
	if err := (Measurement{CampaignID: "C", SubjectID: "s", Status: "PARTIAL"}).Validate(0); err == nil {
		t.Fatal("statut inconnu")
	}
}

func TestMedians(t *testing.T) {
	t.Parallel()
	if got := MedianFloat(nil); got != 0 {
		t.Fatalf("MedianFloat(nil) = %v", got)
	}
	if got := MedianInt(nil); got != 0 {
		t.Fatalf("MedianInt(nil) = %v", got)
	}
	if got := MedianFloat([]float64{4, 1, 3, 2}); got != 2.5 {
		t.Fatalf("médiane paire = %v, attendu 2.5", got)
	}
	source := []float64{3, 1, 2}
	_ = MedianFloat(source)
	if source[0] != 3 {
		t.Fatal("MedianFloat ne doit pas trier l'échantillon d'origine")
	}
}

func TestHypothesisAndVerdictValidate(t *testing.T) {
	t.Parallel()
	hypothesis := Hypothesis{ID: "H-001", SourcePages: "p. 1", Statement: "énoncé", RefutationCriterion: "critère"}
	if err := hypothesis.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	for name, mutate := range map[string]func(*Hypothesis){
		"sans id":      func(h *Hypothesis) { h.ID = "" },
		"sans pages":   func(h *Hypothesis) { h.SourcePages = "" },
		"sans critère": func(h *Hypothesis) { h.RefutationCriterion = "" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			invalid := hypothesis
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
		})
	}
	verdict := Verdict{HypothesisID: "H-001", CampaignID: "C-1", Outcome: OutcomeConfirmed, Rationale: "r", ResultFiles: []string{"f"}}
	if err := verdict.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	for name, mutate := range map[string]func(*Verdict){
		"sans hypothèse": func(v *Verdict) { v.HypothesisID = "" },
		"issue inconnue": func(v *Verdict) { v.Outcome = "MAYBE" },
		"sans rationale": func(v *Verdict) { v.Rationale = "" },
		"sans fichiers":  func(v *Verdict) { v.ResultFiles = nil },
	} {
		t.Run("verdict "+name, func(t *testing.T) {
			t.Parallel()
			invalid := verdict
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("BR-005-2 : Validate aurait dû refuser")
			}
		})
	}
}

// TestUC003_ValidationsDuModeleDEntites verrouille A-005 à A-009 et A-016 : le modèle d'entités
// déclare ces champs requis ou bornés, mais rien ne l'appliquait. Un enregistrement incomplet ou
// aberrant traversait les validations jusque dans `results/`, où plus rien ne le rattrape.
func TestUC003_ValidationsDuModeleDEntites(t *testing.T) {
	t.Parallel()
	prov := Provenance{GoVersion: "g", GOOS: "o", GOARCH: "a", CPUModel: "c", CapturedAt: time.Unix(1, 0)}
	campaign := func() Campaign {
		return Campaign{ID: "C-1", MatrixID: "M-1", HarnessDigest: "h", HypothesesDigest: "d",
			HypothesisIDs: []string{"H-001"}, Count: MinCount, Status: CampaignRunning,
			Provenance: prov, StartedAt: time.Unix(1, 0)}
	}
	t.Run("campagne", func(t *testing.T) {
		t.Parallel()
		for name, mutate := range map[string]func(*Campaign){
			"sans hypothèse gelée": func(c *Campaign) { c.HypothesisIDs = nil },
			"sans startedAt":       func(c *Campaign) { c.StartedAt = time.Time{} },
			"close sans finishedAt": func(c *Campaign) {
				c.Status = CampaignCompleted
			},
			"abandonnée sans raison": func(c *Campaign) {
				c.Status, c.FinishedAt = CampaignAborted, time.Unix(2, 0)
			},
			"en cours avec finishedAt": func(c *Campaign) { c.FinishedAt = time.Unix(2, 0) },
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				c := campaign()
				mutate(&c)
				if err := c.Validate(); err == nil {
					t.Fatalf("Validate accepte une campagne %s", name)
				}
			})
		}
	})

	t.Run("mesure", func(t *testing.T) {
		t.Parallel()
		complete := func() Measurement {
			return Measurement{CampaignID: "C-1", SubjectID: "s", Status: MeasurementComplete,
				NsPerOp: []float64{1}, BytesPerOp: []int64{8}, AllocsPerOp: []int64{1}}
		}
		for name, mutate := range map[string]func(*Measurement){
			"durée négative":        func(m *Measurement) { m.NsPerOp = []float64{-1} },
			"durée infinie":         func(m *Measurement) { m.NsPerOp = []float64{math.Inf(1)} },
			"durée NaN":             func(m *Measurement) { m.NsPerOp = []float64{math.NaN()} },
			"octets négatifs":       func(m *Measurement) { m.BytesPerOp = []int64{-8} },
			"allocations négatives": func(m *Measurement) { m.AllocsPerOp = []int64{-1} },
			"occupation > 1": func(m *Measurement) {
				m.QuietudeMeasured, m.QuietudeOccupancy = true, 1.5
			},
			"occupation négative": func(m *Measurement) {
				m.QuietudeMeasured, m.QuietudeOccupancy = true, -0.1
			},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				m := complete()
				mutate(&m)
				if err := m.Validate(1); err == nil {
					t.Fatalf("Validate accepte une mesure avec %s", name)
				}
			})
		}
		// Une occupation nulle est une valeur légitime, pas une absence (C-010).
		m := complete()
		m.QuietudeMeasured = true
		if err := m.Validate(1); err != nil {
			t.Fatalf("une occupation nulle est légitime : %v", err)
		}
	})

	t.Run("verdict d'échappement", func(t *testing.T) {
		t.Parallel()
		v := EscapeVerdict{CellID: "c", Status: EscapeStatusOK, Category: CategoryNone,
			CompilerError: "un message qui n'a pas lieu d'être"}
		if err := v.Validate(); err == nil {
			t.Fatal("compilerError n'a de sens qu'en COMPILE_ERROR (BR-002-1)")
		}
	})

	t.Run("verdict d'hypothèse", func(t *testing.T) {
		t.Parallel()
		v := Verdict{HypothesisID: "H-001", Outcome: OutcomeConfirmed,
			Rationale: "parce que", ResultFiles: []string{"results/x.json"}}
		if err := v.Validate(); err == nil {
			t.Fatal("un verdict sans campaignId ne peut plus être rattaché à ses mesures (BR-005-2)")
		}
	})

	t.Run("hypothèse", func(t *testing.T) {
		t.Parallel()
		h := Hypothesis{ID: "H-001", SourcePages: "p. 1", RefutationCriterion: "critère"}
		if err := h.Validate(); err == nil {
			t.Fatal("l'énoncé entre dans l'empreinte gelée : il est requis (BR-003-5)")
		}
	})
}
