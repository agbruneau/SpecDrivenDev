# Contre-épreuves hors campagne — la cellule écartée par H-012 et la matrice `M-8f03757ac206`

2026-09-22, lot 7 du plan d'implantation de l'évaluation (constats E-18 et E-17, décision D-61). Rien n'a été écrit sous `results/` ni `matrices/` : sujets copiés, compilés et mesurés dans un répertoire temporaire, hors dépôt.

## En bref

- **E-18, réduit, non clos.** Le désassemblage confirme que le bras pointeur de `Size0008FieldsPtr/LOCAL` exécute un chargement de plus (`MOVQ 0(AX), AX`). Ce chargement n'explique pas l'écart : la paire témoin `Size0008FieldsPlain/LOCAL` a exactement la même structure et ne montre pas d'écart. L'écart se reproduit hors campagne, et un décalage du code de 32 octets ne le change pas : l'alignement de la boucle à 32 octets est écarté. Ce qui reste : un seul bras sur quatre passe sous le coût de ses voisins, et les seules différences de code machine sont la disposition des emplacements de pile et une instruction `NOP`. Les trancher demande les compteurs de performance du processeur.
- **E-17, clos.** Les paramètres de `M-8f03757ac206` n'étaient pas perdus : `matrices/` est ignoré par Git, mais le `matrix.json` de la matrice est resté sur le poste. Ses paramètres redonnent l'identifiant, 252 cellules et 4 sondes. Sur une copie du module, le harnais de non-régression, complété de ces paramètres, rejoue 71 verdicts au lieu de 67, sans divergence. Aucun balayage n'a été nécessaire.

## 1. Désassemblage de `Size0008FieldsPtr_LOCAL` (E-18)

### Conditions

| Élément | Valeur |
|---|---|
| Sources | `EscapeBench/matrices/M-57477f022103/subjects/Size0008FieldsPtr_LOCAL_{VALUE,POINTER}` et, comme témoin, `Size0008FieldsPlain_LOCAL_{VALUE,POINTER}`, avec le `go.mod` de la matrice, copiés tels quels |
| Chaîne d'outils | `go1.27.0 windows/amd64`, `GOAMD64=v1` : la version qu'inscrit la provenance de `C-2026-09-10-12` |
| Compilation | `go test -c -o <tmp>/<sujet>.test.exe` dans chaque sujet, sans option de compilation, comme `gotool.go` (le banc passe seulement `-run`, `-bench`, `-benchmem`, `-count`, `-benchtime`, `-cpu`) |
| Désassemblage | `go tool objdump -s 'LOCAL_(VALUE\|POINTER)\.(consumeValue\|consumePointer\|Run)$'` |

Le bras mesuré est `Run(b.N)`. `BenchmarkSubject` ne l'appelle qu'une fois, hors de la boucle.

### Listings (bras à champ pointeur)

Bras valeur, `Size0008FieldsPtr_LOCAL_VALUE` :

```
consumeValue:
  0x140147600  c3              RET

Run (boucle 0x632–0x65c) :
  0x140147632  4889442410      MOVQ AX, 0x10(SP)      ; compteur -> pile
  0x140147637  48894c2408      MOVQ CX, 0x8(SP)       ; somme    -> pile
  0x14014763c  488d058d2d2000  LEAQ anchor(SB), AX    ; t.P, recalculé à chaque tour
  0x140147643  e8b8ffffff      CALL consumeValue
  0x140147648  488b4c2408      MOVQ 0x8(SP), CX       ; somme    <- pile
  0x14014764d  4801c1          ADDQ AX, CX
  0x140147650  488b442410      MOVQ 0x10(SP), AX      ; compteur <- pile
  0x140147655  48ffc8          DECQ AX
  0x140147658  4885c0          TESTQ AX, AX
  0x14014765b  7fd5            JG 0x140147632
```

Bras pointeur, `Size0008FieldsPtr_LOCAL_POINTER` :

