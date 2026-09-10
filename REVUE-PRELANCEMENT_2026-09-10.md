# Revue du dépôt avant lancement du développement

**Date :** 2026-09-10 · **Portée :** `Prospection/` et le noyau `EscapeBench/` (P1) · **Régime :** production — le dépôt est le point de départ d'un développement réel.

## 1. Résultat

Le noyau de spécification est solide ; l'outillage ne l'était pas. Seize défauts corrigés, dont quatre auraient laissé des garde-fous inopérants sans aucun signal : les trois hooks qui filtrent par chemin ne reconnaissaient pas les chemins Windows que Claude Code transmet sous Git Bash natif, et les quatre scripts n'avaient pas le bit exécutable dans l'index Git. Le squelette Go compile et passe `go vet` et `go test -race -shuffle=on` sur go1.27.0. Un contrôle exécutable des hooks (`EscapeBench/.claude/hooks/selftest.sh`) est ajouté et branché en CI.

Cinq points restent des décisions du chercheur, listés au §4. Aucun ne bloque `/spec-review UC-001`.

## 2. Corrections appliquées — outillage

| # | Constat | Correction |
|---|---|---|
| 1 | `guard-paths.sh`, `spec-lint.sh`, `go-check.sh` comparaient `tool_input.file_path` à un préfixe POSIX. Sous Git Bash natif, Claude Code transmet `C:\...\results\x.json` : aucune garde ne se déclenchait, silencieusement. | Normalisation `\` → `/` de `file_path` et de `CLAUDE_PROJECT_DIR` en tête des trois scripts. Vérifié aux deux formes par `selftest.sh`. |
| 2 | Matcher `MultiEdit` : ce nom d'outil ne figure plus dans la documentation hooks de Claude Code. | `"matcher": "Edit\|Write"` dans `.claude/settings.json`. |
| 3 | `/spec-review` déclarait `allowed-tools: … Task` ; l'outil a été renommé `Agent` en 2.1.63 (alias conservé, mais périmé). | `allowed-tools: Read, Grep, Glob, Agent`. |
| 4 | Les quatre hooks étaient enregistrés `100644` dans l'index : `chmod +x` manuel exigé après chaque clone. | `100755` sur les cinq scripts. |
| 5 | Aucun `.gitattributes` : avec `core.autocrlf=true`, un clone Windows convertit les `.sh` en CRLF et `bash` échoue. | `.gitattributes` : tout le dépôt en `eol=lf`. |
| 6 | Cible `bench: campaign compare` non exécutable — `compare` exige `CAMPAIGN=<id>`, produit par `campaign`. | Cible retirée ; la ligne correspondante du guide corrigée (`/bench` passe par le binaire, pas par `make bench`). |
| 7 | Le `Makefile` employait `MATRIX` pour les *paramètres* de `matrix` et pour l'*identifiant* de `escape`/`campaign`. | Variable `PARAMS` pour la génération. |
| 8 | La sous-commande `dashboard` manquait à la liste du *composition root* dans `/implement`. | Ajoutée. |
| 9 | La CI dupliquait la boucle `spec-lint` en ligne ; les trois autres hooks n'étaient jamais exercés. | Étape unique `bash .claude/hooks/selftest.sh`. |

`selftest.sh` couvre : les quatre chemins protégés et un chemin libre, la garde `internal/harness/` avec et sans `results/.campaign-lock`, `spec-lint` sur les cinq UC du dépôt et sur un UC volontairement amputé, et la sortie anticipée de `go-test.sh` sous `stop_hook_active`. Chaque contrôle tourne aux deux formes de chemin.

## 3. Corrections appliquées — spécification

| # | Constat | Correction |
|---|---|---|
| 10 | BR-003-3 (« le runner crée des fichiers et n'en modifie ni n'en supprime aucun ») contredit l'étape 8 de UC-003, qui fait passer `campaign.json` de `RUNNING` à `COMPLETED`, et la pose puis le retrait de `.campaign-lock`. | Deux exceptions nommées dans la règle et dans `results/README.md`. |
| 11 | `Matrix` n'avait ni `harnessDigest` ni `parameters` au modèle d'entités, alors que UC-001 étape 6 les écrit, que A2 compare l'empreinte enregistrée et que BR-001-1 dérive l'identifiant des paramètres normalisés. | Deux attributs ajoutés. |
| 12 | Aucune contrainte ne tenait les sources générées hors du module : `matrices/<id>/*.go` serait compilé par `go vet ./...`, `go test ./...`, le hook `go-check` et la CI. Une matrice de 230 fichiers casserait la boucle de travail à la première génération. | `C-007` : une matrice est un module imbriqué (`go.mod` propre). `UC-001` la référence ; `.gitignore` ignore `matrices/`. |
| 13 | Précondition UC-003 « les verdicts d'échappement de cette Matrix existent » : impossible à satisfaire pour une matrice de Probe seules, que UC-002 ne classe jamais. | Précondition conditionnée à la présence d'au moins une Cell. |
| 14 | `Entities` de UC-005 omettait Matrix, Cell et TypeSpec, alors que les critères de H-001 à H-003 exigent la taille, `hasPointerField` et le profil — champs absents de `Comparison`. | Liste complétée. |
| 15 | UC-002 A3 se déclenche sur « la même Provenance », qui inclut `capturedAt` : le déclencheur n'aurait jamais été vrai. | Champs comparés nommés (`goVersion`, `goos`, `goarch`). |
| 16 | `.gitignore` ignorait `matrices/*/cells/`, chemin absent de tous les UC ; légende du tableau de bord écrite `Review` contre `Reviewed` dans les fichiers de UC. | Corrigés. |

Aucune correction ne touche un critère de réfutation : aucun UC n'est `Approved`, donc rien n'était gelé.

## 4. Points laissés au chercheur

- **`make` absent du poste.** `CLAUDE.md` et `/implement` imposent `make vet test` comme commande de référence. Installer `make`, ou remplacer la commande de référence par `go vet ./... && go test -race -shuffle=on -count=1 ./...` dans `CLAUDE.md`, `/implement` et `LANCEMENT.md`. La seconde option supprime une dépendance ; la première garde le contrat exécutable de BEPG ch. 11.
- **Budget de NFR-005.** C-003 chiffre ≈ 30 min de mesure pure pour 230 sujets ; la compilation d'un binaire de test par sujet (BR-003-4) n'est pas chiffrée. La cible de 60 min tient ou non selon ce coût, à mesurer à la campagne de fumée avant de figer la valeur.
- **Deux Probe sans usage.** Le critère de H-004 ne porte que sur les jeux de travail ≥ 32 MiB ; les Probe 256 KiB et 4 MiB de BR-001-4 sont mesurées sans servir aucune hypothèse. Les garder comme témoins ou les retirer de la matrice de référence.
- **Asymétrie de H-002.** « Non observé » jusqu'à 4096 octets ne l'infirme pas, et tout point de bascule > 24 octets la confirme : l'hypothèse ne peut échouer que par le bas. C'est défendable, le livre ne bornant pas le point de bascule par le haut, mais il vaut mieux l'assumer explicitement à la revue qu'après les mesures.
- **Moment du gel.** Le gel des critères est déclenché par le passage du premier UC porteur à `Approved` (UC-003 pour H-001 à H-005). Décider si UC-003 est approuvé avant ou après UC-004, dont l'étape 5 définit le point de bascule sur lequel repose le critère de H-002.

## 5. Vérifications faites, sans correction

- `go vet ./...`, `go build ./...`, `go test -race -shuffle=on -count=1 ./...` et `gofmt -l` : verts sur go1.27.0 (poste Windows, amd64).
- `-race` fonctionne sur ce poste : le compilateur C requis est présent. `LANCEMENT.md` n'en faisait pas un prérequis ; il n'en est pas besoin.
- Les cinq UC passent `spec-lint` sans constatation.
- Contrat des hooks confirmé dans la documentation Claude Code (consultée le 2026-09-10) : code de sortie 2 bloque en `PreToolUse`, avertit en `PostToolUse`, force la poursuite au `Stop` ; `stop_hook_active` vaut vrai quand un hook `Stop` tourne déjà pour le tour. `LANCEMENT.md` marquait ces deux points *À vérifier* ; le marqueur est levé.
- Décompte de BR-001-4 : 11 tailles × 2 (champ pointeur) × 5 profils × 2 modes = 220 Cell, plus 10 Probe = 230 sujets. Conforme à C-003 et NFR-005.
- `EscapeBench.zip` était un instantané antérieur aux dernières révisions (dix fichiers divergents, dont `requirements.md` et quatre UC). Retiré du suivi Git : le dossier fait autorité.
