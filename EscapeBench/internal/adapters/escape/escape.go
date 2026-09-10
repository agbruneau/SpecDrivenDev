// Package escape classe les lignes de diagnostic du compilateur en catégories d'échappement
// (UC-002, BR-002-1). Les motifs reconnus sont une implémentation : ils vivent ici et non dans le
// cas d'utilisation. La classification n'est pas déduite du profil de la cellule — c'est
// précisément ce que H-006 met à l'épreuve — mais de la sortie du compilateur et de l'usage
// syntaxique de la valeur qui échappe.
package escape

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
)

// diagnostic est une ligne `fichier:ligne:colonne: message` de `-gcflags=-m`.
type diagnostic struct {
	File    string
	Line    int
	Column  int
	Message string
	Raw     string
}

// positionRe capture la position d'une ligne de diagnostic. Le chemin est glouton : un chemin
// Windows absolu (`C:\...`) est donc correctement séparé du numéro de ligne.
var positionRe = regexp.MustCompile(`^(.*):(\d+):(\d+): (.*)$`)

// movedToHeapRe capture l'identifiant déplacé sur le tas.
var movedToHeapRe = regexp.MustCompile(`^moved to heap: (\w+)$`)

// Classifier applique les motifs de classification à la sortie du compilateur.
type Classifier struct{}

// New construit un classificateur.
func New() Classifier { return Classifier{} }

// IsEscape indique si un message du compilateur signale un échappement. « does not escape » est
// exclu : il contient le mot « escape » mais affirme le contraire.
func IsEscape(message string) bool {
	if strings.Contains(message, "does not escape") {
		return false
	}
	return strings.Contains(message, "escapes to heap") || strings.HasPrefix(message, "moved to heap:")
}

// parseDiagnostics retient les lignes qui signalent un échappement, triées par position.
func parseDiagnostics(lines []string) []diagnostic {
	var out []diagnostic
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		match := positionRe.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		if !IsEscape(match[4]) {
			continue
		}
		line, _ := strconv.Atoi(match[2])
		column, _ := strconv.Atoi(match[3])
		out = append(out, diagnostic{File: match[1], Line: line, Column: column, Message: match[4], Raw: trimmed})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Column < out[j].Column
	})
	return out
}

// Classify rend l'EscapeVerdict d'une cellule à partir des lignes brutes du compilateur.
// sourcePath est le chemin du fichier source de la cellule, relatif à matrixDir.
func (Classifier) Classify(_ context.Context, matrixDir, subjectID, sourcePath string, lines []string) (models.EscapeVerdict, error) {
	verdict := models.EscapeVerdict{CellID: subjectID, Status: models.EscapeStatusOK, Category: models.CategoryNone}
	diags := parseDiagnostics(lines)
	if len(diags) == 0 {
		return verdict, nil
	}
	verdict.Escapes = true

	file, err := parseSource(filepath.Join(matrixDir, filepath.FromSlash(sourcePath)))
	if err != nil {
		// Sans arbre syntaxique la cause reste inconnue : c'est une observation OTHER, pas une
		// erreur de classification (BR-002-2).
		verdict.CompilerReason = diags[0].Raw
		verdict.Category = models.CategoryOther
		return verdict, nil
	}

	fallback := diagnostic{}
	for i, diag := range diags {
		category := classifyDiagnostic(file, diag)
		if i == 0 {
			fallback = diag
		}
		if category != models.CategoryOther {
			verdict.Category = category
			verdict.CompilerReason = diag.Raw
			return verdict, nil
		}
	}
	verdict.Category = models.CategoryOther
	verdict.CompilerReason = fallback.Raw
	return verdict, nil
}

// parsedFile associe un arbre syntaxique à son jeu de positions.
type parsedFile struct {
	fset *token.FileSet
	file *ast.File
}

// parseSource analyse un fichier source généré.
func parseSource(path string) (*parsedFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture du source %s : %w", path, err)
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, content, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("analyse du source %s : %w", path, err)
	}
	return &parsedFile{fset: fset, file: parsed}, nil
}

// classifyDiagnostic rend la catégorie d'une ligne de diagnostic.
func classifyDiagnostic(file *parsedFile, diag diagnostic) models.EscapeCategory {
	// Une closure qui échappe est la cause « capture par closure », sans ambiguïté.
	if strings.HasPrefix(diag.Message, "func literal escapes to heap") {
		return models.CategoryClosureCapture
	}
	name := escapedIdentifier(diag.Message)
	if name == "" {
		return models.CategoryOther
	}
	body := enclosingBody(file, diag.Line)
	if body == nil {
		return models.CategoryOther
	}
	return classifyUsage(body, name)
}

// escapedIdentifier rend le nom de la variable déplacée sur le tas, ou la chaîne vide si le
// message ne nomme pas une variable simple.
func escapedIdentifier(message string) string {
	if match := movedToHeapRe.FindStringSubmatch(message); match != nil {
		return match[1]
	}
	if rest, ok := strings.CutSuffix(message, " escapes to heap"); ok {
		rest = strings.TrimPrefix(rest, "&")
		if isIdentifier(rest) {
			return rest
		}
	}
	return ""
}

