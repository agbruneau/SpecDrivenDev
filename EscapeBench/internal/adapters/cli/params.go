// Package cli traduit la ligne de commande vers le vocabulaire du domaine et met en forme ce que
// les cas d'utilisation rendent observable. Il ne contient aucune logique métier : le composition
// root (cmd/escapebench) s'appuie sur lui pour rester réduit au câblage.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// ErrUsage signale une ligne de commande invalide.
var ErrUsage = errors.New("usage")

// ParseParameters lit une spécification de matrice de la forme
// `sizes=8,16,24;pointer=false,true;profiles=LOCAL,RETURNED;modes=VALUE,POINTER;probes=SEQUENTIAL_SCAN:65536,APPEND_GROW:1000`.
// Les clés absentes prennent la valeur de la matrice de référence : `profiles` vaut donc les cinq
// profils de BR-001-4, et non les huit que C-008 a portés au modèle. Prendre LifetimeProfiles() ici
// ferait produire à une spécification antérieure à C-008 une matrice de 352 cellules au lieu de 220,
// sur laquelle les critères gelés H-001 à H-006 seraient réévalués.
func ParseParameters(spec string) (models.MatrixParameters, error) {
	params := models.MatrixParameters{
		PointerFieldVariants: []bool{false, true},
		Profiles:             models.ReferenceProfiles(),
		PassingModes:         models.PassingModes(),
	}
	// A-086 : une clé répétée écrasait la précédente sans rien dire. `sizes=8;sizes=16` produisait
	// une matrice d'une seule taille, sous un identifiant que le chercheur n'attendait pas.
	seen := map[string]bool{}
	for _, clause := range strings.Split(spec, ";") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		key, value, found := strings.Cut(clause, "=")
		if !found {
			return models.MatrixParameters{}, fmt.Errorf("%w : clause %q attendue sous la forme clé=valeurs", ErrUsage, clause)
		}
		key = strings.TrimSpace(key)
		if seen[key] {
			return models.MatrixParameters{}, fmt.Errorf("%w : clé %q répétée ; une dimension s'énumère en une seule clause", ErrUsage, key)
		}
		seen[key] = true
		switch key {
		case "sizes":
			sizes, err := parseInts(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : sizes : %s", ErrUsage, err)
			}
			params.Sizes = sizes
		case "pointer":
			variants, err := parseBools(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : pointer : %s", ErrUsage, err)
			}
			params.PointerFieldVariants = variants
		case "profiles":
			params.Profiles = nil
			for _, item := range splitValues(value) {
				params.Profiles = append(params.Profiles, models.LifetimeProfile(item))
			}
		case "modes":
			params.PassingModes = nil
			for _, item := range splitValues(value) {
				params.PassingModes = append(params.PassingModes, models.PassingMode(item))
			}
		case "layouts":
			params.Layouts = nil
			for _, item := range splitValues(value) {
				params.Layouts = append(params.Layouts, models.Layout(item))
			}
		case "repeats":
			repeats, err := parseInts(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : repeats : %s", ErrUsage, err)
			}
			params.Repeats = repeats
		case "payloads":
			payloads, err := parseInts(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : payloads : %s", ErrUsage, err)
			}
			params.Payloads = payloads
		case "replicates":
			replicates, err := parseInts(value)
			if err != nil || len(replicates) != 1 {
				return models.MatrixParameters{}, fmt.Errorf("%w : replicates attend un seul entier (%q)", ErrUsage, value)
			}
			params.Replicates = replicates[0]
		case "probes":
			probes, err := parseProbes(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : probes : %s", ErrUsage, err)
			}
			params.Probes = probes
		default:
			return models.MatrixParameters{}, fmt.Errorf("%w : clé %q inconnue (sizes, pointer, profiles, modes, layouts, repeats, payloads, replicates, probes)", ErrUsage, key)
		}
	}
	if len(params.Sizes) == 0 {
		return models.MatrixParameters{}, fmt.Errorf("%w : sizes est obligatoire", ErrUsage)
	}
	return params, nil
}

