package models

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func smallParams() MatrixParameters {
	return MatrixParameters{
		Sizes:                []int{8, 24},
		PointerFieldVariants: []bool{false, true},
		Profiles:             []LifetimeProfile{ProfileLocal, ProfileReturned},
		PassingModes:         PassingModes(),
		Probes:               []ProbeSpec{{Kind: ProbeAppendGrow, Parameter: 1000}},
	}
}

func TestReferenceParametersMatchesBR0014(t *testing.T) {
	t.Parallel()
	// BR-001-4 : 11 tailles × 2 variantes × 5 profils × 2 modes = 220 Cell, plus 10 Probe.
	// Mutation : retirer une taille de ReferenceParameters ⇒ échec attendu.
	params := ReferenceParameters()
	cells, probes, err := params.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	if len(cells) != 220 {
		t.Fatalf("%d Cell, 220 attendues (BR-001-4)", len(cells))
	}
	if len(probes) != 10 {
		t.Fatalf("%d Probe, 10 attendues (BR-001-4)", len(probes))
	}
	sizes := map[int]bool{}
	for _, cell := range cells {
		sizes[cell.TypeSpec.SizeBytes] = true
	}
	if !sizes[24] {
		t.Fatal("la matrice de référence doit inclure 24 octets : H-002 porte sur cette borne")
	}
}

func TestMatrixIDDeterministe(t *testing.T) {
	t.Parallel()
	// BR-001-1 : deux demandes de mêmes paramètres normalisés produisent le même identifiant.
	shuffled := MatrixParameters{
		Sizes:                []int{24, 8, 24},
		PointerFieldVariants: []bool{true, false, false},
		Profiles:             []LifetimeProfile{ProfileReturned, ProfileLocal},
		PassingModes:         []PassingMode{PassingPointer, PassingValue},
		Probes:               []ProbeSpec{{Kind: ProbeAppendGrow, Parameter: 1000}, {Kind: ProbeAppendGrow, Parameter: 1000}},
	}
	if shuffled.MatrixID() != smallParams().MatrixID() {
		t.Fatalf("identifiants différents : %s vs %s", shuffled.MatrixID(), smallParams().MatrixID())
	}
	different := smallParams()
	different.Sizes = append(different.Sizes, 32)
	if different.MatrixID() == smallParams().MatrixID() {
		t.Fatal("des paramètres différents doivent produire un identifiant différent")
	}
	if !strings.HasPrefix(smallParams().MatrixID(), "M-") {
		t.Fatalf("identifiant mal formé : %s", smallParams().MatrixID())
	}
}

func TestNormalizeDedupeAndOrder(t *testing.T) {
	t.Parallel()
	normalized := MatrixParameters{
		Sizes:                []int{24, 8, 8},
		PointerFieldVariants: []bool{true, true, false},
		Profiles:             []LifetimeProfile{ProfileStoredInMap, ProfileLocal, ProfileLocal},
		PassingModes:         []PassingMode{PassingPointer, PassingValue},
		Probes:               []ProbeSpec{{Kind: ProbeAppendGrow, Parameter: 2}, {Kind: ProbeAppendPrealloc, Parameter: 1}},
	}.Normalize()
	if !reflect.DeepEqual(normalized.Sizes, []int{8, 24}) {
		t.Fatalf("tailles = %v", normalized.Sizes)
	}
	if !reflect.DeepEqual(normalized.PointerFieldVariants, []bool{false, true}) {
		t.Fatalf("variantes = %v", normalized.PointerFieldVariants)
	}
	if !reflect.DeepEqual(normalized.Profiles, []LifetimeProfile{ProfileLocal, ProfileStoredInMap}) {
		t.Fatalf("profils = %v", normalized.Profiles)
	}
	if !reflect.DeepEqual(normalized.PassingModes, []PassingMode{PassingValue, PassingPointer}) {
		t.Fatalf("modes = %v", normalized.PassingModes)
	}
	if normalized.Probes[0].Kind != ProbeAppendPrealloc {
		t.Fatalf("sondes mal ordonnées : %v", normalized.Probes)
	}
	empty := MatrixParameters{}.Normalize()
	if len(empty.Sizes) != 0 {
		t.Fatal("la normalisation d'une demande vide reste vide")
	}
}

