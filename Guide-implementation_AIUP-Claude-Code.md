# Guide d'implémentation — *Spec-Driven Development* (AI Unified Process) avec Claude Code pour les projets d'exploration Go

**Sources :** Simon Martinelli, *Spec-Driven Development: From Specs to Code with AI Agents*, Apress Pocket Guides, 2026, ISBN 979-8-8688-2851-5, DOI 10.1007/979-8-8688-2851-5, 152 p. (ci-après **SDD**) ; Saeed Shahsavan, *Building Enterprise Projects with Go*, Apress, 2026 (ci-après **BEPG**). Pages citées = folios imprimés.

**Portée :** méthode commune aux huit projets de `Projets-candidats_Building-Enterprise-Projects-with-Go.md`. Le dossier `EscapeBench/` fournit l'exemple travaillé (noyau de spécification du projet P1).

**Date :** 2026-09-10 · **Longueur :** ~3 100 mots.

---

## 1. Conclusion

Chaque projet d'exploration est conduit comme un système spécifié selon l'AI Unified Process : un dossier `/docs` faisant autorité (vision, catalogue d'exigences, modèle d'entités, cas d'utilisation système), un fichier `CLAUDE.md` court portant les règles de construction issues de BEPG, et des commandes Claude Code qui implémentent ou synchronisent **un cas d'utilisation identifié** — jamais une demande libre (SDD, p. 66). L'adaptation propre aux bancs de réfutation tient en trois points : les affirmations du livre deviennent des hypothèses identifiées (`H-###`) rattachées aux cas d'utilisation ; les règles métier deviennent des invariants de protocole ; le tableau de bord de spécification (SDD, p. 142–143) devient un tableau de bord de réfutation. Aucun *plugin* Go n'existe dans le *marketplace* AIUP (*Confirmé*, voir §6) : le cœur `aiup-core` s'installe tel quel, la couche Go se construit en *skills* de projet.

## 2. Hypothèses de travail

- Le lecteur travaille seul, en Go, avec Claude Code comme exécutant ; la revue par un tiers (SDD, p. 149, 151) est remplacée par une revue en deux temps décrite au §7 — *adaptation assumée*, le livre exige une revue humaine (p. 148).
- Les commandes citées sont celles du *marketplace* AIUP tel que consulté le 2026-09-10 ; le livre lui-même précise que « the process [does not] depend on specific command names » (SDD, p. 114).
- Marqueurs : *Confirmé* = dans SDD/BEPG ou vérifié en source primaire ; *Adaptation* = écart délibéré par rapport au livre, justifié ; *À vérifier* = non vérifiable ici.

## 3. Ce que prescrit l'AIUP (rappel fidèle)

**Principes (SDD, p. 14–15).** Spécifications exécutables (assez précises pour piloter génération, tests et validation, sans langage formel) ; minimalisme (trois artefacts, dans un ordre fixe) ; traçabilité par construction (identifiants stables référencés dans artefacts, code, tests et *commits*) ; collaboration IA sous contrôle humain ; tout est versionné dans Git.

**Artefacts (SDD, p. 17–20).** Catalogue d'exigences en table avec `FR-###`, `NFR-###`, `C-###` ; modèle d'entités (noms du domaine, relations, sert de glossaire, diagramme-as-code Mermaid ou PlantUML) ; cas d'utilisation système `UC-###`, un fichier Markdown chacun. Structure type :

```
/specs            (SDD p. 20)          /docs             (SDD p. 68, 114–115 ; marketplace)
  entity_model.md                        vision.md
  requirements_catalog.md                requirements.md
  use_cases.puml                         entity-model.md
  /usecases                              use_cases.puml
    UC-001_assign_task.md                /use-cases
                                           UC-001-create-task.md
```

Le livre utilise les deux dispositions ; ce guide retient `/docs`, celle du dépôt d'étude de cas et des *skills* du *marketplace*.

**Cycle (SDD, p. 23–24).** *Specify → Generate → Validate → Review → Refine*, répété « sometimes several times per day », sans phases ni transferts.

**Rédaction exécutable (SDD, ch. 4).** Trois tests d'exécutabilité (p. 48) : deux lecteurs imaginent des issues différentes ; un testeur ne peut dériver de critères d'acceptation ; l'IA doit inventer des règles. Comportement observable seulement (p. 49–50) ; bannir « normally », « quickly », « as needed », « should » (p. 51) ; une règle par énoncé (p. 53) ; préconditions = ligne de départ, pas validations (p. 55) ; flux principal = étapes ordonnées, observables, alternance acteur/système (p. 55–57) ; flux alternatifs déclenchés « At step N » (p. 57–58), variantes métier distinctes des pannes techniques (p. 58) ; postconditions de succès **et** d'échec (p. 58–59) ; règles métier identifiées `BR-…` (p. 59) ; « just enough », pas de *big design up front* (p. 60–62).

**Exécution (SDD, ch. 5).** L'agent lit le cas d'utilisation, les exigences liées, le modèle d'entités, `CLAUDE.md` ou `AGENTS.md`, le code et les tests existants (p. 68–69). Les éléments du cas d'utilisation se projettent sur le code : préconditions → vérifications, étapes → chemin nominal, alternatives → branches, postconditions → résultat attendu, règles → logique et assertions (p. 71, 84). Le fichier de consignes reste court ; ce qui ne vaut que pour un flux va dans un *skill* (p. 72). Ordre de revue : spécification, comportement, implémentation, conformité code/spec, couverture des flux (p. 76). Synchroniser plutôt que régénérer : « In long-living systems, synchronization is usually the safer default » (p. 79).

**Tests (SDD, ch. 6).** Scénario principal → test positif ; chaque alternative → son scénario ; chaque règle → assertion explicite (p. 86). *Test Trophy* : analyse statique à la base, peu de tests unitaires, large couche d'intégration, peu de bout-en-bout (p. 88) ; un test d'intégration par scénario principal comme base (p. 90). Couverture = complétude de la spécification, pas lignes (p. 96). Trois questions de revue d'un test généré (p. 95) : l'assertion correspond-elle à la postcondition ; le *setup* ne reflète-t-il que les préconditions ; le test échoue-t-il vraiment si la règle est violée. « AI generates tests but cannot validate them » (p. 97).

**Gouvernance (SDD, ch. 9–10).** Quatre règles (p. 139) : le comportement ne change jamais sans mise à jour préalable de la spécification ; les identifiants ne sont jamais réutilisés ; les cas d'utilisation décrivent le comportement observable ; chaque *bounded context* possède son noyau de spécification. Kanban par cas d'utilisation : Draft → Review → Approved → Implemented → Verified → Deployed (p. 140). Modes d'échec (p. 147) : vague, sur-spécification, traçabilité affaiblie, prompts sans identifiant, spécifications figées.

## 4. Adaptation aux bancs de réfutation

Les huit projets ne sont pas des applications métier : leur acteur principal est le chercheur, leur « système » est un banc de mesure ou un outil d'analyse, et leur finalité est un verdict sur des affirmations de BEPG. L'AIUP s'applique sans modification de structure ; quatre conventions sont ajoutées, chacune marquée *Adaptation*.

| Élément AIUP | Lecture pour un banc de réfutation | Statut |
|---|---|---|
| Acteur primaire | `Chercheur` (invoque le banc) ; acteur secondaire `Pipeline CI` pour les campagnes non surveillées | *Adaptation* (le livre parle d'acteurs métier, p. 36) |
| `FR-###` | Capacités du banc : générer une matrice, exécuter une campagne, classer, comparer, produire un rapport | *Confirmé* |
| `NFR-###` | Reproductibilité, provenance des mesures (version Go, `GOOS/GOARCH`, CPU), durée de campagne | *Confirmé* |
| `C-###` | Contraintes de protocole : toolchain, bibliothèques autorisées, drapeaux de mesure imposés, layout BEPG | *Confirmé* |
| `H-###` | **Hypothèse à éprouver**, avec source (livre, page), énoncé réfutable et critère de réfutation ; chaque `H` est liée à ≥ 1 `UC` | *Adaptation* — quatrième type d'entrée du catalogue ; contredit le minimalisme (p. 14) mais évite d'enfouir les hypothèses dans la prose, ce que le livre proscrit pour les règles (p. 59) |
| `BR-…` | Invariants de protocole observables : « le harnais n'est pas modifié pendant une campagne », « `-count` ≥ 20 », « toute mesure porte sa provenance » | *Confirmé* (règles explicites, p. 59) |
| Postconditions | Artefacts produits (fichiers de résultats, rapport) et état du dépôt (aucune modification hors `results/`) | *Confirmé* (état final vérifiable, p. 58–59) |
| Flux alternatifs vs pannes | Variante métier : cellule non compilable, résultat non concluant ; panne technique : Docker absent, broker injoignable — traitées hors cas d'utilisation | *Confirmé* (p. 58) |
| Tableau de bord | `docs/dashboard.md` : Table 9-2 du livre, plus une table `H-###` → verdict (Confirmée / Infirmée / Non concluante) → `UC` → tests | *Adaptation* de p. 142–143 |
| Statut « Deployed » | = campagne exécutée et rapport publié dans `results/` | *Adaptation* |

Ce qui ne change pas : un cas d'utilisation décrit ce que le chercheur fait et ce que le banc rend observable (fichiers, sorties, codes de retour). Les drapeaux du compilateur, la structure des paquets ou la bibliothèque de *benchmark* sont des contraintes (`C-###`) ou des règles de `CLAUDE.md`, jamais des étapes de flux (sur-spécification, SDD p. 147).

## 5. Structure de dépôt d'un projet

Un dépôt par projet (P1 → `EscapeBench`, etc.), assemblé selon BEPG ch. 4 et 14 (p. 68–70, 368–369) et SDD ch. 8 (p. 114–115) :

```
<projet>/
  CLAUDE.md                     règles de construction (court)
  docs/
    vision.md
    requirements.md             FR / NFR / C / H
    entity-model.md             Mermaid + tables d'attributs
    use_cases.puml
    use-cases/UC-001-<slug>.md
    dashboard.md                état des UC et verdicts des H
  cmd/<projet>/main.go          composition root (BEPG p. 361, 414)
  internal/models               stdlib seulement (BEPG p. 366, 369)
  internal/service              cas d'utilisation, sans I/O (BEPG p. 367)
  internal/ports                interfaces attendues par le core
  internal/adapters/...         compilateur, système de fichiers, conteneurs, broker
  results/<campagne>/           mesures brutes + provenance, jamais réécrites
  .claude/skills/<skill>/SKILL.md
  Makefile                      fmt · vet · test · integration_test · bench (BEPG p. 324–325)
```

Les projets sans composante d'exécution longue (P3 LeakLab, P4 HexaGuard) gardent la même structure avec `results/` réduit aux rapports.

## 6. Outillage Claude Code

**Cœur AIUP (*Confirmé*, dépôt `AI-Unified-Process/marketplace` et `unifiedprocess.ai/tools.html`, consultés le 2026-09-10).**

```
/plugin marketplace add ai-unified-process/marketplace
/plugin install aiup-core
```

`aiup-core` fournit `/requirements`, `/entity-model`, `/use-case-diagram`, `/use-case-spec UC-###`, `/reverse-engineer` (et, selon la page outils, `/test-case`) ; ces *skills* « operate only on /docs artifacts » (SDD, p. 103). *Plugins* de pile disponibles : `aiup-vaadin-jooq`, `aiup-angular-jpa`, `aiup-blazor-dotnet`, `aiup-nestjs-nextjs`. **Aucun plugin Go.** Le livre indique qu'un nouveau *stack plugin* se crée sans modifier l'AIUP (p. 103–104) ; il n'en donne aucun exemple Go.

**Couche Go à construire (*Adaptation*).** Skills de projet dans `.claude/skills/`, réutilisables ensuite comme *plugin* `aiup-go` si plusieurs dépôts les partagent (SDD, p. 107 : ajouter l'outillage « when you feel the pain that the tool solves »). Chaque skill déclare ce qu'il lit, ce qu'il produit et ses contraintes (SDD, p. 73).

| Skill | Lit | Produit | Contraintes issues de BEPG |
|---|---|---|---|
| `/implement UC-###` | UC, exigences liées, `entity-model.md`, `CLAUDE.md`, code existant | Code dans `internal/` + tests unitaires nommés par flux ; mode synchronisation si le UC existe déjà | Layout hexagonal (p. 368–369) ; erreurs enveloppées `%w`, `errors.Is/As` aux bords (p. 177–190) ; `context.Context` premier paramètre de tout I/O (p. 539–545) |
| `/go-test UC-###` | UC, code du UC | Tests table-driven, sous-tests nommés `UC###/<flux>` ; `testing/synctest` pour tout comportement temporel ou concurrent | Table-driven + `t.Run` (p. 213–216) ; `-race` et `-shuffle=on` (p. 231) ; horloge injectée plutôt qu'attentes réelles (p. 225–227) |
| `/integration-test UC-###` | UC, ports/adapters | Tests sous `//go:build integration_test`, conteneur partagé par `sync.Once` | Pattern « one container, many tests » (p. 433–435) ; images figées, jamais `latest` (p. 444) |
| `/bench H-###` | H, UC de mesure liés | `Benchmark*` avec `b.ReportAllocs()`, cible `make bench`, sortie `results/` | `-benchmem` (p. 219, 232) ; `-count` fixé par un `C-###` ; *setup* hors boucle `b.N` (p. 219) |
| `/migration` | `entity-model.md` | Scripts `V###__<slug>.sql` + rollback (projets avec base : P4, P8) | Liquibase/Flyway, migrations immuables (p. 423–426) |
| `/spec-coverage` | UC, tests | Matrice UC × (flux, règle) → test ; signale les flux sans test | Couverture = complétude de la spec (SDD, p. 96) |
| `/refute H-###` | `results/`, H | Verdict Confirmée / Infirmée / Non concluante + mise à jour de `dashboard.md` | Critère de réfutation écrit dans H avant la campagne ; jamais ajusté après coup |

**Fichier `CLAUDE.md`.** Court, stable, versionné (SDD, p. 72). Gabarit minimal :

```markdown
# Project Guidelines — <projet>
- /docs fait autorité : tout changement de comportement commence par la spécification.
- Go 1.25+, bibliothèque standard ; dépendances externes uniquement si couvertes par un C-###.
- Layout : cmd/<projet> (composition root), internal/{models,service,ports,adapters}.
  models n'importe que la stdlib ; service n'importe ni I/O ni adapters.
- Toute I/O prend un context.Context en premier paramètre ; erreurs enveloppées avec %w.
- Tests : table-driven, sous-tests nommés d'après le UC et le flux ; -race et -shuffle=on en CI.
- Référencer l'ID du cas d'utilisation dans le nom des tests et en commentaire du code produit.
- Synchroniser, ne pas régénérer ; changements limités au UC demandé.
- results/ n'est jamais modifié par un skill d'implémentation.
```

**Hooks.** `PostToolUse` sur édition de fichiers Go : `go vet ./...` puis `go test -race ./...` ; sur édition sous `docs/use-cases/` : contrôle de la présence des sections obligatoires et de la syntaxe des identifiants. Le contrôle d'architecture (`hexaguard`, projet P4) devient un hook dès qu'il existe.

**Sous-agents.** Un sous-agent « réviseur » reçoit le UC et le *diff*, sans accès au raisonnement de l'agent implémenteur, et répond aux cinq questions de l'ordre de revue (SDD, p. 76) et aux trois questions sur les tests (p. 95). Voir §7.

**MCP.** Les deux livres n'en prescrivent aucun pour Go. Pour la bibliothèque standard, `go doc` suffit et reste vrai par construction. Pour `testcontainers-go`, `pulsar-client-go`, `oapi-codegen`, un serveur de documentation générique (le *marketplace* configure `context7` dans ses `.mcp.json` — *Confirmé*) est une option, *à vérifier* selon la pile.

**Exécution non surveillée.** `claude -p` planifié pour les campagnes longues (P2, P6, P8), avec `CLAUDE.md` interdisant toute modification du harnais ; la sortie attendue est un dossier `results/<campagne>/` et une ligne dans `dashboard.md`, pas un changement de code.

## 7. Cycle de travail par cas d'utilisation

Ordre invariable, aligné sur SDD p. 23–24 et p. 129–130.

1. **Specify.** Écrire ou modifier le UC (`/use-case-spec UC-###`), les `H-###` liées et, si le domaine change, `entity-model.md`. Vérifier les trois tests d'exécutabilité (p. 48). Statut `Draft` → `Review`.
2. **Review de la spécification** (avant tout code). Premier temps : sous-agent réviseur ; second temps : relecture humaine à froid, idéalement à une autre session que celle de rédaction — *Adaptation* de la règle « reviewed by someone who did not write it » (p. 151). Statut `Approved`.
3. **Generate.** `/implement UC-###` (création) ou `/implement UC-###` sur UC modifié (synchronisation ; le *diff* doit être proportionnel au changement de spec, p. 77–78). Puis `/go-test`, `/integration-test` ou `/bench` selon le UC.
4. **Validate.** `make vet test` ; `make integration_test` si adapters touchés ; `/spec-coverage UC-###` doit rapporter zéro flux et zéro règle sans test.
5. **Review du code.** Cinq questions (p. 76) ; pour chaque test généré, les trois questions (p. 95), la troisième exécutée réellement : casser la règle, constater l'échec du test. Statut `Implemented` puis `Verified`.
6. **Refine.** Toute divergence découverte se corrige d'abord dans la spécification (p. 130 : « If the spec is wrong, fix the spec first »). Un *bug* est « a specification incomplete or code that does not match it » (p. 130).
7. **Campagne** (projets de mesure). `/bench H-###` en exécution non surveillée ; `/refute H-###` ; `dashboard.md` mis à jour ; statut `Deployed`.

Critère de sortie d'un UC : approuvé, synchronisé avec le code, protégé par des tests couvrant flux principal et alternatifs (SDD, p. 142).

## 8. Traçabilité dans le code Go

Conventions retenues (*Adaptation* des principes p. 21–23 ; le livre montre une annotation Java `@UseCase(id, scenario)`, p. 96) :

- Nom des tests : `TestUC003_MainFlow`, `TestUC003_A1_HarnessModified`, `TestUC003_BR1_ProvenanceRecorded` ; sous-tests table-driven nommés par le flux ou la règle.
- Commentaire d'en-tête des fonctions de service : `// UC-003 Exécuter une campagne — étapes 2–5.`
- *Commits* : `UC-003: …` ou `H-002: …` en préfixe ; un UC par *pull request*, la spécification et le code dans le même *diff* (p. 75).
- `docs/dashboard.md` régénéré par `/spec-coverage` et `/refute`, jamais édité à la main.
- Identifiants jamais réutilisés ; un UC retiré garde son fichier avec `Status: Retired`.

## 9. Pièges spécifiques et garde-fous

| Piège (SDD, p. 147) | Forme prise dans un banc de mesure | Garde-fou |
|---|---|---|
| Vague | « le banc mesure la performance » ; « campagne suffisante » | Chaque `H` porte un critère numérique de réfutation ; chaque NFR une valeur |
| Sur-spécification | Drapeaux `-gcflags`, structure des paquets ou algorithme statistique dans les étapes du UC | Les déplacer en `C-###` ou dans `CLAUDE.md` ; le UC ne décrit que l'observable |
| Traçabilité affaiblie | Résultats dans `results/` sans lien vers `H`/`UC` ; verdict dans un message de *commit* | Provenance obligatoire dans chaque fichier de résultats (BR) ; verdict uniquement via `/refute` |
| Prompt sans identifiant | « optimise le harnais » | Refus par `CLAUDE.md` : toute demande d'implémentation nomme un `UC-###` ou un `H-###` |
| Spécification figée | Ajustement du critère de réfutation après avoir vu les mesures | Critère gelé au statut `Approved` ; tout changement crée une nouvelle `H` (identifiants jamais réutilisés) |

Le livre exempte du plein processus les prototypes et scripts jetables (p. 150). Dans ce portefeuille, seule la reconnaissance initiale d'un outil (ex. vérifier que `-gcflags=-m` produit une sortie parsable) relève de cette exemption ; elle ne dépasse pas une session et ne produit aucun résultat cité.

## 10. Compromis, alternative, conditions de renversement

**Compromis principal.** L'AIUP ajoute un coût fixe par cas d'utilisation (rédaction, revue en deux temps, synchronisation) que le livre estime rentable dès la première ambiguïté évitée (p. 149) ; pour des bancs de quelques centaines de lignes, ce coût peut dépasser celui du code. Le gain attendu n'est pas la vitesse mais la réfutabilité : critères gelés avant mesure, provenance, diffs proportionnés.

**Alternative.** Appliquer l'AIUP au niveau « use case specifications alone » (SDD, p. 108) : pas de catalogue ni de modèle d'entités, seulement `docs/use-cases/` et `CLAUDE.md`. Le livre affirme qu'on obtient « most of the benefit » ; on perd la liaison `H` → `UC` et le tableau de bord.

**Conditions qui renversent la recommandation.**

- Projet réduit à une campagne unique et non répétée (ex. P6 restreint à trois paramètres) : l'alternative suffit.
- Plusieurs dépôts partageant les mêmes skills : promouvoir `.claude/skills/` en *plugin* `aiup-go` versionné (p. 106).
- Publication visée : le catalogue `H-###` avec critères gelés devient obligatoire, car il constitue la pré-déclaration des hypothèses.

## 11. Références

- Martinelli, S. *Spec-Driven Development: From Specs to Code with AI Agents*. Apress Pocket Guides, 2026. DOI 10.1007/979-8-8688-2851-5. PDF analysé : `Spec-Driven Development.pdf` (167 p.).
- Shahsavan, S. *Building Enterprise Projects with Go*. Apress, 2026. DOI 10.1007/979-8-8688-2370-1.
- AI Unified Process Marketplace : https://github.com/AI-Unified-Process/marketplace (consulté le 2026-09-10).
- AI Unified Process, page outils : https://unifiedprocess.ai/tools.html (consulté le 2026-09-10).
- Dépôt d'étude de cas du livre SDD : https://github.com/ai-unified-process/task-manager (cité SDD p. 112 ; non consulté).

## Annexe — Écarts relevés dans le livre SDD

| Point | Constat |
|---|---|
| Nom des commandes | Le livre écrit `/entity_model` (p. 117) et `/browerless-test`, `/playwright_-est` (p. 130) ; le *marketplace* expose `/entity-model`, `/browserless-test`, `/playwright-test`. Le livre prévient que les noms sont illustratifs (p. 114). |
| Emplacement des artefacts | `/specs` avec `requirements_catalog.md` (p. 20) contre `/docs` avec `requirements.md` (p. 68, 114) ; ce guide retient `/docs`. |
| Extraction | Les tableaux 2-1, 5 (« Regeneration vs. Synchronization »), 7-1 et 10-1 sont partiellement illisibles dans le texte extrait ; seules les cellules lisibles ont été utilisées. |
