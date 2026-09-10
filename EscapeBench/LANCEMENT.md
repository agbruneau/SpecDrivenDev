# Lancement du développement — EscapeBench

Procédure de la première itération complète (UC-001 → UC-005, première campagne, premiers verdicts), fondée sur *Spec-Driven Development* (Martinelli, **SDD**), *Agentic Coding with Claude Code* (Marco, **ACC**) et *Building Enterprise Projects with Go* (Shahsavan, **BEPG**). Pages = folios imprimés. Marqueurs : *Confirmé* (livre ou vérification directe), *Adaptation* (écart assumé), *À vérifier* (comportement de Claude Code non vérifié ici).

## 0. Prérequis

| Élément | Exigence | Source |
|---|---|---|
| Go | ≥ 1.25 (`go version`) ; `go.mod` déclare `go 1.25`, la toolchain se télécharge seule si `GOTOOLCHAIN=auto` | C-001 ; vérifié : le squelette compile avec go1.25.0 et go1.27.0 (poste au 2026-09-10 ; la campagne consigne la version réelle, NFR-001) |
| Claude Code | version 2.x ; sous Windows, Git Bash natif suffit (hooks vérifiés avec des chemins `C:\...` le 2026-09-10) ; WSL2 reste possible | ACC p. 20, 67 ; `.claude/hooks/*.sh` |
| `jq`, `bash`, `git` | requis par les hooks | `.claude/hooks/*.sh` |
| `make` | requis par `CLAUDE.md` (`make vet test`) et par `/implement` ; **absent du poste Windows au 2026-09-10** — installer (`winget install GnuWin32.Make`) ou travailler en WSL2 | `Makefile` |
| Docker | non requis pour P1 | — |
| `benchstat` | optionnel : `go install golang.org/x/perf/cmd/benchstat@latest` | C-002 (hors BEPG) |
| Plugin `aiup-core` | optionnel, pour rédiger de futurs UC : `/plugin marketplace add ai-unified-process/marketplace` puis `/plugin install aiup-core` | unifiedprocess.ai/tools.html ; ACC passe par l'UI `/plugin` (p. 126) |

## 1. Créer le dépôt

1. Copier le dossier `EscapeBench/` hors de `Prospection/` (dépôt dédié, ex. `github.com/agbruneau/escapebench` — module déjà nommé ainsi dans `go.mod`).
2. `git init && git add -A && git commit -m "init: noyau de spécification AIUP et outillage Claude Code"`.
3. `make vet test` doit réussir (aucun test, compilation propre).
4. `bash .claude/hooks/selftest.sh` doit afficher « hooks : tous les contrôles passent ».
5. `chmod +x .claude/hooks/*.sh` si les droits ont été perdus à la copie.

Le squelette contient des `doc.go` par couche et un `main.go` qui refuse toutes les sous-commandes : c'est le point de départ voulu par SDD (p. 121 : le scaffold « is only the foundation »).

## 2. Première session : vérifier l'outillage

```
cd escapebench && claude
```

| Commande | Attendu | Source |
|---|---|---|
| `/hooks` | quatre entrées : PreToolUse `guard-paths`, PostToolUse `go-check` + `spec-lint`, Stop `go-test` ; portée projet | ACC p. 54–55 |
| `/agents` | `spec-reviewer`, `code-reviewer` (projet) | ACC p. 172–173 |
| `/` | skills `spec-review`, `implement`, `go-test`, `spec-coverage`, `bench`, `refute` | ACC p. 70–72 |
| `/mcp` | vide — aucun serveur, par conception | ACC p. 103–105 (pollution de contexte) |
| `/context` | mesure de référence : system prompt + outils + `CLAUDE.md` ; les skills ne coûtent que leur frontmatter (~100–200 tokens chacun, ACC p. 272) | ACC p. 108–109 |

Si un hook ne se déclenche pas : lancer `bash .claude/hooks/selftest.sh`, vérifier le chemin (`$CLAUDE_PROJECT_DIR`) et les droits d'exécution, puis relancer Claude Code (ACC p. 55–56). *Confirmé* dans la documentation hooks de Claude Code (2026-09-10) : code de sortie 2 = blocage en `PreToolUse`, avertissement montré à Claude en `PostToolUse`, poursuite forcée au `Stop` ; `stop_hook_active` vaut vrai quand un hook `Stop` tourne déjà pour le tour. Le matcher `MultiEdit` n'existe plus dans la documentation courante : les hooks portent sur `Edit|Write`.

Ne pas lancer `/init` : `CLAUDE.md` est rédigé à la main et doit rester court (ACC p. 59 ; SDD p. 72) ; `/init` le remplacerait par une description générique du dépôt.

## 3. Revue et approbation des spécifications (avant tout code)

Pour chaque UC, dans l'ordre UC-001 → UC-005 :

