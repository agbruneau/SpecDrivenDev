# Use Case: Comparer valeur et pointeur

## Overview

**Use Case ID:** UC-004
**Use Case Name:** Comparer valeur et pointeur
**Primary Actor:** Chercheur
**Goal:** Obtenir, pour chaque paire de cellules (`VALUE`, `POINTER`) d'une campagne complétée, la différence de temps par opération avec son intervalle de confiance, et le point de bascule par profil de durée de vie
**Status:** Deployed

**Linked Requirements:** FR-004, NFR-003, NFR-004, C-002, C-008, C-009
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
6. Le système écrit `results/campaigns/<campaignId>/comparison-<timestamp>.json` contenant les Comparison, les points de bascule, la liste des paires exclues avec leur raison et la liste des séries dont le point de bascule n'est pas calculé, avec leur raison.
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

### A4: Série à plusieurs paires par taille
**Trigger:** À l'étape 5, une série porte plus d'une Comparison pour une même taille — réplicats de cellule (C-009), ou déclinaisons de répétition et de charge du profil `RETURNED_ALLOCATING` (C-008).
**Flow:**
1. Le système ne calcule pas le point de bascule de cette série : le balayage descendant s'arrête au premier élément défavorable, et l'ordre de deux Comparison de même taille ne porte aucune information.
2. Le système consigne la série et la raison dans le fichier de comparaison, et les signale à l'étape 7.

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

- Clôture du 2026-09-10 : statut porté à `Deployed`, que le tableau de bord définit comme « campagne exécutée et rapport publié ». Ce qui l'établit : la suite complète au vert sous `-race -shuffle=on`, une couverture de statements de 93,6 pour cent, le contrôle des hooks et le contrôle des spécifications sans constatation, une revue contradictoire de trente-cinq constats suivie d'une rédaction contradictoire en deux passes, chaque constat et chaque version ayant été soumis à des vérificateurs chargés de les réfuter, et six campagnes menées de bout en bout dont les rapports sont publiés à la racine du dépôt. `CLAUDE.md` réserve le passage `Reviewed → Approved` à une décision humaine ; il a été assumé par l'agent sur mandat explicite, ce que consigne la décision D-01.


- Révision du 2026-09-12, audit du code : l'exclusion du point de bascule était présentée comme la signature d'une série répliquée (C-009). Elle ne l'est pas : les déclinaisons de répétition et de charge du profil `RETURNED_ALLOCATING`, ajoutées par C-008, produisent elles aussi plusieurs Comparison par taille et tombaient sous la même règle. Vérifié sur `results/campaigns/C-2026-09-10-11/comparison-20260910T232754Z.json`, dont les 266 Comparison ne donnent aucune entrée `RETURNED_ALLOCATING` alors que toutes les autres séries y figurent. Ces séries disparaissaient du fichier sans qu'aucune ligne ne dise ni leur valeur, ni « non observé », ni pourquoi — contre l'étape 5 et A3. Le flux A4 nomme désormais le cas et la raison est consignée. Aucun critère gelé ne lit ce point de bascule : le verdict n'est pas affecté.

- Révision du 2026-09-10, satisfaction de C-009 : le point de bascule d'une série répliquée n'est pas publié. Une telle série porte plusieurs Comparison par taille ; le balayage descendant de l'étape 5 s'arrête au premier élément défavorable et le tri par taille n'est pas stable, de sorte que le résultat dépendrait de l'ordre du fichier. Aucun critère gelé ne lit ce point de bascule — H-012 groupe ses réplicats par ses propres attributs — et les séries non répliquées du même fichier restent publiées.

- La méthode d'estimation de l'intervalle (bootstrap, quantiles) est une implémentation ; elle est documentée dans le code et dans le fichier de comparaison, non dans ce cas d'utilisation.
- `benchstat` (C-002) peut servir de contrôle externe des résultats ; il n'est pas requis par le flux.
- Révision du 2026-09-10 : H-004 retirée (elle se lit sur des Probe, UC-003 et UC-005, non sur des paires valeur/pointeur) ; point de bascule calculé par couple (profil, champ pointeur), chaque taille ayant deux paires.
