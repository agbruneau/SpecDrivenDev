package specs

import "testing"

// TestUC005_ContrainteIntegration verrouille A-278 : la colonne Integration du tableau de bord ne
// se fonde que sur une vraie ligne de contrainte de build en tête de fichier. La chercher comme
// sous-chaîne rendait la colonne entièrement fausse — un littéral de chaîne suffisait.
func TestUC005_ContrainteIntegration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"contrainte en tête", "//go:build integration_test\n\npackage x\n", true},
		{"contrainte composée", "//go:build integration_test && windows\n\npackage x\n", true},
		{"contrainte alternative sans exigence", "//go:build integration_test || windows\n\npackage x\n", false},
		{"autre contrainte", "//go:build windows\n\npackage x\n", false},
		{"négation d'une autre contrainte", "//go:build !windows\n\npackage x\n", false},
		{"aucune contrainte", "package x\n", false},
		{
			name: "littéral de chaîne après la clause de paquet",
			text: "package x\n\nconst tag = \"//go:build integration_test\"\n",
			want: false,
		},
		{
			name: "commentaire de documentation avant le paquet",
			text: "// Les tests taggés //go:build integration_test tournent à part.\npackage x\n",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := requiresIntegrationTag(tc.text); got != tc.want {
				t.Fatalf("requiresIntegrationTag = %v, attendu %v", got, tc.want)
			}
		})
	}
}
