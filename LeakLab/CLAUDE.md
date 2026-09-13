# Project Guidelines — LeakLab

Banc de détectabilité des anti-patrons de concurrence Go (bibliothèque standard) qui éprouve des affirmations de *Building Enterprise Projects with Go* ; processus AI Unified Process : `docs/` fait autorité.

## Règles de processus
- Toute demande d'implémentation nomme un `UC-###` ou un `H-###`. « améliore », « optimise », « nettoie » ne sont pas des instructions valides.
- Tout changement de comportement commence dans `docs/`, puis est synchronisé dans le code ; diff proportionnel au changement de spécification.
- Identifiants (`FR/NFR/C/H/UC/BR`, identifiants de cas) jamais réutilisés ; un critère de réfutation ne se modifie pas, on crée une nouvelle `H-###`.
- Le tableau « Corpus de référence » de `docs/requirements.md` et `lab/corpus/catalog.go` sont identiques ; un test le vérifie.

## Règles de construction
- Go 1.27+, bibliothèque standard uniquement (C-002).
- Deux modules. Principal : `cmd/leaklab` (sous-commandes `run`, `verdict`, `ctxvet`), `internal/spec` (lecture de `docs/requirements.md`), `internal/results` (types et écriture exclusive), `internal/campaign` (UC-001), `internal/verdict` (UC-002), `internal/ctxvet` (UC-003). Imbriqué `lab/` (C-003) : `corpus` (un fichier par cas, `catalog.go`, oracle), `driver` (pilotes des détecteurs et sondes), `cmd/scenario` (détecteur PROGRAM). Layout plat : aucune interface à implémentation unique.
- Le corpus est fautif par construction : ne jamais « corriger » un cas `faulty`, ne jamais lancer `go vet` ou `-race` sur `lab/` hors campagne.
- Erreurs enveloppées avec `%w` ; jamais de `panic` hors pilotes et corpus.
- Tests table-driven, sous-tests nommés d'après le UC ou l'hypothèse (`TestUC001_...`, `TestH005_...`). Commande de référence, depuis `LeakLab/` : `go vet ./... && go test -race -shuffle=on -count=1 ./...`. Oracle du corpus : `cd lab && go test -count=1 ./corpus`.
- `results/` n'est jamais écrit par un agent (hook `guard-paths`) : seul le binaire le produit.
- Préfixer les commits par `UC-###:` ou `H-###:` (ou `LeakLab:` pour ce qui touche tout le projet).

## Contexte
- Langue : spécifications, commentaires et rapports en français ; identifiants Go en anglais.
- Ne pas charger `docs/` en entier : lire le UC demandé, ses exigences liées et ses entités.
