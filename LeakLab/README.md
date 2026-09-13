# LeakLab

Banc reproductible qui éprouve, par des verdicts sur des hypothèses gelées, les affirmations des chapitres 7, 9 et 20 de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur les anti-patrons de concurrence et sur les outils censés les révéler : test ordinaire, `-race`, `testing/synctest`, `runtime.NumGoroutine()`, profil `goroutineleak`, `go vet` et l'analyseur `ctxvet`.

Le projet est conduit selon l'*AI Unified Process* (Martinelli, *Spec-Driven Development*, Apress 2026) avec Claude Code : `docs/` fait autorité, le code est dérivé des cas d'utilisation.

**État : clos le 2026-09-13.** Quatorze hypothèses jugées sur la campagne de référence `R-2026-09-13-2` : six confirmées, huit infirmées. La campagne `R-2026-09-13-1`, non conforme à C-004, reste archivée. Les trois cas d'utilisation sont au statut `Deployed`. Résultats et lecture : [`../Doc/RAPPORT-FINAL_LeakLab.md`](../Doc/RAPPORT-FINAL_LeakLab.md) ; décisions : [`../Doc/DECISION_LeakLab.md`](../Doc/DECISION_LeakLab.md) ; présentation d'ensemble : [`../README.md`](../README.md).

## Où regarder

| Chemin | Rôle |
|---|---|
| [`docs/vision.md`](docs/vision.md) | Question de recherche, objectifs, hors-périmètre |
| [`docs/requirements.md`](docs/requirements.md) | FR-001…007, NFR-001…005, C-001…008, corpus de référence (32 cas et leur vérité terrain), **H-001…014** |
| [`docs/entity-model.md`](docs/entity-model.md) | Vocabulaire du banc (Mermaid + attributs) |
| [`docs/use-cases/`](docs/use-cases/) | UC-001 exécuter une campagne, UC-002 produire les verdicts, UC-003 analyser avec `ctxvet` |
| [`CLAUDE.md`](CLAUDE.md), `.claude/` | Règles lues par Claude Code ; hooks `guard-paths` et `go-check` ; skills `/implement`, `/refute` |
| `lab/` | Module imbriqué (C-003) : `corpus/` (un fichier par cas, `catalog.go`, oracle), `driver/` (pilotes et sondes), `cmd/scenario` (détecteur PROGRAM) |
| `cmd/leaklab`, `internal/` | Binaire : `campaign` (UC-001), `verdict` (UC-002), `ctxvet` (UC-003), `spec`, `results` |
| [`results/`](results/) | Sorties du binaire, jamais réécrites : `runs/<runId>.json`, `verdicts/<runId>-<horodatage>.{json,md}` |

## Rejouer

Prérequis : Go 1.27.0 ou plus récent (C-001) ; aucune dépendance hors bibliothèque standard. Une toolchain plus ancienne qui sait changer de version doit la recevoir explicitement (`GOTOOLCHAIN=go1.27.0`) : la ligne `go 1.27` des `go.mod` lui fait chercher une version `go1.27` qui n'existe pas. Une campagne dure environ 9 minutes. Depuis `LeakLab/` :

```bash
go vet ./... && go test -race -shuffle=on -count=1 ./...
```

```bash
go run ./cmd/leaklab run
```

```bash
go run ./cmd/leaklab verdict -run <runId>
```

- `run` refuse si le catalogue diverge du corpus de référence, si l'oracle contredit la vérité terrain ou si un binaire ne compile pas ; il n'écrit rien dans ce cas.
- `cd lab && go test -count=1 ./corpus` exécute l'oracle seul.
- `go run ./cmd/leaklab ctxvet <répertoire>` analyse un paquet ; code de sortie 1 s'il trouve un appel d'I/O sans contexte.
