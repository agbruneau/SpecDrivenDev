# Use Case: Exécuter une campagne de mesure

## Overview

**Use Case ID:** UC-003
**Use Case Name:** Exécuter une campagne de mesure
**Primary Actor:** Chercheur (ou Pipeline CI)
**Goal:** Obtenir, pour chaque cellule d'une matrice, des mesures répétées de temps, d'octets et d'allocations par opération, rejouables et accompagnées de leur provenance
**Status:** Deployed

**Linked Requirements:** FR-003, FR-006, NFR-001, NFR-003, NFR-004, NFR-005, C-001, C-002, C-003, C-005, C-006, C-008, C-009, C-010
**Révision :** 2026-09-22 — lot 6 du plan d'implantation de l'évaluation (E-14, E-19, D-60) : la Provenance créée à l'étape 3 et affichée à l'étape 4 porte l'état de la machine (quatre champs facultatifs, voir le modèle d'entités), l'étape 5 conserve le `b.N` de chaque répétition, l'étape 6 archive la sortie brute de `go test` ; BR-003-3 nomme ce fichier parmi ceux qui sont créés et jamais réécrits. Spécification modifiée avant tout code. 2026-09-12 — audit du code : A4 précise la vérification de provenance, la reprise du verrou, la conservation des Measurement `FAILED` et la restitution des paramètres de mesure ; A5 (interruption demandée) est ajouté ; l'étape 3 enregistre les paramètres de C-003 et dérive l'identifiant sous le verrou. 2026-09-10 — ajout de l'empreinte des critères (étape 3, BR-003-5) à la demande de UC-005 ; spécification modifiée avant tout code. Même jour : les Probe (FR-006) sont mesurées comme les Cell (étapes 4, 5, 8, A3, A4, BR-003-4) ; `Measurement.subjectId` remplace `cellId`. Revue pré-lancement du 2026-09-10 : BR-003-3 admet la transition de `status` dans `campaign.json` ; la précondition UC-002 ne vaut que si la Matrix contient une Cell.
**Linked Hypotheses:** H-001, H-002, H-003, H-004, H-005, H-007, H-008, H-010, H-011, H-012, H-013
**Entities:** Matrix, Cell, Probe, Campaign, Measurement, Provenance

## Preconditions

- Une Matrix existe avec au moins une Cell ou une Probe.
- Si la Matrix contient au moins une Cell, les verdicts d'échappement de cette Matrix existent pour la toolchain courante (UC-002).
- La toolchain satisfait C-001 ; aucune autre campagne n'est en cours sur la machine.

## Main Success Scenario

1. Le chercheur demande l'exécution d'une campagne sur une Matrix identifiée, avec un nombre de répétitions.
2. Le système vérifie que le nombre de répétitions satisfait NFR-003 et que l'empreinte du harnais est celle de la Matrix.
3. Le système crée une Campaign avec un identifiant nouveau, sa Provenance, l'empreinte du harnais, l'empreinte des critères des hypothèses liées (lus dans `docs/requirements.md`) et les paramètres de mesure de C-003 (`count`, `benchtime`, `cpu`), au statut `RUNNING`. L'identifiant est dérivé sous le verrou de campagne, de sorte que deux lancements simultanés ne puissent pas le choisir en même temps.
4. Le système affiche l'identifiant de la Campaign, la Provenance, le nombre de Cell et le nombre de Probe à mesurer.
5. Le système mesure chaque Cell puis chaque Probe selon C-003 et enregistre, par sujet, une Measurement contenant exactement `count` valeurs de `nsPerOp`, `bytesPerOp`, `allocsPerOp` et `iterations` (le `b.N` de chaque répétition).
6. Le système écrit la sortie brute de `go test` du sujet dans `results/campaigns/<campaignId>/raw/`, puis la Measurement, qui en porte le chemin, dans `results/campaigns/<campaignId>/measurements/`, dès qu'elle est complète.
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
2. Le système capture la Provenance courante et vérifie qu'elle identifie la même toolchain (`goVersion`, `goos`, `goarch`) et le même `cpuModel` que la Provenance de la Campaign ; sinon la reprise est refusée, aucune Measurement n'est écrite et la Campaign reste `RUNNING` (NFR-001, BR-003-2).
3. Le système reprend `results/.campaign-lock` s'il porte l'identifiant de la Campaign désignée ; s'il porte un autre identifiant, la reprise est refusée en nommant le détenteur.
4. Le système reprend au premier sujet (Cell ou Probe) sans Measurement écrite. Les Measurement `FAILED` antérieures sont conservées (BR-003-3) et comptées à l'étape 8 ; un sujet en échec n'est jamais remesuré dans la même Campaign.
5. Le système restitue les paramètres de mesure enregistrés dans la Campaign à l'étape 3 (`count`, `benchtime`, `cpu`) ; une reprise qui désigne une autre Matrix, un autre nombre de répétitions ou d'autres hypothèses est refusée.
6. Le scénario principal continue à l'étape 5.

