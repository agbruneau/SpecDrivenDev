// Command escapebench est le composition root du banc (BEPG ch. 14, p. 361 ; ch. 16, p. 414).
// Chaque sous-commande correspond à un cas d'utilisation de docs/use-cases :
//
//	matrix    UC-001  Générer la matrice de cellules
//	escape    UC-002  Classer l'échappement
//	campaign  UC-003  Exécuter une campagne de mesure
//	compare   UC-004  Comparer valeur et pointeur
//	verdict   UC-005  Produire les verdicts
//	dashboard UC-005 (étape 7) / FR-007
//
// Le câblage des services et adapters est ajouté par /implement UC-### ; ce fichier ne
// contient aucune logique métier.
package main

import (
	"fmt"
	"os"
)

const usage = `usage: escapebench <matrix|escape|campaign|compare|verdict|dashboard> [options]

Sous-commandes (une par cas d'utilisation, voir docs/use-cases) :
  matrix     UC-001  --reference | --params <spec>
  escape     UC-002  --matrix <id>
  campaign   UC-003  --matrix <id> --count <n≥20> [--hypotheses H-001,...]
  compare    UC-004  --campaign <id>
  verdict    UC-005  --campaign <id> [--escape <fichier>]
  dashboard  UC-005  (régénère docs/dashboard.md)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "matrix", "escape", "campaign", "compare", "verdict", "dashboard":
		fmt.Fprintf(os.Stderr, "escapebench %s : non implémenté — voir docs/use-cases et /implement\n", os.Args[1])
		os.Exit(3)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}
