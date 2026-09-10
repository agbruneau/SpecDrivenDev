# Projets académiques candidats — exploration des concepts de *Building Enterprise Projects with Go*

**Source analysée :** Saeed Shahsavan, *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*, Apress, 2026, ISBN 979-8-8688-2370-1, DOI 10.1007/979-8-8688-2370-1, 611 pages PDF, 20 chapitres en deux parties. Dépôt de code du livre : `github.com/shahsavan/building-enterprise-projects-with-go` (vérifié en ligne le 2026-09-10 : dix répertoires numérotés `01-hello-world` → `10-more-examples`, sans README).

**Date :** 2026-09-10 · **Longueur :** ~6 300 mots (tableaux compris).

---

## 1. Conclusion

Le livre fournit une quarantaine d'affirmations réfutables, concentrées dans les chapitres 8, 9, 13, 16, 17, 18 et 20 ; huit projets d'exploration en découlent, dont trois se démarquent par leur rapport réfutabilité/coût d'infrastructure : **P1 EscapeBench** (frontière valeur/pointeur), **P3 LeakLab** (détectabilité des anti-patrons de concurrence) et **P4 HexaGuard** (règle de dépendance hexagonale exécutable). Séquence recommandée : P1 → P3 → P4, puis P5 (évolution de contrats) comme pont vers la messagerie, et P6 (tuning Pulsar) comme projet de synthèse.

## 2. Hypothèses de travail

- **Finalité** : exploration personnelle rigoureuse (question de recherche, hypothèses tirées du livre, protocole de mesure, critère de réfutation), dans la lignée de FibGo/TauGo. Pas de variante « enseignement ».
- **Ancrage** : concepts et outils du livre uniquement. Les outils absents du livre mais nécessaires à un protocole (ex. `benchstat`, `goleak`, `go/analysis`) sont signalés explicitement comme **hors livre**.
- **Ligne de base** : Go 1.25 (celle du livre). Vérifié dans les notes de version officielles (go.dev/doc/go1.25, publiée août 2025) : `testing/synctest` est stable en 1.25 ; `GOMAXPROCS` tient compte de la limite CPU cgroup et se met à jour périodiquement ; Green Tea GC est expérimental (`GOEXPERIMENT=greenteagc`) ; `encoding/json/v2` est expérimental (`GOEXPERIMENT=jsonv2`).
- **Pagination** : les pages citées sont les folios imprimés tels qu'ils figurent dans la table des matières du livre (ex. « ch. 18, p. 469 »). L'extraction texte du PDF présente un décalage variable (de +3 à +10 pages) entre l'index PDF et le folio ; les références ci-dessous ont été alignées sur la table des matières.
- **Marqueurs épistémiques** : *Confirmé* = présent dans le livre ou vérifié dans une source primaire ; *Probable* = inférence directe ; *Hypothèse* = proposition à éprouver ; *À vérifier* = non vérifiable ici.

## 3. Points de vigilance relevés dans le livre

Ces écarts méritent d'être connus avant de bâtir un protocole sur le texte.

| Affirmation du livre | État | Source primaire |
|---|---|---|
| Green Tea GC activable par `GODEBUG=gc=greentea` (ch. 8, p. 245) | *À vérifier* — les notes de version 1.25 indiquent `GOEXPERIMENT=greenteagc` au *build* | go.dev/doc/go1.25 |
| `encoding/json/v2` présenté comme fonctionnalité de la bibliothèque standard (ch. 10, p. 304) | *Confirmé partiellement* — expérimental en 1.25, derrière `GOEXPERIMENT=jsonv2` | go.dev/doc/go1.25 |
| « the new `os.Environ()` iterator » (ch. 12, p. 340) | *À vérifier* — aucune trace d'un itérateur `os.Environ` dans les notes 1.25 consultées | go.dev/doc/go1.25 |
| Client Pulsar Go v0.18.0 « as of late 2026 » (ch. 18, p. 469) | *Confirmé* — annonce de release 0.18.0 sur la liste `users@pulsar.apache.org` | mail-archive.com |
| Layout hexagonal du ch. 14 mentionnant Kafka (`adapters/kafka`, `schema/`) alors que le projet utilise Pulsar (ch. 10, p. 305) | *Confirmé* — incohérence interne, sans conséquence sur les projets | PDF |

## 4. Cartographie des affirmations réfutables du livre

Le tableau retient les affirmations formulées de façon assez précise pour être mesurées ou contredites. Chaque ligne alimente au moins un projet de la section 6.

