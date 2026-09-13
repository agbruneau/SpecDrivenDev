// Package ctxvet repère les appels d'I/O qui ignorent le contexte (UC-003), par analyse syntaxique
// seule : bibliothèque standard (C-002), sans résolution de types.
package ctxvet

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Diagnostic est un appel visé par BR-003-1.
type Diagnostic struct {
	File        string // chemin tel que lu
	Line, Col   int
	Call        string // paquet.Fonction, avec le nom d'import du fichier
	Replacement string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d: ctxvet: %s ignore le contexte ; utiliser %s", d.File, d.Line, d.Col, d.Call, d.Replacement)
}

// targets est la table BR-003-1, indexée par chemin d'import puis par fonction.
var targets = map[string]map[string]string{
	"net": {
		"Dial":        "(*net.Dialer).DialContext",
		"DialTimeout": "(*net.Dialer).DialContext",
		"Listen":      "(*net.ListenConfig).Listen",
	},
	"net/http": {
		"Get":        "http.NewRequestWithContext et (*http.Client).Do",
		"Head":       "http.NewRequestWithContext et (*http.Client).Do",
		"Post":       "http.NewRequestWithContext et (*http.Client).Do",
		"PostForm":   "http.NewRequestWithContext et (*http.Client).Do",
		"NewRequest": "http.NewRequestWithContext",
	},
	"os/exec": {
		"Command": "exec.CommandContext",
	},
}

// Analyze analyse les fichiers .go de dir, tests exclus, et rend les diagnostics triés par fichier
// puis par position. Une erreur de lecture ou de syntaxe est rendue sans diagnostic partiel (A1).
func Analyze(dir string) ([]Diagnostic, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	fset := token.NewFileSet()
	var diags []Diagnostic
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("analyse de %s : %w", path, err)
		}
		diags = append(diags, analyzeFile(fset, f)...)
	}
	slices.SortFunc(diags, func(a, b Diagnostic) int {
		return cmp.Or(cmp.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Col, b.Col))
	})
	return diags, nil
}

func analyzeFile(fset *token.FileSet, f *ast.File) []Diagnostic {
	imports := map[string]string{} // nom local -> chemin
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || targets[path] == nil {
			continue
		}
		local := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			local = spec.Name.Name
		}
		if local != "_" && local != "." {
			imports[local] = path
		}
	}
	var diags []Diagnostic
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		// BR-003-2 : un identifiant résolu dans le fichier (Obj non nul) est une variable locale
		// homonyme, pas le paquet importé.
		if !ok || pkg.Obj != nil {
			return true
		}
		path, ok := imports[pkg.Name]
		if !ok {
			return true
		}
		if repl, ok := targets[path][sel.Sel.Name]; ok {
			pos := fset.Position(call.Pos())
			diags = append(diags, Diagnostic{File: pos.Filename, Line: pos.Line, Col: pos.Column, Call: pkg.Name + "." + sel.Sel.Name, Replacement: repl})
		}
		return true
	})
	return diags
}
