package models

import (
	"strings"
	"testing"
)

// Tests de non-régression de la revue contradictoire du 2026-09-10. Chaque test porte le constat
// qu'il ferme et la raison pour laquelle le comportement d'origine faussait un verdict.

// Le plancher de bruit de H-007 est le plus grand écart relevé en NAMED_FIELDS_SHAM sur toute la
// série, sans borne de taille. Un témoin à 4096 octets, où la seule copie porte l'écart entre deux
// binaires à près d'une nanoseconde, fixait un plancher plusieurs fois supérieur au plus grand
// effet réel des trois tailles examinées, et H-007 ne pouvait plus qu'être confirmée.
//
// La restriction est un saut dans Expand et non un refus dans Validate, parce que le même critère
// gelé exige un témoin de sensibilité d'au moins 80 octets dans la même série : refuser la matrice
// rendrait H-007 insatisfiable.
func TestC008_TemoinNulBorneAuxPetitesTailles(t *testing.T) {
	t.Parallel()
	// La matrice qu'exige H-007 : les trois petites tailles, le témoin de sensibilité à 128 octets,
	// les deux dispositions.
	p := MatrixParameters{
		Sizes:                []int{8, 16, 24, 128},
		PointerFieldVariants: []bool{false, true},
		Profiles:             []LifetimeProfile{ProfileLocal},
		PassingModes:         PassingModes(),
		Layouts:              []Layout{LayoutNamedFields, LayoutNamedFieldsSham},
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("une matrice H-007 complète doit rester légale : %v", err)
	}
	cells, _, err := p.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	var shamSizes, fieldsAtSensitivity int
	for _, c := range cells {
		switch {
		case c.TypeSpec.Layout == LayoutNamedFieldsSham:
			if c.TypeSpec.SizeBytes > SmallStructBytes {
				t.Fatalf("le témoin nul ne doit pas dépasser %d octets : %s", SmallStructBytes, c.ID())
			}
			shamSizes++
		case c.TypeSpec.Layout == LayoutNamedFields && c.TypeSpec.SizeBytes == 128:
			fieldsAtSensitivity++
		}
	}
	// Trois tailles × deux variantes de champ pointeur × deux modes.
	if shamSizes != 12 {
		t.Fatalf("cellules témoins = %d, 12 attendues", shamSizes)
	}
	// Le témoin de sensibilité de H-007 survit : deux variantes × deux modes.
	if fieldsAtSensitivity != 4 {
		t.Fatalf("cellules du témoin de sensibilité à 128 octets = %d, 4 attendues", fieldsAtSensitivity)
	}
	// Le témoin nul reste par ailleurs cantonné au profil LOCAL.
	horsLocal := p
	horsLocal.Profiles = []LifetimeProfile{ProfileReturned}
	if err := horsLocal.Validate(); err == nil {
		t.Fatal("le témoin nul hors du profil LOCAL doit être refusé")
	}
}

// H-010 comparait k charges à k + 1 : à k = 1 le rapport valait 2 sur toute machine et pour toute
// taille, si bien que le critère ne pouvait que confirmer. La dimension rend k mesurable.
func TestC008_ChargesParInstance(t *testing.T) {
	t.Parallel()
	p := MatrixParameters{
		Sizes:                []int{24},
		PointerFieldVariants: []bool{false},
		Profiles:             []LifetimeProfile{ProfileReturnedAlloc, ProfileLocal},
		PassingModes:         PassingModes(),
		Payloads:             []int{1, 2},
	}
	cells, _, err := p.Expand()
	if err != nil {
		t.Fatalf("Expand : %v", err)
	}
	var kSegments, localCells int
	for _, c := range cells {
		if c.Profile == ProfileLocal {
			localCells++
			if strings.Contains(c.ID(), "_K") {
				t.Fatalf("seul le profil qui alloue une charge s'en décline : %s", c.ID())
			}
		}
		if strings.Contains(c.ID(), "_K2") {
			kSegments++
		}
	}
	if kSegments != 2 {
		t.Fatalf("la paire k = 2 doit exister une fois par mode, obtenu %d", kSegments)
	}
	if localCells != 2 {
		t.Fatalf("le profil LOCAL ne se décline pas en charges, obtenu %d cellules", localCells)
	}
	// Une charge par instance est la valeur par défaut : elle ne laisse aucune trace dans
	// l'identifiant ni dans la représentation canonique.
	if got := (Cell{Profile: ProfileReturnedAlloc, Payload: 1}).ProfileSegment(); got != string(ProfileReturnedAlloc) {
		t.Fatalf("segment de profil à k = 1 = %q", got)
	}
	if strings.Contains(p.Normalize().Canonical(), "payloads") == false {
		t.Fatalf("une charge non standard doit entrer dans la représentation canonique")
	}
	standard := ReferenceParameters()
	if strings.Contains(standard.Normalize().Canonical(), "payloads") {
		t.Fatalf("la valeur par défaut ne doit jamais s'écrire : %q", standard.Normalize().Canonical())
	}
	if p.Payloads[0] < 1 {
		t.Fatal("garde-fou de lecture")
	}
	invalide := p
	invalide.Payloads = []int{0}
	if err := invalide.Validate(); err == nil {
		t.Fatal("une charge nulle doit être refusée")
	}
}

// La disposition est entrée dans la clé des séries : sans elle, le témoin nul, dont l'écart est nul
// par construction, tombait dans la même série que les paires réelles et interrompait le suffixe
// favorable au pointeur, retournant le point de bascule de H-002.
func TestC008_CleDeSerieDistingueLaDisposition(t *testing.T) {
	t.Parallel()
	a := TippingKey{Profile: ProfileLocal, Layout: LayoutArrayFill}
	b := TippingKey{Profile: ProfileLocal, Layout: LayoutNamedFields}
	if a == b {
		t.Fatal("deux dispositions doivent donner deux séries")
	}
	// Une Comparison lue d'un fichier antérieur à C-008 ne porte pas de disposition : c'est celle
	// d'origine, pas une quatrième série.
	if got := (Comparison{}).EffectiveLayout(); got != LayoutArrayFill {
		t.Fatalf("disposition par défaut = %q", got)
	}
	if got := (Comparison{Layout: LayoutNamedFields}).EffectiveLayout(); got != LayoutNamedFields {
		t.Fatalf("disposition explicite = %q", got)
	}
}
