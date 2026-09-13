# Catalogue d'exigences — LeakLab

**Date :** 2026-09-13 · Source des affirmations : *Building Enterprise Projects with Go* (BEPG), folios imprimés. Les citations sont réduites à quelques mots ; le reste est paraphrasé.

## Exigences fonctionnelles

| ID | Titre | Énoncé |
|---|---|---|
| FR-001 | Vérité terrain du corpus | En tant que chercheur, je veux que chaque cas du corpus porte sa vérité terrain (fuite, blocage, course, défaut statique) et qu'un oracle indépendant des détecteurs la vérifie avant toute mesure, afin qu'un verdict ne repose jamais sur un corpus mal étiqueté. |
| FR-002 | Détecteurs dynamiques | En tant que chercheur, je veux exécuter chaque cas sous chaque détecteur dynamique (BARE, RACE, SYNCTEST, NUMGOROUTINE, LEAKPROFILE, PROGRAM), en processus isolés et répétés, et obtenir une issue classée par exécution. |
| FR-003 | Détecteurs statiques | En tant que chercheur, je veux passer le corpus à `go vet` et à `ctxvet`, et attribuer chaque diagnostic au cas dont il vise le fichier. |
| FR-004 | Sondes | En tant que chercheur, je veux mesurer les grandeurs que le livre chiffre ou qualifie — durée réelle d'un test en bulle, mémoire et goroutines retenues par un contexte non annulé, mémoire retenue par `time.After` en boucle — avec un bras témoin par sonde. |
| FR-005 | Verdicts | En tant que chercheur, je veux obtenir pour chaque hypothèse un verdict produit par le seul critère gelé, avec les cellules et sondes qui le fondent. |
| FR-006 | Matrice de détectabilité | En tant que chercheur, je veux une matrice cas × détecteur donnant l'issue majoritaire et la stabilité de chaque cellule, publiée avec les verdicts. |
| FR-007 | Analyseur `ctxvet` | En tant que développeur, je veux lancer `ctxvet` seul sur un répertoire Go et obtenir la liste des appels d'I/O qui ignorent le contexte, avec l'API qui le respecte. |

## Exigences non fonctionnelles

| ID | Titre | Énoncé |
|---|---|---|
| NFR-001 | Provenance | Tout fichier de résultats porte la version de Go, `GOOS`, `GOARCH`, l'identifiant du processeur, le nombre de cœurs logiques, la date UTC et l'identifiant de la campagne. |
| NFR-002 | Répétitions | Chaque couple cas × détecteur dynamique et chaque bras de sonde est exécuté au moins cinq fois ; une grandeur de sonde est rapportée par sa médiane. |
| NFR-003 | Isolation | Une observation dynamique = un processus neuf ; aucun état du runtime (goroutines, minuteries, tas) ne passe d'une observation à l'autre. |
| NFR-004 | Résultats immuables | Un fichier sous `results/` est créé en écriture exclusive et jamais réécrit ; un nouveau calcul crée un nouveau fichier horodaté. |
| NFR-005 | Durée | Une campagne complète tient en moins de 30 minutes sur le poste de référence. |

## Contraintes