func TestNormalizeGardeLesValeursInconnues(t *testing.T) {
	t.Parallel()
	// Une valeur inconnue doit survivre à la normalisation pour que Validate la rejette
	// avec son nom (UC-001, A1 : afficher chaque paramètre fautif).
	normalized := MatrixParameters{
		Sizes:                []int{8},
		PointerFieldVariants: []bool{false},
		Profiles:             []LifetimeProfile{"GLOBAL"},
		PassingModes:         []PassingMode{PassingValue},
	}.Normalize()
	err := normalized.Validate()
	if err == nil || !strings.Contains(err.Error(), "GLOBAL") {
		t.Fatalf("Validate doit nommer le profil fautif, obtenu : %v", err)
	}
}

func TestParametersValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		params MatrixParameters
		want   string
	}{
		{"sans taille", MatrixParameters{PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes()}, "aucune taille"},
		{"sans variante", MatrixParameters{Sizes: []int{8}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes()}, "champ pointeur"},
		{"sans profil", MatrixParameters{Sizes: []int{8}, PointerFieldVariants: []bool{false}, PassingModes: PassingModes()}, "LifetimeProfile"},
		{"sans mode", MatrixParameters{Sizes: []int{8}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}}, "mode de passage"},
		{"taille hors bornes", MatrixParameters{Sizes: []int{7}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes()}, "hors de"},
		{"taille non multiple de 8", MatrixParameters{Sizes: []int{20}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes()}, "multiple"},
		{"sonde inconnue", MatrixParameters{Sizes: []int{8}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(), Probes: []ProbeSpec{{Kind: "RANDOM", Parameter: 1}}}, "genre de Probe"},
		{"paramètre de sonde nul", MatrixParameters{Sizes: []int{8}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(), Probes: []ProbeSpec{{Kind: ProbeAppendGrow, Parameter: 0}}}, "> 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.params.Validate()
			if err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("message %q ne cite pas %q", err, tc.want)
			}
		})
	}
	if err := smallParams().Validate(); err != nil {
		t.Fatalf("des paramètres valides sont refusés : %v", err)
	}
}

func TestExpandProduitLesPairesCompletes(t *testing.T) {
	t.Parallel()
	// BR-001-3 : chaque TypeSpec × LifetimeProfile produit une Cell VALUE et une Cell POINTER.
	cells, probes, err := smallParams().Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	if len(cells) != 2*2*2*2 {
		t.Fatalf("%d Cell, 16 attendues", len(cells))
	}
	if len(probes) != 1 {
		t.Fatalf("%d Probe, 1 attendue", len(probes))
	}
	byKey := map[string]map[PassingMode]bool{}
	for _, cell := range cells {
		key := cell.TypeSpec.Name + "/" + string(cell.Profile)
		if byKey[key] == nil {
			byKey[key] = map[PassingMode]bool{}
		}
		byKey[key][cell.PassingMode] = true
	}
	for key, modes := range byKey {
		if !modes[PassingValue] || !modes[PassingPointer] {
			t.Fatalf("paire incomplète pour %s : %v", key, modes)
		}
	}
	if _, _, err := (MatrixParameters{Sizes: []int{7}}).Expand(); err == nil {
		t.Fatal("Expand doit valider les paramètres")
	}
}

func TestSourcePathEtSubjectDir(t *testing.T) {
	t.Parallel()
	if got, want := SubjectDir("Size0024Plain/LOCAL/VALUE"), "Size0024Plain_LOCAL_VALUE"; got != want {
		t.Fatalf("SubjectDir = %q", got)
	}
	if got, want := SubjectDir("probe/APPEND_GROW/1000"), "probe_APPEND_GROW_1000"; got != want {
		t.Fatalf("SubjectDir = %q", got)
	}
	if got, want := SourcePath("probe/APPEND_GROW/1000"), "subjects/probe_APPEND_GROW_1000/subject.go"; got != want {
		t.Fatalf("SourcePath = %q", got)
	}
}

