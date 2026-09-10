# Use Case: Comparer valeur et pointeur

## Overview

**Use Case ID:** UC-004
**Use Case Name:** Comparer valeur et pointeur
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour chaque paire de cellules (`VALUE`, `POINTER`) d'une campagne complétée, la différence de temps par opération avec son intervalle de confiance, et le point de bascule par profil de durée de vie
**Status:** Implemented

**Linked Requirements:** FR-004, NFR-003, NFR-004, C-002, C-009
**Linked Hypotheses:** H-001, H-002, H-007, H-010, H-012
**Entities:** Campaign, Cell, Measurement, Comparison

## Preconditions

- Une Campaign au statut `COMPLETED` existe avec ses Measurement.
- La Matrix de la Campaign contient au moins une paire (`VALUE`, `POINTER`) pour un même TypeSpec et un même LifetimeProfile.

## Main Success Scenario

1. Le chercheur demande la comparaison d'une Campaign identifiée.
2. Le système vérifie que la Campaign est `COMPLETED` et que son empreinte de harnais est celle de la Matrix.
3. Le système apparie les cellules `VALUE` et `POINTER` par TypeSpec et LifetimeProfile.
4. Pour chaque paire, le système calcule `deltaNsPerOp` (médiane du pointeur moins médiane de la valeur), l'intervalle de confiance à 95 % de cette différence et le drapeau `significant` (vrai si l'intervalle exclut zéro).
5. Pour chaque couple (LifetimeProfile, présence d'un champ pointeur), le système détermine le point de bascule : la plus petite taille telle que, pour cette taille et toutes les tailles supérieures mesurées du couple, la Comparison a `significant` vrai et `deltaNsPerOp` < 0 ; s'il n'en existe pas, le point de bascule est « non observé ».
6. Le système écrit `results/campaigns/<campaignId>/comparison-<timestamp>.json` contenant les Comparison, les points de bascule et la liste des paires exclues avec leur raison.
7. Le système affiche un tableau par couple (profil, champ pointeur) : taille, `deltaNsPerOp`, intervalle, significatif, et le point de bascule.

## Alternative Flows

### A1: Campagne non complétée
**Trigger:** À l'étape 2, la Campaign est `RUNNING` ou `ABORTED`.
**Flow:**
1. Le système refuse la comparaison et affiche le statut de la Campaign.
2. Aucun fichier n'est écrit.

### A2: Paire incomplète
**Trigger:** À l'étape 3, l'une des deux cellules d'une paire n'a pas de Measurement complète (cellule `FAILED` en UC-003).
**Flow:**
1. Le système exclut la paire et consigne la raison.
2. Le système poursuit avec les autres paires.
3. À l'étape 7, le système affiche le nombre de paires exclues.

### A3: Aucune bascule observée
**Trigger:** À l'étape 5, aucune taille ne satisfait la condition pour un couple.
**Flow:**
1. Le système enregistre « non observé » pour ce couple.
2. Le système signale ce couple dans l'affichage de l'étape 7.

## Postconditions

**Success:**
- Un fichier de comparaison existe pour la Campaign ; il référence l'identifiant de la Campaign et de la Matrix.
- Aucun fichier de `results/` antérieur n'a été modifié.

**Failure:**
- `results/` est inchangé.

## Business Rules

### BR-004-1: Aucune exclusion silencieuse
Aucune Measurement n'est écartée sans qu'une entrée d'exclusion, avec sa raison, figure dans le fichier de comparaison.

### BR-004-2: Signification par l'intervalle seulement
`significant` est vrai si et seulement si l'intervalle de confiance à 95 % de `deltaNsPerOp` exclut zéro ; aucun autre seuil n'est appliqué.

### BR-004-3: Comparaison immuable
Un fichier de comparaison n'est jamais réécrit ; un nouveau calcul produit un nouveau fichier horodaté.

## Notes de revue

- La méthode d'estimation de l'intervalle (bootstrap, quantiles) est une implémentation ; elle est documentée dans le code et dans le fichier de comparaison, non dans ce cas d'utilisation.
- `benchstat` (C-002) peut servir de contrôle externe des résultats ; il n'est pas requis par le flux.
- Révision du 2026-09-10 : H-004 retirée (elle se lit sur des Probe, UC-003 et UC-005, non sur des paires valeur/pointeur) ; point de bascule calculé par couple (profil, champ pointeur), chaque taille ayant deux paires.
