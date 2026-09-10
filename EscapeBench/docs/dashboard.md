# Tableau de bord — EscapeBench

Généré par `/spec-coverage` et `/refute` (UC-005) ; ne pas éditer à la main. État au 2026-09-10 : noyau de spécification complet, outillage Claude Code en place, aucun code de service.

## Cas d'utilisation

| Use case | Linked FR | UC status | Code | Unit | Integration | Regression | Integrity |
|---|---|---|---|---|---|---|---|
| UC-001 Générer la matrice de cellules | FR-001 | Reviewed | ✕ | ✕ | — | — | Weak |
| UC-002 Classer l'échappement | FR-002 | Reviewed | ✕ | ✕ | ✕ | — | Weak |
| UC-003 Exécuter une campagne de mesure | FR-003, FR-006 | Reviewed | ✕ | ✕ | ✕ | — | Weak |
| UC-004 Comparer valeur et pointeur | FR-004 | Reviewed | ✕ | ✕ | — | — | Weak |
| UC-005 Produire les verdicts | FR-005, FR-007 | Reviewed | ✕ | ✕ | — | — | Weak |

Statuts (SDD, p. 140) : Draft → Review → Approved → Implemented → Verified → Deployed (= campagne exécutée et rapport publié). Le passage `Reviewed` → `Approved` est une décision humaine consignée dans le fichier du UC.

## Hypothèses

| Hypothèse | Source BEPG | UC liés | Critère gelé | Campagne | Verdict |
|---|---|---|---|---|---|
| H-001 | p. 245, 253 | UC-003, UC-004, UC-005 | Oui | — | — |
| H-002 | p. 253 | UC-003, UC-004, UC-005 | Oui | — | — |
| H-003 | p. 256 | UC-003, UC-005 | Oui | — | — |
| H-004 | p. 254 | UC-003, UC-004, UC-005 | Oui | — | — |
| H-005 | p. 114 | UC-003, UC-005 | Oui | — | — |
| H-006 | p. 238–242 | UC-002, UC-005 | Oui | — | — |

## Synthèse

| Indicateur | Total | Complet | En cours | Non démarré | Couverture |
|---|---|---|---|---|---|
| Exigences fonctionnelles | 7 | 0 | 0 | 7 | 0 % |
| Cas d'utilisation | 5 | 0 | 5 | 0 | 0 % |
| Hypothèses avec verdict | 6 | 0 | 0 | 6 | 0 % |
