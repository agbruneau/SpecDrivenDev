# LeakLab — verdicts des quatorze hypothèses

Clôture du 2026-09-13. Le banc a éprouvé douze affirmations de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur les anti-patrons de concurrence et les outils qui les révèlent, une affirmation des notes de version de Go et une hypothèse successeur. Les critères ont été gelés par un commit avant la première ligne de code ; chaque verdict est produit par un évaluateur qui applique leur texte à la lettre.

**Révision du 2026-09-22 :** aucun verdict ne change. Le décompte se lit désormais par portée (tableau des verdicts), et les limites nomment la construction de NUMGOROUTINE, la résolution de l'horloge et la revue humaine préparée.

**Campagne de référence : `R-2026-09-13-2`** (commit `500d954`, 9 min 01). go1.27.0, windows/amd64, 24 cœurs logiques (identifiant Intel64 Family 6 Model 198), 32 cas × 6 détecteurs dynamiques × 5 répétitions en processus isolés, 2 analyseurs statiques, 15 bras de sonde × 5 répétitions. Aucune cellule instable : les cinq répétitions de chaque cellule ont rendu la même issue. La campagne `R-2026-09-13-1` est archivée mais non conforme à C-004 (voir *Ce que le banc a appris sur lui-même*).

## Les quatorze verdicts

*Note du 2026-09-22.* La colonne « Portée » est une lecture ajoutée après coup, la même que dans le README (résumé et §8.3); verdicts et rationales sont ceux du fichier de verdicts de `R-2026-09-13-2`. Une infirmation est *de fond* si elle contredit l'affirmation telle que le livre l'énonce, *restreinte* si elle ne contredit qu'une lecture stricte ou une opérationnalisation, *de banc* si elle vient d'un artefact de mesure. Une confirmation est *de fond* si la mesure pouvait infirmer l'affirmation telle que le livre l'énonce sur ce matériel, *restreinte* si elle ne le pouvait pas (y compris quand la confirmation ne tient que par une tolérance élargie ou un retrait de cellule décidés par le banc en faveur du livre) ou si elle juge un outil construit par le banc, *de banc* si le corpus la rendait acquise d'avance. *(Définition alignée sur celle du README le 2026-09-22; aucune des cinq confirmations de fond ci-dessous ne tient par une tolérance élargie ni par un retrait, et le décompte ne change pas.)*

| Hypothèse | Page | Ce que le livre affirme | Mesure | Verdict | Portée |
|---|---|---|---|---|---|
| H-001 | 561 | Une fuite de goroutine est silencieuse : les tests peuvent passer | Un test ordinaire passe sur les 11 cas de fuite | confirmée | de fond |
| H-002 | 217 | Depuis Go 1.25, le cadre de test signale mieux les goroutines fuitées | Aucune des 55 exécutions `go test -v` d'un cas de fuite ne contient de signalement | **infirmée** | de fond |
| H-003 | 232 | Sous `-race`, une course donne un rapport et un échec | Les 2 cas de course diagnostiqués, aucun faux positif | confirmée | de fond |
| H-004 | 232 | Lancer toute la suite sous `-race` en CI attrape les bogues de concurrence | 0 des 14 défauts autres que les courses révélé par `-race` | **infirmée** | restreinte |
| H-005 | 289–292 | `synctest` panique si une goroutine reste bloquée à la fin du test | Panique sur 8 fuites sur 11 ; blocage de 10 min sur `mutex-leak`, `global-channel-leak`, `io-without-context-leak` | **infirmée** | de fond |
| H-006 | 291 | Le test de délai de 500 ms se termine instantanément dans une bulle | 0 ns à la résolution de l'horloge (5 µs par bulle en contre-épreuve), contre 500,6 ms hors bulle | confirmée | de fond |
| H-007 | 564 | Les deux exemples d'interblocage finissent par « all goroutines are asleep » | Erreur fatale du runtime sur les deux, exécutés comme programmes | confirmée | de fond |
| H-008 | 564 | Même affirmation, éprouvée par `go test` | Les deux cas bloquent ; tués au bout de 5 s, `go test` les aurait attendus 10 min | **infirmée** | restreinte |
| H-009 | 567 | Sans `cancel()`, les ressources vivent jusqu'à l'échéance, puis sont libérées | Retenue 275 et 319 o par contexte ; « résidu » 157 et 172 o après échéance, mesuré par une sonde défectueuse | **infirmée** (par un défaut de mesure, voir H-014) | de banc |
| H-010 | 567 | L'oubli de `cancel()` coûte aussi des goroutines | 0 goroutine avec un parent standard ; 20 000 avec le parent opaque témoin | **infirmée** | de fond |
| H-011 | 276 | `time.After` en boucle mène à une croissance continue de la mémoire | +0,01 o par itération, contre 258 o pour le témoin qui garde ses minuteries | **infirmée** | de fond |
| H-012 | 293, 563 | Une hausse du nombre de goroutines signale une fuite | 11 fuites sur 11 signalées, aucun cas sain signalé | confirmée | restreinte |
| H-013 | notes Go 1.26/1.27 | Le profil `goroutineleak` repère les fuites sur une primitive inaccessible | 6 cas du domaine sur 7 ; manque `mutex-leak` | **infirmée** | de fond |
| H-014 | 567 | Successeur de H-009, résidu net du coût d'expiration des minuteries | Retenue 275 et 319 o ; résidu −51 et −37 o | confirmée | de fond |