// splitValues découpe une énumération séparée par des virgules.
func splitValues(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// parseInts lit une énumération d'entiers.
func parseInts(value string) ([]int, error) {
	var out []int
	for _, item := range splitValues(value) {
		n, err := strconv.Atoi(item)
		if err != nil {
			return nil, fmt.Errorf("%q n'est pas un entier", item)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune valeur")
	}
	return out, nil
}

// parseBools lit une énumération de booléens.
func parseBools(value string) ([]bool, error) {
	var out []bool
	for _, item := range splitValues(value) {
		b, err := strconv.ParseBool(item)
		if err != nil {
			return nil, fmt.Errorf("%q n'est pas un booléen", item)
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune valeur")
	}
	return out, nil
}

// parseProbes lit une énumération `GENRE:paramètre`.
func parseProbes(value string) ([]models.ProbeSpec, error) {
	var out []models.ProbeSpec
	for _, item := range splitValues(value) {
		kind, parameter, found := strings.Cut(item, ":")
		if !found {
			return nil, fmt.Errorf("%q attendu sous la forme GENRE:paramètre", item)
		}
		n, err := strconv.Atoi(strings.TrimSpace(parameter))
		if err != nil {
			return nil, fmt.Errorf("paramètre de %q illisible", item)
		}
		out = append(out, models.ProbeSpec{Kind: models.ProbeKind(strings.TrimSpace(kind)), Parameter: n})
	}
	return out, nil
}

// ParseHypotheses lit une énumération d'identifiants d'hypothèses, dédoublonnée.
//
// Révision du 2026-09-12 (A-087) : `H-001,H-001` produisait une empreinte de critères différente
// de `H-001` — l'empreinte est calculée sur la liste, doublons compris — et deux verdicts pour la
// même hypothèse dans un même rapport. L'ordre de première apparition est conservé ; le service
// trie de toute façon avant de geler.
func ParseHypotheses(value string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range splitValues(value) {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// ValidateBenchTime contrôle la durée passée à `-benchtime` (C-003).
//
// Révision du 2026-09-12 (A-156) : la valeur n'était validée nulle part. Une faute de frappe —
// « 250s » pour « 250ms », « 1min » pour « 1m » — était transmise telle quelle à `go test`, qui
// refusait chaque sujet : la campagne entière se consignait en FAILED, un sujet après l'autre,
// sans qu'aucun contrôle n'ait eu lieu en amont.
func ValidateBenchTime(value string) error {
	if value == "" {
		return fmt.Errorf("%w : --benchtime est requis (C-003)", ErrUsage)
	}
	// `go test -benchtime` accepte aussi la forme `<n>x`, un nombre d'itérations.
	if count, found := strings.CutSuffix(value, "x"); found {
		n, err := strconv.Atoi(count)
		if err != nil || n <= 0 {
			return fmt.Errorf("%w : --benchtime %q : la forme <n>x attend un nombre d'itérations positif", ErrUsage, value)
		}
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%w : --benchtime %q n'est pas une durée Go (par exemple 250ms, 1s) ni un nombre d'itérations (par exemple 100x)", ErrUsage, value)
	}
	if d <= 0 {
		return fmt.Errorf("%w : --benchtime %q doit être une durée positive", ErrUsage, value)
	}
	return nil
}

// FindRoot remonte depuis start jusqu'au répertoire du projet, reconnu à son docs/requirements.md.
// Elle prend un contexte : la remontée touche le système de fichiers à chaque niveau, et une
// arborescence profonde sur un montage réseau lent ne doit pas ignorer une annulation (A-130).
func FindRoot(ctx context.Context, start string) (string, error) {
	return findRoot(ctx, start, true)
}

// ProjectRoot vérifie qu'un chemin désigne directement la racine du projet, sans remontée.
//
// Révision du 2026-09-12 (A-093) : un `--root` fautif se résolvait en silence au projet englobant.
// Le chercheur croyait travailler sur un dépôt, le banc en lisait un autre — et y écrivait.
func ProjectRoot(ctx context.Context, dir string) (string, error) {
	return findRoot(ctx, dir, false)
}

// findRoot reconnaît la racine du projet à son docs/requirements.md, en remontant ou non.
func findRoot(ctx context.Context, start string, climb bool) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("recherche de la racine du projet depuis %s : %w", start, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "docs", "requirements.md")); err == nil {
			return dir, nil
		}
		if !climb {
			return "", fmt.Errorf("%s n'est pas la racine du projet : aucun docs/requirements.md", start)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("racine du projet introuvable depuis %s : aucun docs/requirements.md dans les répertoires parents", start)
		}
		dir = parent
	}
}