| ID | Titre | Énoncé |
|---|---|---|
| C-001 | Toolchain | Go 1.27 ou plus récent : le profil `goroutineleak` n'est disponible sans `GOEXPERIMENT` qu'à partir de 1.27, `testing/synctest` depuis 1.25. La version exacte est consignée (NFR-001). |
| C-002 | Dépendances | Bibliothèque standard uniquement. `goleak` et `golang.org/x/tools/go/analysis` sont hors périmètre ; `ctxvet` est écrit avec `go/parser` et `go/ast`. |
| C-003 | Corpus isolé | Le corpus vit dans un module Go imbriqué, `lab/`, pour que `go vet ./...` et `go test ./...` du module principal ne l'analysent ni ne l'exécutent : ses défauts sont voulus. |
| C-004 | Délai et défauts de `go test` | Chaque observation dynamique est tuée au bout de 5 s de temps réel et classée `HANG`. Les binaires de test gardent le délai par défaut de `go test` (10 min) : le banc observe ce qu'obtient un développeur qui lance ses tests sans option. **Précision du 2026-09-13, après `R-2026-09-13-1` :** un binaire de test lancé directement n'a aucun délai (`testing` le fixe à 0) ; c'est `go test` qui lui transmet `-test.timeout=10m0s`. Le banc passe donc cette option à chaque binaire de test et de sonde. `R-2026-09-13-1` ne la passait pas et n'est pas conforme à cette contrainte. |
| C-005 | Classification des issues | L'issue d'une observation dynamique est la première qui s'applique : `HANG` (processus tué par C-004) ; `RACE` (la sortie contient `WARNING: DATA RACE`) ; `DEADLOCK` (une ligne commence par `fatal error:` ou `panic:` et contient `deadlock`) ; `LEAK` (une ligne commence par `LEAKLAB-LEAK`, écrite par le pilote de NUMGOROUTINE ou de LEAKPROFILE) ; `FAIL` (code de sortie non nul) ; `PASS`. Une observation statique vaut `DIAGNOSTIC` si au moins un diagnostic vise le fichier du cas, `PASS` sinon. |
| C-006 | Pilotes des détecteurs | BARE : le scénario dans un test ordinaire, sortie `-test.v`. RACE : le même test compilé avec `-race`. SYNCTEST : le scénario dans `synctest.Test`. NUMGOROUTINE : le scénario exécuté K = 10 fois de suite ; si, après au plus 1 s d'attente, le nombre de goroutines dépasse le nombre initial d'au moins K/2, le pilote écrit `LEAKLAB-LEAK`. LEAKPROFILE : le scénario exécuté une fois, puis le profil `goroutineleak` écrit ; un total non nul fait écrire `LEAKLAB-LEAK`. **Révision du 2026-09-13, écrite en construisant le pilote, avant toute campagne :** une pause de 100 ms précède l'écriture du profil, pour que les goroutines lancées par le scénario atteignent leur point de blocage ; une goroutine encore exécutable au moment du profil ne peut pas être reconnue fuitée, et le détecteur serait jugé sur un instant plutôt que sur une fuite. PROGRAM : le scénario dans la fonction `main` d'un programme. VET : `go vet` sur le paquet du corpus. CTXVET : `ctxvet` sur le même paquet. Une erreur rendue par le scénario fait échouer le test ou le programme. |
| C-007 | Sondes | SYNCTEST_TIMEOUT : la fonction `QueryWithTimeout` du livre (p. 290), délai de 500 ms, chargement qui attend l'annulation ; bras `SYNCTEST` (dans une bulle) et `REAL` (hors bulle), temps réel mesuré hors de la bulle. CANCEL_RETENTION : N = 20 000 appels à `context.WithTimeout` ; parents `BACKGROUND`, `CANCELABLE` (issu de `WithCancel`, gardé vivant) et `OPAQUE` (enveloppe d'un parent annulable dont `Value` rend toujours `nil`) ; modes `FORGOTTEN` (délai d'une heure, `cancel` jamais appelé), `CANCELLED` (`cancel` appelé aussitôt), `EXPIRED` (délai d'une milliseconde, `cancel` jamais appelé, mesure après 500 ms) ; grandeurs : octets de tas retenus par contexte et écart du nombre de goroutines, après deux ramasse-miettes. **Ajout du 2026-09-13, pour H-014 :** bras `AFTERFUNC_WITNESS`, N appels à `time.AfterFunc` d'une milliseconde sans contexte, mesure après 500 ms, mêmes grandeurs. TIMER_GROWTH : N = 100 000 itérations dont chacune attend, dans un `select`, un résultat produit par une goroutine ; bras `AFTER_IN_LOOP` (un `time.After(1 h)` par itération), `REUSED_TIMER` (une seule minuterie d'une heure), `RETAINED_WITNESS` (une minuterie d'une heure par itération, gardée dans une tranche) ; grandeur : octets de tas retenus par itération après deux ramasse-miettes. |
| C-008 | Empreinte des critères | À la création d'une campagne, l'empreinte SHA-256 du texte du critère de chaque hypothèse est consignée ; les verdicts sont refusés si une empreinte a changé. |

## Corpus de référence

Vérité terrain de chaque cas. `faulty` : le cas porte le défaut ; `leak` : une goroutine au moins reste bloquée pour toujours après le retour du scénario (pour `ctx-watcher-leak`, jusqu'à l'échéance d'une heure) ; `blocks` : le scénario ne rend jamais la main ; `race` : course de données ; `primitive` : ce qui bloque la goroutine fuitée ou le scénario ; `reachable` : la primitive reste accessible depuis une racine du ramasse-miettes (variable globale, minuterie du runtime) ; `static` : défaut qu'un analyseur statique peut viser.

| Cas | Anti-patron | Page | faulty | leak | blocks | race | primitive | reachable | static | Corrige |
|---|---|---|---|---|---|---|---|---|---|---|
| dispatch-leak | GOROUTINE_LEAK | 561 | oui | oui | non | non | CHAN_SEND | non | NONE | — |
| dispatch-buffered-leak | GOROUTINE_LEAK | 566 | oui | oui | non | non | CHAN_SEND | non | NONE | — |
| abandoned-receive-leak | GOROUTINE_LEAK | 561 | oui | oui | non | non | CHAN_RECV | non | NONE | — |
| select-no-cancel-leak | GOROUTINE_LEAK | 562 | oui | oui | non | non | SELECT | non | NONE | — |
| waitgroup-leak | GOROUTINE_LEAK | 562 | oui | oui | non | non | WAITGROUP | non | NONE | — |
| cond-leak | GOROUTINE_LEAK | 562 | oui | oui | non | non | COND | non | NONE | — |
| mutex-leak | GOROUTINE_LEAK | 562 | oui | oui | non | non | MUTEX | non | NONE | — |
| global-channel-leak | GOROUTINE_LEAK | 561 | oui | oui | non | non | CHAN_SEND | oui | NONE | — |
| empty-select-leak | GOROUTINE_LEAK | 290 | oui | oui | non | non | NONE | non | NONE | — |
| ctx-watcher-leak | FORGOTTEN_CANCEL | 567 | oui | oui | non | non | CHAN_RECV | oui | LOST_CANCEL | — |
| io-without-context-leak | CONTEXT_IGNORED_IO | 568 | oui | oui | non | non | NET_READ | non | CTX_IO | — |
| forgotten-cancel | FORGOTTEN_CANCEL | 566 | oui | non | non | non | NONE | non | LOST_CANCEL | — |
| counter-race | DATA_RACE | 232 | oui | non | non | oui | NONE | non | NONE | — |
| append-race | DATA_RACE | 232 | oui | non | non | oui | NONE | non | NONE | — |
| deadlock-send | CHANNEL_DEADLOCK | 564 | oui | non | oui | non | CHAN_SEND | non | NONE | — |
| deadlock-range | CHANNEL_DEADLOCK | 564 | oui | non | oui | non | CHAN_RECV | non | NONE | — |
| dispatch-fix | GOROUTINE_LEAK | 562 | non | non | non | non | NONE | non | NONE | dispatch-leak |
| abandoned-receive-fix | GOROUTINE_LEAK | 563 | non | non | non | non | NONE | non | NONE | abandoned-receive-leak |
| select-no-cancel-fix | GOROUTINE_LEAK | 562 | non | non | non | non | NONE | non | NONE | select-no-cancel-leak |
| waitgroup-fix | GOROUTINE_LEAK | 562 | non | non | non | non | NONE | non | NONE | waitgroup-leak |
| cond-fix | GOROUTINE_LEAK | 562 | non | non | non | non | NONE | non | NONE | cond-leak |
| mutex-fix | GOROUTINE_LEAK | 562 | non | non | non | non | NONE | non | NONE | mutex-leak |
| global-channel-fix | GOROUTINE_LEAK | 565 | non | non | non | non | NONE | non | NONE | global-channel-leak |
| empty-select-fix | GOROUTINE_LEAK | 291 | non | non | non | non | NONE | non | NONE | empty-select-leak |
| ctx-watcher-fix | FORGOTTEN_CANCEL | 567 | non | non | non | non | NONE | non | NONE | ctx-watcher-leak |
| io-without-context-fix | CONTEXT_IGNORED_IO | 568 | non | non | non | non | NONE | non | NONE | io-without-context-leak |
| forgotten-cancel-fix | FORGOTTEN_CANCEL | 567 | non | non | non | non | NONE | non | NONE | forgotten-cancel |
| counter-race-fix | DATA_RACE | 232 | non | non | non | non | NONE | non | NONE | counter-race |
| append-race-fix | DATA_RACE | 232 | non | non | non | non | NONE | non | NONE | append-race |
| deadlock-send-fix | CHANNEL_DEADLOCK | 566 | non | non | non | non | NONE | non | NONE | deadlock-send |
| deadlock-range-fix | CHANNEL_DEADLOCK | 565 | non | non | non | non | NONE | non | NONE | deadlock-range |
| witness-fail | WITNESS | — | non | non | non | non | NONE | non | NONE | — |

Ensembles nommés par les critères :

- **L** (fuites) : cas `faulty`, `leak`, non `blocks`. Onze cas.
- **S** (sains pour les goroutines) : cas ni `leak` ni `blocks`, hors `WITNESS`.
- **R** (courses) : cas `race`.
- **D** (interblocages) : cas `blocks`.
- **N** (défauts autres que les courses) : cas `faulty` non `race`.

Lecture d'une cellule cas × détecteur (répétitions de NFR-002) : le cas est **diagnostiqué X** si plus de la moitié des répétitions ont l'issue X ; il est **détecté** si plus de la moitié ont une issue autre que `PASS` ; la cellule est **instable** si toutes les répétitions n'ont pas la même issue.

## Hypothèses à éprouver

Chaque hypothèse reprend une affirmation de BEPG — H-013 excepté, tirée des notes de version de Go et marquée hors livre —, précise l'énoncé réfutable et gèle son critère avant toute mesure. Les critères sont écrits le 2026-09-13, avant l'écriture du banc et avant toute exécution du corpus.

| ID | Source | Énoncé réfutable | Critère de réfutation | UC liés |
|---|---|---|---|---|
| H-001 | p. 561 — une fuite est silencieuse : « Tests may pass » | Un test ordinaire passe sur un cas de fuite de goroutine. | Infirmée si BARE détecte au moins la moitié des cas de L, ou détecte `dispatch-leak`, exemple du livre. Confirmée sinon. Non concluante si BARE ne détecte pas `witness-fail`. | UC-001, UC-002 |
| H-002 | p. 217 — depuis Go 1.25, le cadre de test signalerait mieux les goroutines fuitées en fin d'exécution | La sortie de `go test -v` signale la goroutine qu'un cas de fuite laisse en vie. | Un signalement est une ligne de sortie BARE, hors lignes commençant par `=== `, `--- `, `PASS`, `FAIL` ou `ok`, qui contient `leak` ou `goroutine` (casse ignorée). Infirmée si aucune exécution BARE d'un cas de L ne contient de signalement ; confirmée si toutes en contiennent ; non concluante sinon, ou si aucune exécution BARE de `witness-fail` ne contient `LEAKLAB-WITNESS`. | UC-001, UC-002 |
| H-003 | p. 232 — une course trouvée donne un rapport détaillé et un échec | Sous `-race`, chaque cas de course est diagnostiqué, et aucun cas sans course. | Infirmée si un cas de R n'est pas diagnostiqué `RACE` par RACE, ou si un cas non `race` et non `blocks` est diagnostiqué `RACE` par RACE. Confirmée sinon. Non concluante si R compte moins de deux cas. | UC-001, UC-002 |
| H-004 | p. 232 — lancer toute la suite sous `-race` en CI attraperait les bogues de concurrence difficiles à trouver à la main | Le détecteur de courses révèle des défauts de concurrence autres que les courses qu'un test ordinaire ne révèle pas. | Un cas de N est attribuable à `-race` s'il est diagnostiqué `RACE` par RACE, ou détecté par RACE sans l'être par BARE. Infirmée si aucun cas de N n'est attribuable ; confirmée si au moins la moitié des cas de N le sont ; non concluante sinon. | UC-001, UC-002 |
| H-005 | p. 289–292 — la bulle avance quand tout attend une minuterie, un canal ou une primitive de synchronisation, et `synctest` échoue par une panique d'interblocage si une goroutine reste bloquée à la fin du test | Exécuté dans une bulle, tout cas de fuite de goroutine est diagnostiqué par une panique d'interblocage. | Infirmée si un cas de L n'est pas diagnostiqué `DEADLOCK` par SYNCTEST. Confirmée sinon. Non concluante si aucun cas de L n'est diagnostiqué `DEADLOCK` par SYNCTEST. | UC-001, UC-002 |
| H-006 | p. 291 — le test de délai de 500 ms « completes instantly », quelle que soit la charge | Dans une bulle, le test de délai de 500 ms du livre dure une fraction négligeable de sa durée simulée. | Sonde SYNCTEST_TIMEOUT, médiane du temps réel par bras. Non concluante si la médiane de `REAL` est inférieure à 450 ms. Confirmée si la médiane de `SYNCTEST` est inférieure à 50 ms ; infirmée si elle est d'au moins 250 ms ; non concluante entre les deux. | UC-001, UC-002 |
| H-007 | p. 564 — les deux exemples d'interblocage finissent par l'erreur fatale « all goroutines are asleep » | Exécutés comme programmes, `deadlock-send` et `deadlock-range` se terminent par l'erreur fatale d'interblocage du runtime. | Infirmée si l'un des deux cas n'est pas diagnostiqué `DEADLOCK` par PROGRAM. Confirmée sinon. | UC-001, UC-002 |
| H-008 | p. 564 — même affirmation que H-007, éprouvée par un test ordinaire | Exécutés par `go test`, les deux mêmes cas se terminent par la même erreur fatale. | Infirmée si l'un des deux cas n'est pas diagnostiqué `DEADLOCK` par BARE. Confirmée sinon. | UC-001, UC-002 |
| H-009 | p. 567 — sans `cancel()`, les ressources d'un contexte à délai restent vivantes jusqu'à l'échéance, d'où plus de mémoire | Un `WithTimeout` non annulé retient de la mémoire jusqu'à son échéance, et la libère ensuite. | Sonde CANCEL_RETENTION, médianes d'octets retenus par contexte. Pour chaque parent `BACKGROUND` et `CANCELABLE` : retenue = `FORGOTTEN` − `CANCELLED` ; résidu = `EXPIRED` − `CANCELLED`. Confirmée si la retenue est d'au moins 32 octets et le résidu inférieur à 8 octets pour les deux parents. Infirmée si la retenue est inférieure à 8 octets pour les deux parents, ou si le résidu est d'au moins 32 octets pour l'un d'eux. Non concluante sinon. | UC-001, UC-002 |
| H-010 | p. 567 — l'oubli répété de `cancel()` coûte aussi davantage de goroutines | Un `WithTimeout` non annulé laisse des goroutines en vie jusqu'à son échéance. | Sonde CANCEL_RETENTION, médianes de l'écart de goroutines ; delta = `FORGOTTEN` − `CANCELLED` par parent, N = 20 000. Non concluante si le delta du parent `OPAQUE` est inférieur à N/2, la sonde étant alors aveugle aux goroutines ; ce parent ne décide jamais du verdict. Infirmée si le delta est au plus 1 pour `BACKGROUND` et pour `CANCELABLE`. Confirmée s'il est d'au moins N/2 pour les deux. Non concluante sinon. | UC-001, UC-002 |
| H-011 | p. 276 — un `time.After` par itération de boucle mène à une « steady memory growth » | La mémoire retenue croît avec le nombre d'itérations d'une boucle qui crée un `time.After` à chaque tour. | Sonde TIMER_GROWTH, médianes d'octets retenus par itération ; croissance = `AFTER_IN_LOOP` − `REUSED_TIMER` ; référence = `RETAINED_WITNESS` − `REUSED_TIMER`. Non concluante si la référence est inférieure à 64 octets. Infirmée si la croissance est inférieure à 10 % de la référence ; confirmée si elle en atteint 50 % ; non concluante entre les deux. | UC-001, UC-002 |
| H-012 | p. 293 et 563 — une hausse lente et continue de `runtime.NumGoroutine()` signale une fuite | La surveillance du nombre de goroutines sépare les cas de fuite des cas sains. | Infirmée si un cas de L n'est pas diagnostiqué `LEAK` par NUMGOROUTINE, ou si un cas de S est diagnostiqué `LEAK` par NUMGOROUTINE. Confirmée sinon. | UC-001, UC-002 |
| H-013 | Notes de version Go 1.26 et 1.27 (hors livre) — le profil `goroutineleak` repère une goroutine bloquée sur une primitive de concurrence devenue inaccessible, et peut manquer celles dont la primitive reste accessible | Le profil `goroutineleak` signale toute fuite bloquée sur une primitive de concurrence inaccessible, et aucun cas sain. | Domaine : cas de L dont `primitive` ∈ {CHAN_SEND, CHAN_RECV, SELECT, WAITGROUP, COND, MUTEX} et `reachable` = non. Infirmée si un cas du domaine n'est pas diagnostiqué `LEAK` par LEAKPROFILE, ou si un cas de S est diagnostiqué `LEAK` par LEAKPROFILE. Confirmée sinon. Les cas de L hors domaine sont rapportés et ne décident pas. | UC-001, UC-002 |
| H-014 | p. 567 — même affirmation que H-009, résidu mesuré net de ce que l'expiration d'une minuterie laisse au runtime | Un `WithTimeout` non annulé retient de la mémoire jusqu'à son échéance, et la libère ensuite, une fois retranché le coût d'expiration commun à toute minuterie. | Sonde CANCEL_RETENTION, médianes d'octets retenus par opération. Pour chaque parent `BACKGROUND` et `CANCELABLE` : retenue = `FORGOTTEN` − `CANCELLED` ; résidu = `EXPIRED` − `AFTERFUNC_WITNESS`. Confirmée si la retenue est d'au moins 32 octets et le résidu inférieur à 8 octets pour les deux parents. Infirmée si la retenue est inférieure à 8 octets pour les deux parents, ou si le résidu est d'au moins 32 octets pour l'un d'eux. Non concluante sinon. | UC-001, UC-002 |

### Tensions et réserves écrites avant les mesures

- H-001 et H-002 se contredisent dans le livre lui-même : p. 561, une fuite passe les tests en silence ; p. 217, le cadre de test la signalerait. Le banc les éprouve séparément, l'une sur l'issue du test, l'autre sur sa sortie.
- H-004 lit une leçon générale (p. 232) comme une promesse de couverture. Le livre ne dit pas que `-race` trouve les fuites ; une infirmation dira que la leçon ne couvre que les courses, pas que le livre se trompe sur `-race`.
- H-008 opérationnalise H-007 par l'outil qu'un lecteur emploierait pour la vérifier. Le livre parle d'un programme ; seul H-007 juge sa lettre.
- L'auteur du banc connaissait avant d'écrire ces critères deux changements de Go qui touchent H-011 (minuteries collectables depuis Go 1.23) et H-013 (profil disponible depuis 1.27). Les seuils sont dérivés du texte du livre et des notes de version, non d'une mesure.

### Ajout du 2026-09-13, après la campagne `R-2026-09-13-1`

H-014 est l'hypothèse successeur de H-009. Elle ne la remplace ni ne la corrige : le critère de H-009 reste gelé et son verdict acquis. Elle reprend la même affirmation sur une mesure qui ne porte plus le défaut de construction que `R-2026-09-13-1` a révélé. Le résidu de H-009 compare des contextes expirés à des contextes annulés ; or l'expiration d'une minuterie lance son rappel dans une goroutine, et le runtime garde ensuite des structures de goroutines et de minuteries qu'aucune annulation ne crée. Une contre-épreuve hors campagne, sur le poste de référence, l'a mesuré : 20 000 `time.AfterFunc` expirés sans aucun contexte laissent 182 à 234 octets par opération ; 20 000 contextes expirés, 147 à 156. Le résidu de H-009 mesurait donc le coût d'expiration, pas la rétention des contextes. H-014 le retranche par le bras `AFTERFUNC_WITNESS` (C-007).

Réserve : ce critère est écrit en connaissant l'ordre de grandeur du témoin. Un verdict `CONFIRMED` y pèse moins qu'un critère écrit à l'aveugle ; une infirmation, en revanche, garderait toute sa force. H-014 est non concluante sur `R-2026-09-13-1`, qui ne porte ni son empreinte ni son bras témoin.
