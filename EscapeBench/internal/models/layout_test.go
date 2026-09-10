package models

import (
	"strings"
	"testing"
	"time"
)

func TestReferenceMatrixIDInchangeParC008(t *testing.T) {
	t.Parallel()
	// Garantie de compatibilité : les dimensions ajoutées par C-008 ne changent pas la
	// représentation canonique tant qu'elles gardent leur valeur par défaut, donc l'identifiant
	// de la matrice de référence reste celui des campagnes antérieures (BR-001-1).
	// Mutation : écrire inconditionnellement les lignes layouts= et repeats= dans Canonical
	// ⇒ échec attendu, et la campagne C-2026-09-10-1 perdrait sa matrice.
	const identifiantHistorique = "M-823d8b5af441"
	if got := ReferenceParameters().MatrixID(); got != identifiantHistorique {
		t.Fatalf("identifiant de la matrice de référence = %s, %s attendu", got, identifiantHistorique)
	}
	// Une demande antérieure à C-008, sans les deux dimensions, donne le même identifiant.
	ancienne := ReferenceParameters()
	ancienne.Layouts = nil
	ancienne.Repeats = nil
	if got := ancienne.MatrixID(); got != identifiantHistorique {
		t.Fatalf("une demande sans les dimensions de C-008 donne %s", got)
	}
	// Demander explicitement les valeurs par défaut ne change rien non plus.
	explicite := ReferenceParameters()
	explicite.Layouts = []Layout{LayoutArrayFill}
	explicite.Repeats = []int{1}
	if got := explicite.MatrixID(); got != identifiantHistorique {
		t.Fatalf("les valeurs par défaut explicites donnent %s", got)
	}
	// Une disposition supplémentaire, elle, produit une matrice différente.
	etendue := ReferenceParameters()
	etendue.Layouts = []Layout{LayoutArrayFill, LayoutNamedFields}
	if got := etendue.MatrixID(); got == identifiantHistorique {
		t.Fatal("une dimension de disposition élargie doit produire un identifiant différent")
	}
}

func TestReferenceMatrixDecompteInchange(t *testing.T) {
	t.Parallel()
	// BR-001-4 reste satisfaite après C-008 : 220 Cell et 10 Probe.
	cells, probes, err := ReferenceParameters().Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	if len(cells) != 220 || len(probes) != 10 {
		t.Fatalf("%d Cell et %d Probe, 220 et 10 attendues", len(cells), len(probes))
	}
	for _, cell := range cells {
		if cell.TypeSpec.Layout != LayoutArrayFill {
			t.Fatalf("la matrice de référence ne porte que la disposition d'origine : %s", cell.ID())
		}
		if cell.Repetitions() != 1 {
			t.Fatalf("la matrice de référence ne porte qu'une instance par opération : %s", cell.ID())
		}
	}
}

func TestTypeSpecNameParDisposition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size    int
		pointer bool
		layout  Layout
		want    string
	}{
		{24, false, LayoutArrayFill, "Size0024Plain"},
		{24, true, LayoutArrayFill, "Size0024Ptr"},
		{24, false, LayoutNamedFields, "Size0024FieldsPlain"},
		{24, true, LayoutNamedFields, "Size0024FieldsPtr"},
		{24, false, LayoutNamedFieldsSham, "Size0024ShamPlain"},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		got := TypeSpecName(tc.size, tc.pointer, tc.layout)
		if got != tc.want {
			t.Fatalf("TypeSpecName(%d, %v, %s) = %q, %q attendu", tc.size, tc.pointer, tc.layout, got, tc.want)
		}
		// Sans marqueur de disposition, les trois dispositions entreraient en collision.
		if seen[got] {
			t.Fatalf("nom en double : %q", got)
		}
		seen[got] = true
	}
	if _, err := NewTypeSpec(24, false, "PACKED"); err == nil {
		t.Fatal("une disposition inconnue doit être refusée")
	}
}