1. `/spec-review UC-00n` — le sous-agent `spec-reviewer` applique les critères SDD ch. 3–4 et répond `APPROVE` ou `REVISE`.
2. Corriger les constatations bloquantes **dans le fichier du UC** (jamais dans le code) ; le hook `spec-lint` vérifie la forme à chaque sauvegarde.
3. Relecture humaine à froid, de préférence dans une autre session (*Adaptation* de SDD p. 151 : « reviewed by someone who did not write it »).
4. Remplacer `**Status:** Reviewed` par `**Status:** Approved` ; commit `UC-00n: spec approved`.

Le critère de réfutation d'une `H-###` est gelé dès que le premier UC qui la porte est `Approved` (guide, §4) ; toute modification ultérieure crée une nouvelle hypothèse.

## 4. Implémentation, UC par UC

Ordre imposé par les préconditions : UC-001 (matrice) → UC-002 (échappement) → UC-003 (campagne) → UC-004 (comparaison) → UC-005 (verdicts et tableau de bord). Ces cinq cas sont couplés (mêmes entités, sorties enchaînées) : ne pas les paralléliser en *worktrees* (ACC p. 316–317 : couplage fort, état partagé, petites tâches).

Pour chaque UC :

| Étape | Commande | Critère de passage |
|---|---|---|
| Generate | `/implement UC-00n` | rapport « Projection » complet, `make vet test` vert |
| Tests | `/go-test UC-00n` | matrice flux/règle → test sans trou |
| Validate | `/spec-coverage UC-00n` | ligne du tableau de bord `Unit ✔` |
| Review | « Demande au sous-agent code-reviewer une revue de UC-00n » | `Verdict : CONFORME` ; sinon corriger la spec ou relancer `/implement` |
| Refine | toute divergence → `docs/` d'abord (SDD p. 130) | — |
| Commit | `git commit -m "UC-00n: <résumé>"` — spec et code dans le même diff (SDD p. 75) | statut `Implemented` puis `Verified` |

Hygiène de contexte (ACC) : `/clear` entre deux UC (p. 49) ; `/compact` si `/context` dépasse la moitié de la fenêtre (p. 36, 50) ; `Esc Esc` / `/rewind` pour annuler une génération ratée sans perdre la conversation (p. 66–69) ; un seul UC par session.

Modèle : plan mode (`/plan`, lecture seule, ACC p. 152) avec Opus pour toute nouvelle spécification ; Sonnet pour `/implement` (ACC p. 33, 36, 47) — *choix de l'auteur d'ACC, à ajuster selon l'abonnement*.

## 5. Première campagne et premiers verdicts

1. `make matrix` (UC-001, matrice de référence) puis `make escape MATRIX=<id>` (UC-002) — sans agent : ce sont des exécutions déterministes du binaire.
2. Contrôle de H-006 sans campagne : `make verdict CAMPAIGN=` n'est pas applicable ; utiliser `./bin/escapebench verdict --escape results/escape/<matrixId>/<fichier>.json`.
3. Campagne de fumée sur la plus petite hypothèse : `/bench H-005` (préallocation, deux Probe) ; vérifier la Provenance et la durée.
4. Campagne complète : `make campaign MATRIX=<id> COUNT=20` en terminal (durée cible NFR-005 : < 60 min), ou en exécution non surveillée `claude -p "/bench H-001 H-002 H-003 H-004"` (*À vérifier* : invocation d'un skill en mode `-p`, ACC p. 269 ne montre que des prompts libres). Pendant la campagne, `results/.campaign-lock` fige `internal/harness/`.
5. `make compare CAMPAIGN=<id>` puis `/refute <id>` ; lire `docs/dashboard.md` ; commit `H-00x: verdict <CONFIRMED|REFUTED|INCONCLUSIVE>` avec les fichiers de `results/`.
6. Répéter sur `arm64` si disponible (C-006) : nouvelle campagne, mêmes critères.

## 6. Définition de « terminé »

Un UC est terminé quand il est `Approved`, synchronisé avec le code et protégé par des tests couvrant scénario principal et flux alternatifs (SDD p. 142). Une hypothèse est terminée quand un verdict cite ses fichiers de résultats et que `docs/dashboard.md` le reflète. Le projet P1 est terminé quand H-001 à H-006 ont un verdict sur au moins une architecture.

## 7. Compromis, alternative, renversement

**Compromis.** Quatre hooks et six skills ajoutent un coût fixe par session ; ils garantissent en échange les invariants de protocole (harnais figé, résultats immuables, critères gelés) sans dépendre de la discipline du moment.

**Alternative.** Démarrer avec `CLAUDE.md` seul et `/implement` (SDD p. 108 : « use case specifications alone… most of the benefit ») ; ajouter hooks et sous-agents « when you feel the pain that the tool solves » (SDD p. 107).

**Renversement.** Si les hooks bloquent trop souvent des actions légitimes (ex. régénération du harnais entre deux campagnes), assouplir `guard-paths.sh` plutôt que de le désactiver ; si le sous-agent réviseur ne trouve jamais de constatation bloquante après trois UC, le réserver aux UC nouveaux et garder la relecture humaine seule pour les synchronisations.
