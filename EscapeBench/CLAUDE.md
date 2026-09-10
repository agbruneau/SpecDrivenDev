# Project Guidelines — EscapeBench

- `/docs` fait autorité : tout changement de comportement commence par la spécification (vision, requirements, entity-model, use-cases). Toute demande d'implémentation nomme un `UC-###` ou un `H-###` ; une demande sans identifiant est refusée.
- Go 1.25 ou plus récent ; bibliothèque standard uniquement dans `cmd/` et `internal/`. Dépendances externes seulement si couvertes par un `C-###` de `docs/requirements.md`.
- Layout : `cmd/escapebench` (composition root), `internal/models` (stdlib seulement), `internal/service` (cas d'utilisation, aucune I/O directe), `internal/ports` (interfaces), `internal/adapters` (compilateur, système de fichiers, benchmark runner). `adapters` dépend du core, jamais l'inverse.
- Toute fonction d'I/O prend un `context.Context` en premier paramètre ; erreurs enveloppées avec `%w`, inspectées avec `errors.Is`/`errors.As` aux bords uniquement.
- Tests : table-driven, sous-tests nommés d'après le cas d'utilisation et le flux (`TestUC003_MainFlow`, `TestUC003_A1_HarnessModified`) ; `go vet ./...` et `go test -race -shuffle=on ./...` avant tout commit.
- Benchmarks : `b.ReportAllocs()`, setup hors de la boucle `b.N`, exécutés uniquement via `make bench` avec les drapeaux fixés par `C-003`.
- `internal/harness/` et `results/` ne sont jamais modifiés par un skill d'implémentation ; `results/` est en écriture seule pour le runner de campagne.
- Synchroniser, ne pas régénérer : un changement de spécification produit un diff proportionnel ; les changements restent limités au UC demandé.
- Référencer l'ID du cas d'utilisation en commentaire d'en-tête de chaque fonction de service produite.
- Langue : spécifications et commentaires en français ; identifiants (`UC-`, `FR-`, `NFR-`, `C-`, `H-`, `BR-`) et noms Go en anglais.
