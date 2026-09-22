package cli

import "testing"

// TestUC001_A081_CleCacheline : la clé cacheline passe la ligne de cache à la Matrix (lot 9 de
// l'audit) et n'accepte qu'un entier.
func TestUC001_A081_CleCacheline(t *testing.T) {
	t.Parallel()
	params, err := ParseParameters("sizes=8;cacheline=128")
	if err != nil || params.CacheLineBytes != 128 {
		t.Fatalf("cacheline=128 : %+v, %v", params.CacheLineBytes, err)
	}
	if _, err := ParseParameters("sizes=8;cacheline=64,128"); err == nil {
		t.Fatal("cacheline doit refuser deux valeurs")
	}
}
