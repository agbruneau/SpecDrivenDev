# Use Case: Exécuter une campagne

## Overview

**Use Case ID:** UC-001
**Use Case Name:** Exécuter une campagne
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour le corpus de référence, toutes les observations des détecteurs et toutes les mesures des sondes, avec leur provenance, dans un fichier de résultats immuable
**Status:** Approved

**Linked Requirements:** FR-001, FR-002, FR-003, FR-004, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, C-001, C-003, C-004, C-005, C-006, C-007, C-008
**Linked Hypotheses:** H-001 à H-013
**Entities:** Case, Detector, Run, Provenance, Observation, ProbeResult, Hypothesis

## Preconditions

- La toolchain Go est en version 1.27 ou plus récente (C-001).
- Le module `lab/` contient le corpus, les pilotes et le programme de scénario.
- `docs/requirements.md` contient le corpus de référence et les hypothèses.

## Main Success Scenario

1. Le chercheur lance une campagne en indiquant, au besoin, le nombre de répétitions (5 par défaut, jamais moins) et le délai par observation (5 s par défaut).
2. Le système lit `docs/requirements.md`, calcule l'empreinte du critère de chaque hypothèse et vérifie que le catalogue du corpus est identique au corpus de référence.
3. Le système exécute l'oracle de vérité terrain du corpus.
4. Le système compile les binaires de mesure : pilotes de test, pilotes de test sous `-race`, programme de scénario.
5. Pour chaque cas, chaque détecteur dynamique et chaque répétition, le système lance un processus neuf, le tue au-delà du délai, classe son issue et consigne l'Observation.
6. Le système exécute `go vet` et `ctxvet` sur le paquet du corpus et consigne une Observation statique par cas et par analyseur.
7. Pour chaque sonde, chaque bras et chaque répétition, le système lance un processus neuf et consigne le ProbeResult.
8. Le système écrit `results/runs/<runId>.json` en écriture exclusive et affiche le nombre d'observations par issue.

## Alternative Flows

### A1: L'oracle contredit la vérité terrain
**Trigger:** À l'étape 3, un cas ne se comporte pas comme le corpus de référence le déclare.
**Flow:**
1. Le système affiche la sortie de l'oracle et refuse la campagne.
2. Aucun fichier n'est écrit.

### A2: Catalogue et spécification divergent
**Trigger:** À l'étape 2, un cas manque, est en trop ou diffère d'un attribut.
**Flow:**
1. Le système nomme les cas et attributs en écart et refuse la campagne.
2. Aucun fichier n'est écrit.

### A3: Échec de compilation
**Trigger:** À l'étape 4, un binaire ne compile pas.
**Flow:**
1. Le système affiche la sortie du compilateur et refuse la campagne. Aucun fichier n'est écrit.

### A4: Répétitions insuffisantes
**Trigger:** À l'étape 1, le nombre de répétitions demandé est inférieur à 5.
**Flow:**
1. Le système refuse la campagne en citant NFR-002.

## Postconditions

**Success:**
- Un seul nouveau fichier existe sous `results/runs/` ; il porte la provenance, les empreintes, toutes les observations et toutes les mesures.

**Failure:**
- `results/` est inchangé.

## Business Rules

### BR-001-1: Corpus conforme à la spécification
Le catalogue Go du corpus et le tableau du corpus de référence de `docs/requirements.md` portent les mêmes cas avec les mêmes attributs ; toute divergence est un refus, jamais un avertissement.

### BR-001-2: Classification des issues
L'issue d'une observation dynamique suit l'ordre de C-005 ; `mentionsLeak` suit la définition de H-002.

### BR-001-3: Attribution des diagnostics statiques
Un diagnostic est attribué au cas dont il vise le fichier ; un diagnostic qui vise un fichier sans cas (outillage commun du corpus) n'est attribué à aucun cas.

### BR-001-4: Isolation
Aucune observation dynamique ne partage un processus avec une autre (NFR-003).

### BR-001-5: Écriture unique
Le fichier de campagne est écrit en une fois, à la fin, et jamais écrasé (NFR-004).

## Notes de revue

- L'oracle (FR-001) s'appuie sur les piles de goroutines (`runtime.Stack`) et non sur le nombre de goroutines ni sur le profil `goroutineleak`, qui sont des détecteurs jugés : la vérité terrain ne doit dépendre d'aucun détecteur qu'elle sert à juger. Les cas de `D` ne sont pas exécutés par l'oracle, qui ne ferait que se bloquer ; les cas de `R` y sont exécutés sans `-race`.
