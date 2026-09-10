# Use Case: Exécuter une campagne de mesure

## Overview

**Use Case ID:** UC-003
**Use Case Name:** Exécuter une campagne de mesure
**Primary Actor:** Chercheur (ou Pipeline CI)
**Goal:** Obtenir, pour chaque cellule d'une matrice, des mesures répétées de temps, d'octets et d'allocations par opération, rejouables et accompagnées de leur provenance
**Status:** Reviewed

**Linked Requirements:** FR-003, FR-006, NFR-001, NFR-003, NFR-004, NFR-005, C-001, C-002, C-003, C-005, C-006
**Révision :** 2026-09-10 — ajout de l'empreinte des critères (étape 3, BR-003-5) à la demande de UC-005 ; spécification modifiée avant tout code. Même jour : les Probe (FR-006) sont mesurées comme les Cell (étapes 4, 5, 8, A3, A4, BR-003-4) ; `Measurement.subjectId` remplace `cellId`.
**Linked Hypotheses:** H-001, H-002, H-003, H-004, H-005
**Entities:** Matrix, Cell, Probe, Campaign, Measurement, Provenance

## Preconditions

- Une Matrix existe avec au moins une Cell ou une Probe.
- Les verdicts d'échappement de cette Matrix existent pour la toolchain courante (UC-002).
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
Le runner de campagne crée des fichiers dans `results/` et n'en modifie ni n'en supprime aucun.

### BR-003-4: Un sujet, un processus
Chaque Cell et chaque Probe est mesurée dans un processus `go test` distinct afin qu'aucun état du runtime ne se propage d'un sujet au suivant.

### BR-003-5: Critères gelés à la création
L'empreinte des critères de réfutation est calculée à la création de la Campaign ; UC-005 refuse tout verdict si les critères ont changé depuis.

## Notes de revue

- Les drapeaux exacts (`-benchmem`, `-count`, `-cpu`) relèvent de C-003, pas de ce cas d'utilisation.
- La détection du CPU et de la version de Go est un adapter (`internal/adapters/provenance`) ; le service ne fait que consigner ce qu'il reçoit.
- Les pannes techniques (disque plein, toolchain absente) sont traitées par l'implémentation et n'apparaissent pas comme flux alternatifs.
