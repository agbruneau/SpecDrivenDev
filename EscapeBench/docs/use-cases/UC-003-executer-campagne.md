# Use Case: Exécuter une campagne de mesure

## Overview

**Use Case ID:** UC-003
**Use Case Name:** Exécuter une campagne de mesure
**Primary Actor:** Chercheur (ou Pipeline CI)
**Goal:** Obtenir, pour chaque cellule d'une matrice, des mesures répétées de temps, d'octets et d'allocations par opération, rejouables et accompagnées de leur provenance
**Status:** Deployed

**Linked Requirements:** FR-003, FR-006, NFR-001, NFR-003, NFR-004, NFR-005, C-001, C-002, C-003, C-005, C-006, C-008, C-009, C-010
**Révision :** 2026-09-10 — ajout de l'empreinte des critères (étape 3, BR-003-5) à la demande de UC-005 ; spécification modifiée avant tout code. Même jour : les Probe (FR-006) sont mesurées comme les Cell (étapes 4, 5, 8, A3, A4, BR-003-4) ; `Measurement.subjectId` remplace `cellId`. Revue pré-lancement du 2026-09-10 : BR-003-3 admet la transition de `status` dans `campaign.json` ; la précondition UC-002 ne vaut que si la Matrix contient une Cell.
**Linked Hypotheses:** H-001, H-002, H-003, H-004, H-005, H-007, H-008, H-010, H-011, H-012, H-013
**Entities:** Matrix, Cell, Probe, Campaign, Measurement, Provenance

## Preconditions

- Une Matrix existe avec au moins une Cell ou une Probe.
- Si la Matrix contient au moins une Cell, les verdicts d'échappement de cette Matrix existent pour la toolchain courante (UC-002).
- La toolchain satisfait C-001 ; aucune autre campagne n'est en cours sur la machine.

## Main Success Scenario

1. Le chercheur demande l'exécution d'une campagne sur une Matrix identifiée, avec un nombre de répétitions.
2. Le système vérifie que le nombre de répétitions satisfait NFR-003 et que l'empreinte du harnais est celle de la Matrix.
3. Le système crée une Campaign avec un identifiant nouveau, sa Provenance, l'empreinte du harnais et l'empreinte des critères des hypothèses liées (lus dans `docs/requirements.md`), au statut `RUNNING`.
4. Le système affiche l'identifiant de la Campaign, la Provenance, le nombre de Cell et le nombre de Probe à mesurer.
5. Le système mesure chaque Cell puis chaque Probe selon C-003 et enregistre, par sujet, une Measurement contenant exactement `count` valeurs de `nsPerOp`, `bytesPerOp` et `allocsPerOp`.
6. Le système écrit chaque Measurement dans `results/campaigns/<campaignId>/` dès qu'elle est complète.
7. Le système vérifie, en fin de campagne, que l'empreinte du harnais est inchangée.
8. Le système passe la Campaign au statut `COMPLETED` et affiche la durée totale, le nombre de Cell et le nombre de Probe mesurées.

## Alternative Flows

### A1: Nombre de répétitions insuffisant
**Trigger:** À l'étape 2, le nombre de répétitions demandé est inférieur au minimum de NFR-003.
**Flow:**
1. Le système refuse la campagne et affiche le minimum exigé.
2. Aucune Campaign n'est créée.

### A2: Le harnais est modifié pendant la campagne
**Trigger:** À l'étape 7, l'empreinte du harnais diffère de celle enregistrée à l'étape 3.
**Flow:**
1. Le système passe la Campaign au statut `ABORTED` et consigne les deux empreintes.
2. Le système conserve les Measurement déjà écrites, marquées comme appartenant à une campagne abandonnée.
3. Le système affiche que la campagne est invalide pour tout verdict (C-005).

### A3: Un sujet échoue à la mesure
**Trigger:** À l'étape 5, la mesure d'une Cell ou d'une Probe échoue (compilation, panic, délai dépassé).
**Flow:**
1. Le système consigne le sujet avec le statut `FAILED` et le message d'erreur.
2. Le système poursuit la mesure des autres sujets.
3. À l'étape 8, le système affiche le nombre de sujets en `FAILED` ; la Campaign est `COMPLETED` si au moins un sujet a été mesuré.

