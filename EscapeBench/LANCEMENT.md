# Lancement du développement — EscapeBench

Procédure de la première itération complète (UC-001 → UC-005, première campagne, premiers verdicts), fondée sur *Spec-Driven Development* (Martinelli, **SDD**), *Agentic Coding with Claude Code* (Marco, **ACC**) et *Building Enterprise Projects with Go* (Shahsavan, **BEPG**). Pages = folios imprimés. Marqueurs : *Confirmé* (livre ou vérification directe), *Adaptation* (écart assumé), *À vérifier* (comportement de Claude Code non vérifié ici).

**État au 2026-09-10 : procédure exécutée, projet clos.** Les treize hypothèses ont un verdict et les cinq UC sont `Deployed` (`../Doc/RAPPORT-FINAL_EscapeBench.md`). Les §1 à §4 restent la marche à suivre pour extraire le banc ou ajouter un UC ou une `H-###` ; le §5 donne les commandes qui rejouent les campagnes de clôture.

## 0. Prérequis

| Élément | Exigence | Source |
|---|---|---|
| Go | ≥ 1.25 (`go version`) ; `go.mod` déclare `go 1.25`, la toolchain se télécharge seule si `GOTOOLCHAIN=auto` | C-001 ; le squelette initial compilait avec go1.25.0 et go1.27.0 ; le banc complet a été construit et mesuré avec go1.27.0 seulement (la campagne consigne la version réelle, NFR-001) |
| Claude Code | version 2.x ; sous Windows, Git Bash natif suffit (hooks vérifiés avec des chemins `C:\...` le 2026-09-10) ; WSL2 reste possible | ACC p. 20, 67 ; `.claude/hooks/*.sh` |
| `jq`, `bash`, `git` | requis par les hooks | `.claude/hooks/*.sh` |
| `make` | **facultatif depuis le 2026-09-10** ; la commande de référence est `go vet ./... && go test -race -shuffle=on -count=1 ./...`, et le `Makefile` reste disponible pour qui a `make` | `Makefile` |
| Docker | non requis pour P1 | — |
| `benchstat` | optionnel : `go install golang.org/x/perf/cmd/benchstat@latest` | C-002 (hors BEPG) |
| Plugin `aiup-core` | optionnel, pour rédiger de futurs UC : `/plugin marketplace add ai-unified-process/marketplace` puis `/plugin install aiup-core` | unifiedprocess.ai/tools.html ; ACC passe par l'UI `/plugin` (p. 126) |

## 1. Créer le dépôt

1. Copier le dossier `EscapeBench/` hors de `Prospection/` (dépôt dédié, ex. `github.com/agbruneau/escapebench` — module déjà nommé ainsi dans `go.mod`). `results/` suit la copie ; `matrices/` et `bin/`, ignorés par Git, se régénèrent (§5). Les liens `../Doc/`, `../Campagnes/` et `../Revue/` de `README.md` et de ce fichier ne suivent pas : copier ces dossiers avec le banc ou retirer les liens.
2. `git init && git add -A && git commit -m "init: banc EscapeBench (spécification, code, résultats)"`.
3. `go vet ./... && go test -race -shuffle=on -count=1 ./...` doit réussir.
4. `bash .claude/hooks/selftest.sh` doit afficher « hooks : tous les contrôles passent ».
5. `chmod +x .claude/hooks/*.sh` si les droits ont été perdus à la copie.

Au lancement, le squelette ne contenait que des `doc.go` par couche et un `main.go` qui refusait toutes les sous-commandes : c'est le point de départ voulu par SDD (p. 121 : le scaffold « is only the foundation »). Le banc livré implémente les six sous-commandes `matrix`, `escape`, `campaign`, `compare`, `verdict` et `dashboard`.

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

Fait le 2026-09-10 pour UC-001 à UC-005, **sans l'étape 3** : la relecture humaine n'a pas eu lieu, le passage à `Approved` ayant été assumé sur mandat (`../Doc/DECISION.md`, D-01). La procédure vaut pour tout nouveau UC.

Pour chaque UC, dans l'ordre UC-001 → UC-005 :

1. `/spec-review UC-00n` — le sous-agent `spec-reviewer` applique les critères SDD ch. 3–4 et répond `APPROVE` ou `REVISE`.
2. Corriger les constatations bloquantes **dans le fichier du UC** (jamais dans le code) ; le hook `spec-lint` vérifie la forme à chaque sauvegarde.
3. Relecture humaine à froid, de préférence dans une autre session (*Adaptation* de SDD p. 151 : « reviewed by someone who did not write it »).
4. Remplacer `**Status:** Reviewed` par `**Status:** Approved` ; commit `UC-00n: spec approved`.