Six confirmations, huit infirmations. Par portée *(lecture ajoutée le 2026-09-22)* : cinq infirmations de fond (H-002, H-005, H-010, H-011, H-013), deux restreintes (H-004, qui lit « bogues de concurrence » comme tous les défauts du corpus; H-008, qui éprouve l'affirmation sous `go test` et non dans un programme) et une de banc (H-009, sonde défectueuse, D-14); cinq confirmations de fond et une restreinte (H-012, qui juge NUMGOROUTINE, détecteur construit par le banc, plus que la pratique du livre).

## Matrice de détectabilité

Issue majoritaire sur cinq répétitions. `DEADLOCK` : panique de `synctest` ou erreur fatale du runtime ; `HANG` : processus tué au délai de 5 s ; `LEAK` : fuite signalée par le pilote ; `DIAG` : diagnostic statique. La matrice complète, cas corrigés compris, est dans [`LeakLab/results/verdicts/R-2026-09-13-2-20260913T113251Z.md`](../LeakLab/results/verdicts/R-2026-09-13-2-20260913T113251Z.md) : les quinze corrections rendent `PASS` partout, sauf `io-without-context-fix`, qui bloque dans une bulle.

| Cas fautif | BARE | RACE | SYNCTEST | NUMGOROUTINE | LEAKPROFILE | PROGRAM | VET | CTXVET |
|---|---|---|---|---|---|---|---|---|
| dispatch-leak, dispatch-buffered-leak, abandoned-receive-leak, select-no-cancel-leak, waitgroup-leak, cond-leak | PASS | PASS | DEADLOCK | LEAK | LEAK | PASS | PASS | PASS |
| mutex-leak | PASS | PASS | HANG | LEAK | PASS | PASS | PASS | PASS |
| global-channel-leak | PASS | PASS | HANG | LEAK | PASS | PASS | PASS | PASS |
| empty-select-leak | PASS | PASS | DEADLOCK | LEAK | LEAK | PASS | PASS | PASS |
| ctx-watcher-leak | PASS | PASS | DEADLOCK | LEAK | PASS | PASS | DIAG | PASS |
| io-without-context-leak | PASS | PASS | HANG | LEAK | PASS | PASS | PASS | DIAG |
| forgotten-cancel | PASS | PASS | PASS | PASS | PASS | PASS | DIAG | PASS |
| counter-race, append-race | PASS | RACE | PASS | PASS | PASS | PASS | PASS | PASS |
| deadlock-send, deadlock-range | HANG | HANG | DEADLOCK | HANG | HANG | DEADLOCK | PASS | PASS |

Coût médian d'une observation : 7 ms pour BARE, SYNCTEST et PROGRAM, 17 ms pour NUMGOROUTINE (environ 1 s sur un cas de fuite, le temps d'attendre une redescente qui ne vient pas), 109 ms pour LEAKPROFILE (dont la pause de 100 ms), 1 024 ms pour RACE. Un binaire instrumenté qui réussit met environ une seconde à sortir, alors qu'un binaire qui échoue sort en 15 ms. Cet écart s'accorde avec l'attente de 1 000 ms (`atexit_sleep_ms`) que la documentation du détecteur de courses annonce avant la sortie ; il n'a pas été vérifié dans le code.

