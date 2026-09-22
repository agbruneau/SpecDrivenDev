package models

import "testing"

// TestUC001_A081_LigneDeCacheParametree : la ligne de cache est un paramètre de Matrix (lot 9 de
// l'audit). À sa valeur par défaut elle ne laisse aucune trace dans l'identifiant — la matrice de
// référence reste M-823d8b5af441 — et à 128 octets elle en donne un autre.
// Mutation : écrire cacheLine dans Canonical sans condition ⇒ l'identifiant historique change.
func TestUC001_A081_LigneDeCacheParametree(t *testing.T) {
	t.Parallel()
	explicit := ReferenceParameters()
	explicit.CacheLineBytes = DefaultCacheLineBytes
	if got := explicit.MatrixID(); got != "M-823d8b5af441" {
		t.Fatalf("ligne de 64 octets explicite : %s, M-823d8b5af441 attendu", got)
	}
	wide := ReferenceParameters()
	wide.CacheLineBytes = 128
	if err := wide.Normalize().Validate(); err != nil {
		t.Fatalf("la matrice de référence doit rester valide à 128 octets : %v", err)
	}
	if wide.MatrixID() == explicit.MatrixID() {
		t.Fatal("une ligne de 128 octets doit donner un autre identifiant")
	}
	odd := ReferenceParameters()
	odd.CacheLineBytes = 32
	if err := odd.Normalize().Validate(); err == nil {
		t.Fatal("une ligne de 32 octets doit être refusée")
	}
	misaligned := MatrixParameters{
		Sizes: []int{8}, PointerFieldVariants: []bool{false}, Profiles: []LifetimeProfile{ProfileLocal},
		PassingModes: PassingModes(), CacheLineBytes: 128,
		Probes: []ProbeSpec{{Kind: ProbePointerChase, Parameter: 16384 + 64}},
	}
	if err := misaligned.Normalize().Validate(); err == nil {
		t.Fatal("un jeu de travail qui n'est pas un multiple de 128 octets doit être refusé")
	}
	if !wide.UsesCacheLine() || (MatrixParameters{Probes: []ProbeSpec{{ProbeAppendGrow, 1000}}}).UsesCacheLine() {
		t.Fatal("UsesCacheLine ne distingue pas les sondes qui parcourent la mémoire")
	}
}