func TestCellRepeatDansLIdentifiant(t *testing.T) {
	t.Parallel()
	spec, err := NewTypeSpec(24, true, LayoutNamedFields)
	if err != nil {
		t.Fatalf("NewTypeSpec : %v", err)
	}
	// Une instance par opération ne laisse aucune trace : les identifiants antérieurs à C-008
	// sont inchangés. Mutation : suffixer même à un ⇒ échec attendu.
	simple := Cell{TypeSpec: spec, Profile: ProfileReturnedAlloc, PassingMode: PassingValue, Repeat: 1, SourceFile: "x"}
	if got, want := simple.ID(), "Size0024FieldsPtr/RETURNED_ALLOCATING/VALUE"; got != want {
		t.Fatalf("ID = %q, %q attendu", got, want)
	}
	repete := simple
	repete.Repeat = 4
	if got, want := repete.ID(), "Size0024FieldsPtr/RETURNED_ALLOCATING_R4/VALUE"; got != want {
		t.Fatalf("ID = %q, %q attendu", got, want)
	}
	// Le suffixe ne doit pas se confondre avec un identifiant de sonde.
	if _, _, ok := ParseProbeID(repete.ID()); ok {
		t.Fatal("un identifiant de cellule ne doit jamais se lire comme une sonde")
	}
	if simple.Repetitions() != 1 || repete.Repetitions() != 4 {
		t.Fatalf("Repetitions = %d et %d", simple.Repetitions(), repete.Repetitions())
	}
	// Une cellule sans répétition explicite compte pour une.
	sansRepeat := Cell{TypeSpec: spec, Profile: ProfileLocal, PassingMode: PassingValue, SourceFile: "x"}
	if sansRepeat.Repetitions() != 1 {
		t.Fatalf("Repetitions = %d", sansRepeat.Repetitions())
	}
	if err := sansRepeat.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	negative := sansRepeat
	negative.Repeat = -1
	if err := negative.Validate(); err == nil {
		t.Fatal("une répétition négative doit être refusée")
	}
}

func TestTemoinNulSeulementEnLocal(t *testing.T) {
	t.Parallel()
	// Le témoin nul mesure l'écart entre deux binaires sur une paire dont le delta vrai est nul :
	// hors du profil LOCAL, la notion n'a pas de sens.
	spec, _ := NewTypeSpec(24, false, LayoutNamedFieldsSham)
	local := Cell{TypeSpec: spec, Profile: ProfileLocal, PassingMode: PassingPointer, SourceFile: "x"}
	if err := local.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	ailleurs := local
	ailleurs.Profile = ProfileReturned
	if err := ailleurs.Validate(); err == nil {
		t.Fatal("le témoin nul hors du profil LOCAL doit être refusé")
	}
	// Révision du 2026-09-10 : la combinaison n'est plus refusée à la validation, elle est sautée à
	// la génération. Une matrice qui demande le témoin nul et d'autres profils est légale et ne
	// produit simplement aucune cellule témoin hors LOCAL.
	params := MatrixParameters{
		Sizes: []int{24}, PointerFieldVariants: []bool{false},
		Profiles: []LifetimeProfile{ProfileLocal, ProfileReturned}, PassingModes: PassingModes(),
		Layouts: []Layout{LayoutNamedFieldsSham},
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("la combinaison est légale depuis que le refus est devenu un saut : %v", err)
	}
	cells, _, err := params.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	if len(cells) == 0 {
		t.Fatal("le profil LOCAL doit produire ses cellules témoins")
	}
	for _, c := range cells {
		if c.Profile != ProfileLocal {
			t.Fatalf("aucune cellule témoin ne doit exister hors du profil LOCAL : %s", c.ID())
		}
	}
}

