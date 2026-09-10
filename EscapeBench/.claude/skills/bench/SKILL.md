---
name: bench
description: >
  Exécute une campagne de mesure EscapeBench (UC-003) pour une ou plusieurs hypothèses (H-###)
  via le binaire, sans modifier le code, et rapporte l'identifiant de campagne et la provenance.
  Invoqué par slash command : /bench H-001 H-002. Refuse si le harnais ou la matrice ne sont pas
  en place.
disable-model-invocation: true
allowed-tools: Read, Glob, Bash(go run:*), Bash(go build:*), Bash(make:*), Bash(ls:*), Bash(cat:*)
---
# /bench $ARGUMENTS

Hypothèses ciblées : **$ARGUMENTS** (une ou plusieurs `H-###`).

1. Lis dans `docs/requirements.md` chaque hypothèse demandée : énoncé, critère de réfutation, UC liés. Si une hypothèse n'existe pas ou n'a pas de critère, arrête-toi.
2. Vérifie les préconditions de UC-003 : une Matrix existe (`ls matrices/`), les verdicts d'échappement existent pour la toolchain courante (`ls results/escape/<matrixId>/`), `results/.campaign-lock` est absent. Si une précondition manque, indique la sous-commande à exécuter (`matrix`, `escape`) et arrête-toi.
3. Lance la campagne avec le binaire — jamais avec `go test` directement :
   `go run ./cmd/escapebench campaign --matrix <matrixId> --count <N ≥ 20> --hypotheses $ARGUMENTS`
   Le binaire pose et retire le verrou, calcule les empreintes et écrit sous `results/campaigns/<campaignId>/`.
4. Ne modifie aucun fichier. Si la campagne échoue, rapporte la sortie brute ; la correction relève de la spécification ou de `/implement UC-003`.
5. Rendre compte :
```
Campagne : <campaignId> — statut : COMPLETED | ABORTED
Provenance : <goVersion, GOOS/GOARCH, CPU, date UTC>
Cellules mesurées / FAILED : <n / m>
Hypothèses couvertes : $ARGUMENTS
Prochaine étape : go run ./cmd/escapebench compare --campaign <id> puis /refute <campaignId>
```
Pour une exécution non surveillée, le chercheur lance la même commande sans agent (voir `LANCEMENT.md`, §5).
