# EscapeBench

Banc de mesure reproductible qui éprouve, par des verdicts sur des hypothèses gelées, les affirmations du chapitre 8 (et des chapitres 5–6) de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur l'analyse d'échappement et la frontière valeur/pointeur en Go.

Le projet est conduit selon l'*AI Unified Process* (Martinelli, *Spec-Driven Development*, Apress 2026) avec Claude Code : `docs/` fait autorité, le code est dérivé des cas d'utilisation.

**État : clos le 2026-09-10.** Les treize hypothèses ont un verdict (sept infirmées, six confirmées) et les cinq cas d'utilisation sont au statut `Deployed`. Résultats et lecture : [`../Doc/RAPPORT-FINAL_EscapeBench.md`](../Doc/RAPPORT-FINAL_EscapeBench.md) ; présentation d'ensemble : [`../README.md`](../README.md).

## Où regarder

| Chemin | Rôle |
|---|---|
| [`docs/vision.md`](docs/vision.md) | Objectifs et hors-périmètre |
| [`docs/requirements.md`](docs/requirements.md) | FR-001…007, NFR-001…005, C-001…010, **H-001…013** (hypothèses et critères de réfutation gelés) |
| [`docs/entity-model.md`](docs/entity-model.md) | Vocabulaire du banc (Mermaid + attributs) |
| [`docs/use_cases.puml`](docs/use_cases.puml), [`docs/use-cases/`](docs/use-cases/) | Cinq cas d'utilisation exécutables, UC-001 à UC-005 |
| [`docs/dashboard.md`](docs/dashboard.md) | Kanban des UC et verdicts des hypothèses (régénéré, jamais édité) |
| [`CLAUDE.md`](CLAUDE.md) | Règles de processus et de construction lues par Claude Code |
| `.claude/skills/*` | `/spec-review`, `/implement`, `/go-test`, `/spec-coverage`, `/bench`, `/refute` |
| `.claude/agents/*` | Sous-agents `spec-reviewer` et `code-reviewer` |
| `.claude/settings.json`, `.claude/hooks/*` | Garde des chemins protégés, `gofmt`/`go vet` après édition, contrôle de forme des UC, suite de tests au `Stop` ; `selftest.sh` vérifie les quatre hooks (CI comprise) |
| `cmd/escapebench` | Composition root : une sous-commande par UC |
| `internal/{models,service,ports,adapters,harness}` | Layout hexagonal (BEPG ch. 14) |
| [`results/`](results/) | Sorties versionnées du binaire, en écriture seule : verdicts d'échappement, campagnes, verdicts par hypothèse (voir [`results/README.md`](results/README.md)) |
| `matrices/`, `bin/` | Sources générées et binaire compilé, ignorés par Git ; une matrice se régénère depuis son `matrix.json` et le harnais |
| [`LANCEMENT.md`](LANCEMENT.md) | Procédure de lancement du développement, session par session, et commandes qui rejouent les campagnes de clôture |
| [`../Doc/DECISION.md`](../Doc/DECISION.md) | Décisions D-01 à D-38 : écarts assumés, conception du harnais, statistiques, clôture |
| [`../Campagnes/`](../Campagnes/), [`../Revue/`](../Revue/) | Rapports des campagnes intermédiaires et revues contradictoires |

## Rejouer une campagne

Commande de référence ; `make` est facultatif, le `Makefile` offrant `make vet test` pour qui l'a :

```bash
go vet ./... && go test -race -shuffle=on -count=1 ./...
```

Chaîne complète, du plus court au plus long :

```bash
go run ./cmd/escapebench matrix --reference
go run ./cmd/escapebench escape --matrix <matrixId>
go run ./cmd/escapebench campaign --matrix <matrixId> --count 20
go run ./cmd/escapebench compare --campaign <campaignId>
go run ./cmd/escapebench verdict --campaign <campaignId>
```

- La matrice de référence (230 sujets) demande environ 29 minutes sur le poste de référence (NFR-005).
- `verdict` retrouve de lui-même les verdicts d'échappement de la matrice pour H-006 ; `--escape <fichier>` en désigne un précis.
- `campaign --hypotheses H-00x,…` restreint les hypothèses gelées par la campagne. H-012 l'exige : elle demande une matrice à cinq réplicats et ne peut pas partager une campagne avec H-007 (C-009).
- `escapebench dashboard` régénère `docs/dashboard.md` sans produire de verdict ; `bash .claude/hooks/selftest.sh` contrôle les quatre hooks.
