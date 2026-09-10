// Package cli traduit la ligne de commande vers le vocabulaire du domaine et met en forme ce que
// les cas d'utilisation rendent observable. Il ne contient aucune logique métier : le composition
// root (cmd/escapebench) s'appuie sur lui pour rester réduit au câblage.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
)

// ErrUsage signale une ligne de commande invalide.
var ErrUsage = errors.New("usage")

// ParseParameters lit une spécification de matrice de la forme
// `sizes=8,16,24;pointer=false,true;profiles=LOCAL,RETURNED;modes=VALUE,POINTER;probes=SEQUENTIAL_SCAN:65536,APPEND_GROW:1000`.
// Les clés absentes prennent la valeur de la matrice de référence.
func ParseParameters(spec string) (models.MatrixParameters, error) {
	params := models.MatrixParameters{
		PointerFieldVariants: []bool{false, true},
		Profiles:             models.LifetimeProfiles(),
		PassingModes:         models.PassingModes(),
	}
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
		case "probes":
			probes, err := parseProbes(value)
			if err != nil {
				return models.MatrixParameters{}, fmt.Errorf("%w : probes : %s", ErrUsage, err)
			}
			params.Probes = probes
		default:
			return models.MatrixParameters{}, fmt.Errorf("%w : clé %q inconnue (sizes, pointer, profiles, modes, probes)", ErrUsage, key)
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

// ParseHypotheses lit une énumération d'identifiants d'hypothèses.
func ParseHypotheses(value string) []string {
	return splitValues(value)
}

// FindRoot remonte depuis start jusqu'au répertoire du projet, reconnu à son docs/requirements.md.
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "docs", "requirements.md")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("racine du projet introuvable depuis %s : aucun docs/requirements.md dans les répertoires parents", start)
		}
		dir = parent
	}
}
