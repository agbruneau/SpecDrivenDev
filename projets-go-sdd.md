# Projets Go en Spec-Driven Development : identification et recommandation

Date : 2026-09-08
Sources : *Building Enterprise Projects with Go* (S. Shahsavan, Apress 2026), *Spec-Driven Development* (S. Martinelli, Apress Pocket Guides 2026), dépôt [agbruneau/Fibonacci](https://github.com/agbruneau/Fibonacci) (README, v4.1.0).

## 1. Point de départ : ce que Fibonacci prouve déjà

Le projet Fibonacci couvre, souvent au-delà du niveau du livre, toute la Partie I de Shahsavan (chapitres 3 à 9). Ce qui manque est concentré dans la Partie II : les systèmes d'entreprise à entrées-sorties, à état persistant et distribués.

| Thème (chapitre du livre Go) | État dans Fibonacci | Écart |
|---|---|---|
| Structure, modules, frontières (4) | Clean architecture, règles d'import testées (`arch_test.go`) | Aucun module multiple, pas de `go.work` |
| Types, fonctions, interfaces (5-6) | Stratégies, observateur, génériques | Aucun |
| Tests (7) | Golden, propriétés, fuzzing, race, 96 % de couverture, e2e CLI | Aucun test d'intégration contre une dépendance réelle |
| Mémoire (8) | `sync.Pool`, arènes, contrôle du GC, PGO | Aucun, niveau supérieur au livre |
| Concurrence (9) | `errgroup`, sémaphores, FFT parallèle | Aucun |
| Multi-modules, Makefile, Dockerfile (11) | Makefile, Docker multi-étages, devcontainer | Espace de travail multi-modules, stratégie de branches |
| Configuration (12) | Drapeaux + variables `FIBCALC_*` | Fichiers YAML par environnement, secrets |
| Serveur HTTP (13) | Rien | `http.Server`, délais, `/healthz`, arrêt gracieux |
| Architecture hexagonale (14) | Ports internes seulement (stratégies de calcul) | Adaptateurs vers base de données, réseau, courtier |
| API-first, OpenAPI (15) | Rien | Contrat OpenAPI, `oapi-codegen`, convertisseurs, erreurs |
| Persistance (16) | Rien | `database/sql`, dépôts comme ports, transactions, migrations |
| Testcontainers (17) | Rien | Conteneurs réels, étiquette de build, étape CI dédiée |
| Événements (18) | Rien | Pulsar, schémas Avro, producteur générique, DLQ |
| gRPC (19) | Rien | Protobuf, intercepteurs, flux, délais côté client |
| Patrons (20) | Stratégie, observateur, pool de travailleurs | Décorateur d'adaptateurs, pipelines à contexte |
| Observabilité (10) | Paquet `metrics` interne | Métriques de domaine Prometheus, `slog` contextuel |

Conséquence pour le SDD : Fibonacci est précisément le type de système que Martinelli exclut de la discipline complète (un acteur, un objectif, aucune règle d'affaires contestée). Le projet suivant doit avoir plusieurs acteurs, des règles d'affaires avec flux alternatifs et une durée de vie, sinon les cas d'utilisation système n'ont rien à décrire.

## 2. Critères de sélection

1. **Couverture de la Partie II** : combien de chapitres 11 à 20 le projet force-t-il à pratiquer, sans artifice.
2. **Adéquation SDD** : acteurs multiples, règles d'affaires, flux alternatifs, comportement observable testable de l'extérieur; sans interface graphique, le test d'intégration est le test primaire (Martinelli, ch. 6).
3. **Non-redondance** avec Fibonacci : éviter le calcul intensif et l'optimisation, déjà maîtrisés.
4. **Faisabilité en solo** : découpage en cas d'utilisation livrables un à un, « juste assez de spécification ».
5. **Intérêt et réemploi** : lien avec vos travaux en cours, réutilisation des artefacts.

## 3. Projets candidats

### A. Système de transport (réplique du livre)
Ride (REST/OpenAPI, Gin), Vehicle (gRPC), Notification (consommateur Pulsar), MySQL. Refaire le projet du livre en écrivant les cas d'utilisation avant chaque chapitre.
Exerce : tout 11 à 19, avec dépôt de référence pour comparer.
Levier SDD : faible; la conception existe déjà, les specs seraient rétro-écrites.
Risque : copier au lieu de concevoir.

### B. Ordonnanceur de travaux pour plateforme hétérogène (HPC/QPU)
Trois contextes délimités : `jobs` (soumission REST/OpenAPI), `nodes` (registre de nœuds et allocation, gRPC interne, télémétrie en flux), `notify` (consommateur d'événements `JobStateChanged`, audit, DLQ). MySQL, Pulsar, Testcontainers.
Exerce : tout 11 à 20; l'arrêt gracieux (13) et les délais gRPC (19) y sont des exigences réelles, pas des exercices.
Levier SDD : fort; quotas, priorités, états des nœuds, propriété des travaux produisent des règles `BR-` et des flux alternatifs naturels. Aucune interface : tests d'intégration primaires.
Risque : dériver vers l'algorithmique d'ordonnancement (rechute Fibonacci). Parade en section 5.

### C. Réservation de salles et d'équipements de laboratoire
Réservations, conflits d'horaire, fenêtres d'annulation, notifications. Application d'affaires classique.
Exerce : 11 à 17 naturellement; gRPC et événements y sont plaqués.
Levier SDD : fort, c'est le cas d'école de Martinelli (proche du gestionnaire de tâches du ch. 8).
Risque : CRUD sans difficulté Go spécifique; faible motivation.

### D. Passerelle de télémétrie de flotte
Ingestion en flux gRPC, pipeline Pulsar, agrégation, API de requête.
Exerce : 18, 19 et la concurrence en profondeur; 15 et 16 restent légers.
Levier SDD : faible; peu d'acteurs, surtout des flux de données.
Risque : retomber dans le tuning de performance.

### E. Greffon de pile Go pour l'AI Unified Process (`aiup-go`)
Compétences (skills) `implement` hexagonal Go, `migration` depuis le modèle d'entités, `integration-test` Testcontainers depuis un UC, `openapi` depuis un UC. Le marché AIUP n'a qu'un greffon Vaadin/jOOQ (Martinelli, ch. 7).
Exerce : peu de Go; surtout de la rédaction d'instructions.
Levier SDD : très fort, mais sans projet Go concret il tourne à vide.

### F. Tableau de bord de traçabilité des spécifications
Outil Go qui lit `/specs`, repère les identifiants `UC-`, `FR-`, `BR-` dans le code et les tests, produit les tableaux 9-1 et 9-2 de Martinelli en CLI et HTML, s'exécute en CI.
Exerce : `go/ast`, `net/http`, `embed`; bibliothèque standard seulement.
Levier SDD : fort, c'est l'outillage que Martinelli annonce comme prochaine étape.
Risque : encore un outil en ligne de commande; aucun chapitre de la Partie II.

## 4. Comparaison

Notes sur 5. Le total n'est qu'un ordre de grandeur.

| Projet | Partie II | SDD | Non-redondance | Faisabilité | Intérêt | Total |
|---|---|---|---|---|---|---|
| A. Transport (réplique) | 5 | 2 | 4 | 5 | 2 | 18 |
| **B. Ordonnanceur HPC/QPU** | 5 | 5 | 4 | 4 | 5 | **23** |
| C. Réservations labo | 4 | 5 | 5 | 5 | 2 | 21 |
| D. Télémétrie de flotte | 4 | 2 | 3 | 3 | 3 | 15 |
| E. Greffon `aiup-go` | 2 | 5 | 5 | 4 | 4 | 20 |
| F. Tableau de bord specs | 2 | 4 | 4 | 5 | 4 | 19 |

## 5. Recommandation : B, l'ordonnanceur de travaux HPC/QPU

### Justification

1. **Isomorphe au projet du livre.** Travail ↔ Assignment, Nœud ↔ Vehicle, Notification ↔ Notification. Chaque chapitre de la Partie II se transpose un pour un, et le dépôt de référence de Shahsavan reste utilisable comme corrigé technique (`oapi-codegen`, producteur Pulsar générique, Testcontainers) sans copier la conception.
2. **Aligné sur votre mémoire.** Votre question de recherche Q-2 porte sur le mécanisme de délégation des charges au meilleur processeur. L'ordonnanceur en est le prototype exécutable, à condition de garder la politique de délégation derrière un port (voir garde-fous). Le Specification Core du mémoire et celui du projet partagent le vocabulaire.
3. **Le SDD y travaille pour vrai.** « Un nœud en maintenance ne reçoit aucune allocation » est la règle jumelle de « un utilisateur inactif ne reçoit aucune tâche » qui traverse tout le livre de Martinelli. Quotas, priorités, annulation par le propriétaire ou un opérateur, capacités déclarées : autant de flux alternatifs à spécifier avant de coder.
4. **Complémentaire de Fibonacci.** Tout ce qui est difficile ici est de l'état partagé, du réseau, de la persistance et de l'exploitation, jamais du calcul.
5. **Faisable par tranches.** Sept cas d'utilisation, chacun livrable seul, chacun rattaché à deux ou trois chapitres.

### Esquisse du Specification Core

Acteurs : Chercheur, Opérateur, Agent de nœud (acteur système), Horloge (déclencheur du cycle d'allocation).

Entités : Projet (quota), Travail (état `QUEUED`, `ALLOCATED`, `RUNNING`, `DONE`, `FAILED`, `CANCELLED`), Nœud (capacités `CPU`, `GPU`, `FPGA`, `QPU`; état `READY`, `BUSY`, `MAINTENANCE`), Allocation, Notification.

Catalogue d'exigences (extrait) :

| ID | Titre |
|---|---|
| FR-001 | Soumettre un travail |
| FR-002 | Annuler un travail |
| FR-003 | Consulter l'état d'un travail |
| FR-004 | Enregistrer un nœud et ses capacités |
| FR-005 | Allouer un travail à un nœud |
| FR-006 | Notifier tout changement d'état |
| FR-007 | Mettre un nœud en maintenance |
| NFR-001 | Soumission acquittée en moins de 200 ms (p95) |
| NFR-002 | Arrêt gracieux sans perte de travail en file |
| NFR-003 | Aucun événement perdu : livraison au moins une fois, consommateurs idempotents |
| C-001 | Go 1.25+, `net/http` de la bibliothèque standard |
| C-002 | MySQL 8, migrations versionnées |
| C-003 | Contrat OpenAPI écrit avant tout code de l'adaptateur HTTP |
| C-004 | gRPC réservé aux appels internes; REST vers l'extérieur |
| C-005 | Architecture hexagonale : `models`, `ports`, `service`, `adapters` |
| C-006 | La politique d'allocation est un port; v1 = FIFO par priorité décroissante |
| C-007 | Tout s'exécute sur le poste de développement (WSL2 Ubuntu avec Docker Engine); aucun service infonuagique. Détail dans [initialisation-qsched.md](initialisation-qsched.md), § 6 |

UC-001 Soumettre un travail (cible : Chercheur)

- Préconditions : le chercheur est identifié; le projet existe.
- Scénario principal :
  1. Le chercheur soumet une demande (projet, capacités requises, priorité, commande).
  2. Le système vérifie que chaque capacité requise est déclarée par au moins un nœud.
  3. Le système vérifie le quota restant du projet.
  4. Le système enregistre le travail à l'état `QUEUED`.
  5. Le système publie `JobStateChanged`.
  6. Le système retourne l'identifiant et l'état du travail.
- A1, à l'étape 2 : capacité inconnue. Le système rejette la demande et indique la capacité manquante. Aucun travail créé.
- A2, à l'étape 3 : quota épuisé. Le système rejette la demande et indique le quota restant. Aucun travail créé.
- Postconditions : succès, travail `QUEUED` et événement publié; échec, aucune modification.
- Règles : BR-001 un travail ne requiert que des capacités déclarées par au moins un nœud; BR-002 le quota d'un projet n'est jamais dépassé.
- Exigences liées : FR-001, NFR-001, C-003.

Autres règles structurantes : BR-003 un nœud en `MAINTENANCE` ne reçoit aucune allocation; BR-004 seul le propriétaire ou un opérateur annule un travail; BR-005 un travail `RUNNING` annulé passe par `CANCELLING` jusqu'à confirmation de l'agent.

### Architecture cible

```
qsched/
├── go.work
├── specs/                 # vision, requirements, entity-model (Mermaid), use-cases.puml, use-cases/UC-00n.md
├── CLAUDE.md
├── Makefile               # fmt, test, integration_test, build; appelé tel quel par le Dockerfile et la CI
├── jobs/                  # REST externe (OpenAPI → oapi-codegen, cible std-http), MySQL
├── nodes/                 # gRPC interne : RegisterNode, Heartbeat (flux client), Allocate, SetMaintenance
├── notify/                # consommateur Pulsar, Avro, DLQ, journal d'audit
└── deploy/                # docker compose, profils infra / hosted / obs : mysql, pulsar, les trois services, prometheus
```

Chaque module est un contexte délimité et possède son propre Specification Core, comme le prescrit Martinelli au chapitre 9. Le livre Go utilise Gin; `net/http` suffit depuis Go 1.22 et `oapi-codegen` le cible. Pour les migrations, le livre cite Flyway et Liquibase; en Go, `golang-migrate` ou `goose` jouent ce rôle.

### Trophée de tests et traçabilité

- Statique : `golangci-lint`, `govulncheck`, test d'architecture des imports repris de Fibonacci.
- Unitaires : règles `BR-` dans `service`, avec ports factices en mémoire.
- Intégration (couche principale) : un test par scénario principal et par flux alternatif touchant la base ou le courtier, MySQL et Pulsar via Testcontainers, étiquette `//go:build integration`, conteneur démarré une fois par paquet avec `sync.Once`, étape CI séparée.
- Bout en bout : sélectif, docker compose, un scénario de soumission jusqu'à notification.

La traçabilité n'a besoin d'aucune annotation : nommer les tests `TestUC001_SoumettreTravail` avec sous-tests `t.Run("A2_quota_epuise", …)`. Alors `go test -run UC001 ./...` exécute la vérification d'un cas d'utilisation, et le tableau de bord (projet F) se réduit à lire `go test -json`.

### CLAUDE.md du projet (règles de départ)

- Toute modification de comportement commence dans `specs/`; le code se synchronise, il ne se régénère pas.
- Référencer l'identifiant `UC-` dans les noms de tests et les messages de commit.
- Hexagonal strict : aucun SQL, JSON ou Protobuf hors de `adapters`; les DTO générés ne traversent pas les ports.
- Ne jamais éditer le code généré (`oapi-codegen`, `protoc`, `avrogen`); régénérer par `make generate`.
- La politique d'allocation reste un port; toute amélioration passe par une nouvelle exigence.

### Feuille de route (unité de progrès : le cas d'utilisation)

| Itération | Livrable | Chapitres Go | Étapes Martinelli |
|---|---|---|---|
| 0 | Vision, catalogue, modèle d'entités, diagramme des cas, UC-001 à UC-007 en brouillon | — | ch. 3-4, ch. 8 étapes 1-4 |
| 1 | Socle : `go.work`, trois modules, Makefile et Dockerfile, config YAML + env, `http.Server` avec `/healthz` et arrêt gracieux, CI en deux étapes | 11, 12, 13 | étape 5 |
| 2 | UC-001, UC-003 : OpenAPI, adaptateur HTTP, dépôt MySQL, migrations, Testcontainers | 14, 15, 16, 17 | étapes 6-10 |
| 3 | UC-004, UC-005, UC-007 : contrat Protobuf, serveur et client gRPC, flux de battements, intercepteurs, cycle d'allocation | 19, 20 | itération |
| 4 | UC-002, UC-006 : `JobStateChanged` en Avro, producteur et consommateur génériques, DLQ, idempotence, preuve de NFR-002 | 18 | itération |
| 5 | Exploitation : métriques de domaine (travaux en file, nœuds disponibles), `slog` contextuel, tableau de bord de specs (projet F) | 10, 20 | ch. 9 |

### Garde-fous contre la dérive

- C-006 verrouille l'algorithmique : FIFO par priorité en v1, la politique n'évolue que par exigence nouvelle.
- L'agent de nœud est factice : il dort la durée demandée puis rapporte `DONE`. Aucune exécution réelle, aucune simulation de QPU; `QPU` n'est qu'une capacité déclarée.
- Sept cas d'utilisation avant l'itération 5, pas un de plus (« juste assez de spécification », Martinelli ch. 4).
- Une dépendance externe par itération; on refuse celle qui n'est pas exigée par un UC de l'itération.

### Critère de fin

Le projet est terminé quand le tableau 9-1 affiche 100 % : chaque FR est liée à un UC approuvé, implémenté, couvert par un test d'intégration passant en CI, et NFR-002 est prouvée par un test qui envoie `SIGTERM` pendant qu'une file contient des travaux et vérifie qu'aucun n'est perdu.

## 6. Compléments

- **F, tableau de bord de specs** : à démarrer à l'itération 5, quand il y a assez d'identifiants à tracer; réutilisable ensuite dans le mémoire et tout projet SDD futur.
- **E, greffon `aiup-go`** : il naît de lui-même. Le `CLAUDE.md` et les trois ou quatre commandes répétées du projet (`implement`, `migration`, `integration-test`) deviennent des skills une fois stabilisées; les extraire avant est prématuré.
- **Kata d'une journée sur Fibonacci** : rédiger le catalogue et trois cas d'utilisation (`calculer`, `calibrer`, `derniers-chiffres`) puis renommer les tests e2e avec les identifiants `UC-`. Utile seulement pour roder le format avant l'itération 0; à ne pas prolonger.

## Ce qui a été vérifié et ce qui est supposé

Vérifié : lecture intégrale du livre de Martinelli; pour Shahsavan, introduction, chapitres 10, 11 et 14 intégralement, tables des matières détaillées et résumés des chapitres 12 à 20; README du dépôt Fibonacci; inventaire du poste (matériel, Go, WSL2, absence de moteur de conteneurs, ports) le 2026-09-08.
Supposé : que l'expérience Go des services réseau et de la persistance se limite à ce que montre Fibonacci, et que le lien avec le mémoire HPC/QPU vous intéresse; les deux sont faciles à corriger et ne changent que la note « Intérêt » du tableau.
