---
name: go-test
description: >
  Complète ou révise les tests Go d'un cas d'utilisation identifié (UC-###) d'EscapeBench :
  un test par flux et par règle, table-driven, sous-tests nommés, testing/synctest pour le
  temporel et le concurrent, exécution avec -race et -shuffle=on. Invoqué uniquement par
  slash command : /go-test UC-003.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(go:*), Bash(gofmt:*), Bash(make:*), Bash(git diff:*)
---
# /go-test $ARGUMENTS

Cas d'utilisation : **$ARGUMENTS**. Sans identifiant `UC-###`, arrête-toi.

## 1. Lire
`docs/use-cases/$ARGUMENTS-*.md`, `CLAUDE.md`, puis les fichiers de `internal/` qui référencent `$ARGUMENTS` et leurs tests existants.

## 2. Construire la matrice attendue
Une ligne par élément vérifiable du UC :
- scénario principal → `TestUC###_MainFlow` ;
- chaque flux alternatif `Ak` → `TestUC###_Ak_<slug>` ;
- chaque règle `BR-<UC>-<n>` → `TestUC###_BRn_<slug>` ;
- chaque postcondition d'échec → assertion « état inchangé » dans le test du flux correspondant.

## 3. Écrire ou compléter les tests
- Table-driven (`[]struct{name string; ...}` + `t.Run`), un cas par ligne de la matrice manquante ; `t.Parallel()` seulement si le cas n'utilise ni disque partagé ni horloge.
- Assertions sur les **postconditions** (état des fichiers, contenu retourné, erreur typée), pas sur des détails internes.
- Adapters factices en mémoire : `internal/service/fakes_test.go` (`memoryStore`, `fakeRunner`, `fakeDigester`, `fakeProvenance`…). Les étendre plutôt qu'en créer d'autres ; aucun appel réel à `go build`/`go test` dans un test unitaire. Les tests qui exercent les adapters réels portent `//go:build integration_test` et tournent par `make integration_test`.
- `testing/synctest` (Go 1.25) pour tout délai, ticker ou goroutine ; jamais de `time.Sleep`.
- Provenance et empreintes : tests avec des valeurs fixes injectées, jamais l'environnement réel.
- Pour chaque test créé, applique le troisième contrôle de SDD (p. 95) : indique en commentaire la mutation qui doit le faire échouer (`// Mutation : retirer la vérification BR-003-1 ⇒ échec attendu`).

## 4. Exécuter
`go test -race -shuffle=on -count=1 ./...` puis `go vet ./...`. En cas d'échec lié au code (et non au test), ne corrige pas le code : rapporte la déviation, elle relève de `/implement` ou de la spécification.

## 5. Rendre compte (format obligatoire)
```
UC : $ARGUMENTS
Matrice : <élément → test → présent avant | ajouté>
Résultat : <go test, une ligne>
Déviations code/spécification détectées : <liste ou « aucune »>
```
