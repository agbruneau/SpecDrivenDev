package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ProbeSpec est la demande d'une sonde : un genre et son paramètre.
type ProbeSpec struct {
	Kind      ProbeKind
	Parameter int
}

// MatrixParameters est la demande de génération d'une Matrix (UC-001, étape 1).
type MatrixParameters struct {
	Sizes                []int
	PointerFieldVariants []bool
	Profiles             []LifetimeProfile
	PassingModes         []PassingMode
	// Layouts, Repeats et Payloads sont ajoutés par C-008. Leurs valeurs par défaut — la seule
	// disposition d'origine, une instance par opération et une charge par instance — reproduisent
	// exactement les matrices antérieures, identifiant compris.
	Layouts  []Layout
	Repeats  []int
	Payloads []int
	// Replicates est ajouté par C-009 : le nombre de mesures indépendantes d'une même paire dans
	// une campagne. Il vaut 1 ou 5 et rien d'autre, de sorte que la composition de la matrice ne
	// reprenne pas le pouvoir sur le plancher de bruit que la fixation à cinq lui retire (H-012).
	Replicates int
	Probes     []ProbeSpec
}

// DefaultLayouts rend la dimension de disposition par défaut : celle d'avant C-008.
func DefaultLayouts() []Layout { return []Layout{LayoutArrayFill} }

// DefaultRepeats rend la dimension de répétition par défaut : une instance par opération.
func DefaultRepeats() []int { return []int{1} }

// ReplicateCount est le nombre de réplicats qu'exige le critère gelé de H-012. Toute autre valeur
// que celle-ci ou l'absence de réplicat est refusée : le plancher de bruit ne doit ni grandir ni
// rétrécir avec l'effort de mesure.
const ReplicateCount = 5

// DefaultReplicates rend la dimension de réplicat par défaut : une seule mesure par paire.
func DefaultReplicates() int { return 1 }

// DefaultPayloads rend la dimension de charge par défaut : une charge allouée par instance.
//
// Cette dimension existe parce que sans elle H-010 n'est pas réfutable. Le bras valeur alloue k
// charges par instance, le bras pointeur ces mêmes k charges plus la valeur retournée : le rapport
// mesuré vaut (k+1)/k, fixé par le gabarit et non par le mode de passage. À k = 1 il vaut 2 sur
// toute machine et pour toute taille, et le critère de H-010 ne peut alors que confirmer. Faire
// varier k est ce qui met l'énoncé « le surcoût du pointeur est multiplicatif, non additif » à
// l'épreuve : à k = 2 le rapport tombe à 1,5 et le critère infirme.
func DefaultPayloads() []int { return []int{1} }

// ReferenceParameters rend les paramètres de la matrice de référence (BR-001-4) :
// 11 tailles × 2 variantes de champ pointeur × 5 profils × 2 modes = 220 Cell, plus 10 Probe.
func ReferenceParameters() MatrixParameters {
	const (
		kiB = 1024
		miB = 1024 * kiB
	)
	return MatrixParameters{
		Sizes:                []int{8, 16, 24, 32, 64, 128, 256, 512, 1024, 2048, 4096},
		PointerFieldVariants: []bool{false, true},
		Profiles:             ReferenceProfiles(),
		PassingModes:         PassingModes(),
		Layouts:              DefaultLayouts(),
		Repeats:              DefaultRepeats(),
		Payloads:             DefaultPayloads(),
		Replicates:           DefaultReplicates(),
		Probes: []ProbeSpec{
			{ProbeSequentialScan, 256 * kiB}, {ProbeScatteredScan, 256 * kiB},
			{ProbeSequentialScan, 4 * miB}, {ProbeScatteredScan, 4 * miB},
			{ProbeSequentialScan, 32 * miB}, {ProbeScatteredScan, 32 * miB},
			{ProbeSequentialScan, 128 * miB}, {ProbeScatteredScan, 128 * miB},
			{ProbeAppendPrealloc, 100000}, {ProbeAppendGrow, 100000},
		},
	}
}

