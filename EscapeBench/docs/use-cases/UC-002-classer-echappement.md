# Use Case: Classer l'échappement

## Overview

**Use Case ID:** UC-002
**Use Case Name:** Classer l'échappement
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour chaque cellule d'une matrice, un verdict d'échappement reproductible accompagné de la raison rapportée par le compilateur et de sa catégorie
**Status:** Implemented

**Linked Requirements:** FR-002, NFR-001, NFR-002, C-001, C-003, C-005
**Linked Hypotheses:** H-006, H-009
**Entities:** Matrix, Cell, EscapeVerdict, Provenance

## Preconditions

- Une Matrix existe sur disque avec au moins une Cell (produite par UC-001).
- La toolchain Go installée satisfait C-001.
- Le harnais est intact : l'empreinte de `internal/harness/` est celle enregistrée avec la Matrix.

## Main Success Scenario

1. Le chercheur demande la classification d'une Matrix identifiée.
2. Le système vérifie l'empreinte du harnais et affiche la Provenance qu'il va consigner.
3. Le système compile chaque Cell avec le diagnostic d'échappement activé (C-003).
4. Le système extrait, pour chaque Cell, la présence ou l'absence d'un échappement et la ligne de raison du compilateur.
5. Le système attribue à chaque Cell une catégorie parmi `RETURN_POINTER`, `CLOSURE_CAPTURE`, `CHANNEL_SEND`, `CONTAINER_STORE`, `OTHER`, `NONE`.
6. Le système écrit un fichier de verdicts `results/escape/<matrixId>/<capturedAt>.json` contenant la Provenance et un EscapeVerdict par Cell.
7. Le système affiche le décompte par catégorie et le nombre de cellules classées `OTHER`.

## Alternative Flows

### A1: Le harnais a été modifié
**Trigger:** À l'étape 2, l'empreinte de `internal/harness/` diffère de celle enregistrée avec la Matrix.
**Flow:**
1. Le système refuse la classification et affiche les deux empreintes.
2. Le système n'écrit aucun fichier de verdicts.
3. Le cas d'utilisation se termine sans changement dans `results/`.

### A2: Une cellule ne compile pas
**Trigger:** À l'étape 3, la compilation d'une Cell échoue.
**Flow:**
1. Le système consigne la Cell avec le statut `COMPILE_ERROR` et le message du compilateur.
2. Le système poursuit la classification des autres cellules.
3. À l'étape 7, le système affiche le nombre de cellules en `COMPILE_ERROR`.

### A3: Verdicts déjà présents pour la même toolchain
**Trigger:** À l'étape 6, un fichier de verdicts existe déjà pour la même Matrix dont la Provenance porte les mêmes `goVersion`, `goos` et `goarch` (les autres champs de Provenance, dont `capturedAt`, ne sont pas comparés).
**Flow:**
1. Le système écrit le nouveau fichier sous un nouvel horodatage (NFR-004).
2. Le système compare les deux fichiers et affiche le nombre de cellules dont le verdict diffère.
3. Si ce nombre est supérieur à zéro, le système signale une violation de NFR-002.

## Postconditions

**Success:**
- Un fichier de verdicts existe pour la Matrix, avec une Provenance complète et un EscapeVerdict par Cell.
- Aucun fichier existant de `results/` n'a été modifié.

**Failure:**
- `results/` est inchangé.

## Business Rules

### BR-002-1: Une raison, une catégorie
Chaque EscapeVerdict dont `escapes` est vrai porte exactement une catégorie ; la ligne brute du compilateur est conservée telle quelle.

### BR-002-2: `OTHER` est un résultat, pas une erreur
Une raison d'échappement qui n'appartient à aucune des quatre causes de BEPG p. 238–242 est classée `OTHER` et compte comme observation pour H-006 ; elle n'est jamais forcée dans une autre catégorie.

### BR-002-3: Verdicts immuables
Un fichier de verdicts n'est jamais réécrit ; toute nouvelle classification crée un nouveau fichier.

## Notes de revue

- Les motifs de classification (expressions reconnues dans la sortie du compilateur) sont une implémentation ; ils vivent dans `internal/adapters/escape` et non dans ce cas d'utilisation.
- La liste des catégories est celle du modèle d'entités ; l'ajout d'une catégorie passe par une mise à jour de `entity-model.md` puis de ce cas d'utilisation.
