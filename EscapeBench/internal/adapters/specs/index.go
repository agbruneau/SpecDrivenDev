package specs

import (
	"context"
	"fmt"
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

// scan parcourt cmd/ et internal/ une seule fois.
func (i *Index) scan() {
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
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)
			integration := strings.Contains(text, "//go:build integration_test")
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
func (i *Index) References(_ context.Context, useCaseID string) (bool, bool, error) {
	i.once.Do(i.scan)
	if i.err != nil {
		return false, false, i.err
	}
	return i.code[useCaseID], i.itg[useCaseID], nil
}
