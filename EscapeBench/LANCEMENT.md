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
5. Répéter sur `arm64` si disponible (C-006) : nouvelle campagne, mêmes critères. **Reste ouvert à la clôture.** Préparé le 2026-09-22 sur la branche `lot-9-arm64` (D-63) : voir « Série arm64 » ci-dessous.

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

### Matrice des campagnes `C-2026-09-10-2` et `C-2026-09-10-3`

Ces deux campagnes de la seconde génération (H-007 à H-011) emploient `M-8f03757ac206` : 252 cellules et 4 sondes, 256 sujets. Ses paramètres, longtemps tenus pour perdus (D-42), ont été retrouvés le 2026-09-22 dans le `matrix.json` resté sur le poste, `matrices/` n'étant pas versionné (D-61). Vérifié : ils redonnent l'identifiant, les 252 cellules et les 4 sondes, et des sources de sujets identiques octet pour octet à celles du poste.

```bash
go run ./cmd/escapebench matrix --params "sizes=8,16,24,128,1024;pointer=false,true;profiles=LOCAL,STORED_IN_MAP,STORED_IN_SLICE,STORED_IN_STRUCT,RETURNED_ALLOCATING;modes=VALUE,POINTER;layouts=NAMED_FIELDS,NAMED_FIELDS_SHAM;repeats=1,2,4,16;payloads=1,2;probes=APPEND_PREALLOC:100000,APPEND_GROW:100000,POINTER_CHASE:16384,POINTER_CHASE:268435456"
```

### Série arm64 (branche `lot-9-arm64`, D-63)

Préparée le 2026-09-22, **aucune campagne lancée** : le catalogue ne dispose d'aucune machine arm64 (décision Q6 du plan d'implantation de l'évaluation). La branche n'est pas fusionnée dans `main` : les campagnes Linux du lot 8 se lancent sur `main` et gardent l'empreinte `551ce66b…`.

**Ce qui distingue la série.** Le harnais y porte l'empreinte `fd4a470c1fea2dc1a366d5ffb26921cf5c14f8a40540c67a2fcf91a962d06100` (C-005, lot 9 de l'audit) : ses campagnes ne se comparent qu'entre elles. Le type de 8 octets en `ARRAY_FILL` y fait réellement 8 octets, là où la série `551ce66b…` en mesurait 16. `go test ./internal/harness -run D63` vérifie l'empreinte avant de lancer ; un échec signifie que le harnais a changé depuis D-63.

**Avant la première campagne.**

1. Choisir la machine. Linux arm64 de préférence : sous macOS, la Provenance ne relève ni les tailles de cache ni la ligne, et l'attestation de quiétude n'existe pas (C-010), si bien que H-008 et H-013 y sont non concluantes par construction et que la garde de ligne de cache (C-006) ne s'y exerce pas. H-013 exige aussi un dernier niveau de cache d'au moins 64 fois le L1 de données.
2. Lire la ligne de cache : `cat /sys/devices/system/cpu/cpu0/cache/index0/coherency_line_size` sous Linux, `sysctl hw.cachelinesize` sous macOS. À 64, les commandes ci-dessous s'emploient telles quelles ; à 128, ajouter `;cacheline=128` aux deux matrices qui portent des sondes de parcours ou de chaîne (référence et clôture). La matrice des successeurs n'en porte aucune. UC-003 refuse une matrice dont la ligne diffère de celle qu'il relève.
3. Machine au repos, aucun processus de contre-épreuve, campagnes en série.

**Matrices et campagnes, dans cet ordre.** Chaque identifiant est vérifié par `TestUC001_D63_MatricesDeLaSerieArm64`.

| Ordre | Campagne | Hypothèses | Matrice (ligne de 64 / de 128) | Sujets | Durée au rythme du poste du catalogue |
|---|---|---|---|---|---|
| 1 | Référence (BR-001-4) | H-001 à H-006 | `M-823d8b5af441` / `M-9face550a02e` | 230 (220 Cell, 10 Probe) | 29 min |
| 2 | Clôture | H-001 à H-011, H-013 | `M-b44a93baae51` / `M-13da29cdebb6` | 541 (532 Cell, 9 Probe) | 1 h 08 |
| 3 | Successeurs | H-012, H-014, H-015, H-016 | `M-ebbb95f809c8` (sans sonde de parcours) | 162 (160 Cell, 2 Probe) | 21 min |

La troisième campagne est **la première à nommer H-014, H-015 et H-016** : elle gèle leurs critères (BR-003-5). Aucune campagne amd64 ne doit les nommer avant elle. Elle ne gèle ni H-001 ni H-002, que UC-003 refuse sur une matrice qui réplique `ARRAY_FILL` (C-011), ni H-007, refusée sur toute matrice à réplicats (C-009).

