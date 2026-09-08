# Initialisation du projet `qsched` : de l'idéation au premier cas d'utilisation

Date : 2026-09-08
Référence : [projets-go-sdd.md](projets-go-sdd.md), recommandation B retenue (ordonnanceur de travaux HPC/QPU).
Ordre suivi : celui du chapitre 8 de Martinelli (vision, exigences, entités, diagramme, échafaudage, migration, spécification d'un UC, implémentation, tests). Aucune date; chaque étape a un critère de sortie et l'unité de progrès reste le cas d'utilisation.

## Vue d'ensemble

| Étape | Livrable | Critère de sortie |
|---|---|---|
| 0. Décisions de cadrage | `specs/decisions.md` (DEC-1 à DEC-7) | Chaque décision a une justification d'une ligne |
| 1. Idéation | `specs/ideation.md`, `specs/vision.md` | Vision d'une page, tout acteur présent dans une histoire, chaque point chaud tranché ou listé en question ouverte |
| 2. Specification Core | `requirements.md`, `entity-model.md`, `use-cases.puml`, `dashboard.md` | Revue humaine passée, chaque FR a un UC prévu |
| 2b. Préparation du poste (§ 6) | Docker Engine, Go, `make`, `protoc` dans WSL2 Ubuntu; VS Code Remote-WSL | `docker run hello-world` passe, test de fumée Testcontainers vert |
| 3. Échafaudage Go | Dépôt `qsched`, `go.work`, trois modules, socle HTTP, CI | `make test` vert, `/healthz` 200, arrêt gracieux prouvé par un test |
| 4. Premier cycle SDD | Migration, UC-001 approuvé, implémenté, testé | Ligne UC-001 du tableau de bord : Code, Unit, Int cochés |
| 5. Régime de croisière | Kanban par UC | Un UC à la fois, spec d'abord, synchronisation ensuite |

## 0. Décisions de cadrage

À consigner dans `specs/decisions.md` avec le format `DEC-n` déjà utilisé dans le mémoire. Valeurs proposées :

| ID | Décision | Proposition | Pourquoi |
|---|---|---|---|
| DEC-1 | Dépôt | Nouveau dépôt `github.com/agbruneau/qsched`; Prospection garde les documents de cadrage | Même séparation que Fibonacci; le `go.work` doit être à la racine du dépôt |
| DEC-2 | Langues | Spécifications en français, identifiants de code et d'API en anglais | Cohérent avec le mémoire; le code reste lisible par l'outillage Go |
| DEC-3 | Dossier des specs | `specs/` | Convention du mémoire. Vérifier d'abord que les skills `aiup-core` acceptent ce chemin; sinon prendre `docs/` sans discuter |
| DEC-4 | Base de données | MySQL 8, migrations `goose` | Fidèle au livre Go; `goose` est l'équivalent Go de Flyway |
| DEC-5 | Courtier | Apache Pulsar, schémas Avro | Fidèle au livre Go. Ne pas rouvrir avant l'itération 4 |
| DEC-6 | Identité en v1 | En-tête `X-User` et `X-Role` posés par le client, aucun fournisseur d'identité | Les règles de propriété et de rôle se testent sans authentification réelle |
| DEC-7 | Outillage | Go 1.25 ou plus, Docker, Claude Code avec `aiup-core`, `golangci-lint` v2; générateurs épinglés par directives `tool` de `go.mod` (Go 1.24+) et appelés `go tool <nom>` | Même intention que `scripts/tools.env` de Fibonacci, mais portée par la chaîne d'outils standard |
| DEC-8 | Hébergement local | Tout s'exécute dans WSL2 Ubuntu 24.04 : Docker Engine, Go, `make`, `protoc`; Windows reste le poste de travail (VS Code Remote-WSL, navigateur, `curl`). Solution de repli : Docker Desktop, dorsale WSL2 | Gratuit, sans produit supplémentaire, identique à la CI Ubuntu et aux conteneurs de production; détail en § 6 |
| DEC-9 | Emplacement du dépôt | `~/src/qsched` sur le système de fichiers Linux de la distribution; sauvegarde par le dépôt GitHub, jamais par OneDrive | Un dépôt sous `/mnt/c` et OneDrive est lent et sujet aux conflits de synchronisation |

## 1. Idéation

But : faire sortir les règles, le vocabulaire et les questions avant d'écrire une seule exigence. Méthode de Martinelli (ch. 3) adaptée au solo : histoires de domaine, tempête d'événements, puis résumé, extraction et classification par l'agent, puis revue humaine sans exception.

### 1.1 Histoires de domaine (trois, écrites à la main)

Format : acteur, action, objet, en phrases courtes numérotées. Amorces :

1. **Soumission QPU.** Une chercheuse soumet un travail demandant `QPU`; le système vérifie les capacités et le quota, le met en file, l'alloue au nœud QPU libre, l'agent l'exécute, la chercheuse reçoit une notification de fin.
2. **Maintenance sous charge.** Un opérateur met le seul nœud QPU en maintenance alors que trois travaux QPU sont en file; les travaux restent en file, aucun n'est alloué, l'opérateur voit pourquoi; à la sortie de maintenance, l'allocation reprend.
3. **Nœud perdu.** Un agent s'enregistre, bat le cœur, puis cesse; le système déclare le nœud perdu, le travail en cours passe à `FAILED`, une notification part, le travail n'est pas re-soumis automatiquement.

### 1.2 Tempête d'événements (amorce)

- Événements de domaine, au passé : `JobSubmitted`, `JobRejected`, `JobQueued`, `JobAllocated`, `JobStarted`, `JobCompleted`, `JobFailed`, `JobCancelled`, `NodeRegistered`, `HeartbeatReceived`, `NodeMarkedMaintenance`, `NodeMarkedReady`, `NodeLost`, `QuotaExceeded`, `NotificationSent`.
- Commandes : soumettre, annuler, enregistrer un nœud, battre le cœur, mettre en maintenance, allouer (déclenchée par l'horloge).
- Politiques : « quand `JobQueued` et un nœud `READY` compatible existe, allouer »; « quand trois battements manquent, `NodeLost` »; « quand `NodeLost` avec travail `RUNNING`, `JobFailed` ».
- Points chauds à trancher pendant l'atelier : le quota se compte-t-il en travaux ou en heures-nœud; un travail peut-il exiger deux capacités; l'annulation d'un travail `RUNNING` attend-elle l'agent; que fait-on d'un travail dont la capacité disparaît (dernier nœud QPU retiré).

### 1.3 Portée de la v1

- Dedans : les sept FR de la recommandation, agent de nœud factice (dort puis rapporte), trois capacités déclaratives `CPU`, `GPU`, `QPU`.
- Dehors, et écrit noir sur blanc dans la vision : exécution réelle de commandes, simulation de QPU, interface graphique, authentification réelle, facturation, multi-locataire, re-soumission automatique, politique d'allocation autre que FIFO par priorité.

### 1.4 Glossaire initial

| Terme (spec) | Identifiant (code) | Définition d'une ligne |
|---|---|---|
| Travail | `Job` | Demande d'exécution avec capacités requises, priorité et projet |
| Nœud | `Node` | Ressource d'exécution déclarant des capacités et un état |
| Capacité | `Capability` | Étiquette `CPU`, `GPU` ou `QPU` |
| Allocation | `Allocation` | Lien d'un travail à un nœud pour une exécution |
| Projet | `Project` | Regroupement porteur du quota |
| Battement | `Heartbeat` | Signal périodique de vie d'un agent |

### 1.5 Hypothèses et risques

`H-n` pour ce qu'on suppose (un agent par nœud, horloge unique, pas de reprise après panne de l'ordonnanceur en v1), `R-n` pour ce qui peut faire dérailler (dérive algorithmique, Pulsar lourd en local, sur-spécification). Chaque `R-n` reçoit une parade d'une ligne.

### 1.6 Sorties

- `specs/ideation.md` : histoires, événements, points chauds tranchés, questions ouvertes, hypothèses, risques. Notes brutes, non maintenues ensuite.
- `specs/vision.md` : une page, écrite à la main (Martinelli n'a pas de commande pour cette étape). Gabarit :

```markdown
# Vision — qsched
## Problème
## Buts d'affaires (3 à 5 puces)
## Acteurs
## Portée v1 / Hors portée
## Hypothèses (H-n) et risques (R-n)
## Lien avec le mémoire HPC/QPU (Q-2)
```

## 2. Specification Core

Commandes `aiup-core`, chacune suivie d'une revue humaine; la commande propose, la revue décide.

| Ordre | Commande | Fichier | Cible et points de revue |
|---|---|---|---|
| 2.1 | `/requirements` | `specs/requirements.md` | 7 FR, 3 NFR, 6 C de la recommandation. Chasser « normalement », « rapidement », « au besoin »; fusionner les doublons; un FR = une user story |
| 2.2 | `/entity-model` | `specs/entity-model.md` | Diagramme Mermaid `erDiagram` + tables d'attributs avec règles de validation. Ajouter deux `stateDiagram` (Travail, Nœud) : ce sont les règles les plus contestables, autant les figer ici |
| 2.3 | `/use-case-diagram` | `specs/use-cases.puml` | Sept UC, quatre acteurs; aucune relation include/extend sauf gain de clarté évident |
| 2.4 | à la main | `specs/dashboard.md` | Tableau 9-2 de Martinelli : UC, FR liée, statut, Code, Unit, Int, Régression. Tout en `Brouillon` |

Grille de revue avant de passer à l'étape 3 : aucune contradiction entre exigences; identifiants stables et jamais réutilisés; chaque FR pointe vers un UC du diagramme; chaque entité du modèle est nommée dans au moins une exigence; les états des machines à états correspondent aux événements de l'idéation.

## 3. Échafaudage Go (itération 1 de la feuille de route)

Chapitres 11 à 13 du livre Go. Bibliothèque standard seulement; aucune dépendance externe à cette étape. Tout se fait dans le terminal Ubuntu de WSL2, une fois la § 6 exécutée.

```bash
mkdir -p ~/src/qsched && cd ~/src/qsched && git init
mkdir -p jobs nodes notify deploy specs scripts
( cd jobs   && go mod init github.com/agbruneau/qsched/jobs )
( cd nodes  && go mod init github.com/agbruneau/qsched/nodes )
( cd notify && go mod init github.com/agbruneau/qsched/notify )
go work init ./jobs ./nodes ./notify
```

À produire :

- `CLAUDE.md` : les cinq règles de la recommandation, plus la liste de ce que l'agent lit avant d'implémenter (`specs/vision.md`, `requirements.md`, `entity-model.md`, l'UC visé, `CLAUDE.md`, le code existant du module).
- Repris de Fibonacci tels quels : `.golangci.yml`, `scripts/check.sh` et `check.ps1`, `scripts/tools.env`, le test d'architecture des imports, `.devcontainer`.
- `Makefile` à la racine : `fmt`, `lint`, `test`, `integration_test` (étiquette `integration`), `generate`, `build`; le Dockerfile de chaque service appelle `make build`, jamais `go build` directement.
- `jobs/cmd/jobs/main.go` : chargement de `configs/jobs.yaml` avec surcharge par variables `QSCHED_*`, `http.Server` avec `ReadTimeout` et `WriteTimeout`, routes `/` et `/healthz`, arrêt gracieux sur `SIGTERM` avec délai borné.
- `deploy/compose.yaml` : `mysql:8`, `apachepulsar/pulsar` en mode standalone, les trois services, organisés en profils `infra`, `hosted` et `obs` (§ 6.4). Pulsar peut rester commenté jusqu'à l'itération 4.
- `.gitattributes` : `* text=auto eol=lf`, repris de Fibonacci; un script `#!/bin/sh` en CRLF casse un `docker build`.
- `.github/workflows/ci.yml` : deux travaux, `unit` puis `integration`, le second seulement si le premier passe.

Critère de sortie : `make test` vert; `curl localhost:8080/healthz` répond 200, depuis Ubuntu et depuis Windows; un test envoie `SIGTERM` pendant une requête lente et vérifie que la réponse se termine et que le processus sort avec le code 0. Ce test porte l'étiquette `//go:build !windows` : Go sous Windows ne reçoit jamais `SIGTERM`, seulement `os.Interrupt`. Il devient plus tard la preuve de NFR-002.

## 4. Premier cycle SDD : UC-001 Soumettre un travail

| Sous-étape | Action | Sortie |
|---|---|---|
| 4.1 Migration | Dériver de `entity-model.md` les scripts `goose` : `project`, `node`, `job`, `allocation` | `jobs/migrations/0000n_*.sql`, appliqués par `make migrate` et par le conteneur de test |
| 4.2 Spécification | `/use-case-spec UC-001`, puis revue avec les trois tests d'exécutabilité de Martinelli (deux lecteurs, même résultat; un testeur en tire les critères; l'agent n'a rien à inventer). Passer le statut à `Approuvé` | `specs/use-cases/UC-001-soumettre-travail.md` |
| 4.3 Contrat | Écrire `jobs/api/openapi.yaml` à la main : `POST /jobs`, `GET /jobs/{id}`, erreurs 422 (capacité inconnue, quota) au format problem details. Générer avec `oapi-codegen` (cible `std-http`, sortie dans `internal/adapters/http/gen`) | `make generate` reproductible |
| 4.4 Implémentation | Prompt : « Implémente UC-001 ». Arborescence attendue dans `jobs/internal/` : `models`, `ports` (`JobRepository`, `NodeCatalog`, `EventPublisher`), `service`, `adapters/http`, `adapters/mysql`, `adapters/events` (factice en mémoire à cette itération) | Diff revu section par section contre l'UC : préconditions, étapes 2 et 3, postconditions d'échec |
| 4.5 Tests | Unitaires : BR-001 et BR-002 dans `service` avec ports factices. Intégration : `TestUC001_SoumettreTravail` avec sous-tests `principal`, `A1_capacite_inconnue`, `A2_quota_epuise`, MySQL via Testcontainers, conteneur partagé par `sync.Once`, étiquette `//go:build integration` | `go test -tags integration -run UC001 ./...` vert en local et en CI |
| 4.6 Revue et fusion | Une PR contenant l'UC, le contrat, le code et les tests. Ordre de lecture : spec, puis code, puis tests. Message de commit préfixé `UC-001:` | Ligne UC-001 du tableau de bord mise à jour |

Vérification que les tests gardent vraiment la règle : retirer temporairement la vérification du quota et constater que `A2_quota_epuise` échoue. Sans cet échec, le test ne protège rien.

## 5. Régime de croisière

- Tableau Kanban à six colonnes : Brouillon, Revu, Approuvé, Implémenté, Vérifié, Déployé. Un seul UC en cours d'implémentation à la fois.
- Ordre suggéré après UC-001 : UC-003 (consulter l'état, ferme la boucle REST), puis l'itération gRPC UC-004, UC-005, UC-007, puis l'itération événements UC-002, UC-006.
- Toute modification de comportement commence par l'UC; l'agent synchronise, ne régénère pas. Un correctif de bogue commence par vérifier si la spec est fausse ou incomplète.
- Après trois UC implémentés, extraire les prompts répétés en skills de dépôt (`implement`, `migration`, `integration-test`) : c'est la graine du greffon `aiup-go`. Avant, c'est prématuré.
- Une nouvelle dépendance externe par itération au plus, et seulement si un UC de l'itération l'exige.

## 6. Hébergement et exécution sur ce poste

Exigence ajoutée au catalogue : **C-007** — l'ensemble (trois services, base de données, courtier, tests d'intégration, observabilité) s'exécute sur ce seul poste, sans service infonuagique.

### 6.1 Inventaire du poste (vérifié le 2026-09-08)

| Élément | Constat |
|---|---|
| Matériel | Intel Core Ultra 9 275HX, 24 fils; 63,4 Go de RAM (45 Go libres); 571 Go libres sur `C:` |
| Windows | 11 Pro Insider Preview, build 26220; Hyper-V présent; `winget` 1.29 |
| Go côté Windows | Go 1.27.0 `windows/amd64`; `golangci-lint`, `gosec`, `govulncheck`, `mockgen`, `benchstat`, `staticcheck` dans `~/go/bin` |
| Éditeur | VS Code avec `golang.go`; aucune extension Remote-WSL ni Dev Containers |
| WSL2 | Ubuntu 24.04.4 LTS, noyau 6.18, 24 vCPU, 31 Go visibles (50 % par défaut), 947 Go libres sur son disque virtuel; utilisateur dans `sudo` |
| Absents partout | Docker Desktop, Docker Engine, Podman, Rancher, `make`, `protoc`, `buf`, MySQL, `goose`, `oapi-codegen`; ni Go ni `gcc` dans Ubuntu |
| Ports | 3306, 6650, 8080 à 8082, 9090, 3000 libres; aucun service MySQL, PostgreSQL ou Docker installé |

Non vérifié : l'état des fonctionnalités Windows (accès non élevé). WSL2 fonctionne, donc la plateforme de machine virtuelle est active, ce qui suffit à Docker Engine comme à Docker Desktop.

Conclusion : la puissance dépasse largement le besoin. Le seul manque bloquant est un moteur de conteneurs, exigé par Testcontainers, MySQL et Pulsar.

### 6.2 Topologie retenue (DEC-8)

```
Windows 11 : VS Code (Remote-WSL), navigateur, curl, Claude Code
    │ localhost partagé (WSL2 relaie les ports publiés)
    ▼
WSL2 Ubuntu 24.04 : Go, make, protoc, dépôt ~/src/qsched
    └── Docker Engine (systemd)
         ├── mysql:8            3306
         ├── pulsar standalone  6650, admin 8085
         ├── jobs / nodes / notify   (profil hosted)
         └── prometheus, grafana     (profil obs, itération 5)
```

Pourquoi ce choix plutôt que Docker Desktop :

- Aucun produit ni licence supplémentaire; Docker Engine s'installe par `apt` depuis le dépôt officiel.
- Parité exacte avec la CI Ubuntu et les conteneurs de production : `Makefile`, `SIGTERM`, chemins Linux, Testcontainers par `/var/run/docker.sock`.
- Le dev container de Fibonacci redevient utilisable, avec l'extension Dev Containers, pour les deux projets.
- Windows garde tout ce qu'il a : les deux Go cohabitent, le Go Windows sert aux outils déjà installés.

Repli si vous préférez rester en Go natif Windows : `winget install Docker.DockerDesktop` (dorsale WSL2). Testcontainers y fonctionne sans réglage par le tube nommé, et le démarrage à l'ouverture de session est intégré. En contrepartie : `make` à installer séparément (`winget install ezwinports.make`), pas de `SIGTERM` natif, et la licence Docker Personal, gratuite pour un usage personnel, à confirmer selon vos conditions.

### 6.3 Préparation du poste, une seule fois, dans Ubuntu

1. **systemd dans WSL** : écrire `[boot]\nsystemd=true` dans `/etc/wsl.conf`, puis `wsl --shutdown` depuis PowerShell et rouvrir Ubuntu.
2. **Docker Engine** : suivre la page officielle *Install Docker Engine on Ubuntu* (dépôt `apt` de Docker), installer `docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`, puis `sudo usermod -aG docker $USER`, se reconnecter, et vérifier par `docker run --rm hello-world`.
3. **Go** : archive officielle de go.dev dans `/usr/local/go` (1.25 ou plus; 1.27 pour aligner avec Windows), `PATH` mis à jour dans `~/.profile`. Le paquet `apt` d'Ubuntu 24.04 est trop ancien.
4. **Outils de construction** : `sudo apt install make build-essential protobuf-compiler`. Les générateurs Go (`goose`, `oapi-codegen`, `protoc-gen-go`, `protoc-gen-go-grpc`, `avrogen`, `golangci-lint`) sont épinglés par `go get -tool <module>@<version>` dans le `go.mod` de chaque module et appelés par `go tool <nom>` depuis le `Makefile` (DEC-7).
5. **VS Code** : depuis PowerShell, `code --install-extension ms-vscode-remote.remote-wsl`; ensuite `code .` depuis `~/src/qsched` dans Ubuntu ouvre le projet côté Linux.
6. **Mémoire WSL, facultatif** : `%UserProfile%\.wslconfig` avec `[wsl2]\nmemory=32GB\nswap=8GB` puis `wsl --shutdown`. Les défauts suffisent au projet; ce réglage protège seulement Windows si une expérience dérape.
7. **Test de fumée** : un test Go de dix lignes qui démarre le module `mysql` de `testcontainers-go`, exécute `SELECT 1` et s'arrête. S'il passe, l'étape 3 peut commencer.

### 6.4 Modes d'exécution

| Mode | Commande | Contenu | Usage |
|---|---|---|---|
| Développement | `docker compose --profile infra up -d` puis `go run ./cmd/jobs` dans le module | MySQL et Pulsar en conteneurs, services en processus natifs Ubuntu | Boucle rapide pendant l'implémentation d'un UC |
| Hébergé | `docker compose --profile hosted up -d --build` | Les trois services construits par leurs `Dockerfile`, `restart: unless-stopped`, volumes nommés `mysql-data` et `pulsar-data` | Système complet et persistant sur le poste, accessible de Windows sur `localhost` |
| Intégration | `make integration_test` | Conteneurs éphémères de Testcontainers, indépendants de compose | Preuve par UC, exécutable en parallèle du mode hébergé |
| Observabilité | `docker compose --profile obs up -d` | Prometheus et Grafana | Itération 5 |

Persistance entre sessions Windows : WSL ne démarre pas seul à l'ouverture de session. Une tâche planifiée « à l'ouverture de session » lançant `wsl.exe -d Ubuntu --exec /bin/sh -c "sleep infinity"` en fenêtre masquée démarre la distribution; avec systemd, le démon Docker suit et les conteneurs `unless-stopped` remontent. À valider au premier essai; c'est le seul point où Docker Desktop est plus simple.

### 6.5 Ports et budget

| Composant | Port hôte | Mémoire attendue |
|---|---|---|
| `jobs` HTTP | 8080 | moins de 100 Mo |
| `nodes` gRPC | 8082 | moins de 100 Mo |
| `notify` santé | 8084 | moins de 100 Mo |
| MySQL 8 | 3306 | 0,5 Go |
| Pulsar standalone | 6650, admin 8085 (8080 interne remappé) | 1 à 2 Go avec `PULSAR_MEM="-Xms512m -Xmx1g"` |
| Prometheus, Grafana | 9090, 3000 | 0,3 Go |

Total inférieur à 4 Go sur les 31 Go visibles par WSL; les tests d'intégration ajoutent au plus un second MySQL et un second Pulsar éphémères.

### 6.6 Conséquences pour les specs et le code

- C-007 au catalogue; NFR-002 se prouve en mode hébergé par `docker compose stop jobs`, qui envoie `SIGTERM` au conteneur.
- Le test d'arrêt gracieux et tout test envoyant un signal portent `//go:build !windows`. En CI, le travail Windows se limite à `go build` et aux tests unitaires; Testcontainers ne tourne que sur le coureur Ubuntu.
- Dépôt sur le système de fichiers Linux (DEC-9); accès occasionnel depuis l'Explorateur par `\\wsl.localhost\Ubuntu\home\<utilisateur>\src\qsched`.
- Après une mise en veille prolongée, l'horloge de WSL2 peut dériver de plusieurs minutes, ce qui fausse les délais de `NodeLost`; vérifier `date` et corriger par `sudo hwclock -s` avant de conclure à un bogue.
- Deux Go coexistent : celui d'Ubuntu construit et teste le projet; celui de Windows ne sert qu'aux outils déjà en place. Ne pas partager `GOPATH` entre les deux.

### 6.7 Critère de sortie de la préparation

`docker compose --profile infra up -d` laisse MySQL et Pulsar sains (`docker compose ps`); le test de fumée Testcontainers passe; après l'étape 3, `curl http://localhost:8080/healthz` répond 200 depuis PowerShell.

## Prêt et terminé, par cas d'utilisation

Prêt à implémenter : statut `Approuvé`; FR liées; au moins un flux alternatif; postconditions de succès et d'échec; règles `BR-` numérotées; aucun mot vague; entités nommées comme dans le modèle.

Terminé : code synchronisé avec l'UC; test unitaire par règle `BR-`; test d'intégration par scénario principal et par flux alternatif touchant la base; CI verte; ligne du tableau de bord cochée; tests des UC précédents toujours verts.
