# EscapeBench

Banc de mesure reproductible qui éprouve, par des verdicts sur des hypothèses gelées, les affirmations du chapitre 8 (et des chapitres 5–6) de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur l'analyse d'échappement et la frontière valeur/pointeur en Go.

Le projet est conduit selon l'*AI Unified Process* (Martinelli, *Spec-Driven Development*, Apress 2026) avec Claude Code : `docs/` fait autorité, le code est dérivé des cas d'utilisation.

## Où regarder

| Chemin | Rôle |
|---|---|
| `docs/vision.md` | Objectifs et hors-périmètre |
| `docs/requirements.md` | FR-001…007, NFR-001…005, C-001…006, **H-001…006** (hypothèses et critères de réfutation gelés) |
| `docs/entity-model.md` | Vocabulaire du banc (Mermaid + attributs) |
| `docs/use_cases.puml`, `docs/use-cases/UC-00[1-5]-*.md` | Cinq cas d'utilisation exécutables |
| `docs/dashboard.md` | Kanban des UC et verdicts des hypothèses (régénéré, jamais édité) |
| `CLAUDE.md` | Règles de processus et de construction lues par Claude Code |
| `.claude/skills/*` | `/spec-review`, `/implement`, `/go-test`, `/spec-coverage`, `/bench`, `/refute` |
| `.claude/agents/*` | Sous-agents `spec-reviewer` et `code-reviewer` |
| `.claude/settings.json`, `.claude/hooks/*` | Garde des chemins protégés, `gofmt`/`go vet` après édition, contrôle de forme des UC, suite de tests au `Stop` ; `selftest.sh` vérifie les quatre hooks (CI comprise) |
| `cmd/escapebench` | Composition root : une sous-commande par UC |
| `internal/{models,service,ports,adapters,harness}` | Layout hexagonal (BEPG ch. 14) |
| `results/`, `matrices/` | Sorties du binaire, en écriture seule |
| `LANCEMENT.md` | Procédure de lancement du développement, session par session |

## Démarrage

```
make vet test                    # squelette : compile, aucun test encore
bash .claude/hooks/selftest.sh   # contrôle des quatre hooks
claude                           # dans ce répertoire ; puis /hooks, /agents, /context
/spec-review UC-001
```

Voir `LANCEMENT.md`.