## Lecture

**Aucun détecteur ne voit tout, et le livre en recommande deux qui ne voient pas les fuites.** Un test ordinaire, un test sous `-race` et le programme lui-même passent sur les onze fuites : la p. 561 a raison, la p. 217 non (H-001, H-002). La leçon de la p. 232 sur `-race` en CI ne couvre que les courses (H-004). Le seul détecteur dynamique qui a vu les onze fuites sans faux positif est le plus rudimentaire, le compte de goroutines avant et après dix exécutions (H-012), à condition de répéter le scénario et d'attendre qu'une goroutine légitime ait fini. Rien de cela n'est neuf : c'est le principe de `goleak`, dont Saioc et coll. (2024) décrivent l'usage chez Uber, et Tu et coll. (2019) mesuraient déjà la portée limitée de `-race` sur des bogues réels. Le banc le confirme sur un corpus à vérité terrain *(ajout du 2026-09-22)*.

**`synctest` tient sa promesse dans le périmètre de ses propres documents, pas dans celui du livre (H-005).** La p. 289 dit que la bulle avance quand tout attend une minuterie, un canal ou une primitive de synchronisation. La documentation du paquet exclut pourtant nommément `sync.Mutex`, l'I/O et les canaux créés hors de la bulle. Sur ces trois cas, une goroutine fuitée n'est pas « durablement bloquée » : la bulle l'attend, et le test ne panique pas, il bloque. Le livre décrit le cas particulier sans le dire. Même `ctx-watcher-leak`, qui attend une minuterie d'une heure, est bien signalé, parce que le temps de la bulle cesse d'avancer quand sa goroutine racine se termine.

**Un interblocage que le livre dit fatal bloque dix minutes sous `go test` (H-007, H-008).** Exécutés comme programmes, les deux exemples de la p. 564 finissent bien par l'erreur fatale annoncée. Sous `go test`, ils bloquent. La cause se lit dans le runtime : `checkdead` renonce à déclarer l'interblocage dès qu'une minuterie est en attente (`proc.go`, lignes 6511 à 6519), et `go test` arme justement une minuterie d'alarme de dix minutes. L'affirmation est vraie pour un programme et fausse pour le test qu'on écrirait pour la vérifier. Seule la bulle `synctest` rend l'interblocage immédiat. Le symptôme n'est pas une trouvaille *(ajout du 2026-09-22)* : l'*issue* golang/go n° 69188 (2024) signale que le délai de `go test` empêche la détection d'interblocage, et Tu et coll. (2019) montraient que le détecteur intégré ne voit que 2 des 21 bogues bloquants qu'ils reproduisent. Le banc le réplique sur les exemples du livre; il y ajoute la lecture de la cause dans le runtime, que le texte de l'*issue* ne nomme pas ([`ETAT-DE-L-ART.md`](ETAT-DE-L-ART.md), §2).

**Oublier `cancel()` coûte de la mémoire jusqu'à l'échéance, pas des goroutines (H-009, H-010, H-014).** Un `context.WithTimeout` non annulé retient 275 octets avec un parent `Background`, 319 avec un parent annulable, et les rend après l'échéance : net du coût d'expiration de toute minuterie, le résidu est négatif (H-014). Aucune goroutine n'est créée avec un parent standard. La p. 567 n'a raison sur les goroutines que pour un parent que `context` ne sait pas reconnaître : l'enveloppe opaque du témoin en a lancé une par enfant, 20 000 pour 20 000 contextes.

