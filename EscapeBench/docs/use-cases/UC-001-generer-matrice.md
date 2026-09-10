# Use Case: Générer la matrice de cellules

## Overview

**Use Case ID:** UC-001
**Use Case Name:** Générer la matrice de cellules
**Primary Actor:** Chercheur
**Goal:** Obtenir une Matrix de cellules compilables couvrant les tailles, profils de durée de vie et modes de passage demandés, identifiée de façon déterministe et liée à l'empreinte du harnais
**Status:** Deployed

**Linked Requirements:** FR-001, NFR-002, NFR-004, C-001, C-004, C-005, C-007, C-008, C-009
**Linked Hypotheses:** H-007, H-008, H-009, H-010, H-012, H-013 — par les sujets qu'il doit générer (C-008, C-009) ; UC préalable à toutes les autres hypothèses
**Entities:** Matrix, TypeSpec, LifetimeProfile, Cell, Probe

## Preconditions

- La toolchain Go installée satisfait C-001.
- `internal/harness/` est présent et compile.
- Aucune campagne n'est au statut `RUNNING`.

## Main Success Scenario

1. Le chercheur demande la génération d'une Matrix en précisant les tailles (bornes et pas), la présence ou non d'un champ pointeur, les dispositions de type, les LifetimeProfile, les modes de passage, le nombre d'instances produites par opération, le nombre de charges allouées par instance et les Probe (genre et paramètre), ou demande la matrice de référence.
2. Le système valide les paramètres : tailles multiples de 8 dans [8, 4096], dispositions, profils et genres de Probe appartenant aux listes du modèle d'entités, au moins un mode de passage, nombre d'instances par opération et nombre de charges par instance ≥ 1, paramètres de Probe > 0 et, pour les sondes qui parcourent la mémoire, multiples de 64 octets couvrant au moins deux nœuds.
3. Le système calcule l'identifiant de la Matrix à partir des paramètres normalisés. Les dimensions ajoutées par C-008 n'entrent dans la représentation canonique que si elles s'écartent de leur valeur par défaut, de sorte qu'une demande antérieure à C-008 produit le même identifiant.
4. Le système génère, pour chaque combinaison taille × présence de pointeur × disposition, un TypeSpec, pour chaque TypeSpec × LifetimeProfile × nombre d'instances par opération × nombre de charges par instance, une Cell par mode de passage, et une Probe par genre × paramètre demandé, chacune avec son fichier source sous `matrices/<matrixId>/`. Deux combinaisons ne sont pas produites : le témoin nul au-delà de trois mots machine, et les déclinaisons d'instances ou de charges pour un profil qui n'en dépend pas, qui ne créeraient que des doublons.
5. Le système compile l'ensemble des cellules et des sondes.
6. Le système calcule l'empreinte de `internal/harness/` et écrit `matrices/<matrixId>/matrix.json` contenant les paramètres, la liste des cellules et des sondes, l'empreinte et la date de génération.
7. Le système affiche l'identifiant de la Matrix, le nombre de TypeSpec, le nombre de Cell, le nombre de Probe et l'empreinte du harnais.

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

### A3: Une cellule ou une sonde ne compile pas
**Trigger:** À l'étape 5, la compilation d'au moins une Cell ou Probe échoue.
**Flow:**
1. Le système affiche chaque Cell ou Probe fautive avec le message du compilateur.
2. Le système supprime `matrices/<matrixId>/`.
3. Le cas d'utilisation se termine sans Matrix.

## Postconditions

**Success:**
- `matrices/<matrixId>/` contient un fichier source par Cell et par Probe et un `matrix.json` complet.
- Toutes les cellules et sondes compilent.
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
La matrice de référence couvre les tailles 8, 16, 24, 32, 64, 128, 256, 512, 1024, 2048, 4096 octets, avec et sans champ pointeur, les cinq LifetimeProfile et les deux modes de passage (220 Cell), plus dix Probe : `SEQUENTIAL_SCAN` et `SCATTERED_SCAN` pour des jeux de travail de 256 KiB, 4 MiB, 32 MiB et 128 MiB, `APPEND_PREALLOC` et `APPEND_GROW` pour n = 100 000.