| Ch. | Section (p.) | Affirmation (paraphrase fidèle) | Projet |
|---|---|---|---|
| 5 | Slices (112–114) | Préallocation d'un slice : « about 6× » plus rapide, « one-fifth the memory », 28 allocs → 1 | P1 |
| 5 | Strings / Quick Practice (89–91) | `uint16` vs `int64` : 6 octets de plus par enregistrement, « gigabytes » sur des millions | P1 |
| 8 | Escape Analysis (238–242) | Quatre causes d'échappement : retour de pointeur, capture par closure, pointeur envoyé sur un canal, pointeur stocké dans map/slice/struct | P1 |
| 8 | Heap (245) ; Choosing Between Values and Pointers (252–253) | Valeur si 1–3 mots machine, court-vécu, non partagé ; « copying a small struct can be cheaper than passing a pointer » | P1 |
| 8 | The Reality of Stack vs. Heap (254) | Cache hit < 10 ns, miss > 100 ns, « roughly 10× to 200× » | P1 |
| 8 | Profiling Memory with pprof (255–256) | Struct par valeur vs pointeur : jusqu'à « double the number of allocations » | P1 |
| 9 | Goroutines vs. threads (261) ; What Happens When You Use go (266) | Pile initiale « usually about 2 KB » ; création « almost as cheap as calling a normal function » | P2 |
| 9 | Goroutine Overhead (284–285) | « Tens of thousands of goroutines consume significant memory » ; sur 8 cœurs, des dizaines de milliers de goroutines exécutables sont « so expensive to schedule » | P2 |
| 9 | Timeout Handling with time.After (276) | `time.After` en boucle ⇒ « small but steady memory growth » | P2, P3 |
| 9 | Deterministic Concurrency Tests (289–292) | `synctest` : un timeout de 500 ms se teste « instantly » ; panic si une goroutine reste bloquée en fin de bulle | P3 |
| 9 | Monitoring Goroutines (292–293) | `runtime.NumGoroutine()` : stable = sain, hausse lente = fuite, pics = concurrence non bornée | P2, P3 |
| 7 | Speed and Determinism (200) ; Flaky Test Prevention (225) | « A flaky test is worse than no test at all » | P3, S1 |
| 7 | Running with the Race Detector (231) | `-race` ralentit les tests ; obligatoire en CI | P3 |
| 20 | Worker Pool (555–556) ; Anti-patterns (560–569) | Le pool « consumes far less memory » ; fuites : « Tests may pass » ; `cancel()` oublié ⇒ mémoire, goroutines, overhead d'ordonnancement ; « the overhead of scheduling dominates the actual work » | P2, P3 |
| 4 | Circular Dependencies (69) ; Best Practices (70) | Le compilateur refuse les cycles ; éviter `pkg/` « dumping ground » | P4 |
| 10 | Why “Edge API” Modules Are a Trap (307–308) | Tests brittle, angles morts d'observabilité quand l'API est séparée du domaine | P4 |
| 14 | Rules to Keep Your Architecture Clean (369) ; Architecture As Leverage (372) | Dépendance adapters → core uniquement ; changer de base = un nouvel adapter, « service layer and the models remain the same » | P4 |
| 14 | Testing Strategies Across Hexagonal Layers (373–375) | Tests de service sans vraie base ni broker ; tests d'adapters « fewer, slower, and more expensive » | P4, S1 |
| 20 | Small Interfaces, Big Wins (514–517) | Interface à implémentation unique = « ceremony without adding value » | P4 |
| 15 | Struct Tags and JSON (390–398) | Une faute de frappe de tag change silencieusement la clé JSON ; `omitempty` sur un bool supprime `false` ; champs inconnus ignorés ⇒ *over-posting* | P5 |
| 15 | Forward/Backward Compatibility (399) ; Versioning (400) | Changements additifs seulement ; le versioning par query interagit mal avec les caches | P5 |
| 18 | Avro Schema Versioning (456–458) | Défaut obligatoire sur tout nouveau champ ; BACKWARD ⇒ consommateurs d'abord, FORWARD ⇒ producteurs d'abord ; schéma rejeté ⇒ `CreateProducer`/`Subscribe` échouent immédiatement | P5 |
| 19 | Why Protobuf Improves Performance (486) ; Defining Protobuf Contracts (490–493) | Protobuf « much smaller than JSON » ; ne jamais réutiliser un tag ; `reserved` | P5, S2 |
| 18 | Performance Tuning Guide by Design Goal (469–477) | « either low latency or high throughput (not both) » ; batching off « dramatically increases per-message overhead » ; `ReceiverQueueSize` 5000+ ↑ débit au prix de la mémoire ; Shared/Key_Shared « greatly increasing end-to-end throughput » ; `MaxConnectionsPerBroker` > 1 « may improve both » ; baisser `MemoryLimitBytes` « caps throughput » | P6 |
| 13 | Timeouts (347–348) ; Why Graceful Shutdown Matters (360) | Plages : read 5–10 s, write 10–15 s, idle 1–2 min ; sans `ReadTimeout`, Slowloris ; arrêt brutal ⇒ « connection reset or 502 » pendant un rollout | P7 |
| 19 | Resilience and Error Handling (506–507) | Max 5 tentatives avec backoff ; trop de retries « can overload already unhealthy services » ; disjoncteur ≥ 10 requêtes et ≥ 50 % d'échec | P7 |
| 1 | Team Topologies (22–23) | Boucle de retry devenue « a denial-of-service attack » ; « Wouldn't a circuit breaker have prevented all of this? » | P7 |
| 16 | Connection Pooling (416–418) | Pool trop petit ⇒ latence ; trop grand ⇒ base goulot ; « not an optimization… a correctness and stability requirement » | P8 |
| 16 | The Hidden Costs (422–423) ; Performance Tips (426–428) | SQL généré par ORM « harder than writing the queries directly » ; « nearly 80% of performance issues are rooted in design » ; index multi-colonnes, cardinalité la plus haute en premier ⇒ gain « dramatic » | P8 |
| 17 | One Container, Many Tests (433) ; Run Parallel (443) ; CI (443–445) | Démarrage conteneur « several seconds » ; `-p` réduit « dramatically » le temps CI ; snapshots ⇒ reset « instant » | S1, P8 |

## 5. Grille d'évaluation

Cinq critères notés de 1 à 5. *Réfutabilité* : nombre et précision des affirmations du livre mises à l'épreuve. *Centralité* : poids du concept dans le livre. *Infra légère* : 5 = `go test` seul, 1 = broker + base + orchestrateur. *Adéquation Claude Code* : présence d'une boucle « modifier → mesurer → comparer » automatisable et d'une matrice de cas à générer. *Réutilisation* : le projet laisse un outil ou un banc réutilisable au-delà du livre.

