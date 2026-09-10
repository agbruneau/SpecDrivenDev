---
name: spec-coverage
description: >
  Calcule la couverture de spécification d'un cas d'utilisation (UC-###) : matrice flux/règle →
  test, à partir de docs/use-cases et des tests Go, puis met à jour la ligne du UC dans
  docs/dashboard.md. Invoqué par slash command : /spec-coverage UC-002.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Bash(go test:*), Bash(go vet:*), Bash(go run:*), Bash(git log:*)
---
# /spec-coverage $ARGUMENTS

Couverture = complétude de la spécification, pas lignes de code (SDD, p. 96).

1. Lis `docs/use-cases/$ARGUMENTS-*.md` et énumère : scénario principal, flux alternatifs `Ak`, règles `BR-<UC>-<n>`, postconditions d'échec.
2. Liste les tests existants : `grep -rn "func TestUC$(echo $ARGUMENTS | tr -d 'UC-')" internal cmd` (les noms suivent `TestUC###_...`).
3. Exécute `go test -race -count=1 -run 'TestUC<###>' ./...` et consigne le résultat.
4. Construis la matrice :

| Élément | Test | Présent | Passe |
|---|---|---|---|

5. Détermine les colonnes du tableau de bord pour ce UC : `Code` = ✔ si du code référence le UC ; `Unit` = ✔ si tous les éléments ont un test qui passe ; `Integration` = ✔ si un test sous `//go:build integration_test` existe et passe, `—` si le UC n'en requiert pas ; `Regression` = ✔ si la suite complète passe ; `Integrity` = Strong (tout ✔), Partial (Code ✔ mais Unit ✕), Weak (Code ✕).
6. Régénère le tableau de bord avec `go run ./cmd/escapebench dashboard` (UC-005 étape 7, FR-007) si la sous-commande existe ; sinon rapporte la ligne calculée pour `$ARGUMENTS` et signale que le tableau sera régénéré à la livraison de UC-005. N'édite jamais `docs/dashboard.md` directement (BR-005-3, hook guard-paths).
7. Rapporte la matrice, la ligne du tableau de bord et la liste des éléments sans test — ces derniers sont des trous de protection, pas des lignes non couvertes.
