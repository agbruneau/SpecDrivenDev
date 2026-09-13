// Command scenario est le détecteur PROGRAM (C-006) : il exécute un cas du corpus dans la fonction
// main d'un programme ordinaire.
package main

import (
	"fmt"
	"os"

	"github.com/agbruneau/leaklab/lab/corpus"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage : scenario <cas>")
		os.Exit(2)
	}
	c, ok := corpus.Lookup(os.Args[1])
	if !ok {
		fmt.Fprintf(os.Stderr, "cas inconnu : %s\n", os.Args[1])
		os.Exit(2)
	}
	if err := c.Scenario(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