| Projet | Réfutabilité | Centralité | Infra légère | Adéquation CC | Réutilisation | Total |
|---|---|---|---|---|---|---|
| P1 EscapeBench | 5 | 3 | 5 | 5 | 4 | **22** |
| P2 GoroutineCost | 4 | 4 | 3 | 4 | 3 | 18 |
| P3 LeakLab | 4 | 4 | 5 | 5 | 5 | **23** |
| P4 HexaGuard | 4 | 5 | 4 | 5 | 5 | **23** |
| P5 ContractEvo | 4 | 5 | 3 | 4 | 4 | 20 |
| P6 PulsarTune | 5 | 4 | 2 | 3 | 3 | 17 |
| P7 EdgeResilience | 4 | 4 | 3 | 4 | 3 | 18 |
| P8 PersistEcon | 4 | 4 | 3 | 4 | 3 | 18 |

Les scores de P1, P3 et P4 sont indiscernables ; l'ordre P1 → P3 → P4 est dicté par la vitesse de la boucle de rétroaction (P1 se mesure en secondes) et par la dépendance de P3 sur le vocabulaire de P1 (allocations, pression GC).

## 6. Fiches des projets candidats

Chaque fiche suit le même plan : concepts du livre, question de recherche, affirmations à éprouver, protocole, livrables, rôle de Claude Code, compromis principal, alternative, conditions de renversement, effort (hypothèse).

### P1 — EscapeBench : frontière valeur/pointeur et analyse d'échappement

**Concepts du livre.** Ch. 5 (Pointers, p. 101–106 ; Slices, p. 112), ch. 6 (Structs, p. 135 ; exercice « slice of 1,000,000 large structs by value vs. by pointer », p. 177), ch. 7 (Benchmarks, p. 219 ; Understanding the Numbers, p. 232), ch. 8 en entier (Escape Analysis, p. 238 ; Choosing Between Values and Pointers, p. 252 ; pprof, p. 255).

**Question de recherche.** À partir de quelle taille de struct, et sous quelles conditions d'échappement, le passage par pointeur devient-il moins coûteux que la copie par valeur en Go 1.25, et la règle « 1–3 mots machine » du livre prédit-elle correctement ce point de bascule sur amd64 et arm64 ?

**Affirmations à éprouver.** (a) Copier une petite struct peut être moins cher que passer un pointeur (p. 245, 253). (b) Le seuil valeur/pointeur se situe autour de 1–3 mots machine (p. 253). (c) Le passage par pointeur peut doubler le nombre d'allocations (p. 255–256). (d) Cache hit/miss : rapport de 10× à 200× (p. 254). (e) Préallocation d'un slice : ≈ 6× plus rapide, un cinquième de la mémoire (p. 114). (f) Les quatre causes d'échappement listées p. 238–242 sont exhaustives pour du code applicatif ordinaire — *Hypothèse* : le livre ne prétend pas à l'exhaustivité ; c'est la question intéressante.

**Protocole.** Générer une matrice de types `struct` de 8 à 4 096 octets (par pas géométriques), avec et sans champ pointeur (le livre signale le *padding* introduit par un pointeur, p. 250–252), et trois profils de vie : locale pure, retournée, stockée dans une map. Pour chaque cellule : `go build -gcflags=-m` parsé pour classer l'échappement ; `go test -bench -benchmem -count=20` pour `ns/op`, `B/op`, `allocs/op` ; profil `-memprofile` pour vérifier (c). Comparaison statistique des paires valeur/pointeur (`benchstat`, **hors livre**). Critère de réfutation de (b) : point de bascule observé hors de l'intervalle [1, 3] mots sur au moins une architecture. Critère pour (d) : mesurer l'accès séquentiel vs aléatoire sur une slice de structs et un ensemble de pointeurs dispersés ; ratio hors [10, 200] réfute la borne.

**Livrables.** Module Go `escapebench` (générateur de matrice + banc), rapport de résultats par architecture et par version de Go (1.24, 1.25, et la version courante), catalogue des motifs d'échappement observés comparé à la liste p. 238–242.

**Rôle de Claude Code.** Génération de la matrice de types et des benchmarks table-driven (ch. 7, p. 213) ; parseur de la sortie `-gcflags=-m` ; boucle agentique « générer une cellule → exécuter → classer → consigner » avec un `CLAUDE.md` fixant les invariants (ne jamais modifier le harnais pendant une campagne, `-count` minimal, seuils statistiques). Sous-agents pour paralléliser les campagnes par architecture (conteneurs `arm64` via QEMU ou machine distante).

**Compromis principal.** Micro-benchmarks : précision élevée, validité externe faible. Le livre lui-même prévient que l'analyse d'échappement « varies between compiler versions » (p. 242) : les résultats sont datés par construction.

**Alternative.** Mesurer sur le projet Transport du livre (ride service) avec pprof en charge, plutôt que sur des types synthétiques : validité externe supérieure, contrôle des variables inférieur.

**Conditions de renversement.** Si l'objectif est un résultat transposable à une base de code réelle, préférer l'alternative ; si Go 1.26+ modifie l'analyse d'échappement (à surveiller dans les notes de version), la campagne doit être rejouée avant toute conclusion.

**Effort (hypothèse).** 3–5 jours avec Claude Code, sans infrastructure.

### P2 — GoroutineCost : coût réel des goroutines, worker pools et `GOMAXPROCS` conscient des conteneurs

**Concepts du livre.** Ch. 2 (Concurrency: Goroutines vs. threads, p. 41 ; Memory Model and GC, p. 44), ch. 9 (The Go Scheduler, p. 261 ; Goroutine Overhead, p. 284 ; errgroup, p. 287 ; Monitoring Goroutines, p. 292), ch. 20 (Fan-Out/Fan-In, p. 550 ; Worker Pool, p. 555 ; sur-utilisation des goroutines, p. 569).

**Question de recherche.** Quelle est la fonction de coût (mémoire résidente, latence p99, débit, temps GC) d'une goroutine par tâche par rapport à un pool de *W* workers, en fonction de la durée de tâche et de la limite CPU cgroup, et à partir de quel rapport tâches/cœurs « the overhead of scheduling dominates the actual work » (p. 569) ?