Le critère de réfutation d'une `H-###` est gelé dès que le premier UC qui la porte est `Approved` (guide, §4) ; toute modification ultérieure crée une nouvelle hypothèse.

## 4. Implémentation, UC par UC

Fait : UC-001 à UC-005 implémentés, puis passés à `Deployed` à la clôture (D-37).

Ordre imposé par les préconditions : UC-001 (matrice) → UC-002 (échappement) → UC-003 (campagne) → UC-004 (comparaison) → UC-005 (verdicts et tableau de bord). Ces cinq cas sont couplés (mêmes entités, sorties enchaînées) : ne pas les paralléliser en *worktrees* (ACC p. 316–317 : couplage fort, état partagé, petites tâches).

Pour chaque UC :

| Étape | Commande | Critère de passage |
|---|---|---|
| Generate | `/implement UC-00n` | rapport « Projection » complet, `go vet` et la suite de tests au vert |
| Tests | `/go-test UC-00n` | matrice flux/règle → test sans trou |
| Validate | `/spec-coverage UC-00n` | ligne du tableau de bord `Unit ✔` |
| Review | « Demande au sous-agent code-reviewer une revue de UC-00n » | `Verdict : CONFORME` ; sinon corriger la spec ou relancer `/implement` |
| Refine | toute divergence → `docs/` d'abord (SDD p. 130) | — |
| Commit | `git commit -m "UC-00n: <résumé>"` — spec et code dans le même diff (SDD p. 75) | statut `Implemented` puis `Verified` |

Hygiène de contexte (ACC) : `/clear` entre deux UC (p. 49) ; `/compact` si `/context` dépasse la moitié de la fenêtre (p. 36, 50) ; `Esc Esc` / `/rewind` pour annuler une génération ratée sans perdre la conversation (p. 66–69) ; un seul UC par session.

Modèle : plan mode (`/plan`, lecture seule, ACC p. 152) avec Opus pour toute nouvelle spécification ; Sonnet pour `/implement` (ACC p. 33, 36, 47) — *choix de l'auteur d'ACC, à ajuster selon l'abonnement*.

## 5. Campagnes et verdicts

Exécutions déterministes du binaire, sans agent. `make` étant absent du poste de référence, les commandes passent par `go run`. Le `Makefile` en offre les équivalents : `make matrix PARAMS="…"`, `make escape MATRIX=…`, `make campaign MATRIX=… HYPOTHESES=H-###,…`, `make compare CAMPAIGN=…`, `make verdict CAMPAIGN=…` (recettes corrigées le 2026-09-11 : `PARAMS` est désormais cité, et `HYPOTHESES` transmis ; vérifié avec GNU Make 4.4.1).