### A4: Reprise après interruption
**Trigger:** À l'étape 1, le chercheur désigne une Campaign existante au statut `RUNNING` dont le processus n'est plus actif.
**Flow:**
1. Le système vérifie l'empreinte du harnais contre celle de la Campaign.
2. Le système reprend au premier sujet (Cell ou Probe) sans Measurement complète.
3. Le scénario principal continue à l'étape 5.

## Postconditions

**Success:**
- Une Campaign au statut `COMPLETED` existe, avec une Provenance complète, l'empreinte des critères des hypothèses liées et une Measurement par Cell et par Probe mesurée.
- Aucun fichier de `results/` antérieur à la campagne n'a été modifié.
- `internal/harness/` est identique à son état de l'étape 3.

**Failure:**
- La Campaign est absente (A1) ou au statut `ABORTED` (A2) ; les fichiers déjà écrits restent en place et sont identifiables comme invalides.

## Business Rules

### BR-003-1: Harnais figé
Une Campaign n'est valide que si l'empreinte de `internal/harness/` est identique au début et à la fin de l'exécution.

### BR-003-2: Provenance obligatoire
Aucune Measurement n'est écrite sans une Campaign portant une Provenance complète (NFR-001).

### BR-003-3: Écriture seule
Le runner de campagne crée des fichiers dans `results/`. Il ne modifie ni ne supprime aucun fichier existant, à deux exceptions nommées : le `campaign.json` de la campagne courante, dont seul le champ `status` passe de `RUNNING` à `COMPLETED` ou `ABORTED`, et `results/.campaign-lock`, créé au démarrage et retiré à la fin. Aucune Measurement, aucun fichier de comparaison, de verdicts ou d'échappement n'est réécrit.

### BR-003-4: Un sujet, un processus
Chaque Cell et chaque Probe est mesurée dans un processus `go test` distinct afin qu'aucun état du runtime ne se propage d'un sujet au suivant.

### BR-003-5: Critères gelés à la création
L'empreinte des critères de réfutation est calculée à la création de la Campaign ; UC-005 refuse tout verdict si les critères ont changé depuis.

## Notes de revue

- Clôture du 2026-09-10 : statut porté à `Deployed`, que le tableau de bord définit comme « campagne exécutée et rapport publié ». Ce qui l'établit : la suite complète au vert sous `-race -shuffle=on`, une couverture de statements de 93,6 pour cent, le contrôle des hooks et le contrôle des spécifications sans constatation, une revue contradictoire de trente-cinq constats suivie d'une rédaction contradictoire en deux passes, chaque constat et chaque version ayant été soumis à des vérificateurs chargés de les réfuter, et six campagnes menées de bout en bout dont les rapports sont publiés à la racine du dépôt. `CLAUDE.md` réserve le passage `Reviewed → Approved` à une décision humaine ; il a été assumé par l'agent sur mandat explicite, ce que consigne la décision D-01.


- Révision du 2026-09-10, satisfaction de C-010 : chaque mesure est encadrée de deux relevés des temps processeur de la machine, et la fraction d'occupation des cœurs non mesurés est consignée dans la Measurement. Le temps retranché est celui de tout l'arbre de processus, `go test` compilant, liant puis exécutant dans des processus enfants ; sans cette agrégation, le travail légitime de la campagne gonflerait la fraction et la garde de H-013 refuserait les campagnes saines. Une plateforme qui n'expose pas ces compteurs laisse le champ absent, ce qui rend H-013 non concluante plutôt que fausse.

- Révision du 2026-09-10, satisfaction de C-009 : une campagne dont la liste gelée contient H-007 est refusée sur une matrice à réplicats. H-007 indexe ses Comparison par taille et n'en retient qu'une, arbitrairement ; sur une série répliquée elle jugerait donc un réplicat tiré de l'ordre du fichier, sans erreur ni trace. Le refus est une précondition, donc aucune Campaign n'est créée. C'est H-012 qui lit une série répliquée. Une campagne sans sélection explicite retient tout le catalogue, donc H-007 : sur une matrice à réplicats, `--hypotheses` devient obligatoire, et c'est le comportement voulu.

- Les drapeaux exacts (`-benchmem`, `-count`, `-cpu`) relèvent de C-003, pas de ce cas d'utilisation.
- La détection du CPU et de la version de Go est un adapter (`internal/adapters/provenance`) ; le service ne fait que consigner ce qu'il reçoit.
- Les pannes techniques (disque plein, toolchain absente) sont traitées par l'implémentation et n'apparaissent pas comme flux alternatifs.