**Affirmations à éprouver.** (a) Pile initiale ≈ 2 KB (p. 243, 266). (b) Des dizaines de milliers de goroutines exécutables sur 8 cœurs sont « so expensive to schedule » (p. 285). (c) Le worker pool « consumes far less memory » et réduit la pression GC (p. 555–556). (d) Go 1.25 aligne `GOMAXPROCS` sur la limite cgroup et s'adapte si elle change (p. 260–262 ; *Confirmé* par go.dev/doc/go1.25). (e) `GOMAXPROCS` trop haut ⇒ contention et latence instable (p. 262). (f) `time.After` en boucle ⇒ croissance mémoire régulière (p. 276).

**Protocole.** Charge synthétique paramétrée par durée de tâche (1 µs → 10 ms), nature (CPU vs I/O simulée), nombre de tâches (10³ → 10⁶) ; deux exécuteurs : `go` par tâche avec `errgroup` (limite optionnelle) et `StartPool(ctx, workerCount)` du ch. 20. Conteneur Docker avec `--cpus` variable (0.5, 1, 2, 4, 8) et modification à chaud de la limite pour (d). Métriques : `runtime.MemStats`, `runtime.NumGoroutine()`, `GODEBUG=gctrace=1`, `pprof` goroutine et heap, latence par tâche. Critère de réfutation de (c) : à tâches longues et peu nombreuses, le pool ne devrait pas dominer ; trouver la frontière où l'écart mémoire descend sous 10 %.