**`time.After` en boucle ne fait plus croître la mémoire (H-011).** Depuis Go 1.23, une minuterie qui n'est plus référencée est collectée même si elle n'a pas été arrêtée, comme l'annoncent les notes de version. Le conseil de la p. 276 est dépassé pour la version de Go que le livre cible (1.25) : sur 100 000 itérations bloquantes, la boucle à `time.After` retient 0,01 octet de plus par itération que la minuterie réutilisée, quand le témoin qui garde ses minuteries en retient 258.

**Le profil `goroutineleak` a un angle mort que ses notes de version ne décrivent pas (H-013).** Il a signalé six des sept fuites de son domaine, et aussi le `select {}` de la p. 290, que le runtime traite comme inévitablement bloqué. Il a manqué la fuite sur mutex cinq fois sur cinq, alors que le runtime prévoit ce cas. Contre-épreuve : un `sync.Mutex` alloué seul (8 octets) n'est jamais signalé, le même mutex au début d'une structure de 72 octets l'est chaque fois. Le mécanisme probable, non vérifié dans le code, est l'allocateur *tiny*, dont les blocs partagés restent marqués. Hors domaine, le profil manque comme annoncé les primitives accessibles (canal global, contexte tenu par une minuterie) et l'I/O réseau.

**Les analyseurs statiques voient ce que les détecteurs dynamiques ne voient pas, et rien d'autre.** `go vet` diagnostique les deux contextes non annulés (analyseur `lostcancel`), que `go test` ne voit pas : son sous-ensemble de `vet` exclut `lostcancel` (`go help test`). `ctxvet` diagnostique l'I/O sans contexte. Aucun des deux ne voit une fuite de canal ni un interblocage.

## Recommandation pour l'intégration continue

Tirée de la matrice, pour ce corpus et ce poste :

1. **`go vet ./...` explicite**, en plus de `go test` : c'est le seul moyen, dans ce banc, d'attraper un `cancel` oublié avant l'exécution.
2. **Un analyseur de contexte** du type `ctxvet`, pour l'I/O qui ignore le contexte ; aucune autre ligne de la matrice ne la voit statiquement.
3. **`go test -race`** pour les courses, et pour elles seules.
4. **Un contrôle de fuite dans les tests** : le compte de goroutines avant et après un scénario répété couvre les onze fuites ; `synctest` couvre le code sans mutex, sans canal global et sans réseau, et rend en outre les interblocages immédiats.
5. **Un `-timeout` court en CI** : sans lui, un interblocage coûte dix minutes de pipeline avant d'échouer.
6. **En production**, l'endpoint `/debug/pprof/goroutineleak` : il ne demande aucune répétition, mais manque les mutex isolés et les primitives accessibles depuis une variable globale.

## Ce que le banc a appris sur lui-même

La première campagne, `R-2026-09-13-1`, a produit trois résultats faux ou fragiles, tous trouvés en relisant les résultats contraires à l'attente avant de les publier.

- **Le banc n'appliquait pas sa propre contrainte C-004.** Lancé directement, un binaire de test n'a aucun délai ; seul `go test` lui en transmet un. Sans minuterie d'alarme, le runtime déclarait des interblocages qu'il ne déclare pas sous `go test` : H-008 était confirmée, et `synctest` semblait diagnostiquer `mutex-leak` et `global-channel-leak`. Le banc passe désormais `-test.timeout=10m0s` ; la campagne a été refaite en entier, et la première archivée. H-008 passe de confirmée à infirmée, H-005 de un à trois cas manqués.
- **La sonde de H-009 confondait deux coûts.** Son résidu après échéance comptait les structures que le runtime garde après l'expiration de toute minuterie : un rappel de minuterie s'exécute dans une goroutine. Le critère gelé n'a pas été retouché ; H-014 le reprend en retranchant un témoin sans contexte. Réserve écrite avant la campagne qui la juge : son critère connaissait l'ordre de grandeur du témoin.
- **Une mesure nulle n'était pas un défaut.** Les 0 ns de H-006 sont la résolution de l'horloge monotone de ce poste, établie par contre-épreuve.