func TestExpandAvecDispositionsEtRepetitions(t *testing.T) {
	t.Parallel()
	params := MatrixParameters{
		Sizes:                []int{24},
		PointerFieldVariants: []bool{true},
		Profiles:             []LifetimeProfile{ProfileLocal, ProfileReturnedAlloc},
		PassingModes:         PassingModes(),
		Layouts:              []Layout{LayoutArrayFill, LayoutNamedFields},
		Repeats:              []int{1, 4},
	}
	cells, _, err := params.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	// Seul le profil qui produit plusieurs instances par opération se décline en répétitions :
	// ailleurs, une répétition supérieure à un ne créerait que des doublons.
	// Mutation : décliner tous les profils par répétition ⇒ échec attendu.
	byProfile := map[string]int{}
	for _, cell := range cells {
		byProfile[cell.ProfileSegment()]++
	}
	expected := map[string]int{
		"LOCAL":                  4, // 2 dispositions × 2 modes
		"RETURNED_ALLOCATING":    4,
		"RETURNED_ALLOCATING_R4": 4,
	}
	for segment, want := range expected {
		if byProfile[segment] != want {
			t.Fatalf("%s : %d cellules, %d attendues (%v)", segment, byProfile[segment], want, byProfile)
		}
	}
	if len(byProfile) != len(expected) {
		t.Fatalf("segments de profil inattendus : %v", byProfile)
	}
	// Les paires restent complètes et n'apparient jamais deux répétitions différentes.
	matrix, err := NewMatrix(params, "digest", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	for _, pair := range matrix.ValuePointerPairs() {
		if pair[0].Repetitions() != pair[1].Repetitions() {
			t.Fatalf("paire mal appariée : %s / %s", pair[0].ID(), pair[1].ID())
		}
		if pair[0].TypeSpec.Layout != pair[1].TypeSpec.Layout {
			t.Fatalf("paire de dispositions différentes : %s / %s", pair[0].ID(), pair[1].ID())
		}
	}
	if len(matrix.ValuePointerPairs()) != len(cells)/2 {
		t.Fatalf("%d paires pour %d cellules", len(matrix.ValuePointerPairs()), len(cells))
	}
}

func TestValidateSondesEtRepetitions(t *testing.T) {
	t.Parallel()
	base := MatrixParameters{
		Sizes: []int{8}, PointerFieldVariants: []bool{false},
		Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(),
	}
	cases := map[string]struct {
		mutate func(*MatrixParameters)
		want   string
	}{
		"disposition inconnue": {func(p *MatrixParameters) { p.Layouts = []Layout{"PACKED"} }, "disposition"},
		"répétition nulle":     {func(p *MatrixParameters) { p.Repeats = []int{0} }, "instances"},
		"chaîne mal alignée": {func(p *MatrixParameters) {
			p.Probes = []ProbeSpec{{Kind: ProbePointerChase, Parameter: 100}}
		}, "multiple de 64"},
		"parcours mal aligné": {func(p *MatrixParameters) {
			p.Probes = []ProbeSpec{{Kind: ProbeSequentialScan, Parameter: 100}}
		}, "multiple de 64"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := base
			tc.mutate(&params)
			err := params.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate = %v, %q attendu dedans", err, tc.want)
			}
		})
	}
	// Une sonde d'append n'a pas de contrainte d'alignement : son paramètre compte des éléments.
	append1000 := base
	append1000.Probes = []ProbeSpec{{Kind: ProbeAppendGrow, Parameter: 1000}}
	if err := append1000.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
}

func TestProvenanceTaillesMemoireFacultatives(t *testing.T) {
	t.Parallel()
	// Les tailles relevées par C-008 ne sont pas exigées par NFR-001 : un fichier de résultats
	// antérieur reste valide, et l'hypothèse qui en dépend se déclare non concluante.
	ancienne := Provenance{GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "cpu", CapturedAt: time.Unix(1, 0)}
	if err := ancienne.Validate(); err != nil {
		t.Fatalf("une provenance sans tailles mémoire reste valide : %v", err)
	}
	complete := ancienne
	complete.L1DataCacheBytes = 49152
	complete.LastLevelCacheBytes = 36 << 20
	complete.PageSizeBytes = 4096
	complete.GOMAXPROCS = 24
	if err := complete.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
	if !complete.SameToolchain(ancienne) {
		t.Fatal("les tailles mémoire ne participent pas à l'identité de la toolchain")
	}
}
