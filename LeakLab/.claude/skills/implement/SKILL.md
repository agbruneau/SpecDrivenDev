---
name: implement
description: >
  Implémente ou synchronise UN cas d'utilisation identifié (UC-###) de LeakLab à partir de sa
  spécification dans docs/use-cases, selon l'AI Unified Process. Invoqué uniquement par slash
  command : /implement UC-002. Ne traite jamais une demande sans identifiant.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(go:*), Bash(gofmt:*), Bash(git diff:*), Bash(git status:*), Bash(git log:*)
---
# /implement $ARGUMENTS

Cas d'utilisation demandé : **$ARGUMENTS** (forme `UC-###`). Sans cette forme, arrête-toi et demande l'identifiant.

1. Lis `docs/use-cases/$ARGUMENTS-*.md`, puis dans `docs/requirements.md` seulement les FR/NFR/C/H liés, puis dans `docs/entity-model.md` seulement les entités listées, puis `CLAUDE.md`.
2. Vérifie que le statut du UC est au moins `Approved` ; sinon arrête-toi.
3. `grep -rn "$ARGUMENTS" internal cmd lab` détermine le mode : **création** (rien ne le référence) ou **synchronisation** (diff proportionnel au changement de spécification, rien d'autre).
4. Projection : étapes du scénario → fonctions du paquet de `internal/` nommé par `CLAUDE.md` ; flux alternatifs → erreurs nommées `UC-### A<n>` ; règles `BR-` → fonctions commentées `// BR-<UC>-<n>`.
5. Tests dans le même passage : `TestUC###_MainFlow`, `TestUC###_A<n>_<slug>`, `TestUC###_BR<n>_<slug>`, table-driven. Toute règle d'un critère gelé a un test `TestH###`.
6. Exécute `go vet ./... && go test -race -shuffle=on -count=1 ./...` ; si le corpus a changé, `cd lab && go test -count=1 ./corpus`. Ne conclus pas tant que ce n'est pas vert.

Rends compte : UC, mode, fichiers, projection (élément → fichier:fonction → test), ce qui reste non couvert.
Interdits : modifier `docs/`, écrire dans `results/`, ajouter une dépendance hors bibliothèque standard, « corriger » un cas `faulty` du corpus.