// isIdentifier indique si s est un identifiant Go simple.
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// enclosingBody rend le corps de la fonction (déclarée ou littérale) la plus interne qui contient
// la ligne donnée.
func enclosingBody(file *parsedFile, line int) *ast.BlockStmt {
	var best *ast.BlockStmt
	bestSpan := 1 << 30
	ast.Inspect(file.file, func(node ast.Node) bool {
		var body *ast.BlockStmt
		switch fn := node.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		default:
			return true
		}
		if body == nil {
			return true
		}
		start := file.fset.Position(body.Pos()).Line
		end := file.fset.Position(body.End()).Line
		if line < start || line > end {
			return true
		}
		if span := end - start; span < bestSpan {
			bestSpan = span
			best = body
		}
		return true
	})
	return best
}

// classifyUsage détermine la cause d'échappement d'une variable d'après son usage syntaxique dans
// le corps englobant. Les porteurs d'adresse (`p := &v`) sont suivis sur un niveau d'alias.
//
// Deux règles évitent des attributions fausses. D'abord, seul un usage qui emporte l'adresse de la
// variable compte pour le retour, l'envoi sur canal et le stockage : `return v` rend une copie et
// ne fait rien échapper. Ensuite, la marche ne descend pas dans le corps d'une closure : ce qui s'y
// trouve appartient à la closure, pas à la fonction englobante — un `return p` interne serait
// autrement lu comme un retour de pointeur.
func classifyUsage(body *ast.BlockStmt, name string) models.EscapeCategory {
	aliases := addressCarriers(body, name)
	captured := map[string]bool{name: true}
	for alias := range aliases {
		captured[alias] = true
	}
	category := models.CategoryOther
	rank := func(c models.EscapeCategory) int {
		switch c {
		case models.CategoryReturnPointer:
			return 4
		case models.CategoryClosureCapture:
			return 3
		case models.CategoryChannelSend:
			return 2
		case models.CategoryContainerStore:
			return 1
		}
		return 0
	}
	consider := func(c models.EscapeCategory) {
		if rank(c) > rank(category) {
			category = c
		}
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.FuncLit:
			if referencesAny(stmt.Body, captured) {
				consider(models.CategoryClosureCapture)
			}
			return false
		case *ast.ReturnStmt:
			for _, result := range stmt.Results {
				if carriesAddress(result, name, aliases) {
					consider(models.CategoryReturnPointer)
				}
			}
		case *ast.SendStmt:
			if carriesAddress(stmt.Value, name, aliases) {
				consider(models.CategoryChannelSend)
			}
		case *ast.AssignStmt:
			for i, lhs := range stmt.Lhs {
				// Le livre range parmi les conteneurs déjà sur le tas la map, la tranche et la
				// struct : une affectation dans un élément indexé comme dans un champ compte.
				if !isContainerTarget(lhs) {
					continue
				}
				if i < len(stmt.Rhs) && carriesAddress(stmt.Rhs[i], name, aliases) {
					consider(models.CategoryContainerStore)
				}
			}
		case *ast.CallExpr:
			if ident, ok := stmt.Fun.(*ast.Ident); ok && ident.Name == "append" {
				for _, arg := range stmt.Args[min(1, len(stmt.Args)):] {
					if carriesAddress(arg, name, aliases) {
						consider(models.CategoryContainerStore)
					}
				}
			}
		}
		return true
	})
	return category
}

// isContainerTarget indique si la cible d'une affectation désigne l'intérieur d'un conteneur :
// un élément indexé, ou le champ d'une struct. Une simple variable n'en est pas un.
//
// Révision du 2026-09-10 : la branche `*ast.StarExpr` a été retirée. Aucun gabarit du harnais ne
// produit d'affectation par déréférencement (`*x = ...`) ; la branche était inatteignable et
// n'élargissait que la surface de faux positifs.
func isContainerTarget(lhs ast.Expr) bool {
	switch lhs.(type) {
	case *ast.IndexExpr, *ast.SelectorExpr:
		return true
	}
	return false
}

// carriesAddress indique si l'expression emporte l'adresse de name, directement (`&name`) ou par
// un alias déjà reconnu.
func carriesAddress(node ast.Node, name string, aliases map[string]bool) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		switch expr := n.(type) {
		case *ast.UnaryExpr:
			if expr.Op == token.AND && referencesAny(expr.X, map[string]bool{name: true}) {
				found = true
				return false
			}
		case *ast.Ident:
			if aliases[expr.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// addressCarriers rend les identifiants qui portent l'adresse de name, name exclu. Deux passes
// suffisent aux chaînes d'alias produites par le harnais (`p := &t`).
func addressCarriers(body *ast.BlockStmt, name string) map[string]bool {
	carriers := map[string]bool{name: true}
	for pass := 0; pass < 2; pass++ {
		ast.Inspect(body, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || i >= len(assign.Rhs) {
					continue
				}
				if unary, ok := assign.Rhs[i].(*ast.UnaryExpr); ok && unary.Op == token.AND {
					if referencesAny(unary.X, carriers) {
						carriers[ident.Name] = true
					}
				}
			}
			return true
		})
	}
	delete(carriers, name)
	return carriers
}

// referencesAny indique si l'expression cite au moins un des identifiants donnés.
func referencesAny(node ast.Node, names map[string]bool) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && names[ident.Name] {
			found = true
			return false
		}
		return !found
	})
	return found
}
