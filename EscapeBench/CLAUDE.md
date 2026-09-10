# Project Guidelines — EscapeBench

Banc de réfutation Go (bibliothèque standard) qui éprouve des affirmations de *Building Enterprise Projects with Go* ; processus AI Unified Process : `docs/` fait autorité.

## Règles de processus
- Toute demande d'implémentation nomme un `UC-###` ou un `H-###` et passe par un skill : `/spec-review`, `/implement`, `/go-test`, `/spec-coverage`, `/bench`, `/refute`. Une demande sans identifiant est refusée ; « améliore », « optimise », « nettoie » ne sont pas des instructions valides.
- Tout changement de comportement commence dans `docs/` (cas d'utilisation, exigences, modèle d'entités), puis est synchronisé dans le code. Synchroniser, ne jamais régénérer : diff proportionnel au changement de spécification.
- Un UC se code uniquement au statut `Approved` ; le passage `Reviewed → Approved` est une décision humaine.
- Identifiants (`FR/NFR/C/H/UC/BR`) jamais réutilisés ; un critère de réfutation ne se modifie pas, on crée une nouvelle `H-###`.

## Règles de construction (BEPG ch. 4, 6, 14, 20)
- Go 1.25+, bibliothèque standard uniquement ; dépendance externe seulement si couverte par un `C-###`.
- Layout : `cmd/escapebench` (composition root, sous-commandes = UC), `internal/models` (stdlib seulement, sans tags), `internal/service` (UC, aucune I/O), `internal/ports` (interfaces côté consommateur), `internal/adapters/<nom>`, `internal/harness` (code de mesure figé pendant une campagne). `adapters` dépend du core, jamais l'inverse.
- Toute I/O prend un `context.Context` en premier paramètre ; erreurs enveloppées avec `%w`, inspectées avec `errors.Is`/`errors.As` aux bords ; jamais de `panic` sur un chemin de requête.
- Tests : table-driven, sous-tests nommés d'après le UC et le flux (`TestUC003_MainFlow`, `TestUC003_A1_HarnessModified`, `TestUC003_BR1_...`), `testing/synctest` pour tout comportement temporel ou concurrent, jamais de `time.Sleep`. Commande de référence : `go vet ./... && go test -race -shuffle=on -count=1 ./...`. Le `Makefile` offre `make vet test` quand `make` est disponible ; il ne l'est pas sur le poste de référence, et tout le projet a été bâti et vérifié avec `go` directement.
- Benchmarks : `b.ReportAllocs()`, setup hors de la boucle `b.N`, exécutés par le binaire selon `C-003`.
- `results/`, `matrices/` et `docs/dashboard.md` ne sont jamais écrits par un agent (hook `guard-paths`) ; `internal/harness/` est figé si `results/.campaign-lock` existe.
- Référencer l'ID du UC en commentaire d'en-tête de chaque fonction de service ; préfixer les commits par `UC-###:` ou `H-###:`.

## Contexte
- Langue : spécifications, commentaires et rapports en français ; identifiants et noms Go en anglais.
- Aucun serveur MCP n'est configuré (pas de `.mcp.json`) ; `go doc <pkg>` suffit pour la bibliothèque standard.
- Ne pas charger `docs/` en entier : les skills lisent uniquement le UC demandé, ses exigences liées et ses entités.
