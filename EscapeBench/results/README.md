# results/

Écriture seule par le binaire (BR-003-3, NFR-004). Arborescence :

- `escape/<matrixId>/<capturedAt>.json` — verdicts d'échappement (UC-002)
- `campaigns/<campaignId>/` — `campaign.json`, `measurements/<subjectId>.json`, `comparison-<timestamp>.json` (UC-003, UC-004)
- `verdicts/<campaignId>-<timestamp>.json` — verdicts par hypothèse (UC-005)
- `.campaign-lock` — présent pendant une campagne ; fige `internal/harness/` (hook guard-paths)

Aucun fichier n'est jamais réécrit ; un nouveau calcul crée un nouveau fichier horodaté.
