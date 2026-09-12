package specs

import (
	"context"
	"fmt"
	"go/build/constraint"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Index parcourt une fois les sources Go du module et retient, par cas d'utilisation, la présence
// de code et de tests d'intégration qui le référencent (colonnes Code et Integration du tableau
// de bord).
type Index struct {
	root string
	once sync.Once
	err  error
	code map[string]bool
	itg  map[string]bool
}

// NewIndex construit un index enraciné sur le répertoire du projet.
func NewIndex(root string) *Index { return &Index{root: root} }

// scan parcourt cmd/ et internal/ une seule fois. Le contexte est honoré à chaque fichier : le
// parcours lit tout le code du module, et une annulation ne doit pas attendre sa fin (A-130).
func (i *Index) scan(ctx context.Context) {
	i.code = map[string]bool{}
	i.itg = map[string]bool{}
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(i.root, dir)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)
			integration := requiresIntegrationTag(text)
			for _, id := range useCaseIDs(text) {
				i.code[id] = true
				if integration {
					i.itg[id] = true
				}
			}
			return nil
		})
		if err != nil {
			i.err = fmt.Errorf("parcours de %s : %w", dir, err)
			return
		}
	}
}

// requiresIntegrationTag indique si le fichier ne se compile que sous le tag `integration_test`.
//
// Révision du 2026-09-12 (A-278) : la présence du tag était cherchée comme sous-chaîne n'importe
// où dans le fichier. Le littéral de chaîne d'un test, ou ce fichier-ci, suffisait à faire
// apparaître ✔ dans la colonne Integration du tableau de bord pour tout cas d'utilisation cité au
// même endroit : la colonne était un faux positif intégral. Seule une ligne de contrainte de
// build, en tête de fichier et au sens de go/build/constraint, compte désormais.
func requiresIntegrationTag(text string) bool {
	for line := range strings.Lines(text) {
		trimmed := strings.TrimSpace(line)
		// Le préambule de contraintes s'arrête à la clause de paquet.
		if strings.HasPrefix(trimmed, "package ") {
			return false
		}
		if !constraint.IsGoBuild(trimmed) {
			continue
		}
		expr, err := constraint.Parse(trimmed)
		if err != nil {
			return false
		}
		// La contrainte exige le tag si elle est satisfaite quand tous les tags sont posés,
		// et ne l'est plus dès qu'on retire celui-ci.
		withAll := expr.Eval(func(string) bool { return true })
		withoutTag := expr.Eval(func(tag string) bool { return tag != "integration_test" })
		return withAll && !withoutTag
	}
	return false
}

// useCaseIDs rend les identifiants de cas d'utilisation cités dans un texte, sous leurs deux
// formes : `UC-003` dans les commentaires, `TestUC003_` dans les noms de tests.
func useCaseIDs(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, match := range ucCommentRe.FindAllString(text, -1) {
		add(match)
	}
	for _, match := range ucTestRe.FindAllStringSubmatch(text, -1) {
		add("UC-" + match[1])
	}
	return out
}

// References indique si du code et un test d'intégration référencent le cas d'utilisation.
func (i *Index) References(ctx context.Context, useCaseID string) (bool, bool, error) {
	i.once.Do(func() { i.scan(ctx) })
	if i.err != nil {
		return false, false, i.err
	}
	return i.code[useCaseID], i.itg[useCaseID], nil
}