## Notes de revue

- Clôture du 2026-09-10 : statut porté à `Deployed`, que le tableau de bord définit comme « campagne exécutée et rapport publié ». Ce qui l'établit : la suite complète au vert sous `-race -shuffle=on`, une couverture de statements de 93,6 pour cent, le contrôle des hooks et le contrôle des spécifications sans constatation, une revue contradictoire de trente-cinq constats suivie d'une rédaction contradictoire en deux passes, chaque constat et chaque version ayant été soumis à des vérificateurs chargés de les réfuter, et six campagnes menées de bout en bout dont les rapports sont publiés à la racine du dépôt. `CLAUDE.md` réserve le passage `Reviewed → Approved` à une décision humaine ; il a été assumé par l'agent sur mandat explicite, ce que consigne la décision D-01.


- La matrice de référence inclut 24 octets (trois mots machine) parce que H-002 porte sur cette borne ; la règle de l'étape 2 admet tout multiple de 8 dans [8, 4096].
- Le code source généré par cellule est un détail d'implémentation ; le cas d'utilisation n'impose que son existence et sa compilabilité.
- Révision du 2026-09-10 : ajout des Probe (FR-006, H-004, H-005) — la matrice porte les sondes pour qu'une seule campagne, une seule empreinte et une seule Provenance couvrent H-001 à H-006.
- Révision du 2026-09-10, satisfaction de C-009 : la génération se décline en réplicats, dimension la plus extérieure, de sorte que deux mesures d'une même paire soient séparées par une passe complète de la matrice. Le nombre de réplicats vaut un ou cinq et rien d'autre, l'étape 2 refusant toute autre valeur : le plancher de bruit de H-012 ne doit ni grandir ni rétrécir avec l'effort de mesure. Seule la disposition `NAMED_FIELDS` en profil `LOCAL` s'en décline. Un réplicat unique ne laisse aucune trace dans les identifiants, les répertoires de sujets ni la représentation canonique, donc `M-823d8b5af441` est inchangée. Les gabarits ne changent pas — un réplicat rend la même source dans un répertoire différent, ce qui est précisément ce qui garantit un code machine identique —, donc l'empreinte du harnais est inchangée et les matrices antérieures restent extensibles. C'est la seule contrainte de capacité du catalogue qui n'invalide rien.
- Révision du 2026-09-10, revue contradictoire : la génération se décline en nombre de charges allouées par instance (k). Sans cette dimension, H-010 comparait k charges à k + 1 avec k figé à un, donc un rapport de deux par arithmétique du gabarit et non par propriété du mode de passage. Le témoin nul n'est plus produit au-delà de trois mots machine, et c'est un saut à la génération, non un refus à la validation : le critère gelé de H-007 exige dans la même série un témoin de sensibilité d'au moins 80 octets, qu'un refus rendrait inatteignable. Les deux dimensions gardent leur valeur par défaut hors de la représentation canonique, donc `M-823d8b5af441` et son décompte de 220 Cell sont inchangés ; l'empreinte du harnais, elle, a de nouveau changé.
- Révision du 2026-09-10, satisfaction de C-008 : la génération se décline en dispositions de type et en nombre d'instances par opération. La matrice de référence de BR-001-4 est inchangée — une seule disposition, une instance par opération, cinq profils — donc son décompte de 220 Cell et son identifiant `M-823d8b5af441` le sont aussi. Les gabarits du harnais ayant changé, son empreinte a changé : les matrices générées avant C-008 ne peuvent plus être étendues, ce que A2 signale, et leurs résultats archivés restent valides puisque chaque fichier porte l'empreinte sous laquelle il a été produit.
