# Tableau de bord — EscapeBench

Généré par `escapebench dashboard` et `escapebench verdict` (UC-005, FR-007) ; ne pas éditer à la main (BR-005-3). Régénéré le 2026-09-10 14:30 UTC.

## Cas d'utilisation

| Use case | Linked FR | UC status | Code | Unit | Integration | Regression | Integrity |
|---|---|---|---|---|---|---|---|
| UC-001 Générer la matrice de cellules | FR-001 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |
| UC-002 Classer l'échappement | FR-002 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |
| UC-003 Exécuter une campagne de mesure | FR-003, FR-006 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |
| UC-004 Comparer valeur et pointeur | FR-004 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |
| UC-005 Produire les verdicts | FR-005, FR-007 | Implemented | ✔ | ✔ | ✔ | ✔ | Strong |

Statuts (SDD, p. 140, colonne `Review` écrite `Reviewed` dans les fichiers de UC) : Draft → Reviewed → Approved → Implemented → Verified → Deployed (= campagne exécutée et rapport publié). Le passage `Reviewed` → `Approved` est une décision humaine consignée dans le fichier du UC.

## Hypothèses

| Hypothèse | Source BEPG | UC liés | Critère gelé (au statut `Approved`) | Campagne | Verdict |
|---|---|---|---|---|---|
| H-001 | p. 245, 253 — « copying a small struct can be cheaper than passing a pointer to it » | UC-003, UC-004, UC-005 | Oui | C-2026-09-10-1 | REFUTED |
| H-002 | p. 253 — valeur « when the data type is small (typically one to three machine words) » | UC-003, UC-004, UC-005 | Oui | C-2026-09-10-1 | REFUTED |
| H-003 | p. 256 — un changement valeur/pointeur « doubles the number of allocations » | UC-003, UC-005 | Oui | C-2026-09-10-1 | CONFIRMED |
| H-004 | p. 254 — accès mémoire « roughly 10× to 200× » entre cache hit et miss | UC-003, UC-005 | Oui | C-2026-09-10-1 | REFUTED |
| H-005 | p. 114 — préallocation : « about 6× » plus rapide, « one-fifth the memory » | UC-003, UC-005 | Oui | C-2026-09-10-1 | CONFIRMED |
| H-006 | p. 238–242 — quatre causes d'échappement listées | UC-002, UC-005 | Oui | C-2026-09-10-1 | CONFIRMED |

## Synthèse

| Indicateur | Total | Complet | En cours | Non démarré | Couverture |
|---|---|---|---|---|---|
| Exigences fonctionnelles | 7 | 7 | 0 | 0 | 100 % |
| Cas d'utilisation | 5 | 5 | 0 | 0 | 100 % |
| Hypothèses avec verdict | 6 | 6 | 0 | 0 | 100 % |
