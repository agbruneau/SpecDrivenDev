# Projets Go en Spec-Driven Development : identification et parcours recommandé

Date : 2026-09-08
Sources : *Building Enterprise Projects with Go* (S. Shahsavan, Apress 2026); *Spec-Driven Development: From Specs to Code with AI Agents* (S. Martinelli, Apress Pocket Guides 2026); dépôt compagnon du livre Go [shahsavan/building-enterprise-projects-with-go](https://github.com/shahsavan/building-enterprise-projects-with-go) (un dossier par chapitre, `01-hello-world` à `06-database` et suivants); dépôt [agbruneau/Fibonacci](https://github.com/agbruneau/Fibonacci) (README et arborescence); inventaire du poste. Tout lu ou vérifié le 2026-09-08.

## 1. Ce que les deux livres exigent d'un projet

**Shahsavan.** La Partie I (ch. 1 à 9) installe le langage et les réflexes. La Partie II (ch. 10 à 20) construit un seul système, un gestionnaire de transport, en trois modules Go reliés par `go.work` : `ride` (REST généré d'un contrat OpenAPI, Gin), `vehicle` (gRPC interne) et `notification` (consommateur Pulsar). Autour : MySQL par `database/sql`, migrations versionnées, Testcontainers derrière une étiquette de build, schémas Avro comme langue publiée, Makefile et Dockerfile comme contrat de build exécutable, configuration YAML surchargée par variables d'environnement, `http.Server` avec délais, `/healthz` et arrêt gracieux, architecture hexagonale à quatre paquets (`models`, `ports`, `service`, `adapters`), métriques de domaine et patrons de concurrence. Le livre nomme lui-même l'espace laissé libre : « driver management or route planning » (ch. 10).

**Martinelli.** L'AI Unified Process tient en trois artefacts, le Specification Core : catalogue d'exigences (`FR-`, `NFR-`, `C-`), modèle d'entités (Mermaid, qui sert de glossaire) et cas d'utilisation système (`UC-`, un fichier Markdown chacun). Un UC est un contrat exécutable : préconditions, scénario principal numéré en étapes observables, flux alternatifs rattachés à une étape, postconditions de succès et d'échec, règles `BR-` nommées, exigences liées. Cycle : specify, generate, validate, review, refine. Tout changement de comportement commence dans la spec; on synchronise le code, on ne le régénère pas. Les tests dérivent de l'UC selon le test trophy, et sans interface graphique le test primaire d'un UC est un test d'intégration (ch. 6). À l'échelle : un Specification Core par contexte délimité, flux kanban par UC (Draft, Review, Approved, Implemented, Verified, Deployed) et tableau de bord de traçabilité (tables 9-1 et 9-2). Le livre dit aussi quand ne pas appliquer la discipline : prototype jetable, script, système sans règle contestée (ch. 10).

**Point de départ.** Fibonacci couvre la Partie I, souvent au-delà du livre : module unique, `internal/`, test d'architecture des imports, golden files, fuzzing, `-race`, couverture 96 %, Makefile, Dockerfile, devcontainer, pooling, contrôle du GC, PGO. Mais c'est un système à un acteur, sans entrée-sortie ni état persistant : exactement celui que Martinelli exclut. L'écart à combler est la Partie II, et le SDD n'a de prise que sur un système à plusieurs acteurs, avec des règles d'affaires, des flux alternatifs et une durée de vie.

**Votre axe académique.** Le mémoire en cours porte sur l'ingénierie des systèmes appliquée à une plateforme HPC hétérogène (CPU, GPU, FPGA, QPU) avec délégation des charges au processeur optimal, cadrée par ISO/IEC/IEEE 15288 et trois questions : Q-1, une architecture de référence qui traite l'instabilité du QPU comme un état de conception; Q-2, les critères et le mécanisme de délégation au meilleur processeur; Q-3, les exigences d'exploitabilité propres au QPU (étalonnage, disponibilité, observabilité). Un projet Go n'a de valeur académique que s'il fournit un système d'intérêt au sens de 15288 : éléments hétérogènes, interfaces définies, états d'exploitation, compromis à arbitrer, vérification et validation distinctes. Martinelli décrit un processus logiciel; il reste silencieux sur les besoins des parties prenantes, les méthodes de vérification autres que le test, la validation, l'exploitation et l'analyse système. C'est là que votre intérêt ajoute du contenu au projet, et c'est le sixième critère.

## 2. Critères de sélection

1. **Couverture de la Partie II** : combien de chapitres 11 à 20 le projet force à pratiquer sans artifice.
2. **Adéquation SDD** : acteurs multiples, règles contestables, flux alternatifs, comportement vérifiable de l'extérieur.
3. **Non-redondance** avec Fibonacci : pas de calcul intensif ni d'optimisation.
4. **Faisabilité en solo** : découpage en UC livrables un à un, « juste assez de spécification ».
5. **Réemploi** : réutilisation des artefacts (specs, outils, skills, `CLAUDE.md`) dans les projets suivants.
6. **Ingénierie des systèmes** : le projet offre-t-il un système d'intérêt au sens de 15288 (éléments hétérogènes, interfaces, états d'exploitation, compromis, vérification et validation) et alimente-t-il Q-1, Q-2 ou Q-3.

## 3. Contrainte de poste (vérifiée le 2026-09-08)

| Environnement | Go | Moteur de conteneurs |
|---|---|---|
| Windows 11 natif | 1.27.0 | aucun |
| WSL2 Ubuntu | 1.22.2 | aucun (`docker` introuvable) |

Conséquence : les chapitres 16 à 19 (MySQL, Testcontainers, Pulsar) et le test d'intégration primaire de Martinelli exigent un moteur de conteneurs. Prérequis avant l'itération 2 du parcours, à faire par vous :

- Docker Engine dans WSL2 Ubuntu (dépôt apt officiel de Docker), ou Podman avec `DOCKER_HOST` pointé sur son socket; testcontainers-go accepte les deux.
- Go 1.25 ou plus dans WSL2, pour suivre le livre (`GOMAXPROCS` conscient des conteneurs, `encoding/json/v2`).
- Outils : `make`, `protoc` avec `protoc-gen-go` et `protoc-gen-go-grpc`, `oapi-codegen`, `avrogen` (hamba), `golang-migrate` ou `goose`, `golangci-lint`, `govulncheck`.

Vérification une fois installé :

```bash
wsl -d Ubuntu -- bash -lc 'docker run --rm hello-world && go version'
```

Sans moteur de conteneurs, seules les itérations 0 et 1 sont réalisables (adaptateurs en mémoire).

## 4. Projets candidats

### A. Système de transport du livre, refait spec-first

Reprendre Ride, Vehicle et Notification en écrivant catalogue, modèle d'entités et UC avant chaque chapitre, puis comparer avec le dépôt compagnon.
Acteurs et règles : ceux du livre, déjà décidés.
Chapitres : 11 à 19 intégralement, avec corrigé.
Levier SDD : faible; les specs seraient rétro-écrites à partir d'une conception existante, le contraire de ce que prescrit Martinelli.
Ingénierie des systèmes : faible; interfaces et compromis sont déjà tranchés par l'auteur.
Risque : copier au lieu de concevoir.

### B. Contexte délimité « conducteurs » greffé au transport du livre

Nouveau module `driver` à côté des trois du livre : affectation d'un conducteur à une Assignment, permis compatible avec la classe du véhicule, heures maximales par jour, aucun chevauchement, conducteur en congé jamais affecté (règle jumelle du « user inactif » de Martinelli). Il consomme `AssignmentCreated` (Pulsar, Avro), interroge Vehicle par gRPC et expose `/duties` en REST. Le livre cite lui-même l'événement `DriverRetired` venu des RH (ch. 18).
Acteurs : Répartiteur, Conducteur, système RH (acteur système), services Ride et Vehicle.
Chapitres : 11 à 19, plus le ch. 9 de Martinelli pour vrai : un contexte qui ne possède pas les contrats de ses voisins.
Levier SDD : fort; les règles sont neuves, donc à décider.
Ingénierie des systèmes : moyenne; les contrats des voisins deviennent de vrais documents de contrôle d'interface, mais le domaine ne touche ni Q-1, ni Q-2, ni Q-3.
Risque : dépendre du dépôt compagnon (Gin, MySQL, Pulsar) qu'il faut d'abord faire tourner tel quel.

### C. Ordonnanceur de travaux pour plateforme hétérogène (`qsched`)

Trois contextes délimités : `jobs` (soumission, REST/OpenAPI, MySQL), `nodes` (registre de nœuds CPU, GPU, FPGA, QPU et allocation, gRPC interne, battements en flux) et `notify` (consommateur `JobStateChanged`, journal d'audit, DLQ).
Acteurs : Chercheur, Opérateur, Agent de nœud (acteur système), Horloge (déclencheur du cycle d'allocation).
Chapitres : 11 à 20; l'arrêt gracieux (13) et les délais gRPC (19) y sont des exigences réelles.
Levier SDD : fort; quotas, priorités, états des nœuds, propriété des travaux produisent des `BR-` et des flux alternatifs naturels. Isomorphe au projet du livre (Travail ↔ Assignment, Nœud ↔ Vehicle, Notification ↔ Notification), donc le dépôt compagnon reste un corrigé technique sans être un modèle de conception.
Ingénierie des systèmes : forte; nœuds hétérogènes, machine d'états d'exploitation où l'instabilité d'un élément est un état de conception (Q-1), délégation comme compromis explicite derrière un port (Q-2), exploitabilité mesurable par des métriques de disponibilité et d'étalonnage (Q-3).
Risque : dériver vers l'algorithmique d'ordonnancement, rechute Fibonacci. Parade au § 6.7.

### D. Réservation de salles et d'équipements de laboratoire

Réservations, conflits d'horaire, fenêtres d'annulation, rôles, notifications. Le cas d'école de Martinelli, proche du gestionnaire de tâches du ch. 8.
Chapitres : 11 à 17 naturellement; gRPC et événements y sont plaqués.
Levier SDD : très fort.
Ingénierie des systèmes : nulle; un seul élément logiciel, sans état d'exploitation ni compromis.
Risque : CRUD sans difficulté Go propre; faible motivation.

### E. Passerelle de télémétrie de flotte

Ingestion en flux gRPC, pipeline Pulsar, agrégation, API de requête.
Chapitres : 18, 19 et la concurrence en profondeur; 15 et 16 restent légers.
Levier SDD : faible; peu d'acteurs, surtout des flux de données.
Ingénierie des systèmes : moyenne à forte; observabilité et disponibilité relèvent de Q-3, mais sans règles d'affaires ni compromis d'architecture.
Risque : retomber dans le réglage de performance.

### F. Outillage SDD en Go, bibliothèque standard seulement

Deux petits outils réutilisables sur tout dépôt SDD, le mémoire compris.
`specs-lint` applique les règles de rédaction de Martinelli (ch. 3, 4 et table 10-1) : identifiants uniques et jamais réutilisés, chaque UC lié à une `FR-` existante, chaque flux alternatif rattaché à une étape existante, postconditions de succès et d'échec présentes, entités citées présentes dans le modèle, mots flous interdits (« normalement », « rapidement », « au besoin », « de façon appropriée », « devrait »).
`specs-dash` produit les tables 9-1 et 9-2 en Markdown et HTML à partir de `specs/`, de `go test -json` et des identifiants `UC-` trouvés dans le code; s'exécute en CI.
Chapitres : 4 à 7 de la Partie I, 11 (Makefile, CI), 20 (pipeline); `bufio`, `regexp`, `encoding/json`, `text/template`, `embed`.
Levier SDD : fort; c'est l'outillage que Martinelli annonce comme prochaine étape (ch. 10).
Ingénierie des systèmes : forte; la matrice de traçabilité exigences → UC → preuves et la couverture de vérification sont les artefacts de base de la V&V, réutilisables tels quels sur le Specification Core du mémoire.
Risque : encore un outil en ligne de commande; rien de la Partie II.

### G. Greffon de pile Go pour l'AI Unified Process (`aiup-go`)

Skills `implement` (hexagonal Go), `migration` (depuis le modèle d'entités), `integration-test` (Testcontainers depuis un UC), `openapi` et `proto` (depuis un UC). Le marché AIUP n'a qu'un greffon Vaadin/jOOQ (Martinelli, ch. 7).
Chapitres : presque aucun; c'est de la rédaction d'instructions.
Levier SDD : très fort, mais il tourne à vide sans projet Go concret.
Ingénierie des systèmes : faible; c'est de l'ingénierie de processus, pas de système.
Risque : l'extraire avant que le `CLAUDE.md` d'un vrai projet se soit stabilisé.

### H. Simulateur à événements discrets de la plateforme HPC-QPU

Jumeau d'analyse pour comparer des politiques de délégation sur des charges synthétiques (Q-2), avec instabilité et fenêtres d'étalonnage des QPU comme paramètres.
Chapitres : 9 et 20 seulement.
Levier SDD : faible; un acteur, aucune règle d'affaires.
Ingénierie des systèmes : forte en apparence (processus d'analyse système), mais sur un modèle, pas sur un système.
Risque : c'est du calcul et de la mesure, le terrain de Fibonacci; le simulateur remplace le système. La même analyse s'obtient sans simulateur en rejouant le journal d'audit de `notify` (Shahsavan, ch. 18, « auditability and time travel ») contre une autre politique : c'est l'extension du § 6.9, pas un projet.

## 5. Comparaison

Notes sur 5; le total n'est qu'un ordre de grandeur.

| Projet | Partie II | SDD | Non-redondance | Faisabilité | Réemploi | Ing. systèmes | Total |
|---|---|---|---|---|---|---|---|
| A. Transport refait spec-first | 5 | 2 | 4 | 4 | 2 | 2 | 19 |
| B. Contexte « conducteurs » | 5 | 5 | 5 | 3 | 3 | 3 | 24 |
| **C. Ordonnanceur `qsched`** | 5 | 5 | 4 | 4 | 4 | 5 | **27** |
| D. Réservations labo | 4 | 5 | 5 | 5 | 2 | 1 | 22 |
| E. Télémétrie de flotte | 4 | 2 | 3 | 3 | 3 | 4 | 19 |
| F. `specs-lint` et `specs-dash` | 2 | 4 | 4 | 5 | 5 | 4 | 24 |
| G. Greffon `aiup-go` | 1 | 5 | 5 | 4 | 4 | 2 | 21 |
| H. Simulateur HPC-QPU | 1 | 2 | 2 | 4 | 3 | 5 | 17 |

Le sixième critère creuse l'écart en faveur de C, remonte F au niveau de B et écarte D, qui perd son seul avantage.

## 6. Recommandation : un parcours en trois temps

- **Temps 0, kata d'une journée sur Fibonacci.** Exactement l'exercice de départ de Martinelli (ch. 10) : un comportement connu, un acteur, un but, au moins une règle qui a déjà fait débat. Écrire le catalogue minimal et deux UC (`calculer`, `calibrer`), les faire relire, puis renommer les tests e2e existants avec les identifiants `UC-`. But : roder le format, pas prolonger. Ne pas dépasser la journée : Fibonacci reste un système que Martinelli exclut.
- **Temps 1, projet principal : C, `qsched`.** Justification et plan aux § 6.1 à 6.9. B est l'alternative si vous préférez rester dans le domaine du livre; la feuille de route du § 6.6 s'y transpose module pour module, mais le § 6.9 n'y a pas d'équivalent.
- **Temps 2, outillage : F**, en deux temps. `specs-lint` tout de suite après l'itération 0 (un ou deux jours, bibliothèque standard) : il valide les specs qui viennent d'être écrites et celles du mémoire. `specs-dash` à l'itération 3, quand il y a des tests à tracer. G naît de lui-même ensuite : le `CLAUDE.md` et les trois ou quatre commandes répétées du projet deviennent des skills une fois stabilisés.

### 6.1 Pourquoi C

1. **Isomorphe au projet du livre.** Chaque chapitre de la Partie II se transpose un pour un; le dépôt compagnon, organisé par chapitre, sert de corrigé technique (`oapi-codegen`, producteur Pulsar générique, Testcontainers) sans imposer sa conception.
2. **Le SDD y travaille pour vrai.** « Un nœud en maintenance ne reçoit aucune allocation » est la règle jumelle de « un utilisateur inactif ne reçoit aucune tâche » qui traverse tout Martinelli. Quotas, priorités, annulation par le propriétaire ou un opérateur, capacités déclarées : autant de flux alternatifs à décider avant de coder.
3. **Trois contextes délimités dès le départ.** Chacun possède son Specification Core et ne connaît des voisins que leurs contrats (OpenAPI, Protobuf, Avro) : le ch. 9 de Martinelli en grandeur nature.
4. **Complémentaire de Fibonacci.** Tout ce qui est difficile ici est de l'état partagé, du réseau, de la persistance et de l'exploitation; jamais du calcul.
5. **Un système d'intérêt, pas seulement un logiciel.** Éléments hétérogènes, trois interfaces sous contrôle, machine d'états d'exploitation, compromis de délégation, exigences d'exploitabilité mesurables : les trois questions du mémoire y trouvent un prototype exécutable (§ 6.9), à condition de garder la politique d'allocation derrière un port.

### 6.2 Esquisse du Specification Core

Acteurs : Chercheur, Opérateur, Agent de nœud (système), Horloge (déclencheur).

Entités : Projet (quota en heures-nœud), Travail (états `QUEUED`, `ALLOCATED`, `RUNNING`, `CANCELLING`, `DONE`, `FAILED`, `CANCELLED`), Nœud (capacités `CPU`, `GPU`, `FPGA`, `QPU`; états `READY`, `BUSY`, `MAINTENANCE`), Allocation, Notification.

Catalogue (extrait) :

| ID | Titre |
|---|---|
| FR-001 | Soumettre un travail |
| FR-002 | Annuler un travail |
| FR-003 | Consulter l'état d'un travail |
| FR-004 | Enregistrer un nœud et ses capacités |
| FR-005 | Allouer les travaux en file aux nœuds prêts |
| FR-006 | Rapporter la fin d'un travail |
| FR-007 | Mettre un nœud en maintenance |
| NFR-001 | Soumission acquittée en moins de 200 ms (p95) |
| NFR-002 | Arrêt gracieux sans perte de travail en file |
| NFR-003 | Aucun événement perdu : livraison au moins une fois, consommateurs idempotents |
| C-001 | Go 1.25 ou plus; `net/http` de la bibliothèque standard, pas Gin |
| C-002 | MySQL 8, migrations versionnées |
| C-003 | Contrat OpenAPI écrit avant tout code de l'adaptateur HTTP |
| C-004 | gRPC réservé aux appels internes; REST vers l'extérieur |
| C-005 | Hexagonal strict : `models`, `ports`, `service`, `adapters` |
| C-006 | La politique d'allocation est un port; v1 = FIFO par priorité décroissante |
| C-007 | Tout s'exécute dans WSL2 Ubuntu avec Docker Engine; aucun service infonuagique |

UC-001 Soumettre un travail, dans le format exécutable de Martinelli (ch. 4) :

- Acteur principal : Chercheur. But : obtenir un travail en file pour un projet dont le quota le permet. Statut : Draft.
- Préconditions : le chercheur est authentifié; le projet existe et le chercheur en est membre.
- Déclencheur : le chercheur soumet une demande de travail.
- Scénario principal :
  1. Le chercheur soumet une demande : projet, capacités requises, priorité, commande, durée maximale.
  2. Le système vérifie que chaque capacité requise est déclarée par au moins un nœud enregistré.
  3. Le système vérifie que le quota restant du projet couvre la durée maximale demandée.
  4. Le système enregistre le travail à l'état `QUEUED` et réserve la durée sur le quota.
  5. Le système publie `JobStateChanged`.
  6. Le système retourne au chercheur l'identifiant et l'état du travail.
- A1, à l'étape 2, capacité inconnue : le système refuse la demande et nomme la capacité manquante; aucun travail n'est créé.
- A2, à l'étape 3, quota insuffisant : le système refuse la demande et indique le quota restant; aucun travail n'est créé.
- Postconditions. Succès : un travail `QUEUED` existe, le quota est réservé, l'événement est publié. Échec : aucun travail, quota inchangé.
- Règles : BR-001 un travail ne requiert que des capacités déclarées par au moins un nœud enregistré; BR-002 le quota d'un projet n'est jamais dépassé.
- Exigences liées : FR-001, NFR-001, C-003.

Autres règles structurantes : BR-003 un nœud en `MAINTENANCE` ne reçoit aucune allocation; BR-004 seul le propriétaire ou un opérateur annule un travail; BR-005 un travail `RUNNING` annulé passe par `CANCELLING` jusqu'à confirmation de l'agent. Les pannes techniques (base indisponible, délai gRPC) restent hors des flux alternatifs, comme le prescrit Martinelli.

### 6.3 Architecture cible

```
qsched/
├── go.work
├── CLAUDE.md
├── Makefile               # fmt, lint, test, integration_test, generate, build; appelé tel quel par le Dockerfile et la CI
├── jobs/                  # REST externe : OpenAPI → oapi-codegen (cible std-http), MySQL, migrations
│   └── specs/             # Specification Core du contexte
├── nodes/                 # gRPC interne : RegisterNode, Heartbeat (flux client), Allocate, SetMaintenance
│   └── specs/
├── notify/                # consommateur Pulsar, Avro, DLQ, journal d'audit
│   └── specs/
└── deploy/                # docker compose : mysql, pulsar, les trois services, prometheus
```

Chaque module : `cmd/`, `internal/{models,ports,service,adapters}`, comme au ch. 14. Le livre utilise Gin; `net/http` suffit depuis Go 1.22 et `oapi-codegen` le cible. Le livre cite Flyway et Liquibase pour les migrations; en Go, `golang-migrate` ou `goose` jouent ce rôle.

### 6.4 Trophée de tests et traçabilité en Go

- Statique : `golangci-lint`, `govulncheck`, test d'architecture des imports repris de Fibonacci (`internal/arch_test.go`).
- Unitaires : les `BR-` dans `service`, avec ports factices en mémoire.
- Intégration, couche principale : un test par scénario principal et par flux alternatif qui touche la base ou le courtier; MySQL et Pulsar par Testcontainers; `//go:build integration`; conteneur démarré une fois par paquet avec `sync.Once`; étape CI séparée (ch. 17).
- Bout en bout : sélectif, docker compose, un scénario de la soumission jusqu'à la notification.

Martinelli trace par annotation Java (`@UseCase(id, scenario)`). En Go, le nom suffit : `TestUC001_SoumettreTravail` avec sous-tests `t.Run("A2_quota_insuffisant", …)`. Alors `go test -run UC001 ./...` exécute la vérification d'un UC, les messages de commit portent l'identifiant, et `specs-dash` se réduit à lire `go test -json`.

### 6.5 `CLAUDE.md` de départ

Court, comme le veut Martinelli (ch. 5) :

- Toute modification de comportement commence dans `specs/`; le code se synchronise, il ne se régénère pas.
- Référencer l'identifiant `UC-` dans les noms de tests et les messages de commit; ne jamais réutiliser un identifiant.
- Hexagonal strict : aucun SQL, JSON ou Protobuf hors de `adapters`; les DTO générés ne traversent pas les ports.
- Ne jamais éditer le code généré (`oapi-codegen`, `protoc`, `avrogen`); régénérer par `make generate`.
- La politique d'allocation reste un port; toute amélioration passe par une nouvelle exigence.
- Une dépendance externe par itération, et seulement si un UC de l'itération l'exige.

### 6.6 Feuille de route (unité de progrès : l'UC)

| Itération | Livrable | Chapitres Go | Martinelli |
|---|---|---|---|
| 0 | Vision et concept d'exploitation, besoins par acteur, catalogue avec méthode de vérification par exigence, modèle d'entités, diagramme des UC (PlantUML), UC-001 à UC-007 en Draft | 10 | ch. 3-4; ch. 8 étapes 1-4 |
| 0 bis | `specs-lint` sur le `specs/` qui vient d'être écrit | 4 à 7 | ch. 4; table 10-1 |
| 1 | Socle : `go.work`, trois modules, Makefile, Dockerfile, config YAML + env, `http.Server` avec `/healthz` et arrêt gracieux, CI en deux étapes | 11, 12, 13 | étape 5 |
| 2 | UC-001, UC-003 : OpenAPI, adaptateur HTTP, dépôt MySQL, migrations, tests d'intégration | 14, 15, 16, 17 | étapes 6-10 |
| 3 | UC-004, UC-005, UC-007 : Protobuf, serveur et client gRPC, flux de battements, intercepteurs, cycle d'allocation; démarrage de `specs-lint` | 19, 20 | itération; ch. 9 |
| 4 | UC-002, UC-006 : `JobStateChanged` en Avro, producteur et consommateur génériques, DLQ, idempotence, preuve de NFR-002 | 18 | itération |
| 5 | Exploitation : métriques de domaine (travaux en file, nœuds prêts), `slog` contextuel; `specs-dash` en CI; validation bout en bout contre les besoins des acteurs | 10, 20 | ch. 9, tables 9-1 et 9-2 |
| 6 | Extension académique (§ 6.9) : états d'exploitation `CALIBRATING` et `DEGRADED`, fenêtres de disponibilité, métriques d'exploitabilité, rejeu du journal d'audit contre une seconde politique | 18, 20 | itération |

### 6.7 Garde-fous contre la dérive

- C-006 verrouille l'algorithmique : FIFO par priorité en v1; la politique n'évolue que par exigence nouvelle, et une seconde politique n'apparaît qu'à l'itération 6, pour être comparée, pas optimisée.
- L'agent de nœud est factice : il dort la durée demandée puis rapporte `DONE`. Aucune exécution réelle, aucune simulation physique de QPU; `QPU` n'est qu'une capacité déclarée. Ce qui peut devenir réel à l'itération 6, c'est le modèle d'états d'exploitation, spécifié comme tout autre UC.
- Sept UC avant l'itération 5, pas un de plus (« juste assez de spécification », ch. 4).
- Une dépendance externe par itération.
- Le moteur de conteneurs est installé avant l'itération 2, pas pendant.

### 6.8 Critère de fin

Le projet est terminé quand la table 9-1 affiche 100 % : chaque `FR-` est liée à un UC approuvé, implémenté et couvert par un test d'intégration qui passe en CI, et NFR-002 est prouvée par un test qui envoie `SIGTERM` pendant qu'une file contient des travaux et vérifie qu'aucun n'est perdu. Pour l'axe académique, s'ajoute : chaque exigence a une preuve selon la méthode déclarée au catalogue, et les scénarios de validation de l'itération 5 sont rejoués avec succès.

### 6.9 Ce que l'ingénierie des systèmes ajoute au projet

Martinelli couvre les processus techniques de 15288 qui concernent le logiciel et se tait sur les autres. Le tableau place chaque processus en face de l'artefact AIUP correspondant et de ce que `qsched` produit en plus.

| Processus technique 15288 | Artefact AIUP (Martinelli) | Dans `qsched` |
|---|---|---|
| Analyse de mission | `vision.md` | Vision et concept d'exploitation en une page : qui soumet, qui exploite, quels nœuds, quel cycle d'allocation |
| Besoins des parties prenantes | acteurs, user stories des `FR-` | Besoins par acteur, tenus séparés des exigences système; la validation se fait contre eux |
| Définition des exigences système | catalogue `FR-`, `NFR-`, `C-` | Le catalogue, plus une colonne « méthode de vérification » : test, analyse, inspection ou démonstration |
| Définition de l'architecture | contextes délimités, modèle d'entités | Trois contextes; leurs interfaces comme documents de contrôle d'interface : OpenAPI, Protobuf, Avro, chacun avec sa règle de compatibilité (Shahsavan, ch. 15, 18 et 19) |
| Définition de la conception | silence | Hexagonal par module, ports nommés; un ADR par choix structurant (Shahsavan, ch. 1; Fibonacci `docs/adr/`) |
| Analyse système | silence | Compromis explicites : REST ou gRPC par frontière, politique d'allocation derrière C-006; tableau des modes de défaillance (battement perdu, nœud en maintenance en cours d'exécution, courtier indisponible) dont dérivent NFR-002 et NFR-003 |
| Implémentation, intégration | `/implement`, migrations | Itérations 1 à 4; docker compose comme intégration |
| Vérification | tests dérivés des UC | Le trophée du § 6.4; chaque exigence a sa preuve selon sa méthode; `specs-dash` est la matrice de traçabilité |
| Validation | silence (tests bout en bout) | Scénarios bout en bout rejoués contre les besoins des acteurs, pas contre les UC |
| Transition, exploitation, maintenance | silence | `/healthz`, arrêt gracieux, métriques de domaine, `slog`; la synchronisation spec → code est la maintenance (Martinelli, ch. 5) |

Retombées pour le mémoire :

- **Q-1** : la machine d'états des nœuds inclut `CALIBRATING` et `DEGRADED` comme états de conception, avec les règles `BR-` qui disent ce qu'un nœud dans chacun de ces états peut recevoir (itération 6).
- **Q-2** : la politique d'allocation est un port; le journal d'audit de `notify` permet de rejouer l'historique contre une seconde politique et de comparer, sans simulateur.
- **Q-3** : les exigences d'exploitabilité deviennent des `NFR-` mesurées par des métriques : disponibilité par nœud, délai de détection d'un battement perdu, temps passé en `CALIBRATING`.

## 7. Outillage F : périmètre minimal

- `specs-lint` : un binaire, entrée `specs/`, sortie une ligne par violation (`fichier:ligne: règle`), code de retour non nul en CI. Huit règles, celles du § 4 F. Un test golden sur un `specs/` d'exemple qui contient une violation de chaque règle.
- `specs-dash` : entrées `specs/`, un fichier `go test -json` et l'arbre du code; sorties `DASHBOARD.md` (tables 9-1 et 9-2, plus la méthode de vérification et l'état de la preuve par exigence) et une page HTML par `embed`. Un test golden sur le même `specs/` d'exemple. Compatible avec tout Specification Core AIUP, celui du mémoire compris.
- Aucune dépendance hors bibliothèque standard; aucun démon; pas de serveur HTTP tant qu'un UC ne l'exige pas.

## 8. Ce qui a été vérifié et ce qui est supposé

Vérifié le 2026-09-08 : lecture intégrale du livre de Martinelli; pour Shahsavan, table des matières détaillée, introduction, chapitres 10, 11, 14 et 18 intégralement, résumés et exercices des chapitres 12 à 19; existence et organisation par chapitre du dépôt compagnon; README et arborescence de Fibonacci; Go et absence de moteur de conteneurs sous Windows et dans WSL2.

Supposé : que votre expérience Go des services réseau et de la persistance se limite à ce que montre Fibonacci. Le cadrage du mémoire (15288, Q-1 à Q-3) vient de mes notes du 2026-08-29; ses artefacts (`prd.md`, `specs/`) ne sont plus dans le dépôt et je ne les ai pas relus : si les questions ont changé, seuls le § 6.9 et la colonne « Ing. systèmes » bougent. Les processus techniques de 15288 sont cités de mémoire, sans la norme sous les yeux. Une version antérieure de ce document, qui retenait déjà C, a été purgée du dépôt le matin du 2026-09-08; celle-ci est autonome et ne s'y réfère pas.
