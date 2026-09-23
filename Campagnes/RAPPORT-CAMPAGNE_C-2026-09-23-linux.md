# Rapport de campagne — EscapeBench, rejeu sous Linux (`C-2026-09-23-1` et `C-2026-09-23-2`)

**Date :** 2026-09-23 · **Commit :** `3c2dfda` (copie par `git archive`, sur le disque de la distribution WSL) · **Décision :** D-62 · **Lot 8 du plan d'implantation de l'évaluation.**

## 1. Conditions

| | Série de référence (Windows) | Rejeu (Linux) |
|---|---|---|
| Campagnes | `C-2026-09-10-11` (H-001 à H-011, H-013), `C-2026-09-10-12` (H-012) | `C-2026-09-23-1`, `C-2026-09-23-2` |
| Matrices | `M-b44a93baae51`, `M-57477f022103` | mêmes identifiants, régénérées depuis `LANCEMENT.md` §5 |
| Empreinte du harnais | `551ce66b…` | `551ce66b…` (identique) |
| Empreinte des critères | celle de chaque campagne | identique (test D-47 vert sur les deux nouvelles campagnes) |
| Système | Windows 11 (build 26220), go1.27.0 windows/amd64 | Ubuntu 24.04.4 LTS sous WSL2, noyau 6.18.33.2-microsoft-standard-WSL2, go1.27.0 linux/amd64 |
| Machine | Intel Core Ultra 9 275HX, 24 processeurs logiques | même machine, virtualisée |
| Durées | 1 h 08, 11 min | 1 h 02 min 22 s, 10 min 09 s |
| Sujets mesurés / en échec | 541 / 0, 82 / 0 | 541 / 0, 82 / 0 |
| Quiétude (H-013) | 6,5 % au plus | 0,3 % au plus |

Les deux campagnes ont été lancées par l'agent, à la demande explicite du chercheur, puis enchaînées en série, machine au repos. Le lancement contredit la règle 3 du plan (« lancées par le chercheur ») : c'est l'écart que D-62 consigne. Les fichiers sous `EscapeBench/results/` sont ceux que le binaire a écrits, copiés sans retouche depuis la distribution. `docs/dashboard.md`, régénéré par le binaire dans la copie, n'est pas repris : le tableau de bord du dépôt reste celui de la série Windows de référence. La classification d'échappement de `M-b44a93baae51` sous Linux ne diverge sur aucune cellule de celle du 2026-09-12 (NFR-002).

## 2. Verdicts, cellule par cellule

| H | Windows | Linux | Mesure Windows → Linux |
|---|---|---|---|
| H-001 | infirmée | infirmée | paires ≤ 24 o favorables au pointeur : 8, 16 et 24 o → 8 et 24 o (Δ 24 o : −0,67 → −0,31 ns) |
| H-002 | infirmée | infirmée | bascule : sans champ pointeur 8 → 24 o; avec champ pointeur 24 → 8 o |
| H-003 | confirmée | confirmée | 20 paires sur 20, 0 → 1 allocation |
| H-004 | infirmée | infirmée | rapport dispersé/séquentiel ×1,9 et ×2,8 → ×2,0 et ×2,3 |
| H-005 | confirmée | confirmée | ns/op ×0,225 → ×0,219; B/op ×0,196 inchangé |
| H-006 | confirmée | confirmée | 380 cellules qui échappent, toutes dans les quatre causes |
| **H-007** | confirmée | **infirmée** | série avec champ pointeur : avantage du pointeur sur 1 → **2** des trois petites tailles (8 et 16 o) |
| **H-008** | confirmée | **infirmée** | latence non résidente 137,05 → 171,65 ns/accès; rapport ×169,9 → **×202,3**, au-dessus de la borne de 200 |
| H-009 | confirmée | confirmée | les trois conteneurs font échapper la locale |
| H-010 | infirmée | infirmée | 60 paires sur 120 ne doublent pas (2 → 3, 8 → 12, 32 → 48) |
| H-011 | infirmée | infirmée | gain en temps ×4,45 → ×4,58, sous le plancher de 4,8 |
| **H-012** | confirmée | **infirmée** | 16 o avec champ pointeur : 0/5 → **5/5** réplicats où le pointeur franchit le seuil |
| **H-013** | confirmée | **infirmée** | même rapport que H-008, ×202,3; machine attestée au repos |