Les verdicts de `R-2026-09-13-1` restent lisibles dans `results/verdicts/`. Sur les treize hypothèses qu'elle portait, deux diffèrent de la campagne de référence, H-005 par ses cas et H-008 par son verdict, pour la raison ci-dessus.

## Limites

- **Validité externe.** Corpus synthétique de 32 cas, un poste Windows amd64, Go 1.27.0. *(Révisé le 2026-09-23.)* La campagne a été rejouée sous Linux dans WSL2, sur le même poste (`R-2026-09-23-1`, D-20) : treize verdicts sur quatorze sont identiques; voir la section du rejeu.
- **Vérité terrain.** L'oracle vérifie par les piles de goroutines les fuites, leur primitive et l'absence de fuite des cas corrigés. Il ne vérifie ni les courses, ni l'accessibilité d'une primitive, ni les défauts statiques, ni les interblocages, qui tiennent par construction.
- **Détecteurs idéalisés.** NUMGOROUTINE mesure un scénario isolé répété dix fois ; dans une vraie suite, les tests parallèles partagent le compte, ce que le banc n'a pas mesuré. C'est en outre un pilote écrit pour le banc (K = 10, seuil K/2, attente d'au plus 1 s, C-006), pas un outil du livre : H-012 juge ce pilote plus que la surveillance « lente et continue » de la p. 293 *(précision du 2026-09-22)*.
- **Résolution de l'horloge** *(ajout du 2026-09-22)*. Sur ce poste Windows, H-006 borne la durée d'une bulle sans la mesurer : 0 ns est la résolution de l'horloge monotone, environ 5 µs par bulle en contre-épreuve (D-15).
- **Revue humaine.** Les cas d'utilisation ont été approuvés par l'agent sur mandat (D-01) : la revue prévue par l'AIUP n'a pas eu lieu avant la clôture. Une revue à froid par un tiers est préparée ([`Revue/DOSSIER-REVUE-UC_2026-09-22.md`](../Revue/DOSSIER-REVUE-UC_2026-09-22.md)); elle n'a pas encore eu lieu *(ajout du 2026-09-22)*. LeakLab n'a pas eu non plus de revue par agents ([`QR2-MESURES.md`](QR2-MESURES.md)).
- **Travaux antérieurs** *(ajout du 2026-09-22)*. Plusieurs résultats répliquent des travaux connus (Tu et coll., 2019, pour H-002 et H-004; l'*issue* golang/go n° 69188 pour H-008); positionnement dans [`ETAT-DE-L-ART.md`](ETAT-DE-L-ART.md).

## Rejeu sous Linux (2026-09-23)

`R-2026-09-23-1`, sous Ubuntu 24.04 dans WSL2, même poste, go1.27.0 ([rapport](../Campagnes/RAPPORT-CAMPAGNE_R-2026-09-23-linux.md), D-20). Un seul verdict change : **H-007 passe de confirmée à infirmée**. Exécutés comme programmes, les deux interblocages tournent jusqu'au délai au lieu de finir en erreur fatale. Le corpus importe `net`, ce qui lie cgo au binaire sous Linux, et `checkdead` (`runtime/proc.go`) ne déclare pas l'interblocage d'un programme cgo. Construit avec `CGO_ENABLED=0`, le même binaire finit en erreur fatale en 0,01 s (vérifié). Portée : de banc, puisque l'infirmation tient à la composition du corpus. Elle révèle toutefois une condition que le livre ne donne pas : l'erreur fatale suppose un programme sans cgo. Les deux angles morts propres au dépôt (H-008, H-013) se retrouvent à l'identique sous Linux.

## Ce qui reste ouvert

- Rejouer la campagne sur arm64 (le rejeu Linux est fait, D-20).
- Vérifier dans le runtime le mécanisme de l'angle mort du profil `goroutineleak` sur les petits mutex, et l'étendre à `sync.RWMutex`.
- Mesurer NUMGOROUTINE dans une suite réelle, sous `t.Parallel`.
- Passer au projet suivant de la séquence recommandée, P4 (HexaGuard).
