# Tableau de bord — EscapeBench

Généré par `escapebench dashboard` et `escapebench verdict` (UC-005, FR-007) ; ne pas éditer à la main (BR-005-3). Régénéré le 2026-09-10 20:29 UTC.

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
| H-001 | p. 245, 253 — « copying a small struct can be cheaper than passing a pointer to it » | UC-003, UC-004, UC-005 | Oui | — | — |
| H-002 | p. 253 — valeur « when the data type is small (typically one to three machine words) » | UC-003, UC-004, UC-005 | Oui | — | — |
| H-003 | p. 256 — un changement valeur/pointeur « doubles the number of allocations » | UC-003, UC-005 | Oui | — | — |
| H-004 | p. 254 — accès mémoire « roughly 10× to 200× » entre cache hit et miss | UC-003, UC-005 | Oui | — | — |
| H-005 | p. 114 — préallocation : « about 6× » plus rapide, « one-fifth the memory » | UC-003, UC-005 | Oui | C-2026-09-10-4 | CONFIRMED |
| H-006 | p. 238–242 — quatre causes d'échappement listées | UC-002, UC-005 | Oui | — | — |
| H-007 | p. 253 — valeur « when the data type is small (typically one to three machine words) », éprouvée sur une disposition assignable aux registres | UC-001, UC-003, UC-004, UC-005 | Oui | — | — |
| H-008 | p. 254 — « roughly 10× to 200× » entre un succès de cache annoncé sous 10 ns et un défaut servi par la mémoire principale annoncé au-delà de 100 ns | UC-001, UC-003, UC-005 | Oui | — | — |
| H-009 | p. 241 — la quatrième cause est illustrée par une map, puis généralisée : « This rule also applies to slices, structs, or any container that is already heap-allocated » | UC-001, UC-002, UC-005 | Oui | — | — |
| H-010 | p. 256 — « a small change in how you pass a struct (by value vs. pointer) doubles the number of allocations » | UC-001, UC-003, UC-004, UC-005 | Oui | — | — |
| H-011 | p. 114 — « runs significantly faster (about 6×) and uses roughly one-fifth the memory, with only a single allocation », pour `const n = 100_000` | UC-003, UC-005 | Oui | C-2026-09-10-4 | REFUTED |
| H-012 | p. 253 — valeur « when the data type is small (typically one to three machine words) », même affirmation que H-007, éprouvée sur un seuil apparié à la taille et à la série et mesuré sur cinq réplicats de la paire réelle (C-009). Deux écarts à H-007, écrits ici et non dans une justification hors tableau. Premier écart : la tolérance « au plus une exception sur trois » est abandonnée, parce qu'elle rend l'infirmation inatteignable sur toute convention d'appel qui passe trois mots nommés dans les registres d'argument, amd64 comme arm64 (C-006) — à 24 octets le bras valeur y gagne structurellement, mesuré à plus 0,357 et plus 0,359 ns dans les deux séries sur C-2026-09-10-2, donc au plus une taille peut basculer et un critère qui en exige deux ne peut que confirmer. Une seule taille suffit désormais, lecture sans exception, plus stricte que le « typically » du livre : un verdict REFUTED contredit cette lecture stricte et non la réserve de la page 253. Second écart : la taille de 8 octets de la série avec champ pointeur est retirée du corpus jugé. À un mot, la variante à champ pointeur est le pointeur lui-même — le type généré est une structure à un seul champ de type pointeur vers uint64, vérifié dans la source de Size0008FieldsPtr — de sorte que les deux bras y passent un pointeur et que la règle de la page 253 n'y a pas d'objet. Prix de ce retrait, déclaré parce qu'il va dans le sens du livre : c'est la seule cellule qui infirmait sur le corpus du 2026-09-10, avec un avantage pointeur de 0,171 ns en campagne et de 0,250 ns sur 0,819 remesuré aux drapeaux de C-003, alors que son bras pointeur exécute un chargement de plus que son bras valeur — un écart de sens contraire à la mécanique, que le banc ne sait pas expliquer et que le critère fait consigner sans le laisser décider. | UC-001, UC-003, UC-004, UC-005 | Oui | C-2026-09-10-4 | CONFIRMED |
| H-013 | p. 254 — « roughly 10x to 200x » entre un succès de cache annoncé sous 10 ns et un défaut servi par la mémoire principale annoncé au-delà de 100 ns, même affirmation que H-008, éprouvée sous attestation de quiétude de la machine (C-010) et sur trois bandes de résidence pincées. Deux écarts à H-008, écrits ici et non dans une justification hors tableau. Premier écart : le verdict est adossé à une mesure de la charge de la machine extérieure aux sondes. La contre-épreuve du 2026-09-10 a établi que la branche du rapport bascule dès que la latence non résidente dépasse 200 fois la médiane résidente, soit environ 157 ns sur la machine du catalogue, et qu'une charge modérée y produit des infirmations à 166, 168 et 172 ns pour une valeur de repos de 133 ; aucun plafond absolu ne les sépare, puisque le poser là où il protège revient à le poser là où l'infirmation commence. Second écart : la bande intermédiaire est pincée et non ouverte jusqu'à la moitié du dernier niveau de cache, un paramètre libre sur un rapport de 192 décidant sinon seul du verdict, ce que la même contre-épreuve a mesuré en relevant 1,8 ns au bas de la bande large et 103 à 111 ns à son sommet. Prix déclaré : tant que C-010 n'est pas satisfaite, H-013 est non concluante par construction, et son ancre des 100 ns restera de surcroît inatteignable en infirmation tant que le catalogue ne disposera pas d'un boîtier dont la mémoire principale sert un accès dépendant sous 100 ns, C-006 souhaitant l'arm64 qui trancherait. | UC-001, UC-003, UC-005 | Oui | — | — |

## Synthèse

| Indicateur | Total | Complet | En cours | Non démarré | Couverture |
|---|---|---|---|---|---|
| Exigences fonctionnelles | 7 | 7 | 0 | 0 | 100 % |
| Cas d'utilisation | 5 | 5 | 0 | 0 | 100 % |
| Hypothèses avec verdict | 13 | 3 | 0 | 10 | 23 % |