func TestParseProbeID(t *testing.T) {
	t.Parallel()
	kind, parameter, ok := ParseProbeID("probe/SCATTERED_SCAN/33554432")
	if !ok || kind != ProbeScatteredScan || parameter != 33554432 {
		t.Fatalf("ParseProbeID = %v, %v, %v", kind, parameter, ok)
	}
	for _, id := range []string{
		"Size0024Plain/LOCAL/VALUE",
		"probe/RANDOM/10",
		"probe/APPEND_GROW/zero",
		"probe/APPEND_GROW/0",
		"probe/APPEND_GROW",
	} {
		if _, _, ok := ParseProbeID(id); ok {
			t.Fatalf("%q ne devrait pas être un identifiant de sonde", id)
		}
	}
}

func TestNewMatrixEtValidate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	matrix, err := NewMatrix(smallParams(), "digest", now)
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	if matrix.TypeSpecCount() != 4 {
		t.Fatalf("TypeSpecCount = %d, attendu 4", matrix.TypeSpecCount())
	}
	if len(matrix.SubjectIDs()) != len(matrix.Cells)+len(matrix.Probes) {
		t.Fatal("SubjectIDs doit couvrir cellules et sondes")
	}
	if _, ok := matrix.Cell(matrix.Cells[0].ID()); !ok {
		t.Fatal("Cell doit retrouver une cellule existante")
	}
	if _, ok := matrix.Cell("inconnue"); ok {
		t.Fatal("Cell ne doit pas inventer de cellule")
	}
	if _, ok := matrix.Probe(matrix.Probes[0].ID()); !ok {
		t.Fatal("Probe doit retrouver une sonde existante")
	}
	if _, ok := matrix.Probe("inconnue"); ok {
		t.Fatal("Probe ne doit pas inventer de sonde")
	}
	if _, err := NewMatrix(smallParams(), "", now); err == nil {
		t.Fatal("C-005 : une matrice sans empreinte de harnais doit être refusée")
	}

	for name, mutate := range map[string]func(*Matrix){
		"sans id":          func(m *Matrix) { m.ID = "" },
		"sans cellule":     func(m *Matrix) { m.Cells = nil },
		"sans date":        func(m *Matrix) { m.GeneratedAt = time.Time{} },
		"cellule invalide": func(m *Matrix) { m.Cells[0].SourceFile = "" },
		"doublon":          func(m *Matrix) { m.Cells = append(m.Cells, m.Cells[0]) },
		"sonde en double":  func(m *Matrix) { m.Probes = append(m.Probes, m.Probes[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			invalid, err := NewMatrix(smallParams(), "digest", now)
			if err != nil {
				t.Fatalf("NewMatrix : %v", err)
			}
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate aurait dû refuser")
			}
		})
	}
}

