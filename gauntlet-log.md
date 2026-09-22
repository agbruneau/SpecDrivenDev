# Journal de la boucle bâtisseur/critique — implantation de l'évaluation

Barre : grille C1–C10 de `Doc/EVALUATION-ACADEMIQUE_2026-09-15.md`, réappliquée à l'aveugle à l'état témoin (arbre de `1eec98d`, copié hors dépôt avant la réécriture) et à l'état modifié. Budget : environ 20 agents. Journal en ajout seulement.

## Tour 0 — Lot 0 (binaire, sans critique)

- Exécutant : agent principal, actions autorisées par le chercheur le 2026-09-22.
- Barre de sortie : clone neuf de `origin` sans aucun objet des PDF, des psaumes ni des rapports agentiques (0 objet trouvé), 62 commits, paquet de 1,28 Mo; `git ls-remote` : `main` et `refs/pull/1/head` (reste au chercheur).
- Résultat : atteinte. Commit `06294b4`, étiquette locale `v1.0`.
- Coût : 0 sous-agent.

## Tour 1 — Bâtisseurs des lots 1, 3, 4, 5+6 (EscapeBench, LeakLab), 7, puis lot 2

- Coût : 7 sous-agents.
- Résultat : sept artefacts livrés, barre de sortie verte sous Windows (rejouée par l'orchestrateur) ; harnais porté de 67 à 71 verdicts (lot 7). Commits `2812c68` à `5db3bab`.

## Tour 1 bis — Lot 8 (binaire, sans critique)

- Exécutant : agent principal. Barre : suite des deux bancs sous WSL2, go1.27.0, sur `5db3bab`. Atteinte, sauf `selftest.sh` (pas de `jq` dans WSL, gardes fermées comme documenté). Commit `2c0f67f`. Coût : 0 sous-agent.

## Tour 2 — Critiques aveugles, premier passage (état témoin contre `5db3bab`)

| Morceau | Ordre | Verdict | Critères touchés | Plus gros écart retenu |
|---|---|---|---|---|
| W1 présentation des résultats et limites (lot 2) | A = témoin, B = modifié | **B (modifié) gagne** | C5, C6, C7, C8, C9, C10 : B | H-012 (et H-005) classées « de fond » alors que le corps établit que le retrait de la cellule de 8 octets, décidé en connaissant les mesures, favorise le livre ; classement de H-012 non justifié au §4.2 |
| W2 positionnement dans la littérature (lot 3) | A = modifié, B = témoin | **A (modifié) gagne** | C2, C5, C6, C9, C10 : A | Le positionnement reste confiné à §1.4 et à `ETAT-DE-L-ART.md` : les passages d'interprétation (README §4.3, §8.4, rapport final LeakLab H-007/H-008) ne citent ni l'issue 69188 ni Tu et coll. ; la section « Références » du README ne liste que les trois ouvrages |
| W3 mesure de QR2 et transparence des revues (lot 4) | A = témoin, B = modifié | **B (modifié) gagne** | C6, C7, C8, C9, C10 : B | « Environ dix fois plus de temps et de jetons » : les données donnent 3,2 fois pour le temps et 9,4 fois pour les jetons ; écart de décompte de l'audit nommé mais non résolu |

- Même tour, bâtisseurs binaires : lot 10 (commits `fb196e4`, `6543fb2`, test D-47 vert, 71 verdicts) ; lot 9 sur la branche `lot-9-arm64` (sept commits, empreinte `fd4a470c…`, barre verte). Le lot 9 a trouvé un défaut dans la série publiée : en `ARRAY_FILL`, les cellules « 8 octets » mesuraient un type de 16 octets.
- Coût : 5 sous-agents (cumul 12).

## Tour 3 — Rebâtisseurs sur l'écart retenu (en série : même README)

| Morceau | Écart traité | Résultat |
|---|---|---|
| W1 | Classement de H-012 et H-005 | Les deux passent de « de fond » à « restreinte » (définition précisée et datée); EscapeBench : 1 confirmation de fond, 5 restreintes, 2 de banc; erratum daté. En plus : erratum « 8 octets » vérifié sur C-7 et C-11, section « Série arm64 préparée », renvois au journal hors poste. |
| W2 | Antériorité absente des passages d'interprétation | README §4.3, §8.4 et rapports finaux citent l'ABI de Go, McVoy et Staelin, Drepper, l'issue 69188, Tu et coll., goleak; « Références » complétée (15 travaux); sept références montées de M à R ou T. |
| W3 | « Facteur dix » | Corrigé par erratum : environ 3 fois le temps actif, 9 fois les jetons; écart de l'audit renvoyé à D-65. |

- Orchestrateur : contradiction résiduelle signalée par W2 (retrait de la cellule « déclaré avant la mesure » contre « décidé en connaissant les mesures ») levée par une précision datée dans le rapport final d'EscapeBench : déclaré avant `C-2026-09-10-12`, après `C-2026-09-10-2`. Lot 11 : `CITATION.cff`, section « Citer ce dépôt », section « Suivi » de l'évaluation.
- Coût : 3 sous-agents (cumul 15).

## Tour 4 — Critiques aveugles neufs, second passage (état témoin contre l'état après tour 3), puis lissage

| Morceau | Ordre | Verdict | Critères touchés | Plus gros écart retenu |
|---|---|---|---|---|
| W1 | A = modifié, B = témoin | **A (modifié) gagne** | C2, C5, C6, C8, C10 : A ; C9 : B (lisibilité : correctifs en couches, résumé plus long) | La prose du rapport final d'EscapeBench ne suit pas son propre décompte par portée (H-004, H-005, H-003, H-006) ; « de 4 à 6 » absent du fichier de verdicts de C-11 |
| W2 | A = témoin, B = modifié | **B (modifié) gagne** | C2, C5, C6, C9, C10 : B | Les deux apports revendiqués reposent sur une absence de source, établie par une recherche limitée (suivi des issues de Go non fouillé) ; lignes « Praticiens » sans source ; aucune référence lue par le chercheur |
| W3 | A = modifié, B = témoin | **A (modifié) gagne** | C3, C6, C7, C8, C9, C10 : A | `AUDIT.md` publie encore la contre-vérification « 16/16 » sans aucune trace ; A-133 compté implanté et ouvert |

- Lissage (agent neuf) : « 4 → 6 » retiré par erratum (il venait de C-3) ; prose du rapport final alignée sur la portée ; point 6 de l'erratum d'`AUDIT.md` (contre-vérification sans trace) ; A-133 tranché ouvert (123 implantés, 57 ouverts) ; apports ramenés à « possibles, non établis ». 0 lien cassé sur 22 fichiers ; test d'empreintes vert. Non traité, parce que c'est un travail de recherche ou de lecture humaine : fouille du suivi des issues de Go, sources des lignes « Praticiens », lecture des références par le chercheur.
- Coût : 4 sous-agents (cumul 19).

## Sortie de boucle

Budget atteint (19 sous-agents pour un plafond d'environ 20), et deux passages à l'aveugle consécutifs gagnés contre l'état témoin sur les trois morceaux. La barre (grille C1–C10) est donc battue sur chaque morceau. L'écart restant le plus lourd, la part originale non établie (W2), ne se ferme pas par la rédaction : il demande une recherche et une lecture humaines.