```bash
# 1. Référence (ligne de 64 ; à 128, --params "<paramètres de la référence>;cacheline=128", voir TestUC001_D63_…)
go run ./cmd/escapebench matrix --reference
go run ./cmd/escapebench escape --matrix M-823d8b5af441
go run ./cmd/escapebench campaign --matrix M-823d8b5af441 --count 20 --hypotheses H-001,H-002,H-003,H-004,H-005,H-006

# 2. Clôture (paramètres de « Rejouer les campagnes de clôture » ; ajouter ;cacheline=128 si besoin)
go run ./cmd/escapebench matrix --params "sizes=8,16,24,128,1024;pointer=false,true;profiles=LOCAL,RETURNED,CAPTURED_BY_CLOSURE,SENT_ON_CHANNEL,STORED_IN_MAP,STORED_IN_SLICE,STORED_IN_STRUCT,RETURNED_ALLOCATING;modes=VALUE,POINTER;layouts=ARRAY_FILL,NAMED_FIELDS,NAMED_FIELDS_SHAM;repeats=1,4,16;payloads=1,2;probes=SEQUENTIAL_SCAN:33554432,SEQUENTIAL_SCAN:134217728,SCATTERED_SCAN:33554432,SCATTERED_SCAN:134217728,APPEND_PREALLOC:100000,APPEND_GROW:100000,POINTER_CHASE:16384,POINTER_CHASE:262144,POINTER_CHASE:268435456"
go run ./cmd/escapebench escape --matrix M-b44a93baae51
go run ./cmd/escapebench campaign --matrix M-b44a93baae51 --count 20 --hypotheses H-001,H-002,H-003,H-004,H-005,H-006,H-007,H-008,H-009,H-010,H-011,H-013

# 3. Successeurs : NAMED_FIELDS pour H-012 et H-016, ARRAY_FILL pour H-014 et H-015 (C-009, C-011)
go run ./cmd/escapebench matrix --params "sizes=8,16,24,128;pointer=false,true;profiles=LOCAL;modes=VALUE,POINTER;layouts=ARRAY_FILL,NAMED_FIELDS;replicates=5;probes=APPEND_PREALLOC:100000,APPEND_GROW:100000"
go run ./cmd/escapebench escape --matrix M-ebbb95f809c8
go run ./cmd/escapebench campaign --matrix M-ebbb95f809c8 --count 20 --hypotheses H-012,H-014,H-015,H-016
```

Puis `compare` et `verdict` pour chaque campagne. Sur un poste qui garde des matrices de la série `551ce66b…` sous `matrices/`, UC-001 refuse de régénérer un même identifiant sous le nouveau harnais (`ErrHarnessChanged`) : travailler dans un clone neuf plutôt que de supprimer ces répertoires.

**Ce que la matrice des successeurs exige, et pourquoi.** Quatre tailles, dont 8, 16 et 24 octets pour les cellules jugées et 128 octets pour le témoin de sensibilité de chaque série ; les deux séries, avec et sans champ pointeur ; le seul profil `LOCAL` ; `ARRAY_FILL` pour H-014 et H-015, `NAMED_FIELDS` pour H-012 et H-016 ; cinq réplicats, la matrice ne déclarant que des combinaisons répliquées, de sorte qu'une passe est la matrice entière et que deux réplicats d'une même paire sont séparés d'environ quatre minutes. Les deux sondes `APPEND_*` n'y servent aucun critère ; elles reprennent celles de `M-57477f022103`.

**Rapports.** Comme au lot 8 : `Campagnes/RAPPORT-CAMPAGNE_C-<date>-arm64.md`, verdicts par architecture. H-008 et H-013 deviennent infirmables si la mémoire de la machine sert un accès dépendant sous 100 ns : le dire dans un sens ou dans l'autre. H-014 à H-016 se lisent contre l'attente amd64 que leurs critères consignent.

## 6. Définition de « terminé »

Un UC est terminé quand il est `Approved`, synchronisé avec le code et protégé par des tests couvrant scénario principal et flux alternatifs (SDD p. 142). Une hypothèse est terminée quand un verdict cite ses fichiers de résultats et que `docs/dashboard.md` le reflète. Le projet P1 est terminé quand H-001 à H-006 ont un verdict sur au moins une architecture. **Atteint le 2026-09-10, au-delà de la lettre** : H-001 à H-013 ont un verdict sur `windows/amd64` ; le rejeu sur `arm64` reste ouvert.

## 7. Compromis, alternative, renversement

**Compromis.** Quatre hooks et six skills ajoutent un coût fixe par session ; ils garantissent en échange les invariants de protocole (harnais figé, résultats immuables, critères gelés) sans dépendre de la discipline du moment.

**Alternative.** Démarrer avec `CLAUDE.md` seul et `/implement` (SDD p. 108 : « use case specifications alone… most of the benefit ») ; ajouter hooks et sous-agents « when you feel the pain that the tool solves » (SDD p. 107).

**Renversement.** Si les hooks bloquent trop souvent des actions légitimes (ex. régénération du harnais entre deux campagnes), assouplir `guard-paths.sh` plutôt que de le désactiver ; si le sous-agent réviseur ne trouve jamais de constatation bloquante après trois UC, le réserver aux UC nouveaux et garder la relecture humaine seule pour les synchronisations.