```
consumePointer:
  0x140147600  488b00          MOVQ 0(AX), AX         ; le chargement de plus
  0x140147603  c3              RET

Run, avant la boucle :
  0x14014762a  4883ec20        SUBQ $0x20, SP
  0x14014762e  488d0d9b2d2000  LEAQ anchor(SB), CX
  0x140147635  48894c2418      MOVQ CX, 0x18(SP)      ; t vit à 0x18(SP), écrit une fois
Run (boucle 0x63e–0x666) :
  0x14014763e  4889442410      MOVQ AX, 0x10(SP)      ; compteur -> pile
  0x140147643  48894c2408      MOVQ CX, 0x8(SP)       ; somme    -> pile
  0x140147648  488d442418      LEAQ 0x18(SP), AX      ; &t
  0x14014764d  e8aeffffff      CALL consumePointer
  0x140147652  488b4c2408      MOVQ 0x8(SP), CX
  0x140147657  4801c1          ADDQ AX, CX
  0x14014765a  488b442410      MOVQ 0x10(SP), AX
  0x14014765f  48ffc8          DECQ AX
  0x140147662  4885c0          TESTQ AX, AX
  0x140147665  7fd7            JG 0x14014763e
```

Témoin sans champ pointeur, boucles seulement (appelés identiques : `RET` d'un côté, `MOVQ 0(AX), AX; RET` de l'autre) :

```
Size0008FieldsPlain_LOCAL_VALUE (0x632–0x65a), cadre 0x18 :
  MOVQ AX,0x10(SP) · MOVQ CX,0x8(SP) · MOVL $0x1,AX · CALL · MOVQ 0x8(SP),CX · ADDQ · MOVQ 0x10(SP),AX · DECQ · TESTQ · JG
Size0008FieldsPlain_LOCAL_POINTER (0x63b–0x664), cadre 0x20, t à 0x8(SP) écrit une fois :
  MOVQ AX,0x18(SP) · MOVQ CX,0x10(SP) · LEAQ 0x8(SP),AX · CALL · MOVQ 0x10(SP),CX · ADDQ · MOVQ 0x18(SP),AX · DECQ · NOPL · TESTQ · JG
```

Les listings complets, `BenchmarkSubject` compris, sont reproductibles par les commandes ci-dessus. Ils n'ont pas été versionnés.

### Mesure de contrôle, hors campagne

Les quatre binaires et leurs variantes décalées (voir plus bas) ont été lancés en alternance, huit tours, `-test.benchtime 1s -test.count 1`, sur le poste de référence le 2026-09-22. Pendant ce temps, d'autres processus tournaient (d'autres agents travaillaient dans le dépôt) : ces mesures ne satisfont pas l'attestation de quiétude et n'ont pas valeur de campagne. Elles ne servent qu'à savoir si l'écart se reproduit et s'il suit un décalage du code.

| Binaire | Médiane ns/op | Min – max | Campagne `C-2026-09-10-12`, réplicat 1 |
|---|---|---|---|
| `Size0008FieldsPtr_LOCAL_VALUE` | 0,903 | 0,853 – 0,918 | 0,787 |
| `Size0008FieldsPtr_LOCAL_POINTER` | **0,652** | 0,600 – 0,677 | **0,619** |
| `Size0008FieldsPlain_LOCAL_VALUE` | 0,888 | 0,851 – 0,939 | 0,843 |
| `Size0008FieldsPlain_LOCAL_POINTER` | 0,926 | 0,878 – 0,975 | 0,750 |
| `…Ptr_LOCAL_VALUE_PAD32` | 0,882 | 0,862 – 0,901 | — |
| `…Ptr_LOCAL_POINTER_PAD32` | **0,670** | 0,643 – 0,715 | — |
| `…Plain_LOCAL_VALUE_PAD32` | 0,879 | 0,851 – 0,923 | — |
| `…Plain_LOCAL_POINTER_PAD32` | 0,902 | 0,862 – 0,976 | — |

Dans la campagne, les cinq réplicats de la paire à champ pointeur donnent un écart de −0,17 à −0,33 ns, tous significatifs. Ceux de la paire témoin donnent un écart significatif sur un réplicat seulement (−0,09), les quatre autres étant non significatifs. Hors campagne, le constat tient : **c'est le seul bras `Size0008FieldsPtr_LOCAL_POINTER` qui est anormalement rapide.** Les trois autres bras tournent au même coût, à 0,05 ns près.