Neuf infirmées et quatre confirmées sous Linux, contre cinq et huit sous Windows. Les quatre verdicts qui changent vont tous dans le même sens : de confirmée à infirmée.

## 3. Lecture

**H-008 et H-013 : un effet de la virtualisation, supposé.** Sous WSL2, l'accès non résident coûte 171,65 ns au lieu de 137,05, alors que l'accès résident reste à 0,85 ns au lieu de 0,81. Le rapport passe donc de ×169,9 à ×202,3 et franchit la borne de 200 de leur critère, de 2,3 unités seulement. Le rapport de campagne 1 et le README notaient que ces deux hypothèses ne pouvaient pas être infirmées par le bas sur cette machine (latence sous 100 ns). Elles le sont ici par le haut. La cause la plus probable du surcoût est la traduction d'adresses à deux niveaux d'une machine virtuelle, qui renchérit chaque défaut de TLB d'une chaîne de pointeurs dispersée : c'est supposé, rien n'a été instrumenté. Le verdict vaut pour Linux sous WSL2, pas pour un Linux natif sur cette machine.

**H-007 et H-012 : le pointeur gagne à 16 octets avec champ pointeur.** Sous Windows, la cellule de 16 o avec champ pointeur donnait l'avantage à la valeur dans les cinq réplicats (`ciHigh` de +0,05 à +0,13 ns). Sous Linux, elle le donne au pointeur dans les cinq (`ciHigh` de −0,07 à −0,17 ns, seuil 0,061 ns). La cellule de 8 o avec champ pointeur, écartée du jugement de H-012, franchit toujours le seuil dans ses cinq réplicats. H-007 compte désormais deux petites tailles favorables au pointeur, ce que son critère refuse. Même chaîne d'outils, même ABI, même processeur : ce qui diffère, c'est le système, l'éditeur de liens et donc la disposition du code. L'écart est de même nature que celui de la cellule de 8 o, que la contre-épreuve du lot 7 n'a pas su expliquer (D-61). Il confirme la limite que ces deux hypothèses portaient déjà : à ces tailles, l'écart entre les bras est de l'ordre d'une fraction de nanoseconde, et il dépend d'autre chose que de la taille. Le mécanisme n'a pas été cherché.

**Ce qui ne bouge pas.** Tous les verdicts qui ne dépendent pas d'un écart de quelques dixièmes de nanoseconde ni de la latence mémoire sont identiques : l'échappement (H-003, H-006, H-009), les allocations (H-010), la préallocation (H-005, H-011), la disposition en tableau (H-001, H-002, H-004).

## 4. Conséquences

- Les verdicts Windows publiés restent ceux de la série de référence : rien n'est substitué (règle 5 du plan). Le README et le rapport final donnent désormais les verdicts par plateforme.
- H-007, H-008, H-012 et H-013 passent d'un verdict unique à un verdict **dépendant de la plateforme**. Sous Linux, H-012 tombe même sans la cellule écartée, ce qui répond à la réserve qui la classait « restreinte » (retrait décidé en connaissant les mesures).
- Le harnais de non-régression rejoue désormais 84 verdicts sur 14 campagnes (71 + 13).
- E-03 reste ouvert pour sa moitié « autre machine, autre architecture » : le rejeu éprouve le système, sur le même processeur.

## 5. Reproduire

Commandes : [`Doc/VERIFICATION-HORS-POSTE.md`](../Doc/VERIFICATION-HORS-POSTE.md). Fichiers : `EscapeBench/results/campaigns/C-2026-09-23-{1,2}/` (avec `raw/` et `iterations`, D-60), `EscapeBench/results/verdicts/C-2026-09-23-*`, `EscapeBench/results/escape/*/20260923T*.json`.
