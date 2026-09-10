# Catalogue d'exigences — EscapeBench

Conventions : `FR-###` exigence fonctionnelle, `NFR-###` exigence non fonctionnelle, `C-###` contrainte, `H-###` hypothèse à éprouver (adaptation du processus AIUP aux bancs de réfutation ; voir `Guide-implementation_AIUP-Claude-Code.md`, §4). Les identifiants sont stables et ne sont jamais réutilisés. Pages citées : folios imprimés de *Building Enterprise Projects with Go* (BEPG).

## Exigences fonctionnelles

| ID | Titre | Récit utilisateur |
|---|---|---|
| FR-001 | Générer la matrice de cellules | En tant que chercheur, je veux générer une matrice de types Go paramétrée par taille, présence d'un champ pointeur et profil de durée de vie, afin de couvrir l'espace des cas décrits par le livre sans les écrire à la main. |
| FR-002 | Classer l'échappement | En tant que chercheur, je veux obtenir pour chaque cellule un verdict d'échappement (pile ou tas) avec la raison rapportée par le compilateur, afin de comparer le comportement observé aux quatre causes listées par le livre. |
| FR-003 | Exécuter une campagne de mesure | En tant que chercheur, je veux exécuter une campagne de benchmarks sur une matrice donnée et obtenir, par cellule, temps, octets et allocations par opération avec leur provenance, afin de disposer de mesures rejouables. |
| FR-004 | Comparer valeur et pointeur | En tant que chercheur, je veux comparer statistiquement, cellule par cellule, le passage par valeur et le passage par pointeur et localiser le point de bascule, afin de confronter la règle « 1–3 mots machine » aux données. |
| FR-005 | Produire un verdict par hypothèse | En tant que chercheur, je veux qu'un verdict (confirmée, infirmée, non concluante) soit produit pour chaque hypothèse à partir des mesures et du critère de réfutation gelé, afin que le rapport ne dépende pas d'une interprétation après coup. |
| FR-006 | Mesurer l'accès séquentiel et dispersé | En tant que chercheur, je veux mesurer le parcours d'une tranche de structures contiguës et d'un ensemble de pointeurs dispersés, afin d'éprouver le rapport de coût cache annoncé par le livre. |
| FR-007 | Régénérer le tableau de bord | En tant que chercheur, je veux que `docs/dashboard.md` soit régénéré à partir des verdicts, de l'état des cas d'utilisation et du résultat des tests par cas d'utilisation, afin que l'avancement se mesure au comportement vérifié et non à l'activité. |

## Exigences non fonctionnelles

| ID | Titre | Description |
|---|---|---|
| NFR-001 | Provenance des mesures | Tout fichier de résultats porte la version de Go, `GOOS`, `GOARCH`, le modèle de CPU, la date UTC et l'identifiant de la campagne ; un résultat sans provenance est invalide. |
| NFR-002 | Reproductibilité des verdicts d'échappement | Pour une même toolchain et une même matrice, deux exécutions de FR-002 produisent des verdicts identiques cellule par cellule. |
| NFR-003 | Mesures répétées | Chaque cellule d'une campagne est mesurée au moins vingt fois ; les comparaisons FR-004 rapportent un intervalle de confiance, jamais une moyenne seule. |
| NFR-004 | Immutabilité des résultats | Un dossier de résultats de campagne n'est jamais réécrit ; une nouvelle exécution crée un nouveau dossier. |
| NFR-005 | Durée de campagne | Une campagne complète sur la matrice de référence se termine en moins de 60 minutes sur un poste de développement à 8 cœurs (valeur cible, à ajuster après la première campagne). |

## Contraintes

