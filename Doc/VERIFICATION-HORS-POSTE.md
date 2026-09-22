# Journal de vérification hors poste

Depuis le retrait de la CI (D-53), aucune exécution automatique ne vérifie les bancs hors du poste Windows de référence. Ce journal en tient lieu (décision Q2 du plan d'implantation de l'évaluation, constat E-13) : une ligne par exécution, ajoutée à la main, jamais réécrite. L'acteur « Pipeline CI » des cas d'utilisation (UC-003 d'EscapeBench et de LeakLab) désigne un usage prévu ; depuis D-53, c'est ce journal qui le tient.

Une exécution vérifie le code et les verdicts archivés ; elle ne mesure rien. Les campagnes de rejeu sous Linux (lot 8, actions 2 et 3) sont lancées par le chercheur, par le binaire, et ont leurs propres rapports sous `Campagnes/`.

## Exécutions

| Date | Commit | Système | Go | Commandes | Résultat |
|---|---|---|---|---|---|
| 2026-09-22 | `5db3bab` (copie par `git archive`, hors OneDrive) | Ubuntu 24.04.4 LTS sous WSL2, noyau 6.18.33.2-microsoft-standard-WSL2, x86_64, même processeur que le poste de référence | go1.27.0 linux/amd64 (`GOTOOLCHAIN=go1.27.0` sur go1.22.2) | EscapeBench : `go vet ./...`; `go test -race -shuffle=on -count=1 ./...`; même commande avec `-tags=integration_test -timeout 300s`; `gofmt -l ./cmd ./internal`; `bash .claude/hooks/selftest.sh`. LeakLab : `go vet ./...`; `go test -race -shuffle=on -count=1 ./...`; `cd lab && go test -count=1 ./corpus` | Tout vert, sauf `selftest.sh` (voir note 1). Harnais de non-régression : 71 verdicts rejoués sur 12 campagnes, 0 sauté; empreintes gelées inchangées. Oracle de LeakLab : vérité terrain confirmée. |

**Note 1 — `selftest.sh` sans `jq`.** `jq` n'est pas installé dans la distribution WSL. Six contrôles échouent, et c'est le comportement documenté (E-28) : `guard-paths` et `guard-bash` refusent tout (code 2) quand ils ne peuvent pas lire l'entrée, et `spec-lint`, qui n'est qu'un avertissement, s'ignore en le disant (code 0). Les hooks ne sont donc vérifiés que sur le poste de référence. Pour les vérifier sous Linux, il faut installer `jq` dans la distribution, ce que cette exécution n'a pas fait.

**Note 2 — ce que Linux sous WSL2 expose à la provenance (D-60, D-19).** `osVersion` : « Ubuntu 24.04.4 LTS, noyau 6.18.33.2-microsoft-standard-WSL2 ». `powerPlan` : vide (WSL2 n'expose pas `cpufreq`). `coreTypes` : non distingués (WSL2 n'expose pas `cpu_core`/`cpu_atom` : les 24 processeurs logiques paraissent identiques). `cpuAffinity` : « non épinglé ». LeakLab lit le nom commercial du processeur dans `/proc/cpuinfo` : « Intel(R) Core(TM) Ultra 9 275HX ». Les chemins Linux de D-60 et D-19, vérifiés jusque-là par compilation croisée seulement, sont ainsi exécutés.

## Rejeu des campagnes sous Linux (à lancer par le chercheur)

Machine au repos, depuis une copie du dépôt sur le système de fichiers de WSL (pas sous `/mnt/c`), avec `export GOTOOLCHAIN=go1.27.0` :

1. LeakLab (environ 9 min) : `go run ./cmd/leaklab run`, puis `go run ./cmd/leaklab verdict`. Comparer à `R-2026-09-13-2` cellule par cellule.
2. EscapeBench : d'abord vérifier que l'attestation de quiétude fonctionne sous Linux (D-31). Si la fraction dépasse 12 %, H-013 rendra une non-conclusion : c'est voulu, et il faut le rapporter tel quel. Ensuite, les deux commandes de `EscapeBench/LANCEMENT.md` §5, en série (environ 1 h 08, puis 11 min), puis `compare` et `verdict`.
3. Rapports `Campagnes/RAPPORT-CAMPAGNE_R-<date>-linux.md` et `Campagnes/RAPPORT-CAMPAGNE_C-<date>-linux.md` : treize verdicts contre treize, quatorze contre quatorze, écart médian des cellules. Puis les décisions D-62 (EscapeBench) et D-20 (LeakLab) : écarts, et verdicts qui changent (erratum, jamais substitution).

Limite : WSL2 tourne sur le même processeur hybride, virtualisé. Le rejeu éprouve l'effet du système et de la chaîne d'exécution, pas celui d'une autre machine (E-03 reste ouvert pour sa moitié « autre machine, autre architecture »). Le lot 9 prépare la série arm64 sur la branche `lot-9-arm64`, sans l'avoir mesurée.
