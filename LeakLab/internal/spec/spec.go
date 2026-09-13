// Package spec lit docs/requirements.md : les hypothèses et l'empreinte de leur critère (C-008),
// et le tableau du corpus de référence que le catalogue Go doit reproduire (BR-001-1).
package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/agbruneau/leaklab/lab/corpus"
)

// Hypothesis est une ligne du tableau des hypothèses.
type Hypothesis struct {
	ID, Source, Statement, Criterion string
}

// Digest rend l'empreinte SHA-256 du texte du critère (C-008).
func (h Hypothesis) Digest() string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(h.Criterion)))
	return hex.EncodeToString(sum[:])
}

// cells découpe une ligne de tableau Markdown ; ok est faux si la ligne n'en est pas une.
func cells(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil, false
	}
	parts := strings.Split(line[1:len(line)-1], "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, true
}

func lines(doc []byte) []string {
	return strings.Split(strings.ReplaceAll(string(doc), "\r\n", "\n"), "\n")
}

// Hypotheses rend les hypothèses du document, dans leur ordre.
func Hypotheses(doc []byte) ([]Hypothesis, error) {
	var hs []Hypothesis
	for _, l := range lines(doc) {
		c, ok := cells(l)
		if !ok || !strings.HasPrefix(c[0], "H-") {
			continue
		}
		if len(c) != 5 {
			return nil, fmt.Errorf("ligne de %s : %d cellules, 5 attendues", c[0], len(c))
		}
		hs = append(hs, Hypothesis{ID: c[0], Source: c[1], Statement: c[2], Criterion: c[3]})
	}
	if len(hs) == 0 {
		return nil, fmt.Errorf("aucune hypothèse dans le document")
	}
	return hs, nil
}

// Digests rend l'empreinte de chaque hypothèse, par identifiant.
func Digests(hs []Hypothesis) map[string]string {
	m := make(map[string]string, len(hs))
	for _, h := range hs {
		m[h.ID] = h.Digest()
	}
	return m
}

// CaseRow est une ligne du tableau « Corpus de référence ».
type CaseRow struct {
	ID          string
	AntiPattern string
	Page        int
	Faulty      bool
	Leak        bool
	Blocks      bool
	Race        bool
	Primitive   string
	Reachable   bool
	Static      string
	Fixes       string
}

func yesNo(s string) (bool, error) {
	switch s {
	case "oui":
		return true, nil
	case "non":
		return false, nil
	}
	return false, fmt.Errorf("booléen %q, « oui » ou « non » attendu", s)
}

// Corpus rend les lignes du tableau du corpus de référence.
func Corpus(doc []byte) ([]CaseRow, error) {
	var rows []CaseRow
	in := false
	for _, l := range lines(doc) {
		if strings.HasPrefix(l, "## ") {
			in = strings.TrimSpace(l) == "## Corpus de référence"
			continue
		}
		c, ok := cells(l)
		if !in || !ok || c[0] == "Cas" || strings.HasPrefix(c[0], "---") {
			continue
		}
		if len(c) != 11 {
			return nil, fmt.Errorf("cas %s : %d cellules, 11 attendues", c[0], len(c))
		}
		row := CaseRow{ID: c[0], AntiPattern: c[1], Primitive: c[7], Static: c[9], Fixes: c[10]}
		if c[2] != "—" {
			p, err := strconv.Atoi(c[2])
			if err != nil {
				return nil, fmt.Errorf("cas %s, page : %w", c[0], err)
			}
			row.Page = p
		}
		if row.Fixes == "—" {
			row.Fixes = ""
		}
		for i, dst := range []*bool{&row.Faulty, &row.Leak, &row.Blocks, &row.Race} {
			b, err := yesNo(c[3+i])
			if err != nil {
				return nil, fmt.Errorf("cas %s : %w", c[0], err)
			}
			*dst = b
		}
		b, err := yesNo(c[8])
		if err != nil {
			return nil, fmt.Errorf("cas %s : %w", c[0], err)
		}
		row.Reachable = b
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tableau du corpus de référence introuvable")
	}
	return rows, nil
}

// CompareCorpus rend les écarts entre le tableau de la spécification et le catalogue (BR-001-1),
// dans l'ordre du tableau ; une liste vide signifie qu'ils sont identiques.
func CompareCorpus(rows []CaseRow, catalog []corpus.Case) []string {
	var diffs []string
	byID := make(map[string]corpus.Case, len(catalog))
	for _, c := range catalog {
		byID[c.ID] = c
	}
	for i, r := range rows {
		c, ok := byID[r.ID]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("%s : absent du catalogue", r.ID))
			continue
		}
		delete(byID, r.ID)
		if i >= len(catalog) || catalog[i].ID != r.ID {
			diffs = append(diffs, fmt.Sprintf("%s : rang %d dans la spécification, autre rang dans le catalogue", r.ID, i+1))
		}
		got := CaseRow{ID: c.ID, AntiPattern: string(c.AntiPattern), Page: c.Page, Faulty: c.Faulty, Leak: c.Leak, Blocks: c.Blocks, Race: c.Race, Primitive: string(c.Primitive), Reachable: c.Reachable, Static: string(c.Static), Fixes: c.Fixes}
		if got != r {
			diffs = append(diffs, fmt.Sprintf("%s : spécification %+v, catalogue %+v", r.ID, r, got))
		}
	}
	for _, c := range catalog {
		if _, extra := byID[c.ID]; extra {
			diffs = append(diffs, fmt.Sprintf("%s : absent de la spécification", c.ID))
		}
	}
	return diffs
}
