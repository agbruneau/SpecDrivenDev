# Use Case: Produire les verdicts

## Overview

**Use Case ID:** UC-002
**Use Case Name:** Produire les verdicts
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour une campagne, le verdict de chaque hypothèse par son seul critère gelé et la matrice de détectabilité
**Status:** Approved

**Linked Requirements:** FR-005, FR-006, NFR-004, C-008
**Linked Hypotheses:** H-001 à H-014
**Entities:** Run, Observation, Cell, ProbeResult, Case, Hypothesis, Verdict

## Preconditions

- `results/runs/<runId>.json` existe.

## Main Success Scenario

1. Le chercheur demande les verdicts d'une campagne.
2. Le système lit la campagne, recalcule l'empreinte du critère de chaque hypothèse depuis `docs/requirements.md` et la compare à celle de la campagne.
3. Le système regroupe les observations en cellules et en déduit l'issue majoritaire, la détection et l'instabilité de chacune.
4. Pour chaque hypothèse, le système applique son évaluateur et obtient un verdict, un rationale chiffré et la liste des cellules ou bras cités.
5. Le système écrit `results/verdicts/<runId>-<horodatage>.json` et `results/verdicts/<runId>-<horodatage>.md` (tableau des verdicts, matrice de détectabilité, médianes des sondes), en écriture exclusive.
6. Le système affiche le tableau des verdicts.

## Alternative Flows

### A1: Critère modifié depuis la campagne
**Trigger:** À l'étape 2, une empreinte diffère.
**Flow:**
1. Le système nomme les hypothèses dont le critère a changé et refuse ; aucun fichier n'est écrit.
2. Le chercheur crée une hypothèse successeur (identifiant neuf) et lance une nouvelle campagne.

### A2: Données manquantes
**Trigger:** À l'étape 4, une cellule ou un bras exigé par un critère est absent de la campagne.
**Flow:**
1. Le verdict de l'hypothèse est `INCONCLUSIVE`, et le rationale nomme ce qui manque.
2. Le scénario continue pour les autres hypothèses.

## Postconditions

**Success:**
- Deux nouveaux fichiers existent sous `results/verdicts/` ; aucun fichier antérieur n'est modifié.

**Failure:**
- `results/` est inchangé.

## Business Rules

### BR-002-1: Verdict par le seul critère gelé
Aucun seuil n'est fourni au moment de l'évaluation ; chaque évaluateur code le texte de son critère, et une hypothèse sans évaluateur rend `INCONCLUSIVE`.

### BR-002-2: Rationale traçable
Chaque verdict cite les cellules (`cas/DÉTECTEUR`) ou les bras (`SONDE/BRAS`) qu'il lit, et les valeurs qui ont décidé.

### BR-002-3: Matrice publiée avec les verdicts
La matrice de détectabilité est produite par ce cas d'utilisation et jamais éditée à la main.