**Livrables.** Banc `goroutinecost`, surface de coût (heatmaps durée × cardinalité), note sur le comportement observé de `GOMAXPROCS` lors du changement de limite cgroup (délai d'adaptation).

**Rôle de Claude Code.** Génération des exécuteurs à partir des listings du ch. 20 ; script Docker de sweep ; collecte et agrégation ; commande `/sweep` dédiée ; hooks pour exécuter `-race` sur les exécuteurs avant chaque campagne.

**Compromis principal.** La charge synthétique isole les variables mais ignore les effets d'un vrai broker ou d'une base (les tâches réelles bloquent sur le réseau, ce qui change l'ordonnancement).

**Alternative.** Instrumenter le consommateur Pulsar du ch. 18 avec les deux exécuteurs et mesurer en charge réelle — recoupe P6.

**Conditions de renversement.** Sans Docker avec cgroups v2 accessible (hôte Windows 11 avec WSL2 : à vérifier que la limite `--cpus` est honorée), abandonner (d) et (e) et réduire P2 à (a)–(c), ce qui ramène l'infrastructure à `go test`.

**Effort (hypothèse).** 5–8 jours.

### P3 — LeakLab : détectabilité des anti-patrons de concurrence

**Concepts du livre.** Ch. 7 (Parallel Tests et suivi des goroutines fuitées en Go 1.25, p. 216–217 ; Testing Concurrency Safely, p. 227 ; Race Detector, p. 231), ch. 9 (Deadlocks and Livelocks, p. 272 ; `testing/synctest`, p. 289 ; Monitoring Goroutines, p. 292), ch. 20 (Context Pattern, p. 539 ; Anti-patterns : Goroutine Leaks p. 560, Channel Deadlocks p. 563, Forgetting to Cancel a Context p. 566, Ignoring Context in I/O p. 568).

**Question de recherche.** Pour chacun des quatre anti-patrons du ch. 20 (et leurs variantes), quel mécanisme de détection — bulle `synctest`, `-race`, surveillance de `runtime.NumGoroutine()`, analyse statique — le révèle, avec quel taux de faux négatifs et à quel coût en temps d'exécution ?

**Affirmations à éprouver.** (a) `synctest` fait paniquer une bulle dont une goroutine reste bloquée (p. 291–292). (b) Un timeout de 500 ms se teste « instantly » (p. 291). (c) Pour une fuite de goroutine, « Tests may pass » (p. 561) — donc les tests ordinaires sont aveugles. (d) Le cadre de test 1.25 signale les goroutines fuitées (p. 217 ; *À vérifier* : le périmètre exact de ce suivi n'est pas précisé par le livre). (e) `-race` détecte les courses, pas les fuites (implicite, p. 231 ; *Probable*). (f) Un `cancel()` oublié maintient timers et goroutines vivants (p. 566).

**Protocole.** Constituer un corpus de 20 à 30 programmes minimaux : les quatre anti-patrons du livre, leurs corrections (p. 561–569), plus des variantes mutées (buffer ajouté « à l'aveugle », p. 267 ; `time.After` en boucle, p. 276 ; `select` sans `default` dans un `Publish`, p. 534–536). Exécuter chaque programme sous cinq détecteurs : test nu, `-race`, `synctest.Test`, sonde `NumGoroutine()` avant/après, et un analyseur `go/analysis` (**hors livre**) codant deux règles dérivées du ch. 20 : « `context.With*` sans `defer cancel()` » et « appel d'I/O sans `ctx` alors qu'un `ctx` est dans la portée ». Matrice détecteur × anti-patron ; mesure du temps ajouté par détecteur. Critère de réfutation de (a) : une fuite non signalée par la bulle.

**Livrables.** Corpus versionné (chaque cas = un `_test.go` documenté), analyseur `go vet`-compatible, matrice de détectabilité, recommandation de pipeline CI (ordre et coût des détecteurs).

**Rôle de Claude Code.** Génération du corpus par sous-agents indépendants (un sous-agent par anti-patron, chacun produisant cas fautif + correction + test), écriture de l'analyseur, boucle « muter → exécuter les cinq détecteurs → consigner ». Hook post-édition exécutant `go vet` avec l'analyseur maison sur tout le dépôt.

**Compromis principal.** Un corpus synthétique surestime la détectabilité : les fuites réelles sont enfouies dans des chemins d'erreur rarement exercés. En contrepartie, le corpus est le seul moyen d'obtenir une vérité terrain.

**Alternative.** Appliquer les détecteurs au projet Transport complet (chapitres 13 à 19) après y avoir injecté des fautes connues : validité externe supérieure, coût de mise en place plus élevé.

**Conditions de renversement.** Si `goleak` (**hors livre**) couvre déjà la matrice avec un taux de faux négatifs nul, l'analyseur maison perd son intérêt et P3 se réduit à une comparaison `synctest` vs `goleak`.

**Effort (hypothèse).** 4–6 jours.

### P4 — HexaGuard : règle de dépendance hexagonale exécutable et coût du changement

**Concepts du livre.** Ch. 4 (Suggested Project Layout, p. 68 ; Circular Dependencies, p. 69 ; Best Practices, p. 70), ch. 10 (Why “Edge API” Modules Are a Trap, p. 307), ch. 14 en entier (Structure in Go, p. 368 ; Rules, p. 369 ; Architecture As Leverage, p. 372 ; Testing Strategies, p. 373 ; exercice 4 « Swap between the two implementations at runtime—Does the service change? », p. 375), ch. 15 (Folder Structure, p. 383 ; Converters, p. 386), ch. 16 (Databases in the Hexagon, p. 406 ; composition root, p. 414), ch. 20 (Small Interfaces, p. 514).

**Question de recherche.** Les règles d'architecture du ch. 14 peuvent-elles être exprimées comme des invariants vérifiables sur le graphe d'imports, et une base de code qui les respecte présente-t-elle un coût de changement (fichiers et paquets touchés, lignes modifiées, tests à réécrire) mesurablement inférieur lors d'un remplacement d'adapter ?

**Affirmations à éprouver.** (a) Dépendance adapters → core uniquement (p. 367, 369). (b) `internal/models` n'importe que la bibliothèque standard (p. 366, 369). (c) `internal/service` n'importe ni HTTP, ni SQL, ni broker (p. 367). (d) Les DTO n'existent que dans les adapters ; les modèles n'ont pas de tags JSON (p. 369 ; ch. 15, p. 383). (e) Changer de base = un nouvel adapter, service et modèles inchangés (p. 372). (f) Une interface à implémentation unique est de la « ceremony » (ch. 20, p. 517) — en tension apparente avec (a)–(c), qui imposent des ports même à implémentation unique ; c'est la contradiction à instruire.

**Protocole.** Reconstruire le ride service selon le layout du ch. 14 (en s'appuyant sur les répertoires `05-rest-api` et `06-database` du dépôt du livre). Écrire `hexaguard`, outil fondé sur `go list -json ./...` (ch. 4, p. 71) qui vérifie (a)–(d) et signale les interfaces à implémentation unique (f). Puis trois expériences de changement, chacune exécutée sur la version conforme et sur une version « edge API » volontairement dégradée (ch. 10, p. 307) : remplacement de l'adapter MySQL par un adapter en mémoire (exercice p. 375), remplacement de Gin par `net/http` pur, ajout d'un champ `driverId` (ch. 15, exercice 2, p. 403). Métriques : `git diff --stat` par couche, nombre de tests modifiés, temps de la suite unitaire vs intégration (p. 374).

**Livrables.** Squelette de référence conforme au ch. 14, outil `hexaguard` (utilisable en CI et comme hook Claude Code), tableau comparatif conforme vs dégradé, note sur la tension (f) vs (a)–(c) avec une règle de décision.

**Rôle de Claude Code.** Scaffolding du squelette à partir des listings des ch. 14–16 ; écriture de `hexaguard` ; exécution des trois changements en boucle agentique avec mesure automatique du diff ; `hexaguard` branché en hook pour empêcher toute violation pendant les changements — ce qui teste en passant si un agent respecte mieux l'architecture avec un garde-fou exécutable qu'avec un `CLAUDE.md` seul (*Hypothèse* secondaire, mesurable par le nombre de violations bloquées).

**Compromis principal.** Le coût de changement mesuré sur trois scénarios choisis par l'expérimentateur est sensible au choix des scénarios ; la conclusion vaut pour la classe « remplacement d'adapter », pas pour les changements de domaine.

**Alternative.** Utiliser un outil de règles d'architecture existant (ex. `go-arch-lint`, **hors livre**) plutôt que d'écrire `hexaguard` ; moins de contrôle sur la sémantique des règles (b) et (d).

**Conditions de renversement.** Si le squelette de référence dérive du dépôt du livre au point de ne plus être comparable, la mesure de coût de changement perd son ancrage ; dans ce cas, réduire P4 à l'outil de conformité.

**Effort (hypothèse).** 6–10 jours.

### P5 — ContractEvo : évolution de contrats OpenAPI, Protobuf et Avro sous scénarios identiques

**Concepts du livre.** Ch. 15 (OpenAPI As Executable Contracts, p. 378 ; Generator Configs, p. 384 ; Struct Tags and JSON, p. 390 ; Forward/Backward Compatibility, p. 399 ; Versioning Strategies, p. 400 ; Error Handling, p. 401), ch. 18 (Avro Schema, p. 455 ; Avro Schema Versioning, p. 456 ; Messaging Ports génériques, p. 462), ch. 19 (Defining Protobuf Contracts, p. 490 ; Generating Go Code, p. 493 ; exercices 3 et 4, p. 510).

**Question de recherche.** Pour un même catalogue de mutations de contrat (ajout de champ optionnel, ajout sans défaut, renommage, suppression avec et sans `reserved`, changement de type, ajout de valeur d'énumération), à quel moment chaque technologie du livre — OpenAPI + `oapi-codegen`, Protobuf + `protoc-gen-go`, Avro + `avrogen` + Pulsar — révèle la rupture : compilation, enregistrement du schéma, ou exécution ?

**Affirmations à éprouver.** (a) Un tag JSON mal orthographié change silencieusement la clé (p. 398). (b) `omitempty` sur un bool fait disparaître `false` (p. 397). (c) Un nouveau champ Avro sans défaut casse la compatibilité (p. 456–458). (d) Sous BACKWARD, déployer les consommateurs d'abord ; sous FORWARD, les producteurs d'abord ; le défaut Pulsar est FULL (p. 458). (e) Un schéma rejeté fait échouer `CreateProducer`/`Subscribe` immédiatement (p. 456). (f) Retirer un champ Protobuf sans `reserved` puis réutiliser le tag corrompt silencieusement les données (p. 492 ; exercice 4, p. 510). (g) Protobuf « much smaller than JSON » (p. 486) — quantifier sur les messages du projet Transport.

**Protocole.** Définir le message `Assignment` dans les trois IDL. Pour chaque mutation du catalogue (≈ 12), générer le code, compiler producteur et consommateur aux versions N et N+1, exécuter les quatre combinaisons de déploiement (P_N/C_N, P_N+1/C_N, P_N/C_N+1, P_N+1/C_N+1) ; pour Avro, répéter sous les stratégies BACKWARD, FORWARD, FULL d'un Pulsar 4.0.x en conteneur (`testcontainers-go`, ch. 17). Consigner le point de détection (compile / enregistrement / runtime / silencieux) et, pour (g), la taille sérialisée. Critère de réfutation de (d) : une combinaison prédite compatible qui échoue, ou l'inverse.

**Livrables.** Matrice mutation × technologie × point de détection, harnais rejouable, note de synthèse sur les mutations « silencieuses » (celles qu'aucune couche ne détecte).

**Rôle de Claude Code.** Génération des trois définitions à partir de la spécification OpenAPI du ch. 15 (p. 379–381) ; application mécanique du catalogue de mutations ; orchestration des quatre combinaisons de déploiement par scénario ; sous-agent par technologie.

**Compromis principal.** Trois chaînes d'outils et un broker : la surface de configuration est large, et une partie des résultats dépend des versions des générateurs (à figer dans `go.mod` et à consigner).

**Alternative.** Se limiter à OpenAPI et Protobuf (sans broker), ce qui ramène l'infrastructure à `go test` mais perd la question (d), la plus originale.

**Conditions de renversement.** Si le sujet visé est l'interopérabilité agentique plutôt que l'entreprise classique, le catalogue de mutations reste valable mais les IDL cibles changent — à traiter dans un autre dépôt, hors périmètre de ce document.

**Effort (hypothèse).** 6–10 jours.

### P6 — PulsarTune : le guide de tuning par objectif de conception comme expérience réfutable

**Concepts du livre.** Ch. 18 (Pulsar Producers and Consumers, p. 461 ; Reusable Generic Producer/Consumer, p. 465–467 ; DLQ, p. 469 ; Performance Tuning Guide : Low Latency p. 470, High Throughput p. 472, Failover p. 474, Memory p. 476 ; Quality Control et Monitoring, p. 477–478), ch. 10 (Pulsar 4.0.8, p. 305).

**Question de recherche.** Les quatre jeux de paramètres recommandés par le livre (latence, débit, résilience, mémoire) déplacent-ils les métriques annoncées (p50/p99 de bout en bout, messages/s, RSS du client, temps de reprise) dans le sens et l'ordre de grandeur annoncés, et le compromis « latence ou débit, pas les deux » (p. 471) est-il une frontière de Pareto ou un artefact des valeurs par défaut ?

**Affirmations à éprouver.** (a) Désactiver le batching sacrifie le débit et « dramatically increases per-message overhead » (p. 471, 476). (b) `BatchingMaxPublishDelay` à 50 ms ajoute jusqu'à 50 ms par lot (p. 472). (c) LZ4/ZSTD élèvent le débit pour les grands payloads (p. 472). (d) `ReceiverQueueSize` ≥ 5 000 élève le débit en rafales au prix de la mémoire (p. 473, 476–477). (e) Shared/Key_Shared « greatly increasing end-to-end throughput » (p. 473). (f) `MaxConnectionsPerBroker` > 1 « may improve both throughput and latency » (p. 471, 473). (g) Baisser `MemoryLimitBytes` plafonne le débit (p. 477). (h) `NackRedeliveryDelay` d'une minute dégrade la latence de reprise (p. 471).

**Protocole.** Producteur et consommateur génériques du livre (ports `EventProducer[T]`, `EventConsumer[T]`, p. 462–467) sur Pulsar 4.0.x en conteneur, client Go v0.18.0. Plan factoriel fractionnaire sur huit paramètres, trois tailles de payload (100 B, 10 KB, 1 MB), deux profils de charge (constante, rafales). Métriques client (durée, acks, nacks, DLQ) exposées sur `/metrics` comme le prescrit p. 478, plus `msgRateIn`/`msgThroughputIn` côté broker (p. 473). Chaque cellule répétée 5 fois, broker redémarré entre cellules. Critère de réfutation de (a) : un écart de débit < 20 % entre batching on/off à payload 10 KB.

**Livrables.** Banc `pulsartune`, frontière latence/débit mesurée, tableau « affirmation → confirmée / infirmée / non concluante », jeu de paramètres recommandé par objectif tel qu'observé.

**Rôle de Claude Code.** Génération du harnais à partir des listings p. 465–467, génération du plan factoriel, exécution non surveillée (mode *headless* `claude -p` planifié), agrégation et tracés. Le `CLAUDE.md` doit interdire toute modification du harnais pendant une campagne et imposer la consignation des versions (broker, client, image).

**Compromis principal.** Coût d'infrastructure et durée des campagnes (des heures) ; la validité dépend fortement de la machine hôte (un broker en conteneur sur portable ne reproduit pas un cluster). C'est le projet le plus proche d'une étude publiable et le plus coûteux.

**Alternative.** Ne tester que (a), (b) et (e) sur un seul payload : un jour de mesure, la moitié de la valeur.

**Conditions de renversement.** Si la machine hôte ne peut dédier au moins 4 cœurs au broker sans contention, les mesures de latence p99 ne sont pas interprétables ; reporter P6 ou l'exécuter sur une machine dédiée.

**Effort (hypothèse).** 8–12 jours, campagnes comprises.

### P7 — EdgeResilience : timeouts, arrêt gracieux, retries et disjoncteur sous clients adverses

**Concepts du livre.** Ch. 1 (retry loop devenue DoS, p. 23), ch. 13 (Timeouts, p. 347 ; Run with Graceful Shutdown, p. 355 ; Why Graceful Shutdown Matters, p. 360 ; exercices 2 et 4, p. 361), ch. 19 (Running the gRPC Server et `GracefulStop`, p. 499 ; Resilience and Error Handling, p. 506 ; Client-Side Load Balancing, p. 508), ch. 20 (Graceful Shutdown, p. 537 ; Context Pattern, p. 539).

**Question de recherche.** Sous un client lent (Slowloris), un rollout continu et une panne partielle du service aval, quelle combinaison de timeouts serveur, de grâce à l'arrêt, de politique de retry et de seuils de disjoncteur minimise le taux d'erreurs vu par le client sans amplifier la panne, et les valeurs indicatives du livre sont-elles proches de l'optimum observé ?

**Affirmations à éprouver.** (a) `ReadTimeout` = 1 s ferme la connexion d'un `curl --limit-rate 10` (p. 361). (b) Plages : read 5–10 s, write 10–15 s, idle 1–2 min (p. 347–348). (c) Sans arrêt gracieux, un rollout produit « connection reset or 502 » (p. 360) ; avec, zéro erreur pour une grâce ≥ durée max des requêtes (*Probable*, implicite p. 359). (d) Plus de 5 tentatives en panne longue surcharge le service (p. 507). (e) Disjoncteur ≥ 10 requêtes et ≥ 50 % d'échec évite l'amplification (p. 506–507). (f) Une boucle de retry sans disjoncteur se comporte comme un DoS (ch. 1, p. 23).

**Protocole.** Ride service (HTTP) appelant Vehicle service (gRPC) ; générateur de charge maison en Go (le livre n'en prescrit aucun) ; trois scénarios : 500 clients Slowloris à 10 B/s ; rollout de 3 réplicas sous Docker Compose avec SIGTERM et grâces de 0, 1, 3, 10 s ; panne de Vehicle (latence 5 s puis refus) avec retries 0/3/5/10 et disjoncteur on/off. Métriques : taux d'erreurs client, descripteurs de fichiers et goroutines serveur (`/debug/pprof/goroutine`, ch. 9, p. 293), requêtes reçues par Vehicle pendant la panne (facteur d'amplification). Critère de réfutation de (e) : facteur d'amplification > 2 malgré le disjoncteur.

**Livrables.** Deux services conformes aux ch. 13 et 19, générateur de charge, courbes erreurs vs grâce et amplification vs retries, recommandation chiffrée.

**Rôle de Claude Code.** Scaffolding des deux services ; écriture du générateur de charge ; exécution des scénarios en boucle avec variation des paramètres ; rédaction du rapport à partir des mesures.

**Compromis principal.** Docker Compose n'est pas Kubernetes : l'ordre SIGTERM → retrait du load balancer diffère, ce qui touche directement (c). Les résultats valent pour un proxy simple, pas pour un ingress.

**Alternative.** Un cluster `kind` local pour (c) uniquement — plus fidèle, plus lourd.

**Conditions de renversement.** Si la question d'intérêt est l'amplification de panne (d)–(f) plutôt que le rollout, Compose suffit et `kind` est superflu.

**Effort (hypothèse).** 6–9 jours.

### P8 — PersistEcon : ORM vs `database/sql`, dimensionnement du pool, index

**Concepts du livre.** Ch. 16 (A Note on Transactions, p. 408 ; Connection Pooling, p. 416 ; ORMs in Go, p. 419 ; Hidden Costs, p. 422 ; My Recommendation, p. 423 ; Database Performance Tips, p. 426 ; exercice 3 sur les affectations chevauchantes, p. 428), ch. 17 (One Container, Many Tests, p. 433 ; Run Parallel, p. 443).

**Question de recherche.** Sur le schéma `assignments` du livre, quelle est la sensibilité de la latence p99 et du débit à `SetMaxOpenConns`/`SetMaxIdleConns`, et l'écart de performance et de maintenabilité entre GORM et `database/sql` que le livre décrit (p. 422–423) se retrouve-t-il sur des requêtes représentatives (upsert, liste par statut, détection de chevauchement) ?

**Affirmations à éprouver.** (a) Pool trop petit ⇒ latence, trop grand ⇒ base goulot (p. 416–418) : existence d'un optimum intérieur. (b) Reconnexions répétées coûtent plus que des connexions oisives (p. 418). (c) SQL généré par ORM plus difficile à optimiser (p. 423), N+1 invisibles à petite échelle (p. 422). (d) Index multi-colonnes : cardinalité la plus haute en premier ⇒ gain « dramatic » (p. 428). (e) `rows affected` = 1 pour insert, 2 pour update sous `ON DUPLICATE KEY UPDATE` (p. 408). (f) Contrainte de non-chevauchement : en Go vs en SQL (exercice 3, p. 428) — comparer correctness sous concurrence et coût.

**Protocole.** MySQL 8.0.44 en conteneur partagé (`sync.Once`, ch. 17) ; jeu de données 10⁶ affectations ; sweep `MaxOpenConns` ∈ {2, 4, 8, 16, 32, 64, 128} × `MaxIdleConns` ∈ {0, 2, 8, MaxOpen} sous 64 clients concurrents ; deux implémentations de `AssignmentRepository` (GORM, `database/sql`) derrière le même port (ch. 14) ; `EXPLAIN` avant/après réordonnancement d'index pour (d) ; test de course sur (f) avec 100 goroutines créant des affectations chevauchantes.

**Livrables.** Deux adapters conformes au port du ch. 16, banc de sweep, surface latence/débit, comparatif ORM vs SQL (plans, lignes de code, requêtes émises), verdict sur (f).

**Rôle de Claude Code.** Génération des deux adapters et des migrations (`V001__create_assignments.sql`, p. 425) ; sweep et collecte ; analyse des plans `EXPLAIN` ; rédaction.

**Compromis principal.** Un MySQL en conteneur sur la même machine que le client mesure aussi la contention CPU locale ; l'optimum de pool observé n'est pas transposable tel quel.

**Alternative.** Base distante (VM séparée) pour découpler client et serveur ; plus fidèle, plus de variables non contrôlées (réseau).

**Conditions de renversement.** Si l'intérêt principal est (c), un jeu de requêtes représentatif suffit et le sweep de pool devient secondaire.

**Effort (hypothèse).** 5–8 jours.

## 7. Idées secondaires (non développées)

- **S1 — Économie des tests d'intégration** (ch. 17) : mesurer temps total et taux de *flakiness* sur 30 exécutions pour conteneur par test vs partagé, avec/sans snapshots, `-p` ∈ {1, 2, 4, 8}. Absorbable dans P8.
- **S2 — gRPC vs REST pour les appels internes** (ch. 19, p. 486–489) : taille sérialisée, p99, sockets ouverts, *streaming* vs *polling* pour les mises à jour d'affectation. Absorbable dans P5 (g) et P7.
- **S3 — MVS et workspaces** (ch. 11, p. 322) : vérifier expérimentalement la résolution MVS sur un conflit `libA v1.2`/`v1.4` entre modules du workspace, et le coût CI d'un monorepo à mesure que les modules s'ajoutent (p. 314). Valeur pédagogique surtout.
- **S4 — Porte de régression de benchmarks** (ch. 7, p. 218–219 : « CI must treat regressions as warnings or failures ») : concevoir le seuil statistique qui minimise les faux positifs sur une machine bruitée. Requiert `benchstat` (**hors livre**) ; peut devenir l'outillage commun de P1, P2, P6 et P8.

## 8. Séquence recommandée et conditions de renversement

**Recommandation.** P1 → P3 → P4, puis P5, puis P6. Les trois premiers n'exigent que `go test` (P4 : Docker optionnel pour l'adapter MySQL) et fournissent l'outillage réutilisé ensuite : banc statistique (P1), corpus et analyseur (P3), garde-fou d'architecture (P4). P5 introduit le broker sur un périmètre borné ; P6 en tire le maximum.

**Compromis principal de la séquence.** Elle privilégie la vitesse d'itération sur la centralité : les chapitres les plus « entreprise » du livre (13, 15–19) n'arrivent qu'à partir de P4.

**Alternative.** Commencer par P4 pour disposer immédiatement du projet Transport complet et y greffer P7, P8 puis P6 ; les projets de fond (P1, P3) viennent en fin de parcours comme approfondissements.

**Conditions qui renversent la recommandation.**

- Absence de Docker fonctionnel sur le poste : retirer P6 et P8, réduire P2 et P5 ; la séquence devient P1 → P3 → P4 (sans MySQL) → P7 (HTTP seul).
- Objectif de publication : P5 et P3 ont la meilleure originalité ; P1 réplique surtout des résultats connus de la littérature Go.
- Contrainte de temps totale < 10 jours : P1 + P3 uniquement.

## 9. Références

- Shahsavan, S. *Building Enterprise Projects with Go*. Apress, 2026. DOI 10.1007/979-8-8688-2370-1. PDF analysé : `Building_Enterprise_Projects_with_Go.pdf` (611 p.).
- Dépôt de code du livre : https://github.com/shahsavan/building-enterprise-projects-with-go (consulté le 2026-09-10).
- Go 1.25 Release Notes : https://go.dev/doc/go1.25 (consulté le 2026-09-10) — `testing/synctest` stable, `GOMAXPROCS` cgroup-aware, `GOEXPERIMENT=greenteagc`, `GOEXPERIMENT=jsonv2`.
- Annonce Apache Pulsar Go Client 0.18.0, liste `users@pulsar.apache.org` : http://www.mail-archive.com/users@pulsar.apache.org/msg02078.html (consulté le 2026-09-10).
- Ouvrages cités par le livre et pertinents pour les protocoles (annexe « Further Reading », p. 571–573) : Cockburn, *Hexagonal Architecture (Ports & Adapters)*, 2005 ; Newman, *Building Microservices*, 2e éd., 2021 ; Harsanyi, *100 Go Mistakes and How to Avoid Them*, 2022 ; *A Guide to the Go Garbage Collector* (go.dev/doc/gc-guide) ; *The Green Tea Garbage Collector* (go.dev/blog/greenteagc).

## Annexe — Limites de l'analyse

- L'extraction texte (pdftotext) a perdu une partie des listings de code et des encadrés « Enterprise Lesson » ; aucune valeur manquante n'a été reconstituée. Les paramètres numériques cités (timeouts, tailles de lots, seuils) proviennent du texte courant, pas des listings.
- Le ch. 18 ne donne aucune mesure chiffrée : toutes ses affirmations de performance sont qualitatives (« dramatically », « greatly »), ce qui rend P6 réfutable seulement au sens de la direction et de l'ordre de grandeur, pas d'une valeur.
- Les estimations d'effort sont des hypothèses non calibrées ; elles supposent Claude Code comme exécutant principal et un examen humain des protocoles avant chaque campagne.
