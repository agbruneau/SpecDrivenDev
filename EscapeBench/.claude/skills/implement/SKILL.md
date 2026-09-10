---
name: implement
description: >
  Implémente ou synchronise UN cas d'utilisation identifié (UC-###) d'EscapeBench à partir de sa
  spécification dans docs/use-cases, selon l'AI Unified Process : code Go hexagonal dans internal/
  et tests nommés par flux. Invoqué uniquement par slash command : /implement UC-003.
  Ne traite jamais une demande sans identifiant de cas d'utilisation.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(go:*), Bash(gofmt:*), Bash(make:*), Bash(git diff:*), Bash(git status:*), Bash(git log:*)
---
# /implement $ARGUMENTS

Cas d'utilisation demandé : **$ARGUMENTS** (forme attendue : `UC-###`). Si l'argument est absent ou n'a pas cette forme, arrête-toi et demande l'identifiant.

## 1. Lire, dans cet ordre
1. `docs/use-cases/$ARGUMENTS-*.md` — la source de vérité du comportement.
2. `docs/requirements.md` — seulement les FR/NFR/C/H listés dans « Linked Requirements » et « Linked Hypotheses ».
3. `docs/entity-model.md` — seulement les entités listées dans « Entities ».
4. `CLAUDE.md` — règles de construction.
5. Le code existant sous `internal/` et `cmd/` : `grep -rn "$ARGUMENTS" internal cmd` détermine le mode.

## 2. Vérifier avant d'écrire
- `**Status:**` du UC est `Approved`. Sinon, arrête-toi : « UC non approuvé ; lancer /spec-review puis approuver ». Aucun code n'est produit pour un UC `Draft` ou `Reviewed`.
- `results/.campaign-lock` est absent si le UC touche `internal/harness/`.

## 3. Mode
- **Création** : aucun code ne référence `$ARGUMENTS`. Produis l'implémentation complète.
- **Synchronisation** : du code référence déjà `$ARGUMENTS`. Compare la spécification actuelle au code, applique uniquement le changement de comportement, ne réécris rien d'autre, ne renomme rien qui n'est pas concerné. Le diff doit être proportionnel au changement de spécification.

## 4. Projeter le cas d'utilisation sur le code (SDD, p. 71)
| Élément du UC | Cible |
|---|---|
| Préconditions | Vérifications en tête du service ; erreurs typées retournées, jamais de `panic` |
| Étapes du scénario principal | Méthode du service dans `internal/service`, une fonction par étape observable si elle dépasse quelques lignes |
| Flux alternatifs A1..An | Branches explicites ; chaque trigger « À l'étape N » correspond à un point de décision nommé |
| Postconditions succès/échec | Garanties par le code (écriture atomique, aucun fichier partiel) et assertées par les tests |
| Règles `BR-` | Fonctions ou vérifications nommées d'après la règle, avec commentaire `// BR-<UC>-<n>` |
| Entités | Types dans `internal/models`, sans tag JSON ; les DTO et l'encodage vivent dans `internal/adapters` |

Ports dans `internal/ports` uniquement pour ce que le service consomme (système de fichiers, compilateur, runner, horloge) ; adapters dans `internal/adapters/<nom>`. Le *composition root* est `cmd/escapebench/main.go` (sous-commande par UC : `matrix`, `escape`, `campaign`, `compare`, `verdict`).

## 5. Tests (obligatoires dans le même passage)
- Un test par flux et par règle : `TestUC###_MainFlow`, `TestUC###_A1_<slug>`, `TestUC###_BR1_<slug>`, table-driven avec sous-tests nommés.
- Adapters factices en mémoire pour les tests de service ; aucun accès réel au compilateur ou au disque dans `internal/service`.
- Tout comportement temporel ou concurrent est testé sous `testing/synctest`.
- Exécute `make vet test` ; ne conclus pas tant que la suite ne passe pas.

## 6. Rendre compte (format obligatoire)
```
UC : $ARGUMENTS — mode : création | synchronisation
Fichiers : <liste>
Projection : <élément du UC → fichier:fonction → test>
Non couvert (à revoir dans la spécification) : <liste ou « aucun »>
Commande de revue suggérée : demander au sous-agent code-reviewer une revue de $ARGUMENTS
```
Interdits : modifier `docs/` (sauf `docs/dashboard.md`, jamais), écrire dans `results/` ou `matrices/`, ajouter une dépendance hors bibliothèque standard, implémenter un comportement absent du UC.
