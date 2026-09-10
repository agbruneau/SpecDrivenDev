package models

import (
	"strings"
	"testing"
)

// Tests de C-009, la dimension de réplicat. Chaque test porte la clause de la contrainte qu'il
// vérifie, parce que le plancher de bruit de H-012 dépend de chacune.

// La contrainte fige le nombre de réplicats : le laisser libre rendrait le plancher de bruit
// dépendant de l'effort de mesure, et l'expérimentateur reprendrait par la composition de sa
// matrice le pouvoir que la fixation lui retire.
func TestC009_NombreDeReplicatsFige(t *testing.T) {
	t.Parallel()
	base := func(n int) MatrixParameters {
		return MatrixParameters{
			Sizes: []int{8}, PointerFieldVariants: []bool{false},
			Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(),
			Layouts: []Layout{LayoutNamedFields}, Replicates: n,
		}
	}
	for _, ok := range []int{0, 1, ReplicateCount} {
		if err := base(ok).Validate(); err != nil {
			t.Fatalf("%d réplicats doit être accepté : %v", ok, err)
		}
	}
	for _, refuse := range []int{2, 3, 4, 6, 20} {
		err := base(refuse).Validate()
		if err == nil || !strings.Contains(err.Error(), "réplicats") {
			t.Fatalf("%d réplicats doit être refusé, obtenu %v", refuse, err)
		}
	}
}

// Le réplicat est la dimension la plus extérieure : deux mesures d'une même paire sont séparées par
// une passe complète de la matrice répliquée. C'est cette séparation, et non le seul nombre de
// réplicats, qui rend le plancher opposable.
func TestC009_ReplicatEstLaDimensionLaPlusExterieure(t *testing.T) {
	t.Parallel()
	p := MatrixParameters{
		Sizes: []int{8, 16}, PointerFieldVariants: []bool{false},
		Profiles: []LifetimeProfile{ProfileLocal}, PassingModes: PassingModes(),
		Layouts: []Layout{LayoutNamedFields}, Replicates: ReplicateCount,
	}
	cells, _, err := p.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	// 2 tailles × 2 modes × 5 réplicats.
	if len(cells) != 20 {
		t.Fatalf("%d cellules, 20 attendues", len(cells))
	}
	// Les quatre cellules d'une passe précèdent toutes celles de la passe suivante.
	passe := 0
	for i, c := range cells {
		if i%4 == 0 {
			passe++
		}
		if c.ReplicateIndex() != passe {
			t.Fatalf("cellule %d (%s) est du réplicat %d, la passe %d attendue : les réplicats ne sont pas séparés par une passe complète",
				i, c.ID(), c.ReplicateIndex(), passe)
		}
	}
}

// Un réplicat de un ne laisse aucune trace : les identifiants, les répertoires de sujets et
// l'identifiant de matrice d'une demande antérieure à C-009 sont inchangés (BR-001-1).
func TestC009_RetrocompatibiliteDesIdentifiants(t *testing.T) {
	t.Parallel()
	if got := ReferenceParameters().MatrixID(); got != "M-823d8b5af441" {
		t.Fatalf("identifiant de la matrice de référence = %s", got)
	}
	sansDimension := ReferenceParameters()
	sansDimension.Layouts, sansDimension.Repeats, sansDimension.Payloads, sansDimension.Replicates = nil, nil, nil, 0
	if got := sansDimension.MatrixID(); got != "M-823d8b5af441" {
		t.Fatalf("identifiant sans les dimensions de C-008 et C-009 = %s", got)
	}
	// Le suffixe n'apparaît qu'au-delà du premier réplicat, et il ne se confond ni avec _R ni
	// avec _K.
	if got := (Cell{Profile: ProfileLocal, Replicate: 1}).ProfileSegment(); got != string(ProfileLocal) {
		t.Fatalf("segment au premier réplicat = %q", got)
	}
	if got := (Cell{Profile: ProfileLocal, Replicate: 3}).ProfileSegment(); got != "LOCAL_X3" {
		t.Fatalf("segment au troisième réplicat = %q", got)
	}
	if got := (Cell{Profile: ProfileReturnedAlloc, Repeat: 4, Payload: 2, Replicate: 5}).ProfileSegment(); got != "RETURNED_ALLOCATING_R4_K2_X5" {
		t.Fatalf("les trois suffixes se composent dans l'ordre : %q", got)
	}
	// Une demande à cinq réplicats produit un autre identifiant de matrice, la représentation
	// canonique s'écartant de sa valeur par défaut.
	replique := ReferenceParameters()
	replique.Replicates = ReplicateCount
	if replique.MatrixID() == "M-823d8b5af441" {
		t.Fatal("une matrice répliquée doit porter un autre identifiant")
	}
	if !strings.Contains(replique.Normalize().Canonical(), "replicates=5") {
		t.Fatalf("la représentation canonique doit porter le réplicat : %q", replique.Normalize().Canonical())
	}
}

// Le réplicat ne se décline que là où H-012 le lit. Ailleurs il ne produirait que des doublons, et
// la clause de séparation ne vaut alors que pour le sous-ensemble répliqué.
func TestC009_ReplicatSeulementEnNamedFieldsLocal(t *testing.T) {
	t.Parallel()
	p := MatrixParameters{
		Sizes: []int{24}, PointerFieldVariants: []bool{false},
		Profiles:     []LifetimeProfile{ProfileLocal, ProfileStoredInMap},
		PassingModes: PassingModes(),
		Layouts:      []Layout{LayoutArrayFill, LayoutNamedFields},
		Replicates:   ReplicateCount,
	}
	cells, _, err := p.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	compte := map[string]int{}
	for _, c := range cells {
		compte[string(c.TypeSpec.Layout)+"/"+string(c.Profile)]++
	}
	// Seule la combinaison que H-012 lit se décline : 2 modes × 5 réplicats.
	if compte["NAMED_FIELDS/LOCAL"] != 10 {
		t.Fatalf("NAMED_FIELDS/LOCAL = %d, 10 attendues", compte["NAMED_FIELDS/LOCAL"])
	}
	for _, autre := range []string{"NAMED_FIELDS/STORED_IN_MAP", "ARRAY_FILL/LOCAL", "ARRAY_FILL/STORED_IN_MAP"} {
		if compte[autre] != 2 {
			t.Fatalf("%s = %d, 2 attendues : le réplicat ne doit s'y décliner que la première passe", autre, compte[autre])
		}
	}
	// Aucun identifiant en double, faute de quoi Matrix.Validate refuserait la matrice.
	vus := map[string]bool{}
	for _, c := range cells {
		if vus[c.ID()] {
			t.Fatalf("identifiant en double : %s", c.ID())
		}
		vus[c.ID()] = true
	}
}
