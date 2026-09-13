---
name: refute
description: >
  Exécute une campagne LeakLab (UC-001) si aucun identifiant n'est donné, puis produit ses verdicts
  (UC-002) par le binaire et les résume sans interpréter au-delà des critères gelés. Invoqué par
  slash command : /refute ou /refute R-2026-09-13-1.
disable-model-invocation: true
allowed-tools: Read, Glob, Bash(go run:*), Bash(ls:*)
---
# /refute $ARGUMENTS

1. Sans argument : `go run ./cmd/leaklab run` (environ neuf minutes ; le binaire refuse si l'oracle, le catalogue ou la compilation échouent). Note l'identifiant `R-…` affiché.
2. `go run ./cmd/leaklab verdict -run <R-…>` : le binaire vérifie l'empreinte des critères (UC-002 A1) et écrit `results/verdicts/`.
3. Rapporte, sans qualificatif ni conclusion générale :
```
Campagne : <R-…>
| Hypothèse | Verdict | Rationale |
Cellules instables de la matrice : <liste ou « aucune »>
Hypothèses INCONCLUSIVE et cause : <liste ou « aucune »>
```
4. Si le binaire refuse pour un critère modifié, nomme les hypothèses et rappelle la règle : créer une `H-###` successeur, ne jamais réécrire un critère.

Interdits : modifier `docs/requirements.md`, écrire dans `results/`, proposer un seuil ou une réinterprétation d'un critère.
