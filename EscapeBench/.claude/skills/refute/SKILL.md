---
name: refute
description: >
  Produit les verdicts d'EscapeBench (UC-005) pour une campagne complétée via le binaire, puis
  résume chaque verdict avec son rationale et la ligne du tableau de bord, sans interpréter
  au-delà du critère gelé. Invoqué par slash command : /refute C-2026-09-14-1.
disable-model-invocation: true
allowed-tools: Read, Glob, Bash(go run:*), Bash(ls:*), Bash(cat:*)
---
# /refute $ARGUMENTS

Campagne : **$ARGUMENTS** (identifiant `C-<date>-<n>`).

1. Vérifie `results/campaigns/$ARGUMENTS/` : statut `COMPLETED`, présence d'au moins un `comparison-*.json` (sinon indique `go run ./cmd/escapebench compare --campaign $ARGUMENTS`).
2. Exécute `go run ./cmd/escapebench verdict --campaign $ARGUMENTS` (ajoute `--escape <fichier>` si H-006 est visée). Le binaire vérifie l'empreinte des critères (BR-003-5, UC-005 A1), écrit `results/verdicts/`, régénère `docs/dashboard.md`.
3. Lis le fichier de verdicts produit et rapporte, sans ajouter de qualificatif ni de conclusion générale :
```
Campagne : $ARGUMENTS
| Hypothèse | Verdict | Rationale (fichiers, cellules ou paires cités) |
Tableau de bord : <lignes mises à jour>
Hypothèses INCONCLUSIVE et cause : <liste ou « aucune »>
```
4. Si le binaire refuse (critères modifiés), rapporte les hypothèses concernées et rappelle la règle : créer une nouvelle `H-###`, ne jamais modifier un critère existant.
Interdits : modifier `docs/requirements.md`, écrire dans `results/`, proposer un seuil ou une réinterprétation des critères.
