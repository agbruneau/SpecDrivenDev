# Use Case: Produire les verdicts

## Overview

**Use Case ID:** UC-005
**Use Case Name:** Produire les verdicts
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour chaque hypothèse liée aux résultats fournis, un verdict fondé uniquement sur le critère de réfutation gelé, avec son rationale, et un tableau de bord à jour
**Status:** Deployed

**Linked Requirements:** FR-005, FR-007, NFR-001, NFR-004
**Linked Hypotheses:** H-001, H-002, H-003, H-004, H-005, H-006, H-007, H-008, H-009, H-010, H-011, H-012, H-013
**Entities:** Hypothesis, Verdict, Campaign, Matrix, Cell, TypeSpec, Comparison, Measurement, Probe, EscapeVerdict

## Preconditions

- `docs/requirements.md` contient les hypothèses avec leur énoncé réfutable et leur critère de réfutation.
- Les résultats fournis existent : une Campaign `COMPLETED` avec au moins un fichier de comparaison (H-001, H-002), des Measurement de Cell (H-003) ou de Probe (H-004, H-005), ou un fichier de verdicts d'échappement (H-006).
- L'empreinte des critères enregistrée par la Campaign correspond aux critères courants de `docs/requirements.md`.

## Main Success Scenario

1. Le chercheur demande les verdicts en désignant une Campaign et, s'il y a lieu, un fichier de verdicts d'échappement.
2. Le système lit les hypothèses et leurs critères depuis `docs/requirements.md` et vérifie leur empreinte contre celle de la Campaign.
3. Le système détermine, pour chaque hypothèse, si les résultats fournis permettent d'évaluer son critère.
4. Pour chaque hypothèse évaluable, le système évalue le critère de réfutation sur les résultats et attribue `REFUTED` si le critère est déclenché, `CONFIRMED` sinon.
5. Pour chaque hypothèse non évaluable, le système attribue `INCONCLUSIVE` avec la raison.
6. Le système écrit `results/verdicts/<campaignId>-<timestamp>.json` contenant, par hypothèse, le verdict, le rationale et la liste des fichiers de résultats utilisés.
7. Le système régénère `docs/dashboard.md` à partir des fichiers de verdicts les plus récents, du statut déclaré dans chaque fichier de cas d'utilisation et du résultat de l'exécution des tests nommés `TestUC###_*` (FR-007) ; la régénération seule est aussi disponible sans nouveau verdict.
8. Le système affiche le tableau des verdicts.

## Alternative Flows

### A1: Critère modifié depuis le démarrage de la campagne
**Trigger:** À l'étape 2, l'empreinte des critères de la Campaign diffère de celle des critères courants.
**Flow:**
1. Le système refuse de produire les verdicts pour cette Campaign et affiche les hypothèses dont le critère a changé.
2. Aucun fichier n'est écrit.
3. Le chercheur crée une nouvelle hypothèse pour le nouveau critère (identifiants jamais réutilisés) et lance une nouvelle campagne.

### A2: Données insuffisantes pour une hypothèse
**Trigger:** À l'étape 3, les paires, cellules ou sondes nécessaires manquent (exclues en UC-004 ou `FAILED` en UC-003).
**Flow:**
1. Le système attribue `INCONCLUSIVE` avec la liste des sujets manquants.
2. Le scénario principal continue à l'étape 4 pour les autres hypothèses.

## Postconditions

**Success:**
- Un fichier de verdicts existe pour la Campaign ; chaque verdict cite ses fichiers de résultats.
- `docs/dashboard.md` reflète les verdicts les plus récents.
- Aucun fichier de `results/` antérieur n'a été modifié.

**Failure:**
- `results/` et `docs/dashboard.md` sont inchangés.

## Business Rules

### BR-005-1: Verdict par le seul critère gelé
Un verdict ne dépend que du critère de réfutation enregistré au démarrage de la Campaign ; aucun paramètre de décision n'est fourni au moment de l'évaluation.

### BR-005-2: Rationale traçable
Chaque Verdict cite les fichiers de résultats et les identifiants de cellules ou de paires sur lesquels il repose.

### BR-005-3: Tableau de bord régénéré
`docs/dashboard.md` n'est jamais édité à la main ; il est produit par ce cas d'utilisation (étape 7), y compris lorsqu'il est invoqué pour la seule couverture des tests.

## Notes de revue

- Clôture du 2026-09-10 : statut porté à `Deployed`, que le tableau de bord définit comme « campagne exécutée et rapport publié ». Ce qui l'établit : la suite complète au vert sous `-race -shuffle=on`, une couverture de statements de 93,6 pour cent, le contrôle des hooks et le contrôle des spécifications sans constatation, une revue contradictoire de trente-cinq constats suivie d'une rédaction contradictoire en deux passes, chaque constat et chaque version ayant été soumis à des vérificateurs chargés de les réfuter, et six campagnes menées de bout en bout dont les rapports sont publiés à la racine du dépôt. `CLAUDE.md` réserve le passage `Reviewed → Approved` à une décision humaine ; il a été assumé par l'agent sur mandat explicite, ce que consigne la décision D-01.


- L'empreinte des critères est calculée par UC-003 à la création de la Campaign (étape 3) ; le modèle d'entités porte l'attribut `Campaign.hypothesesDigest`.
- La formulation des critères dans `docs/requirements.md` doit rester évaluable mécaniquement (comparaisons numériques sur des champs de Comparison, Measurement ou EscapeVerdict) ; une hypothèse dont le critère n'est pas mécanisable est signalée à la revue de spécification.