func TestValuePointerPairs(t *testing.T) {
	t.Parallel()
	matrix, err := NewMatrix(smallParams(), "digest", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	pairs := matrix.ValuePointerPairs()
	if len(pairs) != len(matrix.Cells)/2 {
		t.Fatalf("%d paires pour %d cellules", len(pairs), len(matrix.Cells))
	}
	for _, pair := range pairs {
		if pair[0].PassingMode != PassingValue || pair[1].PassingMode != PassingPointer {
			t.Fatalf("paire mal ordonnée : %s / %s", pair[0].ID(), pair[1].ID())
		}
		if pair[0].TypeSpec.Name != pair[1].TypeSpec.Name || pair[0].Profile != pair[1].Profile {
			t.Fatalf("paire mal appariée : %s / %s", pair[0].ID(), pair[1].ID())
		}
	}
	// Une cellule sans homologue n'est jamais appariée.
	single := Matrix{Cells: []Cell{matrix.Cells[0]}}
	if len(single.ValuePointerPairs()) != 0 {
		t.Fatal("une cellule seule ne forme pas de paire")
	}
}

// TestUC001_ExpandProfilsOrdinairesSansValeurUn verrouille A-001 : les sauts de répétition et de
// charge étaient écrits « si repeat > 1 et profil ≠ RETURNED_ALLOCATING, sauter », ce qui efface
// les autres profils dès que la liste demandée ne contient pas la valeur 1, au lieu de seulement
// les priver de la déclinaison.
func TestUC001_ExpandProfilsOrdinairesSansValeurUn(t *testing.T) {
	t.Parallel()
	cases := map[string]MatrixParameters{
		"répétitions sans 1": {
			Sizes: []int{24}, PointerFieldVariants: []bool{false},
			Profiles:     []LifetimeProfile{ProfileLocal, ProfileReturnedAlloc},
			PassingModes: PassingModes(), Repeats: []int{4},
		},
		"charges sans 1": {
			Sizes: []int{24}, PointerFieldVariants: []bool{false},
			Profiles:     []LifetimeProfile{ProfileLocal, ProfileReturnedAlloc},
			PassingModes: PassingModes(), Payloads: []int{2},
		},
	}
	for name, params := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cells, _, err := params.Expand()
			if err != nil {
				t.Fatalf("Expand : %v", err)
			}
			byProfile := map[LifetimeProfile]int{}
			for _, cell := range cells {
				byProfile[cell.Profile]++
			}
			// LOCAL produit sa paire VALUE/POINTER, une fois, à ses valeurs par défaut.
			if byProfile[ProfileLocal] != 2 {
				t.Fatalf("%d cellules LOCAL, 2 attendues : %v", byProfile[ProfileLocal], byProfile)
			}
			for _, cell := range cells {
				if cell.Profile == ProfileLocal && (cell.Repetitions() != 1 || cell.Payloads() != 1) {
					t.Fatalf("la cellule %s ne se décline pas : repeat %d, payload %d",
						cell.ID(), cell.Repetitions(), cell.Payloads())
				}
			}
			if byProfile[ProfileReturnedAlloc] == 0 {
				t.Fatalf("le profil qui alloue doit se décliner : %v", byProfile)
			}
		})
	}
}

// TestUC001_A1_ReplicatsSansSerieARepliquer verrouille A-002 : le réplicat ne se décline que sur
// la disposition NAMED_FIELDS en profil LOCAL. Demandé sans elle, il ne produisait aucune cellule
// de plus mais entrait dans la représentation canonique : la même matrice recevait un second
// identifiant.
func TestUC001_A1_ReplicatsSansSerieARepliquer(t *testing.T) {
	t.Parallel()
	params := MatrixParameters{
		Sizes: []int{24}, PointerFieldVariants: []bool{false},
		Profiles: []LifetimeProfile{ProfileReturned}, PassingModes: PassingModes(),
		Layouts: []Layout{LayoutNamedFields}, Replicates: ReplicateCount,
	}
	if err := params.Validate(); err == nil {
		t.Fatal("des réplicats sans profil LOCAL doivent être refusés")
	}
	params.Layouts = []Layout{LayoutArrayFill}
	params.Profiles = []LifetimeProfile{ProfileLocal}
	if err := params.Validate(); err == nil {
		t.Fatal("des réplicats sans disposition NAMED_FIELDS doivent être refusés")
	}
	// La série que H-012 lit est demandée : les réplicats sont admis.
	params.Layouts = []Layout{LayoutNamedFields}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate : %v", err)
	}
}

// TestUC001_A1_DemandeSansAucunSujet verrouille A-003 : Expand rendait zéro cellule sans rien
// dire, et le message final ne nommait pas le paramètre fautif.
func TestUC001_A1_DemandeSansAucunSujet(t *testing.T) {
	t.Parallel()
	params := MatrixParameters{
		Sizes: []int{4096}, PointerFieldVariants: []bool{false},
		Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(),
		Layouts: []Layout{LayoutNamedFieldsSham},
	}
	_, _, err := params.Expand()
	if err == nil {
		t.Fatal("une demande sans aucun sujet doit être refusée")
	}
	if !strings.Contains(err.Error(), string(LayoutNamedFieldsSham)) {
		t.Fatalf("le message doit nommer la demande fautive : %v", err)
	}
}