| ID | Titre | Description |
|---|---|---|
| C-001 | Toolchain | Go 1.25 ou plus récent ; la version exacte est consignée dans chaque résultat. |
| C-002 | Dépendances | Bibliothèque standard uniquement pour le banc ; `golang.org/x/perf/cmd/benchstat` autorisé pour l'analyse hors banc (outil hors BEPG, signalé comme tel). |
| C-003 | Drapeaux de mesure | Classification par `go build -gcflags=-m` (BEPG p. 241) ; mesures par `go test -bench -benchmem -count=N` avec `N` ≥ 20 (NFR-003) ; `-cpu` fixé à 1 pour les cellules de coût de copie. |
| C-004 | Layout | Structure hexagonale de BEPG ch. 14 (p. 368–369) : `cmd/`, `internal/{models,service,ports,adapters}` ; `internal/models` n'importe que la bibliothèque standard. |
| C-005 | Harnais figé | Le code sous `internal/harness/` est identique pour toutes les cellules d'une campagne ; toute modification invalide la campagne en cours. |
| C-006 | Architectures | Campagnes exécutées au minimum sur `amd64` ; `arm64` souhaité pour H-002 et H-004. |

## Hypothèses à éprouver

Chaque hypothèse reprend une affirmation de BEPG, précise l'énoncé réfutable et gèle son critère de réfutation avant toute mesure. Le verdict est produit par FR-005 et consigné dans `dashboard.md`.

| ID | Source (BEPG) | Énoncé réfutable | Critère de réfutation | UC liés |
|---|---|---|---|---|
| H-001 | p. 245, 253 — « copying a small struct can be cheaper than passing a pointer to it » | Pour des structures de taille ≤ 24 octets sans champ pointeur, en profil de vie local, le passage par valeur a un `ns/op` inférieur ou égal au passage par pointeur. | Infirmée si, pour au moins deux tailles ≤ 24 octets, la borne inférieure de l'intervalle de confiance à 95 % du passage par pointeur est strictement inférieure à la borne supérieure du passage par valeur. | UC-003, UC-004 |
| H-002 | p. 253 — valeur « when the data type is small (typically one to three machine words) » | Le point de bascule valeur → pointeur se situe entre 8 et 24 octets (1 à 3 mots sur 64 bits) pour le profil local. | Infirmée si le point de bascule observé (première taille où le pointeur devient significativement plus rapide) est inférieur à 8 ou supérieur à 24 octets sur au moins une architecture mesurée. | UC-003, UC-004 |
| H-003 | p. 256 — un changement valeur/pointeur « doubles the number of allocations » | Pour le profil « retournée », passer du retour par valeur au retour par pointeur multiplie `allocs/op` par un facteur ≥ 2 sur au moins une taille. | Infirmée si le rapport `allocs/op` pointeur / valeur reste < 2 pour toutes les tailles du profil. | UC-003 |
| H-004 | p. 254 — accès mémoire « roughly 10× to 200× » entre cache hit et miss | Le rapport `ns/op` entre parcours dispersé et parcours séquentiel (FR-006) se situe dans [10, 200] pour des ensembles excédant le cache L2. | Infirmée si le rapport observé est < 10 ou > 200 sur toutes les tailles d'ensemble mesurées au-delà du cache L2. | UC-003, UC-004 |
| H-005 | p. 114 — préallocation : « about 6× » plus rapide, « one-fifth the memory » | Pour `n` = 100 000 entiers, l'`append` avec préallocation a un `ns/op` ≤ 1/4 de la version sans préallocation et un `B/op` ≤ 1/4. | Infirmée si l'un des deux rapports dépasse 1/2 (marge de deux fois par rapport à l'affirmation). | UC-003 |
| H-006 | p. 238–242 — quatre causes d'échappement listées | Toute cellule dont le compilateur rapporte un échappement se classe dans l'une des quatre causes du livre (retour de pointeur, capture par closure, envoi sur canal, stockage dans map/slice/struct). | Infirmée si au moins une cellule échappe pour une raison rapportée par `-gcflags=-m` qui n'appartient à aucune des quatre catégories (ex. taille excédant la limite de pile, appel d'interface, `fmt`). | UC-002 |

Liens de traçabilité : chaque `UC-###` liste ses `FR`, `NFR`, `C` et `H` ; chaque test référence son `UC` ; chaque fichier de `results/` référence sa campagne et ses `H`.