### A5: Interruption demandée
**Trigger:** Pendant l'étape 5, le chercheur interrompt le processus (SIGINT, SIGTERM).
**Flow:**
1. Le système arrête la boucle de mesure au sujet courant, sans consigner ce sujet comme `FAILED` : une interruption n'est pas un échec de mesure.
2. Le système libère `results/.campaign-lock` et laisse la Campaign au statut `RUNNING`.
3. Le système affiche que la campagne est interrompue et reprenable (A4).

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
Le runner de campagne crée des fichiers dans `results/`. Il ne modifie ni ne supprime aucun fichier existant, à deux exceptions nommées : le `campaign.json` de la campagne courante, dont seul le champ `status` passe de `RUNNING` à `COMPLETED` ou `ABORTED`, et `results/.campaign-lock`, créé au démarrage, repris à la reprise (A4) et retiré à la fin. Aucune Measurement, aucune sortie brute, aucun fichier de comparaison, de verdicts ou d'échappement n'est réécrit. Une sortie brute laissée par une tentative dont la Measurement n'a pas été écrite (processus tué entre les deux écritures) est conservée : la reprise (A4) écrit la sienne sous un nom suffixé, sans toucher la première.

### BR-003-4: Un sujet, un processus
Chaque Cell et chaque Probe est mesurée dans un processus `go test` distinct afin qu'aucun état du runtime ne se propage d'un sujet au suivant.

### BR-003-5: Critères gelés à la création
L'empreinte des critères de réfutation est calculée à la création de la Campaign ; UC-005 refuse tout verdict si les critères ont changé depuis.

## Notes de revue

- Clôture du 2026-09-10 : statut `Deployed`; ce qui l'établit est consigné en D-37.


- Révision du 2026-09-10, satisfaction de C-010 : chaque mesure est encadrée de deux relevés des temps processeur de la machine, et la fraction d'occupation des cœurs non mesurés est consignée dans la Measurement. Le temps retranché est celui de tout l'arbre de processus, `go test` compilant, liant puis exécutant dans des processus enfants ; sans cette agrégation, le travail légitime de la campagne gonflerait la fraction et la garde de H-013 refuserait les campagnes saines. Une plateforme qui n'expose pas ces compteurs laisse le champ absent, ce qui rend H-013 non concluante plutôt que fausse.

- Révision du 2026-09-10, satisfaction de C-009 : une campagne dont la liste gelée contient H-007 est refusée sur une matrice à réplicats. H-007 indexe ses Comparison par taille et n'en retient qu'une, arbitrairement ; sur une série répliquée elle jugerait donc un réplicat tiré de l'ordre du fichier, sans erreur ni trace. Le refus est une précondition, donc aucune Campaign n'est créée. C'est H-012 qui lit une série répliquée. Une campagne sans sélection explicite retient tout le catalogue, donc H-007 : sur une matrice à réplicats, `--hypotheses` devient obligatoire, et c'est le comportement voulu.

- Révision du 2026-09-12, audit du code. Quatre points d'A4 viennent de défauts démontrés, non d'un changement d'intention. (1) La reprise ne considérait « faite » qu'une Measurement `COMPLETE` : un sujet consigné `FAILED` par A3 — le cas normal — était remesuré, et son écriture refusée par l'immutabilité de BR-003-3, rendant toute reprise impossible. Une Measurement écrite vaut désormais « faite », quel que soit son statut ; une nouvelle Campaign, non une reprise, sert à remesurer un sujet en échec. (2) La reprise ne revérifiait pas la Provenance : une mise à jour de la toolchain entre deux tronçons mélangeait deux chaînes d'outils dans une campagne dont la Provenance en affirme une seule (NFR-001). (3) Le verrou orphelin d'un processus mort interdisait la reprise de la campagne qu'il protégeait, alors que c'est exactement le déclencheur d'A4. (4) La Campaign n'enregistrait ni `benchtime` ni `cpu` : la reprise réutilisait les drapeaux de la ligne de commande du moment, et non ceux sous lesquels les mesures déjà écrites ont été prises.

- Révision du 2026-09-12, A5. Une interruption était convertie en échec de mesure sans erreur Go : la boucle ne voyait pas l'annulation, tous les sujets restants échouaient en chaîne, et la campagne se clôturait `COMPLETED`. A4 devenait alors structurellement inatteignable, puisque la reprise exige `RUNNING`. Une interruption n'est pas un résultat de mesure : elle arrête la boucle et laisse la campagne reprenable.

- Révision du 2026-09-22, lot 6 (E-14, E-19). La Provenance d'une campagne ne disait rien de l'état de la machine au-delà du processeur et des caches : ni la version du système, ni le plan d'alimentation, ni l'affinité, ni la répartition des cœurs d'un processeur hybride, alors que chacun déplace un temps par opération. Les Measurement ne gardaient que les valeurs par opération : ni le `b.N` de chaque répétition, ni la sortie brute, reconnus manquants dès la campagne 1. Les nouveaux champs sont facultatifs, constatés et non imposés ; ils ne participent ni à la vérification de provenance d'A4 ni à aucune empreinte, de sorte que les campagnes archivées restent valides et comparables. Note de revue : révision rédigée et vérifiée par un agent (barre de sortie commune, harnais de non-régression sur les campagnes archivées) ; aucune revue humaine ne l'a encore lue, ce que la revue par un tiers des cas d'utilisation (lot 1 du plan) couvrira.
- Acteur « Pipeline CI » : il désigne un usage prévu. Depuis le retrait de la CI (D-53), ce rôle est tenu par le journal manuel de vérification hors poste, `Doc/VERIFICATION-HORS-POSTE.md` à la racine du dépôt.
- Les drapeaux exacts (`-benchmem`, `-count`, `-cpu`) relèvent de C-003, pas de ce cas d'utilisation.
- La détection du CPU et de la version de Go est un adapter (`internal/adapters/provenance`) ; le service ne fait que consigner ce qu'il reçoit.
- Les pannes techniques (disque plein, toolchain absente) sont traitées par l'implémentation et n'apparaissent pas comme flux alternatifs.
