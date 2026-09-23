# Rapport de campagne — LeakLab, rejeu sous Linux (`R-2026-09-23-1`)

**Date :** 2026-09-23 · **Commit :** `3c2dfda` (copie par `git archive`, sur le disque de la distribution WSL) · **Décision :** D-20 de `DECISION_LeakLab.md` · **Lot 8 du plan d'implantation de l'évaluation.**

## 1. Conditions

La série de référence est `R-2026-09-13-2` : Windows 11, go1.27.0 windows/amd64, 9 min 01. Le rejeu s'est fait sous Ubuntu 24.04.4 LTS sous WSL2 (noyau 6.18.33.2-microsoft-standard-WSL2), go1.27.0 linux/amd64, sur la même machine (Intel Core Ultra 9 275HX, 24 processeurs logiques), en 9 min 27. Il a produit 960 observations dynamiques et 64 statiques, comme la référence. L'oracle du corpus est passé avant la campagne. La campagne a été lancée par l'agent à la demande explicite du chercheur (D-20). Les fichiers `LeakLab/results/runs/R-2026-09-23-1.json` et `LeakLab/results/verdicts/R-2026-09-23-1-*` sont ceux du binaire, copiés sans retouche.

## 2. Verdicts

| H | Windows | Linux | Écart |
|---|---|---|---|
| H-001 | confirmée | confirmée | — |
| H-002 | infirmée | infirmée | — |
| H-003 | confirmée | confirmée | — |
| H-004 | infirmée | infirmée | — |
| H-005 | infirmée | infirmée | mêmes trois cas non diagnostiqués |
| H-006 | confirmée | confirmée | médiane REAL 500,6 → 536,1 ms; SYNCTEST 0,000 → 0,025 ms (l'horloge Linux résout ce que celle de Windows arrondissait à zéro, E-30) |
| **H-007** | confirmée | **infirmée** | `deadlock-send` et `deadlock-range` en PROGRAM : DEADLOCK → **HANG** |
| H-008 | infirmée | infirmée | — |
| H-009 | infirmée | infirmée | résidus 157,3 et 171,7 o → 102,5 et 113,2 o |
| H-010 | infirmée | infirmée | — |
| H-011 | infirmée | infirmée | — |
| H-012 | confirmée | confirmée | — |
| H-013 | infirmée | infirmée | même angle mort : `mutex-leak` |
| H-014 | confirmée | confirmée | résidus −51,0 et −36,6 o → −63,5 et −52,7 o |

Neuf infirmées et cinq confirmées sous Linux, contre huit et six sous Windows. Un seul verdict change, H-007.

## 3. H-007 : l'interblocage n'est pas détecté dans un binaire lié à cgo

Le livre affirme que les deux exemples d'interblocage finissent par l'erreur fatale « all goroutines are asleep ». Sous Linux, le détecteur PROGRAM les voit tourner jusqu'à son délai. La cause est établie par une expérience directe, hors campagne, sur le même commit :

| Construction du binaire `scenario` | `deadlock-send` | `deadlock-range` |
|---|---|---|
| par défaut sous Linux (`CGO_ENABLED=1`) | aucun arrêt en 20 s | aucun arrêt en 20 s |
| `CGO_ENABLED=0` | erreur fatale d'interblocage en 0,01 s | erreur fatale d'interblocage en 0,01 s |

Le corpus importe `net` (cas `io-without-context-leak`). Sous Linux, avec un compilateur C présent, cette importation lie cgo au binaire (`go version -m` : `CGO_ENABLED=1`). Or la fonction `checkdead` de `runtime/proc.go` (go1.27.0, lue) ne compte pas comme au repos le M supplémentaire d'un programme cgo. Elle conclut qu'un fil tourne encore et ne déclare pas l'interblocage. Sous Windows, le binaire est construit sans cgo, et l'erreur fatale se produit.

**Portée.** L'infirmation est réelle pour le binaire que le banc construit, mais elle tient à la composition du corpus : un programme minimal qui contient seulement l'un des deux exemples, sans `net`, se termine par l'erreur fatale sous Linux aussi. Selon les définitions du README, c'est une **infirmation de banc** : elle vient d'un artefact du banc, un corpus compilé en un seul binaire qui tire `net` et donc cgo, et non de l'affirmation dans son cas nominal. Elle révèle pourtant une condition que le livre ne donne pas : l'erreur fatale suppose un programme sans cgo, ce qu'un programme réel sous Linux, dès qu'il importe `net`, n'est souvent pas. Le suivi des *issues* de Go n'a rien donné pour les requêtes « deadlock detection cgo », « all goroutines are asleep cgo not detected » et « checkdead cgo » (le 2026-09-23). Le comportement n'y est donc pas retrouvé comme documenté, sans que l'absence soit établie.

## 4. Ce qui ne bouge pas

Les treize autres verdicts, dont les deux angles morts propres au dépôt : le délai de `go test` qui masque l'interblocage (H-008) et le `sync.Mutex` de 8 octets manqué par le profil `goroutineleak` (H-013). Le résidu de H-009 baisse sous Linux sans changer le verdict, et son successeur H-014 reste confirmé.

## 5. Reproduire

Commandes : [`Doc/VERIFICATION-HORS-POSTE.md`](../Doc/VERIFICATION-HORS-POSTE.md). Expérience cgo : depuis `LeakLab/lab`, `CGO_ENABLED=1 go build -o /tmp/s ./cmd/scenario && timeout 20 /tmp/s deadlock-send`, puis la même commande avec `CGO_ENABLED=0`.