// Normalize trie et dédoublonne chaque dimension. Deux demandes équivalentes produisent des
// paramètres normalisés identiques, donc le même identifiant de Matrix (BR-001-1).
func (p MatrixParameters) Normalize() MatrixParameters {
	layouts := p.Layouts
	if len(layouts) == 0 {
		layouts = DefaultLayouts()
	}
	repeats := p.Repeats
	if len(repeats) == 0 {
		repeats = DefaultRepeats()
	}
	payloads := p.Payloads
	if len(payloads) == 0 {
		payloads = DefaultPayloads()
	}
	replicates := p.Replicates
	if replicates == 0 {
		replicates = DefaultReplicates()
	}
	out := MatrixParameters{
		Sizes:                dedupeSorted(p.Sizes, func(a, b int) bool { return a < b }),
		PointerFieldVariants: dedupeSorted(p.PointerFieldVariants, func(a, b bool) bool { return !a && b }),
		Profiles:             orderedSubset(p.Profiles, LifetimeProfiles()),
		PassingModes:         orderedSubset(p.PassingModes, PassingModes()),
		Layouts:              orderedSubset(layouts, Layouts()),
		Repeats:              dedupeSorted(repeats, func(a, b int) bool { return a < b }),
		Payloads:             dedupeSorted(payloads, func(a, b int) bool { return a < b }),
		Replicates:           replicates,
	}
	seen := make(map[ProbeSpec]bool, len(p.Probes))
	for _, spec := range p.Probes {
		if !seen[spec] {
			seen[spec] = true
			out.Probes = append(out.Probes, spec)
		}
	}
	kindRank := make(map[ProbeKind]int, len(ProbeKinds()))
	for i, k := range ProbeKinds() {
		kindRank[k] = i
	}
	sort.Slice(out.Probes, func(i, j int) bool {
		if out.Probes[i].Kind != out.Probes[j].Kind {
			return kindRank[out.Probes[i].Kind] < kindRank[out.Probes[j].Kind]
		}
		return out.Probes[i].Parameter < out.Probes[j].Parameter
	})
	return out
}

