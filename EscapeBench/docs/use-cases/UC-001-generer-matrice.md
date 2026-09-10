# Use Case: Générer la matrice de cellules

## Overview

**Use Case ID:** UC-001
**Use Case Name:** Générer la matrice de cellules
**Primary Actor:** Chercheur
**Goal:** Obtenir une Matrix de cellules compilables couvrant les tailles, profils de durée de vie et modes de passage demandés, identifiée de façon déterministe et liée à l'empreinte du harnais
**Status:** Reviewed

**Linked Requirements:** FR-001, NFR-002, NFR-004, C-001, C-004, C-005
**Linked Hypotheses:** — (UC préalable à toutes les hypothèses)
**Entities:** Matrix, TypeSpec, LifetimeProfile, Cell

## Preconditions

- La toolchain Go installée satisfait C-001.
- `internal/harness/` est présent et compile.
- Aucune campagne n'est au statut `RUNNING`.

## Main Success Scenario

1. Le chercheur demande la génération d'une Matrix en précisant les tailles (bornes et pas), la présence ou non d'un champ pointeur, les LifetimeProfile et les modes de passage, ou demande la matrice de référence.
2. Le système valide les paramètres : tailles multiples de 8 dans [8, 4096], profils appartenant à la liste du modèle d'entités, au moins un mode de passage.
3. Le système calcule l'identifiant de la Matrix à partir des paramètres normalisés.
4. Le système génère, pour chaque combinaison taille × présence de pointeur, un TypeSpec, et pour chaque TypeSpec × LifetimeProfile, une Cell par mode de passage, chacune avec son fichier source sous `matrices/<matrixId>/`.
5. Le système compile l'ensemble des cellules.
6. Le système calcule l'empreinte de `internal/harness/` et écrit `matrices/<matrixId>/matrix.json` contenant les paramètres, la liste des cellules, l'empreinte et la date de génération.
7. Le système affiche l'identifiant de la Matrix, le nombre de TypeSpec, le nombre de Cell et l'empreinte du harnais.

## Alternative Flows

### A1: Paramètres invalides
**Trigger:** À l'étape 2, au moins un paramètre est hors des bornes ou inconnu.
**Flow:**
1. Le système refuse la génération et affiche chaque paramètre fautif avec la règle violée.
2. Aucun fichier n'est écrit.

### A2: La matrice existe déjà
**Trigger:** À l'étape 3, un répertoire `matrices/<matrixId>/` contenant `matrix.json` existe déjà.
**Flow:**
1. Le système compare l'empreinte du harnais enregistrée avec l'empreinte courante.
2. Si elles sont identiques, le système affiche l'identifiant existant et ne régénère rien.
3. Si elles diffèrent, le système refuse la génération et indique que le harnais a changé depuis la génération (C-005) ; le chercheur choisit de nouveaux paramètres ou restaure le harnais.

### A3: Une cellule ne compile pas
**Trigger:** À l'étape 5, la compilation d'au moins une Cell échoue.
**Flow:**
1. Le système affiche chaque Cell fautive avec le message du compilateur.
2. Le système supprime `matrices/<matrixId>/`.
3. Le cas d'utilisation se termine sans Matrix.

## Postconditions

**Success:**
- `matrices/<matrixId>/` contient un fichier source par Cell et un `matrix.json` complet.
- Toutes les cellules compilent.
- Aucun fichier existant hors de `matrices/<matrixId>/` n'a été modifié.

**Failure:**
- Aucun répertoire de matrice partiel ne subsiste.

## Business Rules

### BR-001-1: Identifiant déterministe
Deux demandes portant les mêmes paramètres normalisés produisent le même identifiant de Matrix.

### BR-001-2: Matrice immuable
Une Matrix n'est jamais modifiée après génération ; tout changement de paramètres produit une nouvelle Matrix sous un nouvel identifiant.

### BR-001-3: Paires complètes
Chaque TypeSpec × LifetimeProfile produit exactement une Cell en mode `VALUE` et une Cell en mode `POINTER` lorsque les deux modes sont demandés.

### BR-001-4: Matrice de référence
La matrice de référence couvre les tailles 8, 16, 24, 32, 64, 128, 256, 512, 1024, 2048, 4096 octets, avec et sans champ pointeur, les cinq LifetimeProfile et les deux modes de passage.

## Notes de revue

- La matrice de référence inclut 24 octets (trois mots machine) parce que H-002 porte sur cette borne ; la règle de l'étape 2 admet tout multiple de 8 dans [8, 4096].
- Le code source généré par cellule est un détail d'implémentation ; le cas d'utilisation n'impose que son existence et sa compilabilité.