### Lecture

**Ce qui s'explique (vérifié sur les listings et la mesure de contrôle).**

1. *Le chargement de plus ne ralentit rien, et il n'accélère rien non plus.* Le bras pointeur de la paire témoin exécute le même `MOVQ 0(AX), AX` sur un emplacement écrit une seule fois avant la boucle, et il coûte ce que coûte son bras valeur. Le chargement lit une adresse stable, résidente en L1, et ne se trouve pas sur la chaîne de dépendance qui borne la boucle. L'énoncé du rapport final (« exécute un chargement de plus et va pourtant plus vite ») est exact, mais le « pourtant » suppose un lien de cause qui n'existe pas.
2. *La chaîne qui borne la boucle passe par la pile, dans les quatre bras.* Le compteur et la somme sont vidés sur la pile avant l'appel (`MOVQ AX, 0x10(SP)`), puis rechargés après, parce que l'appel non inliné détruit les registres. Le coût d'un tour est donc dominé par un transfert écriture→lecture sur la pile. Vers 0,9 ns par tour, à la fréquence maximale annoncée pour ce processeur (5,4 GHz, *supposé*, non mesuré), un tour prend environ cinq cycles : l'ordre de grandeur de ce transfert, plus la paire `CALL`/`RET`.
3. *Pas de fusion d'instructions en jeu.* `TESTQ AX, AX; JG` se fusionne en macro-opération dans les quatre bras. Ni `DECQ` ni le chargement de l'appelé n'ont de partenaire de fusion qui différerait d'un bras à l'autre.
4. *L'alignement de la boucle à 32 octets est écarté.* Chaque variante `_PAD32` ajoute une fonction vide `pad0` avant l'appelé. Tout le code du sujet recule donc de 0x20 : appelé en 0x620, boucle du bras pointeur en 0x65e–0x686, qui franchit désormais d'autres frontières de 32 et de 64 octets. Le code machine est sinon identique. L'écart subsiste (0,670 contre 0,652) et le témoin ne bouge pas.

**Ce qui reste inexpliqué.** Le bras rapide tourne à environ 0,65 ns, soit à peu près 3,5 cycles sous la même hypothèse de fréquence. C'est moins que le transfert écriture→lecture ordinaire qui borne ses trois voisins. Il faut donc qu'un mécanisme raccourcisse la chaîne dans ce bras seulement. À code machine égal par ailleurs, deux différences séparent ce bras du bras pointeur témoin :

- *La disposition des emplacements de pile.* Dans ces deux binaires, le compilateur a placé `t` en haut du cadre quand son type contient un pointeur (`0x18(SP)` ; compteur et somme en `0x10` et `0x8`), et en bas sinon (`0x8(SP)` ; compteur et somme en `0x18` et `0x10`). Le cadre fait 0x20 dans les deux cas.
- *Un `NOPL` dans la boucle du témoin,* entre `DECQ` et `TESTQ`, absent du bras rapide.

Hypothèse la plus plausible (*supposée*, non vérifiée) : le renommage mémoire des cœurs Intel récents, qui peut ramener à presque zéro la latence d'une lecture de pile relue juste après son écriture. Il s'enclencherait sur l'un des motifs d'adresses et pas sur l'autre. Rien dans les listings ne permet de le confirmer ni de l'écarter. Ce qui trancherait : les compteurs de performance du processeur (cycles, micro-opérations, événements de transfert écriture→lecture et de renommage mémoire) sur les deux bras pointeurs, par exemple avec `perf stat` sous Linux, sur le même poste. Autre voie : deux boucles écrites en assembleur Go qui ne diffèrent que par l'ordre des emplacements ou que par le `NOP`. Les deux sortent du périmètre du banc (bibliothèque standard, pas de compteurs matériels).

**Conséquence pour H-012 : aucune.** Le critère gelé retire cette cellule du corpus jugé, et le verdict ne change pas. La contre-épreuve renforce plutôt la raison déclarée du retrait. À huit octets avec champ pointeur, la paire ne mesure pas la règle de la page 253. Elle mesure un effet de micro-architecture propre à un seul binaire, que la paire témoin ne partage pas.

