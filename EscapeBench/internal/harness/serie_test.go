package harness

import "testing"

// serieArm64Digest est l'empreinte du harnais consignée en D-63, qui ouvre la série de comparaison
// arm64 (lot 9 de l'audit). Les campagnes d'une même série ne se comparent que sous une même
// empreinte : toute retouche de harness.go ou d'un gabarit doit donc être délibérée, et consignée
// dans une nouvelle décision avec la nouvelle valeur.
const serieArm64Digest = "fd4a470c1fea2dc1a366d5ffb26921cf5c14f8a40540c67a2fcf91a962d06100"

// TestUC001_D63_EmpreinteDeLaSerieArm64 fige l'empreinte de la série ouverte par D-63.
// Mutation : modifier un commentaire de harness.go ⇒ échec attendu (A-246).
func TestUC001_D63_EmpreinteDeLaSerieArm64(t *testing.T) {
	t.Parallel()
	if got := newRenderer(t).Digest(); got != serieArm64Digest {
		t.Fatalf("empreinte du harnais %s, D-63 consigne %s : le harnais a changé depuis l'ouverture de la série arm64 ; "+
			"si c'est voulu, consigner une nouvelle décision et mettre à jour cette constante (C-005)", got, serieArm64Digest)
	}
}
