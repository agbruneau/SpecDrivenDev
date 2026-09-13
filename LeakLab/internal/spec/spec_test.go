package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/leaklab/lab/corpus"
)

func readRequirements(t *testing.T) []byte {
	t.Helper()
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "requirements.md"))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// TestUC001_BR1_CorpusConformeSpec verrouille BR-001-1 sur les vrais documents : le catalogue Go
// reproduit le tableau du corpus de référence, et chaque cas a son fichier.
func TestUC001_BR1_CorpusConformeSpec(t *testing.T) {
	rows, err := Corpus(readRequirements(t))
	if err != nil {
		t.Fatal(err)
	}
	if diffs := CompareCorpus(rows, corpus.Catalog()); len(diffs) > 0 {
		t.Fatalf("catalogue et spécification divergent :\n%s", strings.Join(diffs, "\n"))
	}
	for _, c := range corpus.Catalog() {
		if _, err := os.Stat(filepath.Join("..", "..", "lab", "corpus", c.File())); err != nil {
			t.Errorf("%s : fichier absent (%v)", c.ID, err)
		}
	}
}

func TestUC001_A2_EcartDetecte(t *testing.T) {
	rows, err := Corpus(readRequirements(t))
	if err != nil {
		t.Fatal(err)
	}
	cat := corpus.Catalog()
	cat[0].Leak = false               // attribut modifié
	cat = append(cat[:1], cat[2:]...) // cas retiré, rangs décalés
	if diffs := CompareCorpus(rows, cat); len(diffs) < 2 {
		t.Fatalf("écarts non signalés : %v", diffs)
	}
}

func TestC008_HypothesesEtEmpreintes(t *testing.T) {
	hs, err := Hypotheses(readRequirements(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 13 || hs[0].ID != "H-001" || hs[12].ID != "H-013" {
		t.Fatalf("hypothèses lues : %d, de %s à %s", len(hs), hs[0].ID, hs[len(hs)-1].ID)
	}
	h := hs[0]
	before := h.Digest()
	h.Criterion += " "
	if h.Digest() != before {
		t.Fatal("un espace final ne doit pas changer l'empreinte")
	}
	h.Criterion = strings.Replace(h.Criterion, "moitié", "tiers", 1)
	if h.Digest() == before {
		t.Fatal("un critère modifié garde son empreinte")
	}
	crlf := []byte(strings.ReplaceAll(string(readRequirements(t)), "\n", "\r\n"))
	hs2, err := Hypotheses(crlf)
	if err != nil || Digests(hs2)["H-005"] != Digests(hs)["H-005"] {
		t.Fatal("les fins de ligne CRLF changent l'empreinte")
	}
}