## 2. Paramètres de `M-8f03757ac206` (E-17)

### Ce qui a été fait

Le plan prévoyait un balayage des combinaisons plausibles. Il n'a pas été nécessaire. `EscapeBench/matrices/` est ignoré par Git (`.gitignore`, ligne 5), mais le poste en garde les cinq matrices. `matrices/M-8f03757ac206/matrix.json`, daté du 2026-09-10 à 14 h 05, porte ses paramètres. D-42 les disait « consignés nulle part » ; on ne sait pas si cette recherche avait consulté la copie locale (*supposé* : non, puisqu'elle aurait abouti).

**Périmètre exact « balayé ».** Une seule combinaison candidate, celle du `matrix.json` local. Les quatre autres matrices du poste ont servi de contrôle du programme de recalcul. Temps de calcul : moins d'une seconde, pour un budget d'une heure. Le programme recopie `internal/models` du banc, sans modification, dans un module temporaire. Il relit les paramètres de chaque `matrix.json` puis appelle `MatrixID()` et `Expand()`. Les cinq identifiants recalculés égalent les identifiants stockés.

| Matrice | Identifiant recalculé | Cellules | Sondes |
|---|---|---|---|
| `M-57477f022103` | identique | 80 | 2 |
| `M-823d8b5af441` | identique | 220 | 10 |
| **`M-8f03757ac206`** | **identique** | **252** | **4** |
| `M-abb3d708d0e1` | identique | 2 | 3 |
| `M-b44a93baae51` | identique | 532 | 9 |

### Paramètres retrouvés

| Dimension | Valeur |
|---|---|
| Tailles | 8, 16, 24, 128, 1024 |
| Champ pointeur | sans, avec |
| Profils | `LOCAL`, `STORED_IN_MAP`, `STORED_IN_SLICE`, `STORED_IN_STRUCT`, `RETURNED_ALLOCATING` (cinq des huit) |
| Modes | `VALUE`, `POINTER` |
| Dispositions | `NAMED_FIELDS`, `NAMED_FIELDS_SHAM` |
| Répétitions | 1, 2, 4, 16 |
| Charges | 1, 2 |
| Réplicats | 1 (champ absent, valeur par défaut) |
| Sondes | `APPEND_PREALLOC:100000`, `APPEND_GROW:100000`, `POINTER_CHASE:16384`, `POINTER_CHASE:268435456` |

La combinaison déborde en partie du périmètre qu'envisageait le plan : cinq profils et non huit, la répétition 2 (absente de `C-2026-09-10-11`), aucune disposition `ARRAY_FILL`. Un balayage restreint à ce périmètre l'aurait probablement manquée (*déduit*, non essayé).

### Vérifications

1. Sur une copie du module EscapeBench prise dans un répertoire temporaire, `go run ./cmd/escapebench matrix --params "<paramètres>"` rend `M-8f03757ac206`, 16 `TypeSpec`, 252 `Cell`, 4 `Probe` et l'empreinte de harnais `551ce66b…`. Le `matrix.json` produit égale celui du poste, `generatedAt` excepté. Les 256 sources de sujets sont identiques (`diff -r` muet).
2. Sur la même copie, on ajoute à `archivedParameters()` de `internal/service/regression_integration_test.go` une cinquième entrée portant ces paramètres. `go test -count=1 -tags=integration_test -run TestUC005_NonRegressionVerdictsArchives ./internal/service/` rend alors : « 71 verdicts rejoués sur 12 campagnes, 0 sautés faute de matrice reconstructible, 2 rendus avant l'existence de leur évaluateur ». Sans l'entrée, sur la même copie : 67 rejoués, 4 sautés (H-009 et H-010 sur `C-2026-09-10-2` et `C-2026-09-10-3`). Les quatre nouveaux sous-tests passent : même verdict, même motif.

Le harnais du dépôt n'est pas modifié par ce lot. Le changement à y porter est donné dans D-61, et les paramètres sont consignés dans `EscapeBench/LANCEMENT.md` §5.