1. `go run ./cmd/escapebench matrix --reference` (UC-001) puis `go run ./cmd/escapebench escape --matrix <id>` (UC-002). La sortie d'`escape` donne le décompte par cause et le nombre de cellules `OTHER` : c'est le contrôle de H-006 avant toute campagne. Le verdict formel exige une campagne, `verdict` refusant de tourner sans `--campaign`.
2. Campagne de fumée sur la plus petite hypothèse : `/bench H-005` (préallocation, deux Probe) ; vérifier la Provenance et la durée.
3. Campagne complète : `go run ./cmd/escapebench campaign --matrix <id> --count 20 --hypotheses <H-###,…>` en terminal (NFR-005 : 7,6 à 7,8 s par sujet sur le poste de référence, soit 29 min pour la matrice de référence), ou en exécution non surveillée `claude -p "/bench H-001 H-002 H-003 H-004"` (*À vérifier* : invocation d'un skill en mode `-p`, ACC p. 269 ne montre que des prompts libres). Sans `--hypotheses`, la campagne gèle tout le catalogue. Pendant la campagne, `results/.campaign-lock` fige `internal/harness/`.
4. `go run ./cmd/escapebench compare --campaign <id>` puis `/refute <id>` (ou `go run ./cmd/escapebench verdict --campaign <id>`, qui retrouve seul les verdicts d'échappement de la matrice ; `--escape <fichier>` en désigne un précis) ; lire `docs/dashboard.md` ; commit `H-00x: verdict <CONFIRMED|REFUTED|INCONCLUSIVE>` avec les fichiers de `results/`.
5. Répéter sur `arm64` si disponible (C-006) : nouvelle campagne, mêmes critères. **Reste ouvert à la clôture.**

### Rejouer les campagnes de clôture

Les treize verdicts viennent de deux campagnes, qui ne peuvent pas être fusionnées : H-012 exige une matrice à cinq réplicats, sur laquelle UC-003 refuse H-007 (C-009).

| Campagne | Hypothèses | Matrice | Sujets | Durée mesurée |
|---|---|---|---|---|
| `C-2026-09-10-11` | H-001 à H-011, H-013 | `M-b44a93baae51` | 541 (532 Cell, 9 Probe) | 1 h 08 |
| `C-2026-09-10-12` | H-012 | `M-57477f022103` | 82 (80 Cell, 2 Probe) | 11 min |

```bash
go run ./cmd/escapebench matrix --params "sizes=8,16,24,128,1024;pointer=false,true;profiles=LOCAL,RETURNED,CAPTURED_BY_CLOSURE,SENT_ON_CHANNEL,STORED_IN_MAP,STORED_IN_SLICE,STORED_IN_STRUCT,RETURNED_ALLOCATING;modes=VALUE,POINTER;layouts=ARRAY_FILL,NAMED_FIELDS,NAMED_FIELDS_SHAM;repeats=1,4,16;payloads=1,2;probes=SEQUENTIAL_SCAN:33554432,SEQUENTIAL_SCAN:134217728,SCATTERED_SCAN:33554432,SCATTERED_SCAN:134217728,APPEND_PREALLOC:100000,APPEND_GROW:100000,POINTER_CHASE:16384,POINTER_CHASE:262144,POINTER_CHASE:268435456"
go run ./cmd/escapebench escape --matrix M-b44a93baae51
go run ./cmd/escapebench campaign --matrix M-b44a93baae51 --count 20 --hypotheses H-001,H-002,H-003,H-004,H-005,H-006,H-007,H-008,H-009,H-010,H-011,H-013
```

```bash
go run ./cmd/escapebench matrix --params "sizes=8,16,24,128;pointer=false,true;profiles=LOCAL;modes=VALUE,POINTER;layouts=NAMED_FIELDS;replicates=5;probes=APPEND_PREALLOC:100000,APPEND_GROW:100000"
go run ./cmd/escapebench escape --matrix M-57477f022103
go run ./cmd/escapebench campaign --matrix M-57477f022103 --count 20 --hypotheses H-012
```

Puis `compare` et `verdict` pour chaque campagne (étape 4). Vérifié le 2026-09-11 : ces paramètres redonnent exactement les deux identifiants de matrice, et l'empreinte de harnais `551ce66b…` des campagnes publiées. Mener les deux campagnes en série, jamais en parallèle, et arrêter tout processus de contre-épreuve avant de lancer : une charge concurrente rend H-013 non concluante (D-35, D-36).

## 6. Définition de « terminé »

Un UC est terminé quand il est `Approved`, synchronisé avec le code et protégé par des tests couvrant scénario principal et flux alternatifs (SDD p. 142). Une hypothèse est terminée quand un verdict cite ses fichiers de résultats et que `docs/dashboard.md` le reflète. Le projet P1 est terminé quand H-001 à H-006 ont un verdict sur au moins une architecture. **Atteint le 2026-09-10, au-delà de la lettre** : H-001 à H-013 ont un verdict sur `windows/amd64` ; le rejeu sur `arm64` reste ouvert.

## 7. Compromis, alternative, renversement

**Compromis.** Quatre hooks et six skills ajoutent un coût fixe par session ; ils garantissent en échange les invariants de protocole (harnais figé, résultats immuables, critères gelés) sans dépendre de la discipline du moment.

**Alternative.** Démarrer avec `CLAUDE.md` seul et `/implement` (SDD p. 108 : « use case specifications alone… most of the benefit ») ; ajouter hooks et sous-agents « when you feel the pain that the tool solves » (SDD p. 107).

**Renversement.** Si les hooks bloquent trop souvent des actions légitimes (ex. régénération du harnais entre deux campagnes), assouplir `guard-paths.sh` plutôt que de le désactiver ; si le sous-agent réviseur ne trouve jamais de constatation bloquante après trois UC, le réserver aux UC nouveaux et garder la relecture humaine seule pour les synchronisations.