// dedupeSorted trie selon less et retire les doublons.
func dedupeSorted[T comparable](values []T, less func(a, b T) bool) []T {
	if len(values) == 0 {
		return nil
	}
	sorted := append([]T(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	out := sorted[:1]
	for _, v := range sorted[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

// orderedSubset rend les valeurs demandées dans l'ordre canonique de la liste de référence,
// sans doublon ; les valeurs inconnues sont conservées en fin pour que Validate les rejette.
func orderedSubset[T comparable](requested, canonical []T) []T {
	wanted := make(map[T]bool, len(requested))
	for _, v := range requested {
		wanted[v] = true
	}
	var out []T
	for _, v := range canonical {
		if wanted[v] {
			out = append(out, v)
			delete(wanted, v)
		}
	}
	var unknown []T
	for _, v := range requested {
		if wanted[v] {
			unknown = append(unknown, v)
			delete(wanted, v)
		}
	}
	return append(out, unknown...)
}

// Validate applique l'étape 2 de UC-001 et rapporte chaque paramètre fautif avec la règle violée.
func (p MatrixParameters) Validate() error {
	var problems []string
	if len(p.Sizes) == 0 {
		problems = append(problems, "aucune taille demandée")
	}
	for _, size := range p.Sizes {
		if err := ValidateSize(size); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(p.PointerFieldVariants) == 0 {
		problems = append(problems, "aucune variante de champ pointeur demandée")
	}
	if len(p.Profiles) == 0 {
		problems = append(problems, "aucun LifetimeProfile demandé")
	}
	for _, profile := range p.Profiles {
		if !profile.Valid() {
			problems = append(problems, fmt.Sprintf("LifetimeProfile %q inconnu", profile))
		}
	}
	if len(p.PassingModes) == 0 {
		problems = append(problems, "au moins un mode de passage est requis")
	}
	for _, mode := range p.PassingModes {
		if !mode.Valid() {
			problems = append(problems, fmt.Sprintf("mode de passage %q inconnu", mode))
		}
	}
	for _, layout := range p.Layouts {
		if !layout.Valid() {
			problems = append(problems, fmt.Sprintf("disposition %q inconnue", layout))
		}
	}
	for _, repeat := range p.Repeats {
		if repeat < 1 {
			problems = append(problems, fmt.Sprintf("nombre d'instances par opération doit être ≥ 1 (%d)", repeat))
		}
	}
	for _, payload := range p.Payloads {
		if payload < 1 {
			problems = append(problems, fmt.Sprintf("nombre de charges par instance doit être ≥ 1 (%d)", payload))
		}
	}
	// C-009 fige le nombre de réplicats : le critère de H-012 en dépend, et le laisser libre
	// rendrait le plancher de bruit dépendant de l'effort de mesure.
	if p.Replicates != 0 && p.Replicates != DefaultReplicates() && p.Replicates != ReplicateCount {
		problems = append(problems, fmt.Sprintf("nombre de réplicats doit valoir %d ou %d (%d)",
			DefaultReplicates(), ReplicateCount, p.Replicates))
	}
	// Le témoin nul ne se mesure qu'en profil LOCAL : demandé avec d'autres profils, il
	// produirait des cellules que Cell.Validate refuse.
	for _, spec := range p.Probes {
		if !spec.Kind.Valid() {
			problems = append(problems, fmt.Sprintf("genre de Probe %q inconnu", spec.Kind))
		}
		if spec.Parameter <= 0 {
			problems = append(problems, fmt.Sprintf("paramètre de Probe %s doit être > 0 (%d)", spec.Kind, spec.Parameter))
		}
		// Une sonde qui parcourt la mémoire travaille par nœuds d'une ligne de cache.
		if (spec.Kind == ProbeSequentialScan || spec.Kind == ProbeScatteredScan || spec.Kind == ProbePointerChase) && spec.Parameter%64 != 0 {
			problems = append(problems, fmt.Sprintf("le jeu de travail d'une sonde %s doit être un multiple de 64 octets (%d)", spec.Kind, spec.Parameter))
		}
	}
	if len(problems) > 0 {
		return invalid("%s", strings.Join(problems, " ; "))
	}
	return nil
}

// Canonical rend la représentation stable des paramètres normalisés, base de l'identifiant.
func (p MatrixParameters) Canonical() string {
	n := p.Normalize()
	var b strings.Builder
	b.WriteString("v1\n")
	fmt.Fprintf(&b, "sizes=%v\n", n.Sizes)
	fmt.Fprintf(&b, "pointerField=%v\n", n.PointerFieldVariants)
	fmt.Fprintf(&b, "profiles=%v\n", n.Profiles)
	fmt.Fprintf(&b, "passingModes=%v\n", n.PassingModes)
	// Les deux dimensions de C-008 ne sont écrites que si elles s'écartent de leur valeur par
	// défaut : une demande antérieure à C-008 produit la même représentation canonique, donc le
	// même identifiant de Matrix (BR-001-1).
	if !sameLayouts(n.Layouts, DefaultLayouts()) {
		fmt.Fprintf(&b, "layouts=%v\n", n.Layouts)
	}
	if !sameInts(n.Repeats, DefaultRepeats()) {
		fmt.Fprintf(&b, "repeats=%v\n", n.Repeats)
	}
	if !sameInts(n.Payloads, DefaultPayloads()) {
		fmt.Fprintf(&b, "payloads=%v\n", n.Payloads)
	}
	if n.Replicates != DefaultReplicates() {
		fmt.Fprintf(&b, "replicates=%d\n", n.Replicates)
	}
	b.WriteString("probes=")
	for _, spec := range n.Probes {
		fmt.Fprintf(&b, "%s:%d,", spec.Kind, spec.Parameter)
	}
	b.WriteString("\n")
	return b.String()
}

// sameLayouts compare deux listes de dispositions.
func sameLayouts(a, b []Layout) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sameInts compare deux listes d'entiers.
func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MatrixID rend l'identifiant déterministe de la forme M-<sha256 court> (BR-001-1).
func (p MatrixParameters) MatrixID() string {
	sum := sha256.Sum256([]byte(p.Canonical()))
	return "M-" + hex.EncodeToString(sum[:6])
}

// SubjectDir rend le nom de répertoire du paquet Go d'un sujet, dérivé de son identifiant.
func SubjectDir(subjectID string) string {
	return strings.NewReplacer("/", "_", ".", "_", "-", "_").Replace(subjectID)
}

// Expand dérive les TypeSpec, Cell et Probe des paramètres (UC-001, étape 4). Les paramètres sont
// normalisés au préalable : l'ordre des sujets est donc déterministe.
func (p MatrixParameters) Expand() ([]Cell, []Probe, error) {
	n := p.Normalize()
	if err := n.Validate(); err != nil {
		return nil, nil, err
	}
	var cells []Cell
	// Le réplicat est la dimension la plus extérieure (C-009) : deux mesures d'une même paire sont
	// ainsi séparées par tous les autres sujets répliqués de la campagne, soit une passe complète.
	// C'est cette séparation, et non le seul nombre de réplicats, qui rend opposable le plancher de
	// bruit de H-012 : cinq processus distincts espacés de plusieurs minutes échantillonnent une
	// classe de bruit que cinq exécutions consécutives ne verraient pas.
	for replicate := 1; replicate <= n.Replicates; replicate++ {
		for _, layout := range n.Layouts {
			for _, size := range n.Sizes {
				// Le témoin nul ne se produit que sur les tailles dont il borne le bruit. Le plancher
				// de H-007 est le plus grand |deltaNsPerOp| relevé en NAMED_FIELDS_SHAM sur toute la
				// série : un témoin à 4096 octets, où le seul coût de copie porte l'écart entre deux
				// binaires à près d'une nanoseconde, fixerait un plancher plusieurs fois supérieur au
				// plus grand effet réel des trois tailles examinées, et l'hypothèse ne pourrait plus
				// qu'être confirmée.
				//
				// C'est un saut et non un refus : le même critère gelé exige, dans la même série, un
				// témoin de sensibilité d'au moins 80 octets en NAMED_FIELDS. Refuser la matrice
				// entière rendrait H-007 insatisfiable.
				if layout == LayoutNamedFieldsSham && size > SmallStructBytes {
					continue
				}
				for _, hasPointer := range n.PointerFieldVariants {
					spec, err := NewTypeSpec(size, hasPointer, layout)
					if err != nil {
						return nil, nil, err
					}
					for _, profile := range n.Profiles {
						// Le témoin nul ne se mesure qu'en profil LOCAL : ses deux cellules exécutent
						// le corps du mode VALUE, ce qui n'a de sens que pour une paire dont le bras
						// valeur est le sujet. Saut et non refus, pour la même raison que la borne de
						// taille juste au-dessus : H-007 exige le témoin nul, H-009 exige les profils
						// conteneurs et H-010 le profil qui alloue. Refuser la combinaison forçait à
						// trois campagnes distinctes, donc trois empreintes et trois Provenance, pour
						// des hypothèses qu'une seule campagne peut couvrir.
						if layout == LayoutNamedFieldsSham && profile != ProfileLocal {
							continue
						}
						for _, repeat := range n.Repeats {
							// Seul un profil qui produit plusieurs instances par opération dépend de
							// la répétition ; ailleurs elle ne créerait que des doublons.
							if repeat > 1 && profile != ProfileReturnedAlloc {
								continue
							}
							for _, payload := range n.Payloads {
								// Même règle pour la charge : seul le profil qui en alloue une s'en
								// décline.
								if payload > 1 && profile != ProfileReturnedAlloc {
									continue
								}
								// BR-001-3 : une Cell par mode demandé, donc la paire complète quand
								// les deux le sont.
								for _, mode := range n.PassingModes {
									cell := Cell{TypeSpec: spec, Profile: profile, PassingMode: mode,
										Repeat: repeat, Payload: payload, Replicate: replicate}
									// Le réplicat ne se décline que là où H-012 le lit. Ailleurs il ne
									// produirait que des doublons, et la clause de séparation de C-009
									// vaut alors pour le seul sous-ensemble répliqué.
									if replicate > 1 && (layout != LayoutNamedFields || profile != ProfileLocal) {
										continue
									}
									cell.SourceFile = SourcePath(cell.ID())
									if err := cell.Validate(); err != nil {
										return nil, nil, err
									}
									cells = append(cells, cell)
								}
							}
						}
					}
				}
			}
		}
	}
	var probes []Probe
	for _, spec := range n.Probes {
		probe := Probe{Kind: spec.Kind, Parameter: spec.Parameter}
		probe.SourceFile = SourcePath(probe.ID())
		if err := probe.Validate(); err != nil {
			return nil, nil, err
		}
		probes = append(probes, probe)
	}
	return cells, probes, nil
}

// SourcePath rend le chemin, relatif au répertoire de la Matrix, du fichier source d'un sujet.
func SourcePath(subjectID string) string {
	return "subjects/" + SubjectDir(subjectID) + "/subject.go"
}

// ParseProbeID retrouve le genre et le paramètre d'un identifiant de sonde. Le deuxième segment
// d'un identifiant de Probe est un genre, jamais un code de LifetimeProfile : la forme suffit à
// distinguer une sonde d'une cellule.
func ParseProbeID(id string) (ProbeKind, int, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] != "probe" {
		return "", 0, false
	}
	kind := ProbeKind(parts[1])
	if !kind.Valid() {
		return "", 0, false
	}
	parameter, err := strconv.Atoi(parts[2])
	if err != nil || parameter <= 0 {
		return "", 0, false
	}
	return kind, parameter, true
}

// Matrix est l'ensemble des sujets générés pour un jeu de paramètres.
type Matrix struct {
	ID            string
	Parameters    MatrixParameters
	HarnessDigest string
	Cells         []Cell
	Probes        []Probe
	GeneratedAt   time.Time
}

// NewMatrix construit une Matrix validée à partir de paramètres et d'une empreinte de harnais.
func NewMatrix(params MatrixParameters, harnessDigest string, generatedAt time.Time) (Matrix, error) {
	normalized := params.Normalize()
	cells, probes, err := normalized.Expand()
	if err != nil {
		return Matrix{}, err
	}
	m := Matrix{
		ID:            normalized.MatrixID(),
		Parameters:    normalized,
		HarnessDigest: harnessDigest,
		Cells:         cells,
		Probes:        probes,
		GeneratedAt:   generatedAt.UTC(),
	}
	return m, m.Validate()
}

// Validate applique les règles de validation du modèle d'entités.
func (m Matrix) Validate() error {
	if m.ID == "" {
		return invalid("Matrix.id est requis")
	}
	if m.HarnessDigest == "" {
		return invalid("Matrix.harnessDigest est requis (C-005)")
	}
	if len(m.Cells) == 0 {
		return invalid("Matrix.cells contient au moins une cellule")
	}
	if m.GeneratedAt.IsZero() {
		return invalid("Matrix.generatedAt est requis")
	}
	seen := make(map[string]bool, len(m.Cells)+len(m.Probes))
	for _, cell := range m.Cells {
		if err := cell.Validate(); err != nil {
			return err
		}
		if seen[cell.ID()] {
			return invalid("identifiant de sujet en double : %s", cell.ID())
		}
		seen[cell.ID()] = true
	}
	for _, probe := range m.Probes {
		if err := probe.Validate(); err != nil {
			return err
		}
		if seen[probe.ID()] {
			return invalid("identifiant de sujet en double : %s", probe.ID())
		}
		seen[probe.ID()] = true
	}
	return nil
}

// SubjectIDs rend les identifiants de tous les sujets, cellules puis sondes (UC-003, étape 5).
func (m Matrix) SubjectIDs() []string {
	ids := make([]string, 0, len(m.Cells)+len(m.Probes))
	for _, cell := range m.Cells {
		ids = append(ids, cell.ID())
	}
	for _, probe := range m.Probes {
		ids = append(ids, probe.ID())
	}
	return ids
}

// Cell retrouve une cellule par identifiant.
func (m Matrix) Cell(id string) (Cell, bool) {
	for _, cell := range m.Cells {
		if cell.ID() == id {
			return cell, true
		}
	}
	return Cell{}, false
}

// Probe retrouve une sonde par identifiant.
func (m Matrix) Probe(id string) (Probe, bool) {
	for _, probe := range m.Probes {
		if probe.ID() == id {
			return probe, true
		}
	}
	return Probe{}, false
}

// TypeSpecCount rend le nombre de TypeSpec distincts (UC-001, étape 7).
func (m Matrix) TypeSpecCount() int {
	seen := make(map[string]bool, len(m.Cells))
	for _, cell := range m.Cells {
		seen[cell.TypeSpec.Name] = true
	}
	return len(seen)
}

// ValuePointerPairs apparie les cellules VALUE et POINTER par TypeSpec et LifetimeProfile
// (UC-004, étape 3). Les cellules sans homologue sont ignorées : elles n'ont pas de paire.
func (m Matrix) ValuePointerPairs() [][2]Cell {
	byKey := make(map[string]map[PassingMode]Cell, len(m.Cells))
	var order []string
	for _, cell := range m.Cells {
		// La clé est l'identifiant privé de son mode de passage : deux cellules ne s'apparient que
		// si leur type, leur profil et leur nombre d'instances par opération coïncident.
		key := cell.TypeSpec.Name + "/" + cell.ProfileSegment()
		if _, ok := byKey[key]; !ok {
			byKey[key] = make(map[PassingMode]Cell, 2)
			order = append(order, key)
		}
		byKey[key][cell.PassingMode] = cell
	}
	var pairs [][2]Cell
	for _, key := range order {
		value, hasValue := byKey[key][PassingValue]
		pointer, hasPointer := byKey[key][PassingPointer]
		if hasValue && hasPointer {
			pairs = append(pairs, [2]Cell{value, pointer})
		}
	}
	return pairs
}
