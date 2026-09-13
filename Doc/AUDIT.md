# Audit du code — EscapeBench

**Date :** 2026-09-12 · **Portée :** `EscapeBench/` (74 fichiers Go, 16 402 lignes), gabarits du harnais, hooks, `Makefile`, CI, et la conformité du code aux cinq cas d'utilisation et au catalogue d'exigences. **État du dépôt audité :** `main`, commit `9833da0`, arbre propre. **Toolchain :** `go1.27.0 windows/amd64`.

L'audit ne juge pas la clôture du projet ni les verdicts eux-mêmes. Il cherche les écarts entre ce que le code fait et ce que `docs/` prescrit, et les défauts qui peuvent produire une mesure fausse, un verdict faux ou une perte de résultat.

## Résultat

Le banc est en bon état : la suite complète passe sous `-race -shuffle=on`, `go vet` est muet, `gofmt` n'a rien à réécrire, la couverture de statements va de 83,8 % à 98,1 % selon le paquet et les quatre hooks passent leur propre contrôle. Aucune dépendance externe, le layout hexagonal est respecté, `internal/models` n'importe que la bibliothèque standard et ne porte aucun tag.

Trois défauts bloquants ont été démontrés :
1. Le classificateur d'échappement ne sait pas nommer la variable quand le compilateur rapporte une expression plutôt qu'un identifiant (A-072). Les 120 cellules que le tableau de bord compte « hors des quatre causes du livre » échappent en réalité par un retour de pointeur, la première des quatre causes. Le verdict `REFUTED` de H-006, publié au tableau de bord et au rapport final, est un artefact du classificateur et non une observation.
2. La reprise de campagne est inopérante dès qu'un sujet a échoué (A-020), ce qui est le cas normal prévu par le flux A3 de UC-003. Une campagne interrompue après un échec de sujet ne se reprend pas en raison de la garde d'immutabilité.
3. Une interruption de campagne (SIGINT / Ctrl-C) est masquée en échec de mesure sans arrêt de la boucle (A-261). Tous les sujets suivants échouent immédiatement et la campagne se clôture au statut `COMPLETED`, rendant le flux A4 de reprise inatteignable.

Le reste des constats confirmés se répartit en 20 majeurs (les 12 initiaux plus 8 majeurs confirmés lors de la contre-vérification), 53 mineurs et 35 suggestions. 75 constats (mineurs et suggestions) demeurent sans vérification contradictoire complète.

## Méthode

Dix-huit découvreurs ont lu le dépôt en parallèle, un par paquet et un par lentille transversale : conformité au `CLAUDE.md` du projet, concurrence et ressources, robustesse aux entrées, portabilité, qualité des tests, statistiques, sur-ingénierie, et dérive entre chaque cas d'utilisation et son code. Ils ont produit 290 constats bruts, ramenés à 190 items après premier dédoublonnage (102 confirmés + 4 rejetés + 84 non vérifiés).

Chaque constat vérifié a ensuite reçu trois vérificateurs indépendants, chargés de le faire tomber : un réfutateur qui cherche la garde en amont ou l'erreur de lecture, un reproducteur qui travaille dans une copie isolée du dépôt et exécute un test jetable quand c'est possible, et un juge de spécification qui cherche si `docs/` ou une décision `D-##` prescrit le comportement dénoncé. Un constat n'est retenu que si au moins deux vérificateurs sur trois le maintiennent. Cette règle a écarté 4 constats.

Une contre-vérification indépendante a été menée le 2026-09-12 pour finaliser l'audit avant implémentation. Huit agents auditeurs ont passé en revue les constats majeurs et bloquants contre le code source réel de `EscapeBench` :
- **100 % des constats critiques testés (16/16) ont été confirmés dans le code source** (numéros de ligne exacts, comportements réels conformes à la description).
- Le constat **A-261** (interruption transformée en échec absorbé et campagne `COMPLETED`) a été confirmé dans `internal/adapters/gotool/gotool.go:208` et promu au rang de **Bloquant**.
- **8 constats majeurs** parmi les 84 non vérifiés ont été intégralement validés (A-156, A-141, A-278, A-246, A-263, A-265, A-187, A-185) et intégrés aux lots d'implémentation.
- Les 75 constats restants (37 mineurs, 38 suggestions) ne présentent aucun risque de blocage ou d'invalidité sur les résultats scientifiques et peuvent être traités au fil de l'eau.

Ce qui n'a pas été fait :

- `staticcheck` est installé sur le poste mais ne lit pas les données d'export de `go1.27` ; aucune analyse statique tierce n'a donc tourné.
- Le lot 9 (gabarits du harnais et couverture de `harnessDigest`) est soumis à un arbitrage préalable du chercheur.

## Ce qui tient

Ces points ont été examinés et n'ont donné aucun constat. Ils sont notés parce qu'un audit qui ne dit que ce qui cloche laisse croire que le reste n'a pas été regardé.

- **Fidélité des évaluateurs aux critères gelés.** Les treize évaluateurs de `internal/service/criteria*.go` ont été relus clause par clause contre le texte de `docs/requirements.md` : ordre des clauses de non-conclusion, bornes inclusives ou exclusives, médianes, rapports, bandes de résidence de H-008 et H-013, plancher et barrière de H-012, garde de quiétude. Aucun écart. Le soupçon initial que H-001 compterait des réplicats comme des tailles distinctes a été écarté : `Expand` ne réplique que la disposition `NAMED_FIELDS` en profil `LOCAL`, que H-001 ne lit pas.
- **Gel des critères.** L'empreinte des hypothèses est calculée à la création de la campagne et revérifiée à chaque production de verdict ; le refus est testé.
- **Déterminisme de l'identifiant de matrice.** `Canonical` trie, dédoublonne et omet les dimensions restées à leur valeur par défaut, de sorte qu'une demande antérieure à C-008 rende le même identifiant ; deux tests verrouillent `M-823d8b5af441`.
- **Intervalle de confiance.** Le bootstrap est déterministe par graine dérivée de la paire, et `significant` est vrai si et seulement si l'intervalle exclut zéro, sans autre seuil, conformément à BR-004-2.
- **Séparation des couches.** `internal/service` ne fait aucune entrée-sortie directe, les adaptateurs n'importent jamais le service sauf `cli` pour ses types de rapport, et aucun `panic` ne se trouve sur un chemin de requête.

## Lecture des constats

Quatre niveaux. **Bloquant** : le banc produit un résultat faux ou perd un résultat. **Majeur** : comportement erroné sur un cas limite plausible, règle `BR`/`C`/`NFR` sans garde, ou fuite de ressource. **Mineur** : défaut réel mais contenu. **Suggestion** : simplification, lisibilité, test manquant sans risque immédiat.

Deux marqueurs accompagnent les constats dont le correctif a un coût de processus. `Campaign.harnessDigest` signale un correctif qui touche un gabarit embarqué : il invalide toute matrice et toute campagne antérieures (C-005, BR-003-1). *Critère gelé* signale un correctif qui exigerait de réécrire le texte d'une hypothèse, ce que le processus interdit : il faut alors ouvrir une nouvelle `H-###`.

## Planification d'exécution des correctifs

Destinataire : Claude Opus 5 en mode Ultracode. Chaque lot est une session, un workflow, un commit. Les lots sont ordonnés par dépendance ; à l'intérieur d'un lot, l'ordre est libre.

### Règles qui gouvernent toute la campagne de correction

Elles viennent du `CLAUDE.md` du projet et du catalogue ; les enfreindre invalide le travail plutôt que de le retarder.

1. **La spécification d'abord.** Tout correctif qui change un comportement observable commence par une révision datée dans `docs/`, puis seulement le code, dans le même commit. Un correctif qui ne fait qu'aligner le code sur une spécification déjà écrite n'a pas ce préalable.
2. **Un identifiant par demande.** Chaque lot nomme les `UC-###` et `H-###` qu'il touche ; le message de commit est préfixé de l'identifiant principal.
3. **Ne jamais écrire sous `results/`, `matrices/` ni dans `docs/dashboard.md`.** Le hook `guard-paths` le refuse et c'est voulu. Les fichiers de résultats se produisent en exécutant le binaire, jamais en éditant.
4. **Un critère gelé ne se retouche pas.** Si un correctif exige de changer le texte d'une hypothèse, il faut ouvrir une nouvelle `H-###` ; aucun constat de cet audit n'est dans ce cas.
5. **Les gabarits du harnais sont à part.** Toute modification sous `internal/harness/templates/` change `Campaign.harnessDigest` et rend les matrices et campagnes antérieures incomparables. Le lot 9 les regroupe ; rien d'autre n'y touche.
6. **Barre de sortie de chaque lot.** `go vet ./...`, `go test -race -shuffle=on -count=1 ./...`, `gofmt -l ./cmd ./internal` vide, `bash .claude/hooks/selftest.sh` sans échec, et le harnais de non-régression du lot 0 au vert.

### Lot 0 — Outiller la non-régression et consigner les décisions

Sans préalable. Ne corrige aucun constat, conditionne tous les autres lots.

Aucun test ne rejoue aujourd'hui les dix campagnes archivées sous `results/`. Rien ne garantit donc qu'un correctif aux évaluateurs, au store ou au modèle laisse les verdicts publiés inchangés. C'est le premier manque à combler, et il donne du même coup un contenu à la cible `integration_test` du `Makefile`, qui rejoue aujourd'hui la suite unitaire en se faisant passer pour autre chose (A-101).

- Écrire, sous le tag de build `integration_test`, un test qui lit `results/` en lecture seule, recharge chaque campagne, ses mesures, son fichier de comparaison et son rapport de verdicts, rejoue les évaluateurs et compare verdict par verdict à ce qui est archivé. Toute divergence échoue en nommant l'hypothèse et la campagne.
- Ajouter `-shuffle=on` à la cible `integration_test` et faire appeler `make vet test` par la CI.
- Consigner dans `Doc/DECISION.md` les deux décisions que les lots 1 et 2 exigent : le changement de comportement du classificateur d'échappement et la réexécution qu'il impose, et la conservation des mesures `FAILED` à la reprise.
- Réviser `docs/use-cases/UC-003-executer-campagne.md`, flux A4, pour dire ce que le lot 2 va implémenter : la reprise repart au premier sujet sans mesure écrite, les mesures `FAILED` antérieures sont conservées et comptées, et la reprise vérifie la provenance avant de mesurer.
- Retirer les vingt-six worktrees de vérification laissés par l'audit sous `.claude/worktrees` à la racine du dépôt, soit 2,7 Go (`git worktree list` les énumère, `git worktree remove` les retire).

**Sortie du lot :** le harnais de non-régression passe sur les dix campagnes archivées et les treize verdicts publiés.

### Lot 1 — Le verdict faux (chemin critique)

Dépend du lot 0. Un seul constat, mais c'est le seul qui a déjà produit un résultat publié faux.

- **A-072** — le classificateur rend `OTHER` dès que le compilateur nomme une expression plutôt qu'une variable. Corriger dans `internal/adapters/escape/escape.go` : quand le message est « `<expression>` escapes to heap », retrouver dans le corps englobant l'affectation qui porte cette allocation à la ligne du diagnostic, et classer d'après l'usage de la variable qui la reçoit. Les précisions des vérificateurs sur le repérage du porteur et sur les tests de non-régression sont dans la fiche du constat.

Puis, en exécutant le binaire et non en éditant des fichiers :

1. Rejouer UC-002 sur les matrices concernées, ce qui crée un nouveau rapport de verdicts d'échappement sans réécrire l'ancien.
2. Le flux A3 de UC-002 va signaler un écart de NFR-002, puisque deux exécutions sur la même toolchain donnent des verdicts différents. C'est attendu et c'est exactement ce que la décision du lot 0 consigne ; ne pas le traiter comme une régression.
3. Produire un nouveau verdict pour les campagnes qui portent H-006, et laisser le binaire régénérer le tableau de bord.
4. Ajouter un erratum daté dans `Doc/RAPPORT-FINAL_EscapeBench.md`, qui affirme aujourd'hui que H-006 tombe sur 120 cellules hors des quatre causes. Un erratum, pas une réécriture silencieuse.

**Attention.** Si la reclassification fait passer H-006 de `REFUTED` à `CONFIRMED`, le rapport final doit le dire et expliquer pourquoi le premier verdict était un artefact. C'est le résultat le plus important de cet audit et il ne doit pas se perdre dans un tableau régénéré.

### Lot 2 — Reprise, résilience aux signaux et verrou de campagne

Dépend du lot 0 pour la révision de UC-003 A4.

- **A-020** (bloquant) — la reprise ne saute que les mesures complètes, remesure les sujets en échec et se heurte à l'immutabilité. Marquer comme faites toutes les mesures écrites et propager leur statut au décompte.
- **A-261** (bloquant) — une interruption (Ctrl-C / SIGTERM) fait renvoyer à `Toolchain.Run` un échec de mesure avec `err == nil` : la boucle de mesure continue, tous les sujets restants échouent en chaîne et la campagne finit close au statut `COMPLETED`, rendant le flux A4 inatteignable. Propager l'annulation de contexte (`ctx.Err()`) pour stopper immédiatement la boucle sans clôturer la campagne.
- **A-141** (majeur) — un `go build -gcflags=-m` interrompu par le contexte dans `Toolchain.EscapeAnalysis` devient un `ports.CompileError`, ce qui marque la cellule en `COMPILE_ERROR` et laisse la boucle continuer. Vérifier `ctx.Err()` dans `EscapeAnalysis`.
- **A-147** — après le premier SIGINT, tous les signaux suivants sont avalés dans `main.go:67` : un second Ctrl+C ne peut pas forcer l'arrêt.
- **A-021** — la reprise ne capture ni ne compare la provenance : une reprise après changement de toolchain consigne des mesures sous la provenance de départ.
- **A-044** — un verrou orphelin bloque la reprise de la campagne qu'il protège. Accepter le verrou s'il nomme la campagne qu'on reprend (avec précaution sur la vivacité du processus d'origine).
- **A-265** (majeur) — la reprise réutilise les drapeaux de la ligne de commande (`--benchtime`, `--cpu`) au lieu de ceux de la campagne. Ajouter ces paramètres dans `models.Campaign` et les restaurer à la reprise.
- **A-030** — la reprise ignore en silence `--matrix`, `--count` et `--hypotheses` ; les refuser.
- **A-024** — quand la consignation de l'abandon échoue, l'erreur d'origine est perdue.
- **A-028** — aucun test n'assure la libération du verrou sur les chemins d'erreur.

### Lot 3 — Intégrité des écritures de résultats et ordonnancement

Indépendant des lots 1 et 2, parallélisable avec eux.

- **A-043** (majeur) — la garde d'immutabilité est un contrôle d'existence suivi d'un renommage : deux processus peuvent écraser un fichier de résultats. Rendre l'écriture atomique et exclusive (`os.Link`) pour les créations immuables, en préservant le renommage pour les mises à jour de statut légitimes (`SetCampaignStatus`).
- **A-263** (majeur) — le verrou est acquis dans `measure()`, après que `CreateCampaign` a écrit `campaign.json` au statut `RUNNING` : une campagne orpheline subsiste en cas de conflit de verrou, et la dérivation d'identifiant (`NextCampaignID`) est exposée à une collision. Poser le verrou avant la création de la campagne.
- **A-046** — la transition de statut n'applique pas la seule transition admise par BR-003-3.
- **A-053** — les gardes traitent toute erreur d'accès autre que « absent » comme « absent ».
- **A-054** — deux écritures n'exigent ni identifiant non vide ni campagne existante.
- **A-050** — pas de synchronisation avant renommage : un fichier de résultats peut être vide après coupure.
- **A-052** — l'horodatage à la seconde produit un refus d'immutabilité trompeur.
- **A-048** — le tri des rapports de verdicts n'est pas déterministe entre rapports de même seconde.
- **A-055** — les fichiers temporaires d'une écriture interrompue ne sont ni nettoyés ni ignorés.

### Lot 4 — Interruption et arbre de processus

Indépendant. C'est le lot le plus délicat à vérifier, parce que rien ne le teste aujourd'hui.

- **A-062** (majeur) — l'annulation ne tue que la commande `go`, pas l'arbre : l'attente bloque jusqu'à la fin du benchmark orphelin (`cmd.WaitDelay` absent, Job Object Windows sans `KILL_ON_JOB_CLOSE`, pas de `Setpgid` Unix). Poser un délai d'attente et terminer l'arbre de processus.
- **A-063** — le message d'échec est tronqué au milieu d'une séquence UTF-8.

**Vérification exigée.** Ce lot doit laisser derrière lui un test qui interrompt réellement une commande longue et vérifie qu'aucun processus ne survit et que la campagne n'est pas close `COMPLETED`.

### Lot 5 — Génération de matrice, validation du modèle et ports

Indépendant.

- **A-001** (majeur) — la génération supprime tous les profils sauf un dès que les dimensions de répétition ou de charge ne contiennent pas la valeur 1. Choisir les listes par profil au lieu de sauter la combinaison.
- **A-156** (majeur) — `--benchtime` n'est validé nulle part : une faute de frappe entraîne une campagne entière de sujets FAILED. Valider la durée au CLI et dans le service.
- **A-123** (complément architectural) — définir `ports.ErrNotFound` au niveau du port `internal/ports`, aligner `store.ErrNotFound` et les fakes de test pour permettre une discrimination typée (`errors.Is`).
- **A-002**, **A-003**, **A-010**, **A-011**, **A-012**, **A-013**, **A-017** — réplicats acceptés sans sujet à répliquer, matrice vide sans message utile, commentaire périmé, invariants manquants, identifiant de sonde non canonique, saut évalué trop profond, tests manquants.
- **A-005** à **A-009**, **A-015**, **A-016** — les validations du modèle n'exigent pas ce que le modèle d'entités déclare requis : horodatages de campagne, énoncé d'hypothèse, identifiant de campagne d'un verdict, cohérence du verdict d'échappement, bornes de l'attestation de quiétude, valeurs finies et non négatives.

Ces validations sont le filet du banc : chacune est un `if` de trois lignes et un sous-test. Le lot est volumineux en nombre, faible en risque.

### Lot 6 — Comparaison, verdicts et tableau de bord

Dépend du lot 0 (non-régression) et vient après le lot 1, qui touche le même chemin de production de verdicts.

- **A-032** (majeur) — le point de bascule disparaît silencieusement de toute série portant plusieurs paires par taille, pas seulement des séries répliquées. Consigner l'exclusion ou étendre la clé de série.
- **A-034** (majeur) — le fichier de verdicts est écrit avant la régénération du tableau de bord : un échec de la dernière étape laisse `results/` modifié, contre la postcondition d'échec de UC-005.
- **A-123** (majeur) — la collecte des preuves avale toute erreur de lecture du fichier de comparaison : un fichier corrompu devient un verdict « non concluant, exécuter compare » au lieu d'un échec. Ne rattraper que `ports.ErrNotFound`.
- **A-185** (majeur) — UC-005 A1 : le message d'erreur en cas de critères modifiés ne nomme pas les hypothèses altérées, alors que `specs.ChangedCriteria` existe précisément pour cet usage.
- **A-187** (majeur) — BR-004-2 : aucun test ne valide que « intervalle de confiance excluant zéro implique `Significant == true` », permettant à des mutations à seuil de survivre.
- **A-278** (majeur) — la colonne Integration du tableau de bord est un faux positif intégral issu d'une détection de sous-chaîne `//go:build integration_test` dans le code et les tests sans directive réelle.
- **A-035**, **A-036**, **A-037**, **A-038**, **A-040**, **A-041**, **A-042** — fichier d'échappement non vérifié contre la matrice, verdicts rendus sur corpus partiel sans consigner les écarts, quantile tronqué d'un rang, intervalle dégénéré sur variance nulle, duplication du chemin de campagne, bornes sans test au point exact, méthode recopiée en texte.
- Les constats sur la lecture du catalogue et sur l'écriture du tableau de bord (statut de cas d'utilisation non validé, tableau des hypothèses reconnu sans ancrage, écriture non atomique de `docs/dashboard.md`) appartiennent au même chemin et se traitent ici.

### Lot 7 — Outillage : hooks, CI, Makefile, skills

Indépendant de tous les autres. C'est le lot qui protège les suivants, donc à faire tôt.

- **A-102** (majeur) — la garde des chemins est ouverte en cas de défaillance : sans l'outil d'extraction JSON, toute écriture protégée passe en silence. La fermer.
- **A-103** (majeur) — la garde se contourne par un segment `..` dans le chemin.
- **A-104** (majeur) — le contrôle des hooks pose et supprime le verrou réel du dépôt : lancé pendant une campagne, il la déprotège.
- **A-105** — les motifs sont sensibles à la casse, ce qui les rend contournables sous Windows.
- **A-106**, **A-107** — la CI ne tourne que sous Linux, n'exerce jamais la toolchain des campagnes ni le code Windows, et n'a ni délai ni permissions minimales.
- **A-109** et les constats voisins sur le contrôle de forme des cas d'utilisation, le hook d'arrêt sans l'outil JSON, la couverture de la garde limitée à deux outils d'édition, et les skills qui citent des chemins ou des drapeaux inexistants.

### Lot 8 — Ligne de commande et adaptateurs système

Indépendant.

- Ligne de commande : rapport vide imprimé sur échec, erreur de drapeau imprimée deux fois, `-h` traité comme une erreur, clé répétée écrasée en silence, hypothèses non dédoublonnées, racine de projet résolue au projet englobant, usage qui décrit mal les valeurs par défaut, rationale inséré dans un tableau sans échappement.
- Adaptateurs système : modèle de processeur mal lu sous Windows, disposition mémoire supposant une architecture, décompte de cœurs suivant le masque d'affinité, défauts de page exigés par C-010 et jamais relevés, absence de test sur la mesure du temps de l'arbre de processus.

### Lot 9 — Gabarits du harnais et périmètre de l'empreinte (sur décision explicite)

**Ce lot change `Campaign.harnessDigest`.** Toute matrice et toute campagne antérieures deviennent incomparables, et les résultats archivés ne peuvent plus être étendus. Il ne s'exécute que sur décision du chercheur, et en une seule fois pour ne payer le prix qu'une fois.

- **A-246** — l'empreinte ne couvre que les gabarits, pas le code Go qui façonne la source rendue. Une modification de la dérivation des champs change le code généré sans changer l'empreinte, ce que C-005 et BR-003-1 interdisent. Soit étendre l'empreinte à ce fichier, soit corriger la spécification pour dire ce que l'empreinte couvre réellement. La première voie change l'empreinte, la seconde non : c'est le seul arbitrage du lot qui mérite une décision écrite.
- **A-073** — la garde de taille des types générés ne détecte qu'un type trop petit ; un type trop grand compile.
- **A-081** — la ligne de cache est figée à 64 octets, ce qui bloquera le rejeu sur arm64 que C-006 souhaite.

Si le chercheur préfère garder les campagnes comparables, ce lot reste ouvert et les constats sont consignés comme dette assumée. A-246 mérite alors au moins sa correction documentaire, qui ne coûte rien.

### Lot 10 — Simplifications

Dernier, sans risque, à faire quand tout le reste est vert : code mort, duplications de la bibliothèque standard, commentaires déplacés, paramètres inutiles.

### Conduite en mode Ultracode

Pour chaque lot, une session neuve, un workflow, un commit.

**Forme du workflow.** Un agent par constat pour l'implémentation, en parallèle quand les constats touchent des fichiers différents, en série quand ils touchent le même. Puis, sur le diff complet du lot, une passe de vérification à trois lentilles distinctes : un relecteur de conformité au cas d'utilisation, un chercheur de régression qui rejoue le harnais de non-régression et la suite complète, et un sceptique chargé de trouver ce que le correctif a cassé ailleurs. Un correctif n'est retenu que si les trois passent.

**Ce qu'il faut donner à chaque agent.** L'identifiant du constat, sa fiche complète depuis cet audit, le texte du cas d'utilisation concerné, et la règle du projet qui s'applique. Ne pas donner l'audit entier : les agents doivent lire le code, pas un résumé du code.

**Règle d'arrêt.** Un lot s'arrête quand la barre de sortie est verte et que le harnais de non-régression montre les verdicts archivés inchangés — sauf au lot 1, où le changement de H-006 est précisément le but et doit être constaté explicitement.

**Ordre recommandé.** Lot 0, puis lot 7 qui protège les suivants, puis le lot 1 qui porte le résultat faux, puis les lots 2 à 6 en parallèle, puis 8 et 10. Le lot 9 attend une décision.

## Inventaire des constats

<!-- 290 constats bruts · 190 après dédoublonnage initial · 111 confirmés (3 bloquants, 20 majeurs, 53 mineurs, 35 suggestions) · 4 écartés · 75 sans vérification complète -->


### Bloquants (3)

#### A-072 · « new(payload) escapes to heap » est classé OTHER alors que la cause est le retour du pointeur p : H-006 est réfutée à tort

`internal/adapters/escape/escape.go:155` — correctness — effort M — exigences UC-002, BR-002-1, BR-002-2, H-006, C-008, NFR-002, UC-005

**Défaut.** escapedIdentifier ne reconnaît qu'un identifiant nu (« moved to heap: t », « &t escapes to heap »). Toute expression d'allocation (« new(payload) escapes to heap ») rend la chaîne vide, classifyDiagnostic rend OTHER sans regarder l'AST. Or dans produceValueAlloc (cell.go.tmpl l. 93-101) l'allocation est affectée à p puis « return t, p » : c'est la première cause du livre (retour de pointeur), pas une cause hors des quatre. Vérifié de bout en bout dans le scratchpad (module de remplacement, 304 cellules rendues par le harnais courant, go build -gcflags=-m réel sur go1.27.0, Classify) : les 32 cellules RETURNED_ALLOCATING*/VALUE (8/16/24/4096 octets, ARRAY_FILL et NAMED_FIELDS, avec et sans champ pointeur, k=1 et k=2) sont toutes OTHER avec cette raison ; le bras POINTER n'y échappe que parce que « moved to heap: t » (l. 57) est trié avant « new(payload) » (l. 60) et l'emporte. Dans…

**Scénario d'échec.** Entrée : cellule Size0024Plain/RETURNED_ALLOCATING/VALUE, lignes du compilateur go1.27.0 = ["subjects\\...\\subject.go:60:10: new(payload) escapes to heap"] (sortie réelle, colonne 10 = parenthèse de l'appel). Sortie : Category=OTHER, alors que le corps englobant contient « return t, p » avec p = new(payload). Conséquence : evaluateH006 compte la cellule hors livre → H-006 REFUTED (C-2026-09-10-7, C-2026-09-10-11), verdict repris dans docs/dashboard.md. Règle : BR-002-2 dit qu'OTHER est réservé aux raisons hors des quatre causes ; ici la raison est le retour d'un pointeur.

**Correctif.** Dans classifyDiagnostic (escape.go), quand escapedIdentifier rend "" et que le message est « <expr> escapes to heap » avec expr une allocation (new(T), &T{...}, make(...)), retrouver dans enclosingBody l'AssignStmt dont un Rhs est un *ast.CallExpr/UnaryExpr placé à diag.Line (fset.Position(call.Lparen) == ligne:colonne, ou à défaut la ligne seule) et prendre l'Ident du Lhs correspondant comme porteur ; appeler classifyUsage avec ce nom en semant aliases[nom]=true (le porteur EST le pointeur, il n'y a pas de « & »), de sorte que « return t, p » donne RETURN_POINTER, « ch <- p » CHANNEL_SEND, « m[0] = p » CONTAINER_STORE. Ajouter le cas table-driven « allocation retournée » dans escape_test.go (source : var p *payload; p = new(payload); return t, p ; message « new(payload) escapes to heap » ; attendu RETURN_POINTER). Ne touche pas internal/harness/templates : digest inchangé. Conséquences…

**Précision apportée en vérification.** Le correctif du constat est juste dans son principe ; trois précisions : 1) classifyUsage (escape.go l.222) construit lui-même aliases via addressCarriers et ne peut pas être semé : extraire une variante classifyCarrier(body, carrier string) qui appelle la même marche avec aliases = {carrier: true} ∪ addressCarriers(body, carrier) et captured = idem, de sorte que carriesAddress reconnaisse l'Ident `p` dans `return t, p` (RETURN_POINTER), `ch <- p` (CHANNEL_SEND), `m[0] = p`/`c.P = p`/`append(s, p)` (CONTAINER_STORE). 2) Repérage du porteur : ne pas comparer fset.Position(call.Lparen) à diag.Column — la colonne rapportée par le compilateur ne correspond pas de façon stable au Lparen…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient.

#### A-020 · resume ne considère « faites » que les mesures COMPLETE : un sujet FAILED est remesuré puis son écriture est refusée par l'immuabilité, ce qui fait échouer toute reprise

`internal/service/campaign.go:176` — correctness — effort S — exigences UC-003, BR-003-3

**Défaut.** done n'inclut que MeasurementComplete. Un sujet consigné FAILED avant l'interruption (A3, flux normal) est relancé ; store.WriteMeasurement (store.go:275-279) refuse le fichier existant avec ErrImmutable (« existe déjà (BR-003-3) ») — memoryStore.WriteMeasurement fait de même (fakes_test.go:275-279). measure rend l'erreur, la campagne reste RUNNING, et chaque nouvelle reprise butera sur le même sujet. De plus les FAILED antérieurs ne sont pas comptés dans report.Failed. TestUC003_A4_RepriseApresInterruption n'écrit qu'une mesure COMPLETE.

**Scénario d'échec.** Campagne interrompue après 40 sujets dont le sujet 7 est FAILED (panic pendant la mesure). `--resume` : sujets 1..6 sautés, sujet 7 remesuré (7 s), WriteMeasurement → « results/campaigns/C-…/measurements/…json existe déjà (BR-003-3) » ; Run rend l'erreur, statut RUNNING inchangé. Toute reprise ultérieure échoue au même point ; la campagne est irrécupérable sans intervention manuelle sur results/.

**Correctif.** Dans resume, marquer done pour toute mesure écrite et propager son statut : `done[m.SubjectID] = m.Status` (map[string]models.MeasurementStatus) ; dans measure, `if st, ok := alreadyDone[subjectID]; ok { if st == models.MeasurementComplete { report.Measured++ } else { report.Failed++ }; continue }`. Test : écrire une mesure FAILED avant `--resume`, asserter que le sujet n'est pas dans runner.order, Failed == 1, statut COMPLETED.

**Précision apportée en vérification.** Spec d'abord, conformément au processus (« tout changement de comportement commence dans docs/ ») : réviser docs/use-cases/UC-003-executer-campagne.md, A4 étape 2, en « reprend au premier sujet sans Measurement écrite ; les Measurement FAILED antérieures sont conservées (BR-003-3) et comptées à l'étape 8 », avec une ligne de révision datée ; ajouter une phrase de Notes de revue expliquant qu'un sujet FAILED n'est jamais remesuré dans la même Campaign (une nouvelle campagne sert à cela). Ensuite, synchroniser le code exactement comme le constat le propose : `done map[string]models.MeasurementStatus` alimenté pour toute mesure lue ; dans measure, `if st, ok := alreadyDone[subjectID]; ok { if…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-140, A-183, A-262.

#### A-261 · Une interruption (Ctrl-C / SIGTERM) est consignée comme échec de sujet, la boucle continue et la campagne finit COMPLETED : A4 est inatteignable

`internal/adapters/gotool/gotool.go:208` — robustesse — effort S — exigences UC-003, BR-003-3, NFR-004

**Défaut.** Lors de l'annulation du contexte par signal (SIGINT/SIGTERM), `Toolchain.Run` intercepte l'erreur du processus tué et retourne une mesure au statut `models.MeasurementFailed` avec une erreur Go `err == nil`. Dans la boucle de mesure (`CampaignService.measure`, `campaign.go:198-223`), `err` étant nulle, l'annulation du contexte n'est jamais détectée. Le sujet en cours est écrit comme FAILED sur le disque. Aux itérations suivantes, le contexte étant déjà expiré, tous les sujets restants échouent instantanément en FAILED et sont écrits dans `results/`. Si au moins un sujet avait réussi avant le signal, la campagne termine sa boucle et est clôturée au statut `models.CampaignCompleted`. Le scénario A4 (reprise d'une campagne interrompue au statut RUNNING) devient structurellement impossible à déclencher.

**Scénario d'échec.** L'utilisateur lance une campagne de 250 sujets. Au 10e sujet, il tape `Ctrl+C`. Au lieu de s'arrêter proprement, `gotool` consigne le 10e sujet comme FAILED, puis boucle à toute allure sur les 240 sujets restants qui échouent tous en 0 ms. La campagne est marquée `COMPLETED` avec 9 mesures réussies et 241 échouées. Toute reprise `--resume` est refusée (« la campagne est au statut COMPLETED, la reprise exige RUNNING »).

**Correctif.** (1) Dans `Toolchain.Run` (`gotool.go`), si `ctx.Err() != nil`, retourner immédiatement `models.Measurement{}, ctx.Err()` au lieu de convertir le signal en mesure FAILED avec `err == nil`. (2) Dans la boucle de `CampaignService.measure` (`campaign.go`), vérifier `if ctx.Err() != nil { return report, ctx.Err() }` en tête de boucle. (3) Ne jamais passer le statut de la campagne à `COMPLETED` si le contexte a été annulé : laisser la campagne au statut `RUNNING` pour permettre le flux A4.

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient (contre-vérification 2026-09-12).

### Majeurs (12)

#### A-102 · guard-paths est fail-open : sans jq ou sur JSON non parsable, l'écriture protégée passe en silence

`.claude/hooks/guard-paths.sh:11` — robustesse — effort S — exigences BR-003-3, NFR-004, BR-001-2, BR-005-3, C-005

**Défaut.** `jq` est absorbé par `2>/dev/null \|\| true` ; FILE vide -> `exit 0` sans message. Vérifié : avec une fonction jq qui échoue (code 127), un Write vers results/x.json rend exit 0 ; idem quand le JSON d'entrée est invalide. jq n'est pas dans la bibliothèque standard du poste (installé via WinGet ici) ; un clone sur un poste sans jq désactive la garde promise par CLAUDE.md sans aucun signal. Même motif dans go-check.sh:7 et spec-lint.sh:8 (acceptable là, mais muet).

**Scénario d'échec.** Poste sans jq dans PATH, agent fait `Write docs/dashboard.md` -> hook exit 0, fichier écrasé (BR-005-3 violée sans garde). Reproduit : `printf '{"tool_input":{"file_path":".../results/x.json"}}' \| bash -c 'jq(){ return 127; }; export -f jq; bash guard-paths.sh'` -> exit 0.

**Correctif.** En tête : `command -v jq >/dev/null 2>&1 \|\| { echo 'guard-paths : jq introuvable, écriture refusée par précaution' >&2; exit 2; }` ; puis `FILE=$(... \| jq -r ...) \|\| { echo 'guard-paths : entrée JSON illisible' >&2; exit 2; }`. Dans go-check.sh et spec-lint.sh, remplacer le silence par un avertissement sur stderr. Ajouter au selftest un cas PATH sans jq attendant 2.

**Précision apportée en vérification.** Le correctif du constat convient ; deux précisions. (1) guard-paths.sh : remplacer la ligne 11 par un garde-fou explicite : `command -v jq >/dev/null 2>&1 \|\| { echo 'EscapeBench guard : jq introuvable, écriture refusée par précaution (installer jq)' >&2; exit 2; }` puis `FILE="$(printf '%s' "$INPUT" \| jq -r '.tool_input.file_path // empty')" \|\| { echo 'EscapeBench guard : entrée JSON illisible' >&2; exit 2; }` ; garder `[ -z "$FILE" ] && exit 0` uniquement pour le cas légitime d'un outil sans file_path. Dans go-check.sh:7 et spec-lint.sh:8, conserver exit 0 mais écrire un avertissement sur stderr quand jq manque (`command -v jq >/dev/null 2>&1 \|\| { echo 'go-check : jq introuvable,…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-173.

#### A-103 · guard-paths contournable par un segment `..` : internal/../results/x.json n'est pas bloqué

`.claude/hooks/guard-paths.sh:17` — securite — effort S — exigences BR-003-3, BR-001-2, BR-005-3, C-005

**Défaut.** REL est obtenu par simple suppression du préfixe ROOT, sans normalisation. Un chemin absolu contenant `..` (ou `//`) donne un REL qui ne correspond à aucun motif du `case`. Vérifié en forme POSIX et en forme Windows (`C:\…\internal\..\results\a.json`) : exit 0.

**Scénario d'échec.** Edit avec file_path `<root>/internal/../results/campaigns/C-2026-09-10-1/campaign.json` -> exit 0, le fichier de campagne versionné est modifié (BR-003-3). Idem `<root>//results/x.json` -> exit 0 (vérifié).

**Correctif.** Normaliser avant le `case` : `FILE=$(realpath -m -- "$FILE")` (coreutils, présent sous Git Bash et ubuntu) ou, en bash pur, refuser d'emblée `case "$FILE" in *..*\|*//*) deny "chemin non normalisé";; esac`. Ajouter au selftest `check 2 guard-paths.sh "$DIR" "$(j internal/../results/x.json)"`.

**Précision apportée en vérification.** Le correctif proposé respecte le processus (hook seul, ni results/, ni templates, ni H-###). Précision : préférer la variante bash pure, qui ne dépend pas de realpath et ne change pas la forme du chemin (realpath -m rend `/c/...` pour une entrée `/c/...` mais `C:/...` pour une entrée `C:/...`, donc un mélange FILE en forme POSIX / ROOT en forme Windows casserait la suppression de préfixe ligne 17). Insérer après la ligne 14 : `case "$FILE" in *..*\|*//*) REL="$FILE"; deny "chemin non normalisé (segment .. ou //) ; fournir le chemin canonique.";; esac` — en déplaçant la définition de deny() avant ce case. Effet collatéral acceptable : le faux positif `results/../internal/...` devient un…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient.

#### A-104 · selftest pose et supprime le verrou réel results/.campaign-lock du dépôt, pas une copie

`.claude/hooks/selftest.sh:40` — concurrence — effort S — exigences BR-003-1, BR-003-3, C-005, UC-003

**Défaut.** Le contrôle « harness sous verrou » écrit `$ROOT/results/.campaign-lock` puis le `rm -f`. Store.AcquireLock (store.go:350) pose ce même fichier en O_EXCL pour la durée d'une campagne, et ReleaseLock ignore ErrNotExist. Deux effets : (1) lancé pendant une campagne réelle, le selftest supprime le verrou de la campagne en cours ; une seconde campagne peut alors démarrer et les deux runners écrivent sous results/ (BR-003-1, BR-003-3, C-005) ; (2) interrompu (Ctrl-C) entre les lignes 40 et 42, le trap ne retire que $TMP et laisse un verrou orphelin : `matrix` et `campaign` refusent (service/matrix.go:65 ErrPrecondition) et internal/harness reste gelé jusqu'à suppression manuelle.

**Scénario d'échec.** Campagne C-x en cours (verrou présent, 1 h 08 selon LANCEMENT.md) ; `bash .claude/hooks/selftest.sh` -> « ÉCHEC harness sans verrou », puis `rm -f` retire le verrou ; `go run ./cmd/escapebench campaign --matrix M-…` dans un autre terminal -> AcquireLock réussit, deux campagnes écrivent en parallèle, H-013 non concluante (D-35).

**Correctif.** Pointer CLAUDE_PROJECT_DIR vers un faux projet sous $TMP pour ce contrôle : `mkdir -p "$TMP/proj/results"; : > "$TMP/proj/results/.campaign-lock"; check 2 guard-paths.sh "$TMP/proj" "$TMP/proj/internal/harness/x.go"` (guard-paths ne lit que `$ROOT/results/.campaign-lock`). Sinon : refuser de tourner si le verrou existe déjà, et ajouter le verrou au trap EXIT.

**Précision apportée en vérification.** Le correctif du constat fonctionne — guard-paths.sh:30 ne consulte que `$CLAUDE_PROJECT_DIR/results/.campaign-lock`, vérifié — mais il faut le décliner par forme de chemin, sinon la branche `windows` perd sa raison d'être. Dans selftest.sh, avant la boucle `for form` : `mkdir -p "$TMP/proj/results" "$TMP/proj/internal/harness"`. Puis dans la boucle, en tête (avec `$DIR` déjà calculé) : if [ "$form" = windows ]; then PDIR="$(cygpath -w "$TMP/proj")"; else PDIR="$TMP/proj"; fi jp() { printf '%s%s%s' "$PDIR" "${SEP:0:1}" "$(printf '%s' "$1" \| tr '/' "$SEP")"; } et remplacer les l. 39 à 42 par : check 0 guard-paths.sh "$PDIR" "$(jp internal/harness/x.go)" "$form harness sans verrou" : >…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-171.

#### A-101 · Cible integration_test sans aucun test taggé : elle rejoue la suite unitaire en se faisant passer pour des tests d'intégration

`Makefile:20` — tests — effort M — exigences UC-001, UC-002, FR-007

**Défaut.** Aucun fichier du module ne porte `//go:build integration_test` (grep -rn "go:build" --include=*.go : uniquement des tags windows/linux/!windows). La cible exécute donc exactement la suite unitaire (sortie identique, 7 s), sans `-shuffle=on` (diverge de la commande de référence de CLAUDE.md), et le commentaire promet des adapters réels (compilateur, système de fichiers). La CI ne l'appelle pas non plus ; elle recopie les commandes du Makefile au lieu de l'invoquer, ce qui laisse les deux dériver (l'en-tête dit « le Makefile dit QUOI, le CI dit OÙ »).

**Scénario d'échec.** `make integration_test` sur le dépôt actuel -> `ok github.com/agbruneau/escapebench/internal/service 1.229s` etc. : mêmes paquets, mêmes tests que `make test`. Un adapter gotool cassé sur la toolchain réelle (parsing de -gcflags=-m avec go1.27) n'est détecté par aucune cible.

**Correctif.** Soit écrire les tests taggés (`//go:build integration_test` en tête de fichier) pour gotool (compilation réelle d'une cellule, lecture de -m) et store (round-trip disque) nommés TestUC001_/TestUC002_ ; soit retirer la cible, le commentaire, et la colonne Integration (index.go, dashboard.go, ports.go). Dans les deux cas ajouter `-shuffle=on` et faire appeler `make vet test` par la CI (make est présent sur ubuntu-latest).

**Précision apportée en vérification.** Processus d'abord : (1) docs/use-cases/UC-005-produire-verdicts.md étape 7 (et FR-007 si on veut une exigence) doit définir la colonne Integration — soit « ✔ si un fichier _test.go porte la contrainte //go:build integration_test en tête et référence le UC », soit retirer la colonne — et trancher si UC-001/UC-002 exigent des tests d'intégration ; consigner le choix dans Doc/DECISION.md §5 (point « Colonne Integration ») et .claude/skills/spec-coverage/SKILL.md:22. (2) Code, synchronisé sur la spec : internal/adapters/specs/index.go:48 — ne reconnaître la contrainte que comme ligne de contrainte de build en tête de fichier (go/build/constraint ou préfixe des premières lignes), jamais par…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-099, A-189.

#### A-062 · L'annulation ne tue que `go test`, pas l'arbre : Wait bloque jusqu'à la fin du benchmark orphelin et le Job Object n'est pas utilisé pour terminer

`internal/adapters/gotool/gotool.go:50` — robustesse — effort M — exigences UC-003, C-010, BR-003-4

**Défaut.** exec.CommandContext installe Cancel = Process.Kill, qui ne termine que go.exe ; le binaire de test enfant hérite des tubes stdout/stderr (strings.Builder ⇒ tubes + goroutines de copie) et continue le benchmark. Sans cmd.WaitDelay, Wait attend la fermeture des tubes, donc la fin de la benchtime restante de l'orphelin. Sous Windows, le Job Object créé dans treecpu_windows.go:55 (create.Call(0, 0)) n'a ni JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE ni TerminateJobObject dans Cancel, alors qu'il regroupe précisément l'arbre à tuer. Sous Linux/darwin, pas de Setpgid ni de kill du groupe.

**Scénario d'échec.** Reproduit (scratchpad, Windows) : cancel() à 3 s, `Wait a rendu après 13.7s` — le sub.test.exe orphelin a terminé ses 10 s de benchtime en consommant un cœur, puis Wait a rendu. Avec `--benchtime 5s --count 20`, un `taskkill escapebench` ou un ctx à délai laisse un processus de mesure tourner jusqu'à 100 s après l'annulation, et gonfle la fraction d'occupation de tout autre processus qui mesurerait à ce moment. Le Ctrl-C console masque le cas (le signal atteint tous les processus de la console), pas SIGTERM/taskkill/délai.

**Correctif.** Dans ExecRunner : `cmd.WaitDelay = 2 * time.Second`. Sous Windows, créer le job avant Start via un tracker pré-alloué, poser `cmd.Cancel = func() error { TerminateJobObject(job, 1); return cmd.Process.Kill() }` (kernel32 TerminateJobObject, syscall seulement, C-002) ou SetInformationJobObject avec JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE. Hors Windows : `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` et `cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }` dans treecpu_other.go (tag !windows, à scinder si plan9/js sont visés).

**Précision apportée en vérification.** Le correctif proposé est juste dans l'idée, incomplet sur trois points. 1. `cmd.WaitDelay = 2 * time.Second` dans ExecRunner : confirmé, il débloque Wait mais ne tue pas l'orphelin ; il est nécessaire, pas suffisant. 2. Windows : créer le job AVANT `cmd.Start()` (CreateJobObjectW ne demande pas de pid ; seul AssignProcessToJobObject attend le pid), puis `SetInformationJobObject(job, JobObjectExtendedLimitInformation=9, &JOBOBJECT_EXTENDED_LIMIT_INFORMATION{BasicLimitInformation.LimitFlags: JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE=0x2000})`. Cela couvre aussi le cas `taskkill /F escapebench` (le handle se ferme à la mort du parent et l'arbre meurt), que le `Cancel` proposé ne couvre pas. Poser…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-143, A-177.

#### A-044 · Un verrou orphelin bloque la reprise A4 : AcquireLock refuse même la campagne qu'on reprend

`internal/adapters/store/store.go:357` — robustesse — effort S — exigences UC-003, BR-003-3

**Défaut.** UC-003 A4 se déclenche précisément quand « le processus n'est plus actif » (docs/use-cases/UC-003-executer-campagne.md:56). Si ce processus est mort par plantage, SIGKILL ou coupure, le defer ReleaseLock de campaign.go:195 n'a pas tourné et results/.campaign-lock subsiste. resume() appelle measure() qui appelle AcquireLock avec O_EXCL : le verrou existant fait échouer la reprise, et aussi UC-001 (matrix.go:60, LockHeld). Aucune sous-commande ni cible Makefile ne retire le verrou (grep « lock » dans cmd/ et Makefile : rien) ; le chercheur doit supprimer le fichier à la main, ce que le hook guard-paths interdit à un agent. Le contenu du verrou (l'identifiant de campagne, écrit ligne 364) n'est jamais relu.

**Scénario d'échec.** Campagne C-2026-09-11-1 tuée à mi-course (verrou présent, statut RUNNING). `escapebench campaign --resume C-2026-09-11-1` : LoadCampaign OK, digest OK, LoadMeasurements OK, puis AcquireLock rend « une campagne est déjà en cours » ; la reprise A4 est impossible sans intervention manuelle sous results/.

**Correctif.** Dans AcquireLock, sur ErrExist : lire le contenu du verrou ; s'il vaut exactement campaignID+"\n" (reprise de la même campagne), accepter et continuer ; sinon rendre l'erreur en nommant le détenteur. Ajouter un test TestLock_RepriseMemeCampagne. Alternative plus lourde : sous-commande `unlock`.

**Précision apportée en vérification.** Spec d'abord, conformément au processus (« spécification modifiée avant tout code », UC-003 révision) : dans docs/use-cases/UC-003-executer-campagne.md, A4, ajouter une étape « le système reprend le verrou results/.campaign-lock s'il porte l'identifiant de la Campaign désignée ; s'il porte un autre identifiant, la reprise est refusée en nommant le détenteur », et préciser dans BR-003-3 que le verrou est « créé au démarrage ou repris à la reprise (A4) ». Ni critère gelé H-### ni gabarit internal/harness/templates/*.tmpl ne sont touchés (harnessDigest inchangé) ; aucune écriture sous results/ par un agent. Ensuite, code : dans store.go AcquireLock, sur ErrExist lire le fichier ; si le contenu concorde avec campaignID, accepter le verrou. *Note de contre-vérification :* vérifier également la vivacité du processus d'origine (via détection de PID ou verrou adhésif du noyau) pour éviter qu'un opérateur ne lance deux reprises concurrentes sur la même campagne active.

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-022, A-264.

#### A-043 · writeFileAtomic écrase silencieusement une cible existante : la garde d'immutabilité Stat-puis-Rename est un TOCTOU

`internal/adapters/store/store.go:558` — concurrence — effort M — exigences NFR-004, BR-003-3, BR-001-2, BR-004-3, UC-003

**Défaut.** Toutes les gardes d'immutabilité (Finalize, WriteEscapeReport, CreateCampaign, WriteMeasurement, WriteComparisonSet, WriteVerdictReport) font un os.Stat puis, séparément, writeFileAtomic qui termine par os.Rename. Or os.Rename remplace une cible existante sur Linux comme sur Windows (vérifié : `go run` en scratchpad, « Rename sur cible existante: err=<nil> contenu="nouveau" »). L'immutabilité NFR-004 / BR-001-2 / BR-002-3 / BR-003-3 / BR-004-3 n'est donc garantie que pour un seul processus. Dans le service, le verrou n'est pris qu'après NextCampaignID et CreateCampaign (campaign.go:132-192), donc deux lancements simultanés d'`escapebench campaign` calculent le même identifiant et le second CreateCampaign réécrit le campaign.json du premier sans erreur.

**Scénario d'échec.** Deux processus `escapebench campaign --matrix M` démarrés dans la même seconde : les deux NextCampaignID rendent C-2026-09-11-1 ; les deux Stat de CreateCampaign voient le fichier absent ; les deux Rename réussissent, le second écrasant le campaign.json du premier (startedAt et provenance.capturedAt du perdant). Puis AcquireLock : un seul gagne, l'autre échoue avec « une campagne est déjà en cours » après avoir corrompu le fichier du gagnant.

**Correctif.** Dans writeFileAtomic, ajouter un mode exclusif pour les cibles immuables : os.Link(tmpName, path) puis os.Remove(tmpName) (os.Link échoue avec ErrExist sur Linux et Windows/NTFS — vérifié en scratchpad : « Cannot create a file when that file already exists ») ; convertir errors.Is(err, os.ErrExist) en ErrImmutable ; garder Rename uniquement pour SetCampaignStatus (qui applique une transition légitime de statut RUNNING -> COMPLETED/ABORTED). Complément dans le service : appeler AcquireLock avant NextCampaignID (campaign.go:132) pour que l'identifiant soit choisi sous verrou. Prévoir un repli Rename si Link n'est pas supporté (FAT/exFAT).

**Précision apportée en vérification.** Le correctif du constat est juste sur le fond mais inverse la priorité et oublie un détail d'ordre. Correctif retenu, en deux temps, implémentation seule (aucun .tmpl, aucun H-###, aucune écriture sous results/ par l'agent) : (a) Racine du scénario dénoncé : dans internal/service/campaign.go, poser le verrou avant NextCampaignID (ligne 132) et non dans measure() (ligne 192), puis le relâcher par defer dans start()/resume() ; comme AcquireLock(ctx, campaignID) écrit l'identifiant alors qu'il n'est pas encore connu, et que personne ne lit le contenu du verrou (grep LockName : guard-paths.sh ne teste que l'existence), poser le verrou avec un contenu vide puis, après CreateCampaign, réécrire… (b) Distinguer dans le store `writeFileAtomicExclusive` (utilisant `os.Link` pour les créations immuables) et `writeFileAtomic` (utilisant `os.Rename` pour `SetCampaignStatus`).

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient.

#### A-001 · Expand supprime tous les profils autres que RETURNED_ALLOCATING dès que Repeats ou Payloads ne contient pas la valeur 1

`internal/models/matrix.go:365` — correctness — effort S — exigences UC-001, BR-001-3, H-010, C-008

**Défaut.** Les sauts de répétition et de charge sont écrits « si repeat > 1 et profil ≠ RETURNED_ALLOCATING, sauter » au lieu de « pour les autres profils, produire une seule fois avec repeat = payload = 1 ». La déclinaison des profils ordinaires dépend donc de la présence de 1 dans la liste demandée. Le commentaire (« ailleurs elle ne créerait que des doublons ») et docs/entity-model.md (« seul le profil RETURNED_ALLOCATING s'en décline ») décrivent une dimension inerte pour les autres profils, pas une dimension qui les efface. TestExpandAvecDispositionsEtRepetitions et TestC008_ChargesParInstance utilisent toujours des listes contenant 1 et ne peuvent pas détecter le cas. Les cinq matrices existantes sous matrices/ (`M-57477f022103`, `M-823d8b5af441`, `M-8f03757ac206`, `M-abb3d708d0e1`, `M-b44a93baae51`) contiennent 1 dans repeats et payloads : le correctif ne change ni leur contenu ni leur identifiant (Canonical ne bouge pas).

**Scénario d'échec.** Confirmé par go test sur une copie hors dépôt : MatrixParameters{Sizes:[24], Profiles:[LOCAL, RETURNED_ALLOCATING], Repeats:[4]}.Expand() rend 2 cellules seulement (Size0024Plain/RETURNED_ALLOCATING_R4/VALUE et /POINTER), le profil LOCAL disparaît sans erreur. ReferenceParameters() + Payloads:[2] (la demande naturelle pour éprouver H-010 à k = 2) rend 0 cellule ; NewMatrix échoue ensuite avec « Matrix.cells contient au moins une cellule », sans nommer la cause (UC-001 A1). Via la CLI : `repeats=4` ou `payloads=2` sans le 1.

**Correctif.** Dans Expand, choisir les listes par profil : `repeats, payloads := n.Repeats, n.Payloads; if profile != ProfileReturnedAlloc { repeats, payloads = []int{1}, []int{1} }` puis boucler sur ces listes sans les deux `continue`. Ajouter un test Repeats:[4] / Payloads:[2] sans 1 qui exige que LOCAL produise sa paire.

**Précision apportée en vérification.** Le correctif du constat est juste ; précision vérifiée : remplacer les deux boucles avec `continue` (matrix.go:362-373) par ```go repeats, payloads := n.Repeats, n.Payloads if profile != ProfileReturnedAlloc { repeats, payloads = DefaultRepeats(), DefaultPayloads() } for _, repeat := range repeats { for _, payload := range payloads { ``` (utiliser DefaultRepeats()/DefaultPayloads() plutôt que des littéraux, et déplacer le commentaire des lignes 363-364 au-dessus du `if`). Ajouter dans internal/models un test table-driven (p. ex. TestUC001_ExpandProfilsOrdinairesSansValeurUn) avec Repeats:[4] puis Payloads:[2] sans le 1, exigeant que LOCAL produise exactement sa paire VALUE/POINTER à repeat…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient.

#### A-021 · resume ne capture ni ne compare la Provenance : une reprise sur une autre toolchain consigne des mesures sous la provenance de départ

`internal/service/campaign.go:148` — robustesse — effort S — exigences UC-003, NFR-001, BR-003-2

**Défaut.** resume vérifie l'empreinte du harnais (A4, étape 1) mais n'appelle jamais s.provenance.Capture. Measurement ne porte pas de provenance propre (models.go:548-560) ; c'est Campaign.Provenance qui atteste NFR-001 pour toutes les mesures. Une reprise après mise à jour de Go (ou sur une autre machine partageant results/) mélange donc deux toolchains dans une campagne dont la Provenance affirme une seule. La précondition « verdicts d'échappement pour la toolchain courante » n'est pas non plus revérifiée. UC-003 A4 ne nomme que le contrôle d'empreinte : le correctif passe d'abord par docs/ (CLAUDE.md), sans toucher de H-###.

**Scénario d'échec.** Campagne démarrée sous go1.27.0, interrompue au sujet 100 ; `go` mis à jour en go1.27.1 ; `--resume` : 130 sujets mesurés avec go1.27.1, campaign.json dit go1.27.0, statut COMPLETED. UC-005 rend des verdicts H-### sur des mesures dont la provenance est fausse (NFR-001), sans erreur ni trace.

**Correctif.** Dans resume, `current, err := s.provenance.Capture(ctx)` puis refus `ErrPrecondition` si `!campaign.Provenance.SameToolchain(current) \|\| campaign.Provenance.CPUModel != current.CPUModel`. Ajouter l'étape à UC-003 A4 dans docs/use-cases/UC-003-executer-campagne.md avant le code. Test : fakeProvenance.provenance.GoVersion modifié avant `--resume` → ErrPrecondition, aucune mesure écrite.

**Précision apportée en vérification.** Le correctif du constat est juste sur le fond mais doit respecter l'ordre du processus (CLAUDE.md : « Tout changement de comportement commence dans docs/ ») : 1. docs/use-cases/UC-003-executer-campagne.md, A4 : ajouter une étape entre 1 et 2 — « Le système capture la Provenance courante et vérifie qu'elle identifie la même toolchain (goVersion, goos, goarch) et le même cpuModel que la Provenance de la Campaign ; sinon la reprise est refusée et la Campaign reste RUNNING (NFR-001, BR-003-2). » Mettre à jour la ligne « Révision » de l'en-tête. Aucun H-### touché, aucun gabarit du harnais touché (harnessDigest inchangé, C-005/BR-003-1 intacts), aucune écriture sous results/. Il n'est pas…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-266.

#### A-032 · hasDuplicateSizes supprime le point de bascule de toute série à plusieurs paires par taille, pas seulement des séries répliquées, sans trace dans le fichier

`internal/service/compare.go:219` — conformite-spec — effort M — exigences UC-004, C-009

**Défaut.** Le commentaire et la note de revue de UC-004 présentent l'exclusion comme la signature d'une série répliquée (C-009). Or les profils déclinés par Repeats et Payloads (RETURNED_ALLOCATING) produisent aussi plusieurs Comparison par taille, et la garde les élimine : ni valeur ni « non observé » n'est consigné pour ces couples, contrairement à l'étape 5 et à A3 de UC-004 (« pour chaque couple … s'il n'en existe pas, non observé »). Vérifié sur results/campaigns/C-2026-09-10-11/comparison-20260910T232754Z.json : 266 Comparison, doublons de taille sur (RETURNED_ALLOCATING, ARRAY_FILL, 8/16/24…), et tippingPoints ne contient aucune entrée RETURNED_ALLOCATING alors que toutes les autres séries y figurent. Aucun critère gelé ne lit ce point de bascule, le verdict n'est pas affecté.

**Scénario d'échec.** Matrice avec Repeats {1,2,4} ou Payloads {1,2,3} sur RETURNED_ALLOCATING -> Compare -> tippingPoints sans entrée pour ce profil -> l'affichage de l'étape 7 et le fichier ne disent ni bascule ni « non observé » pour ce couple, sans raison consignée.

**Correctif.** Soit inclure Repeat et Payload dans la clé de série (TippingKey gagne deux champs, DTO du fichier de comparaison à étendre en omitempty), soit consigner l'exclusion : ajouter à ComparisonSet une liste des séries dont le point de bascule n'est pas calculé avec la raison (« plusieurs paires par taille »), et corriger le commentaire de compare.go:215-218 ainsi que la note de revue de UC-004 pour nommer les répétitions et charges.

**Précision apportée en vérification.** Retenir l'option 2 du constat (consigner l'exclusion) et l'étendre ; l'option 1 est déconseillée : docs/requirements.md:39 (C-009) écrit explicitement qu'ajouter un champ à TippingKey casse le littéral de evaluateH002 (internal/service/criteria.go:98) et rendrait H-002 non concluante sans trace ; il faudrait aussi ajouter Repeat/Payload à Comparison et à son DTO, ce que le fichier ne porte pas aujourd'hui. Correctif minimal, docs d'abord (CLAUDE.md) : 1. docs/use-cases/UC-004 : ajouter à l'étape 5/A3 un cas « série à plusieurs paires par taille (réplicats C-009, répétitions et charges C-008) : point de bascule non calculé, raison consignée » et corriger la note de revue C-009 pour nommer…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-285.

#### A-034 · Le fichier de verdicts est écrit avant la régénération du tableau de bord ; un échec de l'étape 7 laisse results/ modifié et perd le chemin écrit

`internal/service/verdict.go:121` — conformite-spec — effort M — exigences UC-005, FR-007, BR-005-3

**Défaut.** Produce écrit le rapport (étape 6) puis appelle Dashboard (étape 7), qui exécute la suite de tests, l'index de code et l'écriture de docs/dashboard.md. Si l'une de ces étapes échoue, Produce rend une erreur sans VerdictReportSummary : la postcondition d'échec de UC-005 (« results/ et docs/dashboard.md sont inchangés ») est violée et le chemin du fichier déjà écrit n'est pas rendu au chercheur. TestUC005_ErreursDesPorts (« tests injoignables », « index de code cassé », « tableau de bord en panne ») n'affirme que err != nil et ne regarde pas f.store.verdicts.

**Scénario d'échec.** go test injoignable (GOFLAGS cassé, disque plein) pendant Dashboard -> results/verdicts/<id>-<stamp>.json existe, Produce rend une erreur, l'utilisateur relance et obtient un second fichier de verdicts pour la même campagne.

**Correctif.** Exécuter les collectes faillibles de Dashboard (LoadUseCases, hypotheses.Load, tests.RunAll, code.References, RunUseCaseTests) avant WriteVerdictReport et ne garder après l'écriture que la composition des lignes et dashboard.Write ; ou rendre le summary partiel (Path renseigné) avec l'erreur. Ajouter dans TestUC005_ErreursDesPorts la vérification que f.store.verdicts est vide pour les trois cas de l'étape 7.

**Précision apportée en vérification.** Le correctif du constat est retenu, précisé sur deux points. 1. Réordonnancement (internal/service/verdict.go) : extraire de Dashboard une fonction de collecte (LoadUseCases, hypotheses.Load, tests.RunAll, code.References, RunUseCaseTests) appelée avant WriteVerdictReport ; ne garder après l'écriture que VerdictReports (qui doit lire le rapport tout juste écrit), la composition des lignes et dashboard.Write. Attention : VerdictReports et dashboard.Write restent nécessairement après l'étape 6, donc une fenêtre résiduelle subsiste (échec de lecture ou d'écriture de docs/dashboard.md). Pour cette fenêtre, ne pas supprimer le fichier de verdicts (NFR-004, immutabilité) : rendre le summary avec…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-186, A-279.

#### A-123 · gather ignore toute erreur de LatestComparisonSet : un fichier de comparaison corrompu ou illisible devient un verdict INCONCLUSIVE « exécuter compare »

`internal/service/verdict.go:184` — robustesse — effort M — exigences UC-005, BR-005-2

**Défaut.** `set, path, err := s.store.LatestComparisonSet(ctx, campaign.ID); if err == nil {…}` : l'erreur n'est ni inspectée avec errors.Is ni propagée. Le port ne définit aucune sentinelle « introuvable » (store.ErrNotFound vit dans l'adapter, que service ne peut pas importer), donc le service ne peut pas distinguer l'absence légitime d'un fichier d'un JSON invalide ou d'une erreur de permission. H-001, H-002, H-007 et H-012 rendent alors INCONCLUSIVE avec le rationale « aucun fichier de comparaison pour la campagne … ; exécuter `escapebench compare` », écrit dans results/verdicts et repris au tableau de bord. Règle CLAUDE.md : erreurs inspectées avec errors.Is aux bords.

**Scénario d'échec.** results/campaigns/C-X/comparison-….json tronqué (disque plein pendant un rename, édition manuelle) → `escapebench verdict --campaign C-X` réussit, écrit un fichier de verdicts où H-001/H-002/H-007/H-012 sont INCONCLUSIVE pour cause d'absence de comparaison, alors que le fichier existe ; le chercheur relance compare, qui crée un second fichier, sans jamais voir l'erreur de lecture.

**Correctif.** Déclarer `var ErrNotFound = errors.New("introuvable")` dans internal/ports (ou models), faire envelopper `%w` par store.LatestComparisonSet (et les autres lectures), puis dans gather : `switch { case err == nil: …; case errors.Is(err, ports.ErrNotFound): // pas de comparaison; default: return Evidence{}, err }`. Test : memoryStore renvoyant une erreur non-ErrNotFound → Produce doit échouer sans écrire.

**Précision apportée en vérification.** Le correctif proposé est bon dans son principe (sentinelle dans ports + switch dans gather) mais plus lourd que nécessaire, et il manque la propagation côté CLI. Version minimale : 1. internal/ports/ports.go : `var ErrNotFound = errors.New("introuvable")` (le paquet ports n'importe aujourd'hui ni errors ni rien d'autre que context/models : l'ajout est sans effet de bord). 2. internal/adapters/store/store.go:27 : remplacer `var ErrNotFound = errors.New("introuvable")` par `var ErrNotFound = ports.ErrNotFound`. Aucun `%w` supplémentaire n'est requis : tous les sites d'absence (store.go:330, 339, 439, 451, 523, 58, 77, 177, 290, 383, 411, 438) enveloppent déjà ErrNotFound, et tous les tests…

Vérification : réfutateur maintient, reproducteur maintient, juge de spécification maintient. Constats fusionnés : A-031, A-045, A-218, A-280.


### Mineurs (53)

**A-110 · Sans jq, stop_hook_active est lu comme false : la garde anti-boucle du hook Stop disparaît**

`.claude/hooks/go-test.sh:9` — robustesse — effort S

`\|\| echo false` masque l'absence de jq. Quand la suite échoue, le hook rend 2, Claude reprend, un nouveau Stop survient avec stop_hook_active=true ; sans jq la valeur est ignorée et la suite (7 s à chaud, 22 s à froid) est relancée à chaque tour, avec exit 2 chaque fois tant que le test échoue. Vérifié : avec jq neutralisé et `{"stop_hook_active":true}`, le hook lance bien la suite (exit 0 ici parce qu'elle passe).

*Correctif.* `command -v jq >/dev/null 2>&1 \|\| exit 0` en tête (comportement dégradé explicite), ou détecter sans jq : `printf '%s' "$INPUT" \| grep -qE '"stop_hook_active"[[:space:]]*:[[:space:]]*true' && exit 0`.

**A-105 · Motifs sensibles à la casse : Results/x.json et docs/Dashboard.md passent sous Windows (NTFS insensible)**

`.claude/hooks/guard-paths.sh:23` — portabilite — effort S — exigences BR-003-3, BR-001-2, BR-005-3

Le `case` compare `results/*`, `matrices/*`, `docs/dashboard.md`, `internal/harness/*` en respectant la casse. Sous Windows (poste de référence), `Results/x.json` ou `RESULTS\x.json` désigne le même répertoire. Vérifié : exit 0 pour `<root>/Results/x.json`, `<root>/docs/Dashboard.md`, `C:\…\RESULTS\x.json`. Peu probable qu'un agent change la casse, mais la garde est contournable et le selftest ne couvre pas ce cas.

*Correctif.* Après le calcul de REL : `case "$OSTYPE" in msys*\|cygwin*\|win*) REL="${REL,,}";; esac` (motifs déjà en minuscules) ; ou `shopt -s nocasematch` sur ces plateformes (s'applique à `case`). Ajouter les cas au selftest en forme windows.

**A-109 · Un chemin relatif docs/use-cases/UC-###-*.md n'est pas linté (motif exige un `/` devant docs)**

`.claude/hooks/spec-lint.sh:10` — robustesse — effort S

Le motif `*/docs/use-cases/...` demande un caractère `/` avant `docs` ; un file_path relatif `docs/use-cases/UC-001-generer-matrice.md` ne correspond pas et le hook sort 0 sans rien vérifier. Vérifié par exécution (exit 0, aucune sortie) alors que guard-paths accepte lui les chemins relatifs. Claude Code envoie normalement des chemins absolus, donc impact contenu.

*Correctif.* `case "$FILE" in docs/use-cases/UC-[0-9][0-9][0-9]-*.md\|*/docs/use-cases/UC-[0-9][0-9][0-9]-*.md) ;; *) exit 0 ;; esac` puis `[ -f "$FILE" ] \|\| FILE="$ROOT/$FILE"`.

**A-108 · Détection des mots vagues : faux positif sur les citations anglaises inline du livre, et `\b` non portable (GNU seulement)**

`.claude/hooks/spec-lint.sh:34` — robustesse — effort S

Seules les lignes de blockquote (`^N:>`) et « Notes de revue » sont exclues. Une citation inline « a small struct should be passed by value » dans un UC est signalée (vérifié sur une copie hors dépôt : exit 2 avec la ligne citée). Aucun UC actuel n'a d'occurrence (selftest passe), mais la première citation ajoutée déclenchera l'avertissement. De plus `\b` avec `grep -E` est une extension GNU : sous BSD grep (macOS)…

*Correctif.* Retirer le contenu entre guillemets avant le grep : `sed -E 's/«[^»]*»//g; s/"[^"]*"//g' "$FILE" \| grep -niE ...` (le numéro de ligne reste correct) ; remplacer `\b…\b` par `grep -wniE '(normalement\|…\|should)'` (`-w` est POSIX). Ajouter au selftest un UC fautif avec citation qui doit passer.

**A-112 · La garde ne couvre que Edit\|Write : Bash et NotebookEdit écrivent librement dans results/, matrices/, docs/dashboard.md**

`.claude/settings.json:5` — conformite-claude-md — effort M — exigences BR-003-3, BR-001-2, BR-005-3

CLAUDE.md affirme : « results/, matrices/ et docs/dashboard.md ne sont jamais écrits par un agent (hook guard-paths) ». Le matcher PreToolUse est `Edit\|Write` ; l'outil Bash n'est pas gardé (`echo > docs/dashboard.md`, `cp`, `tee`), et NotebookEdit passe `notebook_path` (vérifié : exit 0 pour un notebook sous results/). Les skills restreignent Bash (`Bash(go run:*)`…) mais une session hors skill a Bash complet. La…

*Correctif.* Ajouter une entrée PreToolUse `"matcher": "Bash"` avec un hook qui lit `.tool_input.command` et refuse (exit 2) toute commande citant `results/`, `matrices/` ou `docs/dashboard.md` hors du préfixe `go run ./cmd/escapebench` / `./bin/escapebench` ; lire aussi `.tool_input.notebook_path // .tool_input.file_path` dans guard-paths. À défaut, reformuler la phrase de CLAUDE.md : « … jamais écrits via Edit/Write par un agent (hook guard-paths) ; Bash reste sous la responsabilité…

**A-113 · Le skill cite internal/adapters/fake, qui n'existe pas**

`.claude/skills/go-test/SKILL.md:28` — documentation — effort S

Les adapters factices vivent dans internal/service/fakes_test.go (memoryStore, etc.). `ls internal/adapters/fake` : No such file. Un agent suivant le skill peut créer ce paquet (nouveau layout non prévu par CLAUDE.md) au lieu de réutiliser fakes_test.go.

*Correctif.* Remplacer par « Adapters factices en mémoire dans `internal/service/fakes_test.go` (memoryStore…) ; les étendre plutôt qu'en créer d'autres ».

**A-114 · spec-coverage : commande grep en shell non autorisée par allowed-tools, -shuffle absent, conditionnel périmé et description contradictoire**

`.claude/skills/spec-coverage/SKILL.md:15` — documentation — effort S — exigences FR-007, BR-005-3

(1) L'étape 2 prescrit `grep -rn "func TestUC…" internal cmd` en Bash, mais allowed-tools (ligne 8) ne liste que `Bash(go test:*)`, `Bash(go vet:*)`, `Bash(go run:*)`, `Bash(git log:*)` : la commande déclenche une demande de permission ou un refus. (2) L'étape 3 omet `-shuffle=on`, contrairement à la commande de référence de CLAUDE.md. (3) L'étape 5 fonde la colonne Integration sur `//go:build integration_test`,…

*Correctif.* Étape 2 : « avec l'outil Grep, motif `func TestUC002_` sous internal/ et cmd/ » ; étape 3 : ajouter `-shuffle=on` ; étape 6 : retirer le conditionnel ; description : « … puis fait régénérer docs/dashboard.md par le binaire ».

**A-107 · CI sans timeout-minutes, sans permissions minimales, double exécution push+pull_request, cache setup-go sans go.sum**

`.github/workflows/ci.yml:4` — build-ci — effort S

Aucun `timeout-minutes` (défaut GitHub : 360 min : un `go test -race` qui bloque consomme six heures de quota) ; pas de bloc `permissions:` (le GITHUB_TOKEN garde les droits par défaut du dépôt) ; `push:` sans filtre de branche plus `pull_request:` -> chaque PR interne tourne deux fois. actions/setup-go@v5 active le cache par défaut et cherche go.sum, absent ici (aucune dépendance) : d'après la documentation de…

*Correctif.* `on: { push: { branches: [main] }, pull_request: {} }` ; `permissions: { contents: read }` ; `timeout-minutes: 15` sur le job ; `with: { go-version-file: go.mod, cache: false }` (ou `cache-dependency-path: go.mod` si on veut conserver le cache de build).

**A-106 · CI Linux seulement : les tests _windows_test.go ne tournent jamais en CI et la toolchain de référence (go1.27) n'est jamais exercée**

`.github/workflows/ci.yml:10` — build-ci — effort S — exigences C-001, NFR-004, UC-003

Le module contient topology_windows_test.go et quietude_parse_windows_test.go (`//go:build windows`) ; ubuntu-latest les exclut. Le poste de référence est Windows et les campagnes publiées (results/campaigns/*) ont été mesurées avec go1.27.0, alors que `go-version-file: go.mod` (go 1.25, sans ligne toolchain) fait compiler la CI avec la dernière 1.25.x. Une régression Windows ou un changement de format de sortie…

*Correctif.* `strategy: { matrix: { os: [ubuntu-latest, windows-latest], go: ['1.25', '1.27'] } }`, `runs-on: ${{ matrix.os }}`, `go-version: ${{ matrix.go }}` ; l'étape hooks avec `shell: bash` sous Windows (jq est préinstallé sur les runners GitHub Windows et Linux). Alternative minimale : ajouter `toolchain go1.27.0` dans go.mod pour aligner la CI sur la toolchain des campagnes.

**A-055 · Les fichiers temporaires .tmp-* d'une écriture interrompue ne sont ni nettoyés ni ignorés par git**

`.gitignore:6` — robustesse — effort S — exigences NFR-004

writeFileAtomic laisse .tmp-XXXX si le processus meurt entre CreateTemp et Rename. Rien ne les balaie (ni au démarrage, ni dans List/LoadMeasurements, qui les ignorent sans les signaler) et .gitignore ne les exclut pas ; results/ étant versionné, un fichier partiel finit dans un commit. Aucun résidu présent aujourd'hui (find results matrices -name '.tmp-*' : vide).

*Correctif.* Ajouter `.tmp-*` à .gitignore ; optionnellement, dans AcquireLock ou au démarrage de CreateCampaign/resume, supprimer les .tmp-* de la campagne courante.

**A-092 · L'usage affirme que les clés absentes de --params prennent la valeur de la matrice de référence ; c'est faux pour sizes et probes**

`cmd/escapebench/main.go:53` — documentation — effort S — exigences UC-001, BR-001-4

ParseParameters exige `sizes` et laisse `Probes` nil quand la clé est absente (TestParseParametersDefauts l'affirme : « aucune sonde par défaut »), alors que ReferenceParameters porte 10 sondes. La phrase de l'usage (et le commentaire de ParseParameters, params.go:22) n'est vraie que pour pointer, profiles et modes. Un utilisateur qui omet `probes` en croyant reprendre les sondes de référence obtient une matrice…

*Correctif.* Préciser dans `usage` et dans le commentaire de ParseParameters : « pointer, profiles et modes prennent la valeur de référence ; sizes est obligatoire ; probes, layouts, repeats, payloads, replicates sont vides ou minimaux par défaut ». Aucun code touché.

**A-085 · Erreur de drapeau imprimée deux fois et `-h` traité comme une erreur d'usage (code 2)**

`cmd/escapebench/main.go:163` — robustesse — effort S

parse laisse flag.ContinueOnError écrire sur os.Stderr son propre message et l'usage de la sous-commande, puis main réimprime l'erreur enveloppée et l'usage global. Vérifié : `escapebench escape --inconnu` produit sur stderr « flag provided but not defined: -inconnu / Usage of escape: … » puis « usage : flag provided but not defined: -inconnu » suivi des 17 lignes d'usage. De plus flag.ErrHelp (`-h`, `--help`) est…

*Correctif.* `if errors.Is(err, flag.ErrHelp) { fs.PrintDefaults(); return nil }` (ou une sentinelle dédiée sortant 0) et `fs.SetOutput(io.Discard)` pour laisser main seul imprimer ; ajouter un cas `-h` dans TestRunOptionsManquantes.

**A-084 · runMatrix et runCampaign impriment un rapport vide sur stdout quand le service échoue avant de produire quoi que ce soit**

`cmd/escapebench/main.go:196` — robustesse — effort S — exigences UC-001, UC-003

Le rapport est imprimé inconditionnellement pour que A3 (UC-001, sujets non compilables) et A2 (UC-003, abandon) restent visibles ; mais Generate et Run rendent un MatrixReport{} / CampaignReport{} vide sur tout autre chemin d'erreur (précondition, validation, verrou). Vérifié : `escapebench matrix --params sizes=7` imprime sur stdout `Matrice : \nTypeSpec : 0 · Cell : 0 · Probe : 0\nEmpreinte du harnais : \n` ;…

*Correctif.* N'imprimer que si le rapport porte un identifiant : `if err == nil \|\| report.MatrixID != "" { fmt.Print(cli.RenderMatrix(report)) }` (même garde sur report.CampaignID dans runCampaign, ligne 249) ; ou faire rendre par le service un rapport nul explicite et le tester dans main_test.

**A-133 · Aucun test hors internal/service n'est nommé TestUC###_… : 0 sur cmd, adapters, models et harness (205 fonctions de test)**

`internal/adapters/cli/cli_test.go:16` — conformite-claude-md — effort L — exigences FR-007, UC-005

CLAUDE.md : « sous-tests nommés d'après le UC et le flux (TestUC003_MainFlow, TestUC003_A1_HarnessModified, …) ». `go test -list '^TestUC'` : service 90, cmd 0, adapters 0, models 0, harness 0. Conséquence concrète : la colonne Unit du tableau de bord est calculée par RunUseCaseTests avec le motif `^TestUC###_` sur ./... (gotool.go:285) ; les tests d'adapters (immutabilité BR-003-3 dans store_test.go, classification…

*Correctif.* Renommer par UC porteur (ex. TestUC003_BR3_MeasurementsImmuables, TestUC002_BR1_ClassifyParCause, TestUC001_C005_DigestStable) ou documenter dans CLAUDE.md que la règle ne vise que internal/service et cmd ; l'index de specs (ucTestRe) profitera aussi des renommages.

**A-086 · Une clé répétée dans --params écrase silencieusement la précédente**

`internal/adapters/cli/params.go:32` — robustesse — effort S — exigences UC-001, BR-001-1

La boucle ne mémorise pas les clés déjà vues : `sizes=8;sizes=7` retient 7 et perd 8 sans avertissement (vérifié : l'erreur rendue est « taille 7 hors de [8, 4096] », rien sur la première clause). Avec deux valeurs valides (`sizes=8;sizes=16`), la matrice produite ne contient que 16 et son MatrixID diffère de celui attendu par l'utilisateur. Aucun test ne couvre le cas.

*Correctif.* Tenir `seen := map[string]bool{}` et rendre `fmt.Errorf("%w : clé %q répétée", ErrUsage, key)` ; ajouter le cas à TestParseParametersErreurs.

**A-087 · ParseHypotheses ne dédoublonne pas : `H-001,H-001` produit une empreinte de critères différente de `H-001` et deux verdicts pour la même hypothèse**

`internal/adapters/cli/params.go:165` — correctness — effort S — exigences BR-003-5, UC-003, UC-005

ParseHypotheses rend splitValues tel quel ; en aval, CampaignService.freezeCriteria (internal/service/campaign.go:285-293) trie sans dédoublonner et specs.Digest hache chaque identifiant de la liste, doublons compris. La même sélection de critères donne donc deux HypothesesDigest selon la présence d'un doublon, et Campaign.HypothesisIDs conserve le doublon ; VerdictService.Produce (verdict.go:102) itère sur cette…

*Correctif.* Dédoublonner dans ParseHypotheses (map + tri) et rendre ErrUsage si un identifiant ne respecte pas `^H-\d{3}$` ; par défense, dédoublonner aussi dans freezeCriteria (hors périmètre, service). Test : ParseHypotheses("H-001,H-001") == []string{"H-001"}.

**A-091 · Write tronque docs/dashboard.md avant d'écrire ; une interruption laisse un fichier versionné partiel**

`internal/adapters/dashboard/dashboard.go:36` — robustesse — effort S — exigences BR-005-3, FR-007

os.WriteFile ouvre avec O_TRUNC puis écrit ; le contexte reçu est ignoré. Un Ctrl+C (signal.NotifyContext de main.go) ou un plantage pendant l'écriture laisse docs/dashboard.md vide ou tronqué, et BR-005-3 interdit de le corriger à la main. Le fichier est régénérable, d'où la sévérité contenue.

*Correctif.* Écrire dans `w.path + ".tmp"` puis `os.Rename` (atomique sur un même volume, Windows inclus depuis Go 1.5) ; conserver le test TestWrite.

**A-130 · Quatre points d'entrée d'I/O ne prennent pas de context.Context : QuietudeSampler, cli.FindRoot, system.DetectCPUModel, specs.Index.scan**

`internal/adapters/gotool/gotool.go:89` — conformite-claude-md — effort S — exigences C-004, C-010

CLAUDE.md : « Toute I/O prend un context.Context en premier paramètre ». Contre-exemples vérifiés : (1) `type QuietudeSampler func() (busy time.Duration, ok bool)` (gotool.go:89) exécuté deux fois par sujet par sampleQuietude (gotool.go:235-240) et implémenté par system.QuietudeProbe.Sample() (quietude.go:39) qui lit GetSystemTimes//proc/stat ; (2) `cli.FindRoot(start string)` (params.go:170) boucle sur os.Stat ;…

*Correctif.* Ajouter `ctx context.Context` en premier paramètre à QuietudeSampler, QuietudeProbe.Sample, FindRoot, DetectCPUModel (et au type CPUReader) ; dans Index, faire `scan(ctx)` et vérifier `ctx.Err()` dans le callback WalkDir. Mettre à jour main.go:143-146 et les tests correspondants.

**A-063 · truncate coupe au milieu d'une séquence UTF-8 et laisse un octet invalide dans FailureReason**

`internal/adapters/gotool/gotool.go:277` — correctness — effort S — exigences UC-003, NFR-001

s[:max] tranche en octets. Les raisons d'échec contiennent du français (« répétitions mesurées », « aucune ligne de benchmark ») et la sortie de go test avec chemins ; si l'octet 2000 tombe dans un « é », la chaîne est invalide. encoding/json (store.go:533, json.MarshalIndent) remplace l'octet par U+FFFD : le fichier de résultats porte une raison altérée.

*Correctif.* Reculer jusqu'à un début de rune : `for max > 0 && !utf8.RuneStart(s[max]) { max-- }` avant la tranche ; ajouter un cas non ASCII à TestTruncate.

**A-068 · Aucun test n'affirme TreeCPUMeasured : le chemin Job Object de treecpu_windows.go et le chemin ProcessState ne peuvent pas échouer**

`internal/adapters/gotool/gotool_test.go:274` — tests — effort S — exigences C-010, H-013

TestExecRunner exécute `go env` réellement mais ne regarde ni result.TreeCPU ni result.TreeCPUMeasured. Les tests TestC010_* injectent TreeCPUMeasured=true via un runner factice. Une régression dans startTreeCPU (droits d'accès OpenProcess, taille de jobAccounting, ordre des champs, échec silencieux de AssignProcessToJobObject) ferait passer toute attestation à non mesurée sans qu'aucun test ne rougisse ; H-013…

*Correctif.* Dans TestExecRunner, après le `go env` réussi : `if runtime.GOOS == "windows" \|\| runtime.GOOS == "linux" { if !result.TreeCPUMeasured { t.Fatal("le temps de l'arbre doit être relevé (C-010)") } }` ; un `go env` peut consommer 0 tick, ne pas exiger TreeCPU > 0.

**A-128 · time.Sleep dans un test, aucun usage de testing/synctest, parce que Toolchain.Run lit l'horloge réelle (time.Now/time.Since) au lieu de l'horloge injectée**

`internal/adapters/gotool/quietude_test.go:96` — conformite-claude-md — effort S — exigences C-010, H-013

CLAUDE.md : « testing/synctest pour tout comportement temporel ou concurrent, jamais de time.Sleep ». grep synctest sur cmd/ et internal/ : aucun fichier. Le Sleep est là parce que gotool.go:204-206 mesure `elapsed` avec `time.Now()`/`time.Since()` directement, alors que ports.Clock existe et que ExecRunner reçoit déjà ses dépendances par injection. Le test dépend de la résolution de l'horloge du système…

*Correctif.* Ajouter à Toolchain un champ `now func() time.Time` (défaut time.Now, option `WithClock(ports.Clock)`), l'utiliser aux lignes 204-206, et dans le test fournir une horloge qui avance d'une valeur fixe entre deux appels ; supprimer le Sleep. Alternative Go 1.25 : envelopper le corps du test dans `synctest.Test(t, func(t *testing.T){…})` sans t.Parallel.

**A-064 · C-010 exige aussi le nombre de défauts de page (/proc/vmstat, GetSystemTimes) : rien ne le relève, TotalPageFaultCount est lu puis jeté**

`internal/adapters/gotool/treecpu_windows.go:33` — conformite-spec — effort M — exigences C-010, H-013, UC-003

Le texte de C-010 (docs/requirements.md:40) : « UC-003 doit donc acquérir, autour de chaque Measurement, le relevé du temps processeur système consommé hors du processus mesuré et du nombre de défauts de page, par GetSystemTimes sur Windows et par /proc/stat et /proc/vmstat sur Linux ». Le code n'échantillonne que les temps processeur : quietude_linux.go ne lit que /proc/stat, quietude_windows.go n'appelle que…

*Correctif.* Soit implémenter : Linux, lire `pgfault` (et `pgmajfault`) dans /proc/vmstat ; Windows, pas d'équivalent machine dans GetSystemTimes — utiliser GetPerformanceInfo ou consigner la part propre via TotalPageFaultCount ; ajouter un champ facultatif à Measurement. Soit amender le texte de C-010 pour retirer explicitement les défauts de page (C-010 n'est pas un critère H gelé). Dans les deux cas, aucun gabarit touché.

**A-090 · Le statut d'un UC n'est pas validé contre la liste admise ; une faute de frappe dégèle silencieusement les hypothèses portées**

`internal/adapters/specs/specs.go:134` — robustesse — effort S — exigences FR-007, UC-005

useCaseStatRe capture `\w+` sans vérifier l'appartenance à Draft/Reviewed/Approved/Implemented/Verified/Deployed. frozenHypotheses (internal/service/verdict.go:301-304) ne reconnaît que quatre valeurs exactes : un statut `Aproved` ou `approved` rend le UC non approuvé, donc les hypothèses qu'il porte apparaissent « Critère gelé : Non » dans le tableau de bord, sans erreur. LoadUseCases ne vérifie pas non plus que…

*Correctif.* Valider le statut dans LoadUseCases contre une liste fermée (rendre une erreur nommant le fichier), vérifier que `UC-###` du nom de fichier égale l'ID déclaré et refuser les doublons ; tests dans TestLoadUseCasesErreurs.

**A-053 · Les gardes d'immutabilité traitent toute erreur de Stat autre que « absent » comme « absent »**

`internal/adapters/store/store.go:100` — robustesse — effort S — exigences BR-001-2, NFR-004

Sept gardes (lignes 100, 124, 164, 242, 277, 314, 397) testent `err == nil` seulement ; une erreur de permission, un chemin invalide sous Windows ou une erreur d'E/S sur le Stat laisse passer l'écriture, contrairement à Exists (ligne 53-62) qui remonte l'erreur. L'écriture qui suit échouera le plus souvent, mais pas toujours (Stat refusé sur le fichier, écriture permise dans le répertoire).

*Correctif.* Factoriser une fonction exists(path) (bool, error) reprenant la logique de Exists, et l'employer dans les sept gardes ; remonter toute erreur autre que os.ErrNotExist.

**A-046 · SetCampaignStatus n'applique pas la seule transition admise par BR-003-3 (RUNNING → COMPLETED\|ABORTED)**

`internal/adapters/store/store.go:268` — conformite-spec — effort S — exigences BR-003-3, NFR-004

BR-003-3 (UC-003:81) admet uniquement le passage de status de RUNNING à COMPLETED ou ABORTED. Le store réécrit campaign.json quel que soit le statut courant : COMPLETED → RUNNING, ABORTED → COMPLETED, ou un second passage qui change finishedAt et abortReason d'une campagne close. La garde n'existe que dans les appelants (resume exige RUNNING). Note : la règle parle du seul champ status alors que finishedAt et…

*Correctif.* Après LoadCampaign : if campaign.Status != models.CampaignRunning { return fmt.Errorf("%w : la campagne %s est au statut %s (BR-003-3)", ErrImmutable, campaignID, campaign.Status) } ; refuser aussi status == CampaignRunning. Test de mutation dans TestCampaignCycleDeVie.

**A-052 · Horodatage à la seconde : deux écritures de comparaison, d'échappement ou de verdict dans la même seconde échouent avec un ErrImmutable trompeur**

`internal/adapters/store/store.go:313` — robustesse — effort S — exigences BR-004-3, BR-002-3, NFR-004

Stamp a une résolution d'une seconde. WriteComparisonSet, WriteEscapeReport et WriteVerdictReport nomment le fichier par Stamp seul ; une seconde exécution dans la même seconde (script, tests d'intégration) tombe sur la garde d'immutabilité et rend « existe déjà (BR-004-3) », alors qu'il ne s'agit pas d'une réécriture mais d'une collision de nom.

*Correctif.* Soit documenter la résolution et distinguer le message (« un fichier porte déjà cet horodatage, réessayer »), soit ajouter un suffixe de désambiguïsation (-2, -3) tout en gardant le tri lexicographique (largeur fixe).

**A-054 · WriteComparisonSet et WriteMeasurement n'exigent ni identifiant non vide ni campagne existante**

`internal/adapters/store/store.go:317` — robustesse — effort S — exigences UC-003, UC-004, NFR-004

WriteComparisonSet fait MkdirAll sur le répertoire de campagne : un CampaignID inexistant crée results/campaigns/<id>/ sans campaign.json, que NextCampaignID comptera ensuite comme pris et que LoadCampaign signalera introuvable. Un CampaignID vide fait s'effondrer le chemin : results/campaigns/comparison-<stamp>.json et results/campaigns/measurements/<sujet>.json (filepath.Join nettoie le segment vide).…

*Correctif.* Refuser CampaignID ou SubjectID vide (et pour WriteComparisonSet, exiger que campaign.json existe, sans MkdirAll) ; enveloppe d'erreur nommant BR-003-3 / BR-004-3.

**A-124 · LatestComparisonSet convertit toute erreur de ReadDir en ErrNotFound et jette la cause**

`internal/adapters/store/store.go:330` — robustesse — effort S — exigences UC-004, UC-005

`entries, err := os.ReadDir(...); if err != nil { return …, fmt.Errorf("%w : campagne %s", ErrNotFound, campaignID) }` : `err` (permission refusée, chemin non-répertoire, E/S) n'est pas enveloppé. Combiné au constat précédent, une erreur de permission sur results/campaigns/C-X devient silencieusement « pas de comparaison ». Règle CLAUDE.md : %w.

*Correctif.* `if errors.Is(err, os.ErrNotExist) { return …, fmt.Errorf("%w : campagne %s", ErrNotFound, campaignID) }` puis `if err != nil { return …, fmt.Errorf("lecture de la campagne %s : %w", campaignID, err) }` (même schéma que ListEscapeReports lignes 176-182).

**A-125 · AcquireLock perd os.ErrExist et n'expose aucune sentinelle pour « campagne déjà en cours »**

`internal/adapters/store/store.go:358` — conformite-claude-md — effort S — exigences UC-003, BR-003-3

L'erreur de collision de verrou est construite sans %w : ni os.ErrExist ni une sentinelle du paquet ne sont conservés. CampaignService.measure la propage telle quelle ; main ne peut pas la reconnaître par errors.Is (seul cli.ErrUsage l'est). Règle CLAUDE.md : erreurs enveloppées avec %w, inspectées avec errors.Is aux bords. Note : `ErrImmutable` existe déjà pour le même genre de refus.

*Correctif.* `return fmt.Errorf("%w : une campagne est déjà en cours (results/%s)", ErrLocked, LockName)` avec `var ErrLocked = errors.New("campagne en cours")` (ou `%w` sur os.ErrExist) ; adapter TestLock pour vérifier errors.Is.

**A-126 · Si l'écriture du contenu du verrou échoue, le fichier .campaign-lock vide subsiste et bloque UC-001 et UC-003 ; l'erreur est rendue brute**

`internal/adapters/store/store.go:364` — robustesse — effort S — exigences UC-001, UC-003, BR-003-3

Après `os.OpenFile(... O_CREATE\|O_EXCL ...)`, `file.WriteString` peut échouer (disque plein) : le fichier existe déjà, n'est pas retiré, et `return err` n'enveloppe rien. Côté service (campaign.go:192-195), `AcquireLock` en erreur fait sortir de measure avant que `defer ReleaseLock` ne soit posé : le verrou orphelin reste, et LockHeld/AcquireLock refusent ensuite toute matrice et toute campagne jusqu'à suppression…

*Correctif.* `if _, err := file.WriteString(...); err != nil { file.Close(); os.Remove(path); return fmt.Errorf("écriture du verrou de campagne : %w", err) }` et envelopper aussi l'erreur de Close.

**A-057 · LatestVerdictReport est du code mort et son commentaire de doc est accroché à VerdictReports ; Store.Root() n'a pas d'appelant hors tests**

`internal/adapters/store/store.go:406` — documentation — effort S — exigences UC-005

Aucun appelant de LatestVerdictReport hors du store et de ports.VerdictStore (grep sur cmd/ et internal/ : rien). La ligne 406 est le commentaire de doc de LatestVerdictReport mais elle précède VerdictReports ; LatestVerdictReport (ligne 435) n'a donc pas de doc. Root() (ligne 41) n'est utilisé que par store_test.go:407.

*Correctif.* Supprimer LatestVerdictReport de ports.VerdictStore et du store (ou l'implémenter via VerdictReports en trois lignes) ; déplacer le commentaire ; retirer Root() ou le garder documenté comme aide de test.

**A-048 · Tri des rapports de verdicts par horodatage seul avec sort.Slice : ordre non déterministe entre rapports de même seconde**

`internal/adapters/store/store.go:453` — correctness — effort S — exigences UC-005, BR-005-3

VerdictReports (ligne 423) et LatestVerdictReport (ligne 453) trient sur stampOf(name), qui ignore l'identifiant de campagne. Deux rapports produits la même seconde pour deux campagnes ont des clés égales ; sort.Slice n'est pas stable, donc l'ordre relatif — et le « dernier verdict l'emporte » du tableau de bord (verdict.go:261-270) — peut varier d'une exécution à l'autre au-delà de 12 éléments (pdqsort).…

*Correctif.* Trier sur (stampOf(name), name) ou utiliser sort.SliceStable (ReadDir rend déjà les noms triés). Extraire la boucle de collecte commune aux deux fonctions.

**A-050 · writeFileAtomic ne fait pas de Sync avant Rename : un fichier de résultats peut être vide après coupure de courant**

`internal/adapters/store/store.go:554` — robustesse — effort S — exigences NFR-004, UC-003

Le commentaire promet qu'aucun fichier partiel ne subsiste. C'est vrai pour un plantage du processus, pas pour un arrêt brutal de la machine : sans tmp.Sync(), le renommage peut être journalisé avant les données (ext4 delayed allocation, NTFS lazy write) et matrix.json ou une mesure se retrouve tronqué ou vide sous son nom définitif.

*Correctif.* Appeler tmp.Sync() avant tmp.Close() ; optionnellement ouvrir et Sync le répertoire parent sur Linux. Coût négligeable au volume actuel (une écriture par sujet).

**A-065 · Sous Windows, DetectCPUModel consigne PROCESSOR_IDENTIFIER (famille/modèle/stepping), pas le modèle de processeur**

`internal/adapters/system/system.go:89` — conformite-spec — effort S — exigences NFR-001, C-008, UC-003

NFR-001 exige « le modèle de CPU » dans la provenance. PROCESSOR_IDENTIFIER vaut « Intel64 Family 6 Model 198 Stepping 2, GenuineIntel » : c'est la signature CPUID, partagée par tous les SKU d'une même génération (un Core Ultra 5 235 et un Core Ultra 9 275HX portent la même chaîne). Les 16 fichiers de results/ portent cette valeur (`grep -rho cpuModel results` : « Intel64 Family 6 Model 198 Stepping 2, GenuineIntel…

*Correctif.* Sous Windows, lire `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0\ProcessorNameString` par syscall.RegOpenKeyEx/RegQueryValueEx (paquet syscall, C-002 respecté) dans un fichier cpumodel_windows.go, avec PROCESSOR_IDENTIFIER en repli ; conditionner le repli env à runtime.GOOS == "windows". Les provenances antérieures restent valides (SameToolchain ne compare pas cpuModel).

**A-066 · recordSize = 32 et les décalages 8/16/20/24 supposent un ProcessorMask de 8 octets, mais le tag de build est `windows` sans restriction d'architecture**

`internal/adapters/system/topology_windows.go:152` — portabilite — effort S — exigences C-006, C-008, NFR-001

SYSTEM_LOGICAL_PROCESSOR_INFORMATION commence par un ULONG_PTR : 8 octets sur amd64/arm64, 4 sur 386/arm. Sur une plateforme 32 bits l'enregistrement fait 24 octets et la relation est à l'offset 4, le niveau à 8, la taille à 12, le type à 16. Le code compile pourtant pour windows/386 (vérifié : `GOOS=windows GOARCH=386 go vet ./internal/adapters/system` sans erreur) et lit des champs décalés.

*Correctif.* Dériver les décalages de la taille du pointeur : `const ptrSize = unsafe.Sizeof(uintptr(0)); recordSize = 2*ptrSize + 16` et lire à `2*ptrSize` (relation), `2*ptrSize+8` (union). Ou restreindre le tag à `windows && (amd64 \|\| arm64)` et étendre topology_other.go à `!linux && !(windows && (amd64 \|\| arm64))`.

**A-074 · Les tests du harnais vérifient la syntaxe du code généré, jamais ses types ni sa compilation**

`internal/harness/harness_test.go:218` — tests — effort M — exigences UC-001, C-007, C-008, FR-007

assertParses appelle parser.ParseFile : un gabarit qui produirait un import « unsafe » inutilisé, un champ P *byte lu comme *uint64, un appel à une fonction non émise (consumeValue absent) ou une constante Sizeof négative passerait `go test ./...`. La seule vérification réelle est une campagne. La cible Makefile integration_test (`-tags=integration_test`) n'a aucun test derrière elle (confirmé par l'orchestrateur :…

*Correctif.* Deux options, la première suffit : (1) dans harness_test.go, type-checker chaque subject.go rendu avec go/types (`types.Config{Importer: importer.Default()}` ; subject.go n'importe que « unsafe », résolu sans données d'export) pour toutes les combinaisons de TestRenderCellPourTousLesProfilsEtModes, layouts NAMED_FIELDS et NAMED_FIELDS_SHAM inclus ; (2) un fichier `//go:build integration_test` dans internal/harness qui rend une matrice réduite dans t.TempDir() avec…

**A-073 · La garde unsafe.Sizeof ne détecte qu'un type trop petit ; un type trop grand compile**

`internal/harness/templates/cell.go.tmpl:33` — correctness — effort S — exigences C-008, C-005, BR-003-1, UC-001

**Ce correctif change `Campaign.harnessDigest`** : il touche un gabarit embarqué, donc invalide matrices et campagnes antérieures (C-005, BR-003-1).

Le commentaire promet qu'« une taille différente rend la constante négative ». Seul Sizeof < SizeBytes rend la soustraction négative ; Sizeof > SizeBytes donne une constante positive et le paquet compile. Aujourd'hui layoutOf produit exactement words*8 octets pour toutes les dispositions (vérifié : 304 cellules compilent), la garde est donc inerte, mais elle ne protège pas C-008 (a) contre une régression du…

*Correctif.* Ajouter la seconde direction dans cell.go.tmpl (après la l. 33) : `const _ = uint({{.SizeBytes}} - unsafe.Sizeof({{.TypeName}}{}))`, et dans probe.go.tmpl (après la l. 23) : `const _ = uint(nodeBytes - unsafe.Sizeof(node{}))` ; corriger le commentaire. Ajouter dans harness_test.go une assertion sur la présence des deux constantes. ATTENTION : modifie deux gabarits embarqués, donc Campaign.harnessDigest change (C-005, BR-003-1) et toutes les matrices et campagnes antérieures…

**A-002 · Replicates = 5 sans NAMED_FIELDS × LOCAL est accepté et produit une matrice identique sous un autre identifiant**

`internal/models/matrix.go:224` — correctness — effort S — exigences UC-001, H-012, C-009, BR-001-1

Validate n'exige pas que la combinaison où le réplicat se décline (disposition NAMED_FIELDS, profil LOCAL) soit demandée. Expand ne produit alors les cellules qu'à la passe 1, mais Canonical écrit `replicates=5`, donc l'identifiant change. L'utilisateur obtient une matrice au contenu strictement identique à la matrice non répliquée, sous un autre M-…, et croit avoir des réplicats pour H-012.

*Correctif.* Dans MatrixParameters.Validate, après normalisation des dispositions : si Replicates == ReplicateCount et (Layouts ne contient pas LayoutNamedFields ou Profiles ne contient pas ProfileLocal), ajouter le problème « réplicats demandés sans NAMED_FIELDS × LOCAL ». Ne pas toucher Canonical (BR-001-1).

**A-010 · Commentaire périmé dans MatrixParameters.Validate : annonce un refus du témoin nul hors LOCAL qui n'existe plus**

`internal/models/matrix.go:228` — documentation — effort S — exigences H-007, C-008

Le commentaire « Le témoin nul ne se mesure qu'en profil LOCAL : demandé avec d'autres profils, il produirait des cellules que Cell.Validate refuse » précède la boucle des sondes et ne correspond à aucun code : la revue du 2026-09-10 a remplacé le refus par un saut dans Expand (TestTemoinNulSeulementEnLocal, revue_test.go:57-60 le documentent). Un lecteur cherche une règle absente.

*Correctif.* Supprimer les deux lignes, ou les remplacer par « Le témoin nul hors LOCAL et au-delà de SmallStructBytes est sauté dans Expand, jamais refusé ici (revue 2026-09-10) ».

**A-003 · Expand rend zéro cellule sans erreur quand tous les sauts s'appliquent ; le message final ne nomme pas le paramètre fautif**

`internal/models/matrix.go:407` — robustesse — effort S — exigences UC-001, H-007

Trois sauts (témoin nul > 24 octets, témoin nul hors LOCAL, répétition/charge hors RETURNED_ALLOCATING) peuvent vider la matrice. Expand retourne alors (nil, probes, nil) et l'erreur ne surgit qu'à Matrix.Validate (« Matrix.cells contient au moins une cellule »), après calcul de l'empreinte du harnais et vérification d'existence dans le service. UC-001 étape 2 / A1 demande de nommer chaque paramètre fautif.

*Correctif.* À la fin d'Expand : `if len(cells) == 0 { return nil, nil, invalid("aucune cellule : le témoin nul ne se produit qu'en LOCAL et jusqu'à %d octets, ...", SmallStructBytes) }`, ou un compteur par cause de saut pour nommer laquelle a tout absorbé.

**A-008 · EscapeVerdict.Validate n'impose pas compilerError vide en statut OK**

`internal/models/models.go:406` — conformite-spec — effort S — exigences UC-002, BR-002-1

docs/entity-model.md (EscapeVerdict.compilerError) : « Requis si COMPILE_ERROR, vide sinon ». Validate vérifie la première moitié seulement : un verdict OK portant un message de compilateur passe.

*Correctif.* Après le contrôle du statut OK : `if v.CompilerError != "" { return invalid("EscapeVerdict.compilerError doit être vide si OK (%s)", v.CellID) }` ; ajouter le cas à TestEscapeVerdictValidate.

**A-005 · Campaign.Validate n'exige ni finishedAt (COMPLETED/ABORTED), ni abortReason (ABORTED), ni startedAt, ni hypothesisIds**

`internal/models/models.go:533` — conformite-spec — effort S — exigences UC-003, BR-003-5, NFR-001

docs/entity-model.md (Campaign) : startedAt requis, finishedAt requis si COMPLETED ou ABORTED, abortReason requis si ABORTED (UC-003 A2), hypothesisIds requis. Validate ne couvre que id, matrixId, deux empreintes, count, status et provenance. store.SetCampaignStatus (store.go:269-270) écrit le statut terminal sans passer par Validate, donc rien ne garde ces règles.

*Correctif.* Ajouter les cas : StartedAt.IsZero() ; (Status == COMPLETED \|\| ABORTED) && FinishedAt.IsZero() ; Status == ABORTED && AbortReason == "" ; len(HypothesisIDs) == 0. Étendre TestCampaignValidate.

**A-009 · Measurement.Validate ne borne pas quietudeOccupancy à [0, 1]**

`internal/models/models.go:600` — robustesse — effort S — exigences H-013, C-010, UC-005

Le commentaire et docs/entity-model.md disent « vaut de 0 à 1 ». Le producteur (gotool.occupancyOf) borne bien la fraction, mais le DTO (store/dto.go:355) relit n'importe quel flottant et pose QuietudeMeasured = true sans garde ; Validate laisse passer. criteria_c010.go:102 compare `> QuietudeThreshold` : une valeur négative ou NaN est prise pour une machine quiète.

*Correctif.* Ajouter `if m.QuietudeMeasured && !(m.QuietudeOccupancy >= 0 && m.QuietudeOccupancy <= 1) { return invalid("Measurement.quietudeOccupancy %v hors de [0, 1] (%s)", …) }` (la forme `!(a >= 0 && a <= 1)` rejette aussi NaN).

**A-006 · Hypothesis.Validate n'exige pas statement, pourtant requis et inclus dans l'empreinte gelée**

`internal/models/models.go:724` — conformite-spec — effort S — exigences BR-003-5, UC-003, UC-005

docs/entity-model.md (Hypothesis.statement) : « Requis ; énoncé réfutable, participe à l'empreinte gelée (BR-003-5) ». internal/adapters/specs/specs.go:51 lit l'énoncé de la colonne du tableau et specs.go:105 le hache. Une ligne de requirements.md à énoncé vide serait acceptée et son empreinte calculée sur une chaîne vide, ce qui affaiblit la garantie de gel.

*Correctif.* Ajouter `case h.Statement == "": return invalid("Hypothesis.statement est requis (BR-003-5, %s)", h.ID)` et un sous-test « sans énoncé ». Vérifier que l'adaptateur specs appelle Validate après lecture.

**A-007 · Verdict.Validate n'exige pas campaignId**

`internal/models/models.go:753` — conformite-spec — effort S — exigences UC-005, BR-005-2

docs/entity-model.md (Verdict.campaignId) : « Requis ». Validate vérifie hypothesisId, outcome, rationale et resultFiles seulement ; service/verdict.go:111 s'y fie avant l'écriture.

*Correctif.* Ajouter `case v.CampaignID == "": return invalid("Verdict.campaignId est requis (%s)", v.HypothesisID)` et le sous-test correspondant.

**A-030 · Run ignore silencieusement MatrixID, Count et HypothesisIDs quand Resume est renseigné**

`internal/service/campaign.go:68` — robustesse — effort S — exigences UC-003

`--resume C-x --matrix M-autre --hypotheses H-012` est accepté : la reprise porte sur la matrice et les hypothèses gelées de C-x, sans avertissement. Un chercheur qui pense étendre ou rediriger la campagne se trompe sans le savoir.

*Correctif.* Dans resume, refuser `ErrPrecondition` si opts.MatrixID != "" && opts.MatrixID != campaign.MatrixID, et si len(opts.HypothesisIDs) > 0 ; ou faire ce contrôle dans runCampaign (cli.ErrUsage). Test correspondant.

**A-131 · Soixante-trois fonctions de internal/service n'ont pas d'identifiant UC-### dans leur commentaire d'en-tête**

`internal/service/campaign.go:75` — conformite-claude-md — effort S — exigences UC-001, UC-002, UC-003, UC-004, UC-005

CLAUDE.md : « Référencer l'ID du UC en commentaire d'en-tête de chaque fonction de service ». Relevé mécanique (awk sur le bloc de commentaire précédant chaque `func`) : campaign.go start:75, resume:148, measure:185, freezeCriteria:280 ; matrix.go renderAll:139, compileAll:164, merge:177 ; escape.go classifyCell:114, DifferingVerdicts:150 ; compare.go pairProblem:118, compare:136, bootstrapDeltaCI:168,…

*Correctif.* Compléter les en-têtes : « start couvre le scénario principal et A1 de UC-003 », « evaluateH001 (UC-005, étape 4) — … », etc. Commentaires seulement, aucun gabarit touché.

**A-024 · Si SetCampaignStatus(ABORTED) échoue, l'erreur ErrHarnessChanged (ou « aucun sujet mesuré ») est perdue et le rapport ne porte ni statut ni raison**

`internal/service/campaign.go:234` — robustesse — effort S — exigences UC-003, BR-003-1, C-005

Dans les deux branches d'abandon, l'erreur du store remplace la cause métier : le retour ne satisfait plus errors.Is(err, ErrHarnessChanged), report.Status reste vide et report.AbortReason aussi, alors que la raison a été calculée. L'appelant (main.go:248-250 imprime RenderCampaign(report) puis err) affiche une campagne sans statut et une erreur de disque, sans dire que le harnais a changé.

*Correctif.* Renseigner report.Status/AbortReason avant l'appel au store, et rendre `errors.Join(fmt.Errorf("%w : …", ErrHarnessChanged, …), fmt.Errorf("consignation de l'abandon : %w", err))` si SetCampaignStatus échoue. Même chose pour la branche « aucun sujet mesuré ». Test : store.writeErr posé après la première mesure, asserter errors.Is(err, ErrHarnessChanged) et report.AbortReason non vide.

**A-028 · Aucun test n'assure la libération du verrou sur les chemins d'erreur de measure (runner en panne, écriture impossible, abandon)**

`internal/service/campaign_test.go:393` — tests — effort S — exigences UC-003, BR-003-3

Seul TestUC003_MainFlow vérifie LockHeld faux, et seulement sur le succès. TestUC003_ErreursDesPorts se contente de err != nil. Le `defer ReleaseLock` (campaign.go:195) est la seule garde contre un verrou fantôme après échec ; une mutation qui le remplacerait par un appel explicite en fin de chemin nominal passerait la suite.

*Correctif.* Dans TestUC003_ErreursDesPorts (cas « runner en panne », « écriture impossible ») et dans TestUC003_A2 et TestUC003_AucunSujetMesure, ajouter `if held, _ := f.store.LockHeld(ctx); held { t.Fatal("verrou non libéré") }`.

**A-036 · H-003 et H-004 rendent un verdict sur un corpus partiel sans consigner les paires ou sondes écartées**

`internal/service/criteria.go:149` — conformite-spec — effort S — exigences H-003, H-004, UC-005, BR-005-2

Les deux évaluateurs sautent en silence les paires RETURNED (criteria.go:149) et les couples de sondes (criteria.go:206) incomplets ou FAILED, puis tranchent — y compris REFUTED — sur ce qui reste. Le texte gelé dit « pour toutes les tailles du profil RETURNED » et « pour toutes les Probe ≥ 32 MiB » ; UC-005 A2 prévoit INCONCLUSIVE « avec la liste des sujets manquants » quand des paires ou sondes nécessaires…

*Correctif.* Compter les paires/sondes écartées, les nommer dans la rationale (« 1 paire écartée : Size0032…/RETURNED/VALUE FAILED ») et, au choix documenté dans le commentaire d'en-tête, rendre INCONCLUSIVE quand au moins une paire du profil manque (lecture stricte de « toutes ») ; ajouter un cas de test avec une paire FAILED pour H-003 et une sonde FAILED pour H-004.

**A-026 · A3 : si Remove échoue, la liste des sujets non compilables est perdue et le message ne dit pas qu'un répertoire partiel subsiste**

`internal/service/matrix.go:118` — robustesse — effort S — exigences UC-001

Le retour `MatrixReport{}, err` écrase les CompileErrors calculés. UC-001 A3 exige d'afficher « chaque Cell ou Probe fautive avec le message du compilateur » ; ici l'utilisateur ne voit que l'erreur de suppression, doit relancer (et recompiler 230 sujets) pour retrouver les fautifs, et la postcondition d'échec « aucun répertoire partiel ne subsiste » est violée silencieusement.

*Correctif.* `return MatrixReport{MatrixID: matrix.ID, CompileErrors: failures}, errors.Join(fmt.Errorf("%d sujet(s) ne compilent pas", len(failures)), fmt.Errorf("matrices/%s n'a pas pu être supprimée, répertoire partiel à retirer : %w", matrix.ID, err))`. Test : memoryStore gagne un removeErr ; asserter CompileErrors non vide et message citant le répertoire.

**A-027 · TestUC001_MatriceExistanteIllisible ne teste pas ce que son nom annonce : la branche Load en erreur après Exists vrai (matrix.go:88-91) n'est couverte par aucun test**

`internal/service/matrix_test.go:246` — tests — effort S — exigences UC-001, BR-001-2

Le test insère puis supprime aussitôt une entrée (lignes 252-253, sans effet), range la matrice sous un autre identifiant et vérifie qu'une génération normale réussit. Exists rend faux, Load n'est jamais appelé en erreur. Une mutation qui, sur erreur de Load, régénérerait la matrice (écrasant sources et matrix.json d'une matrice existante, BR-001-2) passerait la suite.

*Correctif.* Ajouter à memoryStore un champ loadErr consulté par Load ; dans ce test, poser f.store.matrices[matrix.ID] = matrix et loadErr = boom, puis asserter que Generate rend boom, que compiler.built est vide et que sources n'a pas été réécrit.

**A-035 · Un fichier d'échappement désigné explicitement n'est pas vérifié contre la matrice de la campagne**

`internal/service/verdict.go:205` — robustesse — effort S — exigences UC-005, H-006, H-009, BR-005-2

La découverte automatique liste les rapports par campaign.MatrixID, mais le chemin désigné est lu et adopté sans contrôle de report.MatrixID. H-006 (criteria.go:279) ne lit que les EscapeVerdict et juge donc les cellules d'une autre matrice, en citant ce fichier dans le verdict de la campagne ; H-009 tombe INCONCLUSIVE par Matrix.Cell inconnue, sans dire pourquoi. TestUC005_H006_FichierDesigneExplicitement ne couvre…

*Correctif.* Après ReadEscapeReport : `if report.MatrixID != campaign.MatrixID { return Evidence{}, fmt.Errorf("%w : %s porte la matrice %s, la campagne %s porte %s", ErrPrecondition, escapePath, report.MatrixID, campaign.ID, campaign.MatrixID) }` et un sous-test dans verdict_test.go.


### Suggestions (35)

| Constat | Emplacement | Objet | Correctif | Effort |
|---|---|---|---|---|
| A-117 | `.claude/hooks/go-check.sh:15` | gofmt -w réécrit le fichier juste édité : la vue de Claude devient périmée et le prochain Edit peut échouer | Soit ne pas réécrire et rendre exit 2 avec « fichier non formaté : relire puis gofmt -w » ; soit conserver et préciser dans le message stderr « relire le fichier avant le prochain… | S |
| A-118 | `.claude/hooks/selftest.sh:33` | Le selftest n'exerce ni go-check.sh ni les cas de contournement de guard-paths (`..`, casse, jq absent, chemin relatif spec-lint) | Ajouter : `check 2 guard-paths.sh "$DIR" "$(j internal/../results/x.json)"`, en forme windows `check 2 … "$(j Results/x.json)"`, un appel avec `PATH=/nonexistent` attendant 2,… | S |
| A-119 | `.claude/hooks/spec-lint.sh:22` | DECLARED devient multi-ligne si « **Use Case ID:** UC-### » apparaît deux fois, d'où une fausse incohérence d'identifiant | Ajouter `\| head -n 1` après le second grep, ou comparer sur `sort -u` et exiger une seule valeur. | S |
| A-111 | `.claude/settings.json:29` | Hook Stop sans timeout explicite et relancé à chaque tour même sans modification Go | Ajouter `"timeout": 300` à l'entrée Stop (et `"timeout": 120` au PostToolUse go-check qui fait `go vet ./...`). Optionnel : dans go-test.sh, sortir 0 si `git status --porcelain --… | S |
| A-116 | `.claude/skills/implement/SKILL.md:56` | Phrase d'interdiction ambiguë : « modifier docs/ (sauf docs/dashboard.md, jamais) » | « Interdits : modifier `docs/` ; éditer `docs/dashboard.md` (toujours régénéré par le binaire, BR-005-3) ; écrire dans … » | S |
| A-115 | `Makefile:10` | `make` sans argument exécute fmt (première cible) et réécrit les sources | Ajouter `.DEFAULT_GOAL := test` (ou une cible `all: vet test`) en tête du Makefile. | S |
| A-093 | `internal/adapters/cli/params.go:170` | FindRoot remonte aussi depuis un --root explicite : un mauvais chemin se résout silencieusement au projet englobant | Quand --root est fourni, exiger `docs/requirements.md` directement sous ce chemin (os.Stat) et ne remonter que depuis le répertoire courant. | S |
| A-097 | `internal/adapters/cli/render.go:101` | Round(1e9) au lieu de Round(time.Second) | Remplacer par `report.Duration.Round(time.Second)` (importer time). | S |
| A-095 | `internal/adapters/cli/render.go:169` | RenderVerdicts insère le rationale dans une table Markdown sans échapper `\|` ni les retours de ligne | `strings.NewReplacer("\|", "\\\|", "\n", " ").Replace(verdict.Rationale)` avant l'insertion. | S |
| A-098 | `internal/adapters/cli/render.go:193` | Le paramètre `max` masque la fonction intégrée max de Go 1.21+ | Renommer en `limit`. | S |
| A-096 | `internal/adapters/dashboard/dashboard.go:19` | Le champ Writer.root est assigné et jamais lu | Supprimer le champ et l'initialisation `root: root`. | S |
| A-075 | `internal/adapters/escape/escape.go:48` | IsEscape ignore silencieusement « leaking param » ; le choix n'est documenté nulle part | Ajouter « leaking param: f » → false et « leaking param content: t » → false dans la table de TestIsEscape, et une phrase dans le commentaire de IsEscape : les fuites de… | S |
| A-076 | `internal/adapters/escape/escape.go:56` | parseDiagnostics ne compare jamais le fichier du diagnostic à sourcePath | Dans Classify, ne garder que les diagnostics dont filepath.Base(diag.File) == filepath.Base(sourcePath) (les séparateurs mixtes « subjects\\X/subject.go » de Windows interdisent… | S |
| A-079 | `internal/adapters/escape/escape.go:99` | La variable fallback est diags[0] sous un autre nom | Supprimer fallback et la branche i == 0 ; écrire `verdict.CompilerReason = diags[0].Raw` après la boucle. Aucun gabarit touché. | S |
| A-077 | `internal/adapters/escape/escape.go:317` | carriesAddress prend le sélecteur d'un champ pour un alias s'ils portent le même nom | Dans carriesAddress et referencesAny, sur *ast.SelectorExpr inspecter uniquement expr.X (retourner false après un ast.Inspect explicite de X) pour ne jamais lire Sel comme une… | S |
| A-071 | `internal/adapters/gotool/treecpu_windows.go:52` | syscall.NewLazyDLL("kernel32.dll") et NewProc sont reconstruits à chaque appel dans trois fichiers | Un fichier kernel32_windows.go par paquet : `var kernel32 = syscall.NewLazyDLL("kernel32.dll")` et les procs en variables de paquet (`procCreateJobObjectW =… | S |
| A-094 | `internal/adapters/specs/specs.go:30` | Le tableau des hypothèses est reconnu ligne par ligne, sans ancrage à sa section ni à son en-tête | Ne lire que les lignes situées après l'en-tête `\| ID \| Source (BEPG) \| Énoncé réfutable \| Critère de réfutation \| UC liés \|` jusqu'à la première ligne non-tableau, et exiger… | S |
| A-059 | `internal/adapters/store/store_test.go:204` | Tests manquants : NextCampaignID au-delà de 9, aller-retour de la quiétude C-010 par le DTO, ordre de LatestComparisonSet et de… | Ajouter TestNextCampaignID_ApresDix (créer C-<jour>-9 et C-<jour>-10, attendre -11), TestMeasurement_QuietudeRoundTrip (occupation 0 mesurée ⇒ relue mesurée ; non mesurée ⇒ champ… | S |
| A-070 | `internal/adapters/system/quietude.go:165` | system.Occupancy duplique gotool.occupancyOf et n'est appelée que par ses tests ; CPUTimes.Total et sortedLevels sont aussi morts | Supprimer system.Occupancy, CPUTimes.Total et sortedLevels avec leurs tests (TestC010_Occupancy, TestSortedLevels, les assertions sur Total), ou, si une seule formule est voulue,… | S |
| A-069 | `internal/adapters/system/quietude.go:193` | CPUCount suit le masque d'affinité du processus alors que les sondes comptent toute la machine | Linux : compter les lignes `cpu\d+` de /proc/stat dans sampleCPUTimes et exposer le compte ; Windows : GetActiveProcessorCount(ALL_PROCESSOR_GROUPS = 0xffff) via kernel32. Repli… | S |
| A-080 | `internal/harness/harness.go:262` | Le message de refus parle de « nœuds » pour les sondes de parcours | Formuler « au moins deux lignes de cache (128 octets) » qui vaut pour les trois genres. Aucun gabarit touché. | S |
| A-081 | `internal/harness/templates/probe.go.tmpl:11` | La ligne de cache est figée à 64 octets alors que le rejeu arm64 (C-006) reste ouvert | Rien à faire tant que seul amd64 est mesuré. Pour arm64 : détecter la taille de ligne dans la Provenance (C-008 f en détecte déjà trois) et la passer au gabarit comme paramètre de… ⚠ | M |
| A-017 | `internal/models/layout_test.go:171` | Cas non couverts : Repeats/Payloads sans 1, Replicates négatif, Payloads(), Cell.Validate Payload/Replicate négatifs, Normalize… | Ajouter : sous-tests Repeats:[4] et Payloads:[2] exigeant la paire LOCAL ; Replicates:-1 refusé ; Cell{Payload:-1} et Cell{Replicate:-1} refusés ; Payloads() sur 0 et 3 ;… | S |
| A-014 | `internal/models/matrix.go:281` | sameLayouts, sameInts, dedupeSorted et les cinq Valid() réimplémentent le paquet slices | Remplacer par slices.Equal / slices.Contains ; garder dedupeSorted uniquement si l'ordre des bool par `less` est conservé (sinon `slices.SortFunc` + `slices.Compact`). | S |
| A-013 | `internal/models/matrix.go:382` | Le saut de réplicat est évalué au niveau le plus profond alors qu'il ne dépend que de layout et profile | Déplacer le `if replicate > 1 && (...) { continue }` immédiatement après le saut du témoin nul hors LOCAL (ligne 359), avant `for _, repeat := range n.Repeats`. | S |
| A-012 | `internal/models/matrix.go:427` | ParseProbeID accepte des paramètres non canoniques (« 0100 », « +100 ») qui ne se re-rendent pas à l'identique | Ajouter `\|\| strconv.Itoa(parameter) != parts[2]` à la condition de rejet, et un cas « probe/APPEND_GROW/0100 » dans TestParseProbeID. | S |
| A-011 | `internal/models/matrix.go:476` | Matrix.Validate ne vérifie pas ID == Parameters.MatrixID() et Cell.Validate n'encode pas les règles de déclinaison | Dans Matrix.Validate : `if m.ID != m.Parameters.MatrixID() { return invalid(...) }` et `m.Parameters.Validate()`. Optionnel : dans Cell.Validate, refuser Repeat/Payload > 1 hors… | M |
| A-015 | `internal/models/models.go:481` | Provenance.Validate accepte des tailles de cache, une taille de page et un GOMAXPROCS négatifs | Ajouter un cas `p.L1DataCacheBytes < 0 \|\| p.LastLevelCacheBytes < 0 \|\| p.PageSizeBytes < 0 \|\| p.GOMAXPROCS < 0`. | S |
| A-016 | `internal/models/models.go:551` | Measurement.Validate ne vérifie pas que les valeurs sont finies et non négatives | Dans la branche COMPLETE : boucler et refuser `math.IsNaN \|\| math.IsInf \|\| v < 0` pour NsPerOp, `v < 0` pour les deux listes entières. | S |
| A-134 | `internal/service/campaign.go:250` | Deux erreurs de service sans sentinelle : « aucun sujet mesuré » (campaign.go:250) et « n sujet(s) ne compilent pas »… | Ajouter `var ErrNoSubjectMeasured` et `var ErrCompileFailures` dans matrix.go et envelopper avec %w. | S |
| A-042 | `internal/service/compare.go:19` | ComparisonMethod recopie « 2000 » en texte au lieu de dériver de BootstrapResamples | Remplacer la constante par `func comparisonMethod() string { return fmt.Sprintf("bootstrap percentile …, %d rééchantillonnages, …", BootstrapResamples) }` ou ajouter un test qui… | S |
| A-038 | `internal/service/compare.go:183` | Un échantillon à variance nulle donne un IC dégénéré [d, d] et rend significatif tout delta non nul | Consigner dans Comparison un indicateur de largeur d'intervalle (ou le nombre de valeurs distinctes des deux échantillons) pour que le lecteur du fichier voie la dégénérescence ;… | S |
| A-037 | `internal/service/compare.go:194` | percentile tronque l'indice : l'intervalle bootstrap est asymétrique d'un rang | Utiliser l'arrondi au plus proche (`int(math.Round(p*float64(n-1)))`) ou l'interpolation linéaire entre les deux rangs voisins ; ajuster TestPercentile. Les fichiers de… | S |
| A-041 | `internal/service/criteria_c008_test.go:195` | Bornes inclusives/exclusives de H-001, H-008, H-011 et H-013 sans test au point exact | Ajouter aux tables : H-008 et H-013 (residentNs 1.0, nonResidentNs 100.0 -> REFUTED ; 1.0 et 200.0 -> CONFIRMED), H-011 (grow bytes 3200 et 4800 sur prealloc 800 -> CONFIRMED),… | S |
| A-040 | `internal/service/verdict.go:157` | Evidence.CampaignPath duplique la disposition du store au lieu du port CampaignStore.CampaignPath | Ajouter un champ Evidence.CampaignFile renseigné dans gather par s.store.CampaignPath(campaign.ID) et le lire dans evaluate ; retirer la méthode CampaignPath. | S |

⚠ change `Campaign.harnessDigest`.

### Constats écartés par la vérification (4)

| Constat | Emplacement | Énoncé | Motif du rejet |
|---|---|---|---|
| A-120 | `.gitignore:2` | Entrée `tmpcheck/` : résidu d'un module de brouillon local (go.mod + main.go présents sur le poste) | La prémisse centrale du constat est fausse. L'entrée `tmpcheck/` est documentée à deux endroits : 1. `../Doc/DECISION.md:15` — « **D-03 — Le bac à sable `tmpcheck/` est un module imbriqué ignoré par Git.** Les vérifications jetables faites pendant la… |
| A-137 | `internal/adapters/cli/render.go:9` | internal/adapters/cli importe internal/service (types de rapport) : conforme au texte de CLAUDE.md, mais seul adapter à dépendre d'un… | Le fait rapporté est exact, mais ce n'est pas un défaut. Vérifié ligne par ligne. 1) Exactitude de l'extrait : `internal/adapters/cli/render.go:8-9` contient bien `"github.com/agbruneau/escapebench/internal/models"` et… |
| A-083 | `internal/adapters/specs/index.go:75` | La colonne Code est ✔ pour tout UC cité n'importe où dans un fichier .go, y compris l'usage de main.go et les fixtures de test | Le mécanisme décrit est exact (index.go:47-54 et 75-80 : texte brut de tout .go sous cmd/ et internal/, regex `\bUC-\d{3}\b` à specs.go:139), mais ce n'est pas un défaut : c'est le comportement spécifié et testé. (1) La définition qui fait autorité,… |
| A-058 | `internal/adapters/store/dto.go:411` | comparisonDTO écrit toujours layout (même ARRAY_FILL) alors que tippingPointDTO et cellDTO l'omettent par défaut | La prémisse du constat est fausse et son correctif contredit la spécification. 1) cellDTO n'omet PAS la disposition ARRAY_FILL : dto.go:67 `Layout: string(c.TypeSpec.Layout)` sans aucune garde, exactement comme comparisonDTO à dto.go:411. Vérifié par… |

### Constats confirmés lors de la contre-vérification (8 majeurs)

| Constat | Sévérité | Emplacement | Énoncé | Statut |
|---|---|---|---|---|
| A-156 | Majeur | `cmd/escapebench/main.go:231` | --benchtime n'est jamais validé : une faute de frappe crée une campagne entière de mesures FAILED | Confirmé (3/3) |
| A-141 | Majeur | `internal/adapters/gotool/gotool.go:118` | Un `go build -gcflags=-m` tué par l'annulation du contexte devient un verdict COMPILE_ERROR | Confirmé (3/3) |
| A-278 | Majeur | `internal/adapters/specs/index.go:48` | La colonne Integration du tableau de bord est un faux positif : le tag de build est cherché comme sous-chaîne, et seuls des littéraux de chaîne le matchent | Confirmé (3/3) |
| A-246 | Majeur | `internal/harness/harness.go:51` | L'empreinte du harnais ne couvre que templates/*.tmpl, pas harness.go qui façonne pourtant la source générée | Confirmé (3/3) |
| A-263 | Majeur | `internal/service/campaign.go:141` | Le verrou est pris après la création de la Campaign : campagne RUNNING orpheline si une autre est en cours, et fenêtre de collision d'identifiant | Confirmé (3/3) |
| A-265 | Majeur | `internal/service/campaign.go:180` | La reprise réutilise les drapeaux de la ligne de commande, pas ceux de la campagne : la Campaign n'enregistre ni benchtime ni cpu | Confirmé (3/3) |
| A-187 | Majeur | `internal/service/compare_test.go:119` | BR-004-2 : le sens « IC exclut zéro ⇒ significant » n'est vérifié nulle part ; une mutation à seuil survit à toute la suite | Confirmé (3/3) |
| A-185 | Majeur | `internal/service/verdict_test.go:205` | UC-005 A1 : le message ne nomme pas les hypothèses modifiées et le fake digestOf masque l'absence | Confirmé (3/3) |

### Constats sans vérification contradictoire complète (75)

| Constat | Sévérité annoncée | Emplacement | Énoncé | Votes |
|---|---|---|---|---|
| A-147 | Mineur | `cmd/escapebench/main.go:67` | Après le premier SIGINT, tous les signaux suivants sont avalés : un second Ctrl+C ne peut pas forcer l'arrêt | 0/3 |
| A-268 | Mineur | `cmd/escapebench/main.go:113` | L'étape 7 et A2 sont tautologiques en production : l'empreinte du harnais est calculée une fois sur embed.FS et ne peut pas changer pendant la… | 0/3 |
| A-271 | Mineur | `cmd/escapebench/main.go:249` | L'étape 4 (afficher identifiant, provenance et décomptes) n'a lieu qu'après l'étape 8 : rien n'est affiché pendant la campagne | 0/3 |
| A-254 | Mineur | `docs/entity-model.md:77` | Table Probe cassée : un paragraphe est inséré entre deux lignes et la ligne sourceFile est orpheline | 0/3 |
| A-253 | Mineur | `docs/entity-model.md:85` | Matrix.parameters n'énumère pas les dimensions layouts, repeats, payloads, replicates que le code et le JSON portent — spec à mettre à jour d'abord | 0/3 |
| A-252 | Mineur | `docs/use-cases/UC-001-generer-matrice.md:26` | Le scénario principal de UC-001 ne nomme ni la dimension de réplicat ni deux des quatre sauts de génération — spec à mettre à jour d'abord | 0/3 |
| A-284 | Mineur | `docs/use-cases/UC-004-comparer-valeur-pointeur.md:26` | UC-004 étape 5 définit le point de bascule par couple (profil, champ pointeur) ; le code le calcule par triplet avec la disposition | 0/3 |
| A-133 | Mineur | `internal/adapters/cli/cli_test.go:16` | Aucun test hors internal/service n'est nommé TestUC###_… : 0 sur cmd, adapters, models et harness (205 fonctions de test) | 2/3 |
| A-172 | Mineur | `internal/adapters/escape/escape.go:69` | CompilerReason conserve le séparateur de chemin du système : un rapport d'échappement n'est pas reproductible octet pour octet entre Windows et Linux | 0/3 |
| A-161 | Mineur | `internal/adapters/gotool/gotool.go:119` | La sortie brute de `go build`/`go test` est consignée dans results/ et peut contenir le chemin absolu du poste | 0/3 |
| A-196 | Mineur | `internal/adapters/gotool/gotool_test.go:268` | TestExecRunner dépend du binaire `go` sur le PATH dans la suite unitaire | 0/3 |
| A-128 | Mineur | `internal/adapters/gotool/quietude_test.go:96` | time.Sleep dans un test, aucun usage de testing/synctest, parce que Toolchain.Run lit l'horloge réelle (time.Now/time.Since) au lieu de l'horloge… | 2/3 |
| A-163 | Mineur | `internal/adapters/specs/specs.go:45` | Une ligne d'hypothèse à plus de cinq cellules ou un identifiant en double passent silencieusement dans l'empreinte des critères | 0/3 |
| A-159 | Mineur | `internal/adapters/store/store.go:203` | Identifiants de matrice et de campagne issus des drapeaux joints tels quels aux chemins : `..` sort de matrices/ et de results/ | 0/3 |
| A-162 | Mineur | `internal/adapters/store/store.go:252` | Les lecteurs (Load, LoadCampaign, LoadMeasurements) ne valident pas ce qu'ils décodent, contrairement aux écrivains | 0/3 |
| A-164 | Mineur | `internal/adapters/store/store.go:544` | Tous les fichiers écrits par writeFileAtomic naissent en mode 0600 sur Linux/macOS | 0/3 |
| A-195 | Mineur | `internal/adapters/store/store_test.go:351` | store_test teste LatestVerdictReport (sans appelant en production) et pas VerdictReports (utilisé par le tableau de bord, C-009) | 0/3 |
| A-129 | Mineur | `internal/adapters/system/quietude_test.go:91` | TestC010_SondeReelle est un test temporel sur horloge réelle et compteurs système, sans synctest, à résultat dépendant de la charge de la machine | 1/3 |
| A-188 | Mineur | `internal/harness/harness_test.go:21` | Aucun test du harnais ne détecte une empreinte constante : Digest() figé à 64 « a » passe toute la suite | 0/3 |
| A-247 | Mineur | `internal/models/matrix.go:238` | La règle « au moins deux nœuds » des sondes de parcours n'est pas appliquée à l'étape 2 mais au rendu | 0/3 |
| A-145 | Mineur | `internal/service/campaign.go:195` | L'échec de ReleaseLock est avalé : un verrou orphelin bloque silencieusement UC-001 et UC-003 | 0/3 |
| A-151 | Mineur | `internal/service/campaign.go:227` | Sur une reprise, Duration couvre l'interruption entière (StartedAt d'origine à la fin de la reprise) | 0/3 |
| A-269 | Mineur | `internal/service/campaign.go:243` | Comportement absent du UC : la campagne est passée ABORTED avec la raison « aucun sujet mesuré » | 0/3 |
| A-193 | Mineur | `internal/service/campaign_test.go:100` | Étape 5 « chaque Cell puis chaque Probe » : seul order[0] est vérifié | 0/3 |
| A-194 | Mineur | `internal/service/campaign_test.go:139` | A2 étape 3 (« invalide pour tout verdict ») et UC-004 A1 (« affiche le statut ») : les messages exigés ne sont pas vérifiés | 0/3 |
| A-192 | Mineur | `internal/service/campaign_test.go:176` | A3 : le message d'erreur du sujet FAILED n'est pas vérifié dans la mesure écrite | 0/3 |
| A-205 | Mineur | `internal/service/compare.go:172` | Le déterminisme de l'IC bootstrap n'est garanti que pour une même toolchain 64 bits | 0/3 |
| A-282 | Mineur | `internal/service/criteria.go:55` | evaluateH001 restreint le corpus à ARRAY_FILL, filtre absent du critère gelé de H-001 | 0/3 |
| A-283 | Mineur | `internal/service/criteria.go:98` | evaluateH002 ne lit que les séries ARRAY_FILL alors que le critère gelé parle des « deux séries (avec, sans champ pointeur) » sans disposition | 0/3 |
| A-206 | Mineur | `internal/service/criteria.go:256` | Les rationales arrondissent les rapports à une précision qui peut contredire le seuil appliqué | 0/3 |
| A-255 | Mineur | `internal/service/escape.go:72` | UC-002 étape 2 : la Provenance « qu'il va consigner » n'est affichée qu'à l'étape 7, et jamais en cas d'échec | 0/3 |
| A-256 | Mineur | `internal/service/escape.go:136` | Un fichier de verdicts antérieur illisible bloque définitivement UC-002 pour cette matrice | 0/3 |
| A-250 | Mineur | `internal/service/matrix.go:112` | Échec de WriteSources ou de Finalize : aucun Remove, le répertoire partiel subsiste | 0/3 |
| A-249 | Mineur | `internal/service/matrix.go:168` | compileAll traite toute erreur de Build comme un sujet non compilable (A3), sans distinguer *ports.CompileError | 0/3 |
| A-160 | Mineur | `internal/service/verdict.go:211` | --escape est enregistré tel quel dans resultFiles : chemin absolu ou à barres obliques inverses écrit dans results/verdicts | 0/3 |
| A-191 | Mineur | `internal/service/verdict_test.go:230` | UC-005 A2 : « liste des sujets manquants » ni produite (H-004, H-005, H-011) ni exigée par les tests | 0/3 |
| A-199 | Suggestion | `cmd/escapebench/main_test.go:130` | La chaîne complète exerce `dashboard` sur une racine sans go.mod : colonnes Unit/Regression jamais vérifiées de bout en bout | 0/3 |
| A-212 | Suggestion | `internal/adapters/gotool/gotool.go:203` | La fenêtre d'attestation de quiétude englobe la compilation et l'édition de liens, ce qui dilue une contention pendant le seul banc | 0/3 |
| A-200 | Suggestion | `internal/adapters/gotool/gotool_test.go:139` | BR-003-4 « un sujet, un processus » : TestRunMesureUnSujet n'assert pas un seul appel au runner | 0/3 |
| A-276 | Suggestion | `internal/adapters/store/dto.go:326` | Une Measurement FAILED sérialise ses listes en `null` alors que le modèle d'entités dit « listes vides » | 0/3 |
| A-234 | Suggestion | `internal/adapters/store/store.go:502` | Le comparateur de TippingKey est recopié trois fois (store.sortedTippingKeys, service/compare.sortedKeys, cli.RenderComparison) et le libellé de… | 0/3 |
| A-182 | Suggestion | `internal/adapters/store/store.go:558` | os.Rename par-dessus campaign.json sous OneDrive/antivirus peut échouer transitoirement (ERROR_SHARING_VIOLATION) ; aucune reprise | 0/3 |
| A-203 | Suggestion | `internal/adapters/store/store_test.go:193` | time.Now() dans TestEscapeReportRoundTripEtImmutabilite | 0/3 |
| A-179 | Suggestion | `internal/adapters/system/topology_other.go:7` | darwin et freebsd tombent dans _other : H-008 et H-013 sont non concluantes par construction sur l'arm64 le plus accessible (Apple Silicon), que… | 0/3 |
| A-178 | Suggestion | `internal/adapters/system/topology_windows.go:25` | Les parseurs purs (parseWindowsCaches, cpuTimesFrom, parseProcStat, parseLinuxCacheSize, readLinuxCaches) et leurs tests ne se compilent que sur leur… | 0/3 |
| A-180 | Suggestion | `internal/harness/harness.go:24` | Aucun test ne garantit que les gabarits embarqués sont en LF : un fichier converti en CRLF par un éditeur Windows change harnessDigest sans que rien… | 0/3 |
| A-225 | Suggestion | `internal/models/matrix.go:521` | Matrix.Probe est morte hors tests | 0/3 |
| A-238 | Suggestion | `internal/models/models.go:252` | Repetitions, Payloads, ReplicateIndex et les trois bornages de cellDTO.toModel réécrivent max(1, x) | 0/3 |
| A-224 | Suggestion | `internal/models/models.go:269` | Cell.ReplicateIndex est morte hors tests | 0/3 |
| A-260 | Suggestion | `internal/models/models.go:290` | Cell.Validate accepte repeat, payload et replicate à 0 alors que le modèle les déclare « Requis, ≥ 1 » | 0/3 |
| A-241 | Suggestion | `internal/models/models.go:661` | Comparison.EffectiveLayout double le défaut déjà appliqué par comparisonSetDTO.toModel ; usage incohérent entre évaluateurs | 0/3 |
| A-204 | Suggestion | `internal/models/models_test.go:353` | TestMedians ne couvre pas MedianInt sur un effectif pair | 0/3 |
| A-138 | Suggestion | `internal/models/revue_test.go:19` | Huit fichiers de test sans aucun t.Run : la règle « tests table-driven » de CLAUDE.md n'y est pas appliquée | 0/3 |
| A-240 | Suggestion | `internal/service/campaign.go:122` | Boucle de recherche de « H-007 » remplaçable par slices.Contains | 0/3 |
| A-274 | Suggestion | `internal/service/campaign.go:187` | CampaignReport.Status est vide sur les chemins d'erreur (verrou refusé, runner en panne) : le rendu affiche « statut : » alors que la campagne est… | 0/3 |
| A-134 | Suggestion | `internal/service/campaign.go:250` | Deux erreurs de service sans sentinelle : « aucun sujet mesuré » (campaign.go:250) et « n sujet(s) ne compilent pas » (matrix.go:122) | 2/3 |
| A-201 | Suggestion | `internal/service/campaign_test.go:393` | BR-003-2 (Provenance complète obligatoire) sans test au niveau du service | 0/3 |
| A-290 | Suggestion | `internal/service/compare.go:96` | Quand toutes les paires sont exclues, les raisons d'exclusion ne sont consignées nulle part | 0/3 |
| A-202 | Suggestion | `internal/service/compare_test.go:140` | UC-004 A3 n'a pas de test nommé TestUC004_A3_ ; BR-001-2, BR-002-3, BR-003-3, BR-004-3 n'ont pas de TestUC###_BR#_ | 0/3 |
| A-243 | Suggestion | `internal/service/criteria.go:31` | service.SmallStructBytes est un alias exporté de models.SmallStructBytes que personne n'importe | 0/3 |
| A-210 | Suggestion | `internal/service/criteria.go:68` | H-001 compte des Comparison, non des tailles distinctes, contrairement au texte gelé « au moins deux tailles » | 0/3 |
| A-230 | Suggestion | `internal/service/criteria_c008.go:115` | pointerWins a un paramètre ref jamais différent de c | 0/3 |
| A-245 | Suggestion | `internal/service/criteria_c008.go:223` | evaluateH009 appelle Matrix.Cell (balayage linéaire avec reconstruction de l'ID) dans deux boucles imbriquées sur les verdicts | 0/3 |
| A-228 | Suggestion | `internal/service/criteria_c008.go:416` | abs réinvente math.Abs | 0/3 |
| A-233 | Suggestion | `internal/service/criteria_c009.go:22` | h012Cell.medianValue, medianPointer et significant ne servent qu'à l'intérieur de h012CellsOf | 0/3 |
| A-232 | Suggestion | `internal/service/criteria_c009.go:42` | h012JudgedCells rend un struct anonyme alors que h012Key porte exactement les mêmes champs | 0/3 |
| A-242 | Suggestion | `internal/service/criteria_c009.go:60` | H-012 code en dur deux identifiants de campagne là où H-011 a une constante nommée PreH011CampaignID | 0/3 |
| A-231 | Suggestion | `internal/service/criteria_c009.go:156` | replicateCount est redondante : la valeur zéro de h012Cell a déjà len(deltas) == 0 | 0/3 |
| A-227 | Suggestion | `internal/service/criteria_c009.go:230` | medianOf recopie models.MedianFloat à l'identique | 0/3 |
| A-229 | Suggestion | `internal/service/criteria_c009.go:244` | maxOf réinvente le builtin max (Go 1.21+, module en go 1.25) | 0/3 |
| A-257 | Suggestion | `internal/service/escape.go:162` | DifferingVerdicts ignore compilerReason et compilerError alors que NFR-002 exige des verdicts identiques cellule par cellule | 0/3 |
| A-198 | Suggestion | `internal/service/fakes_test.go:210` | Le memoryStore n'a jamais de collision d'horodatage, le store réel si (résolution 1 s) | 0/3 |
| A-286 | Suggestion | `internal/service/verdict.go:226` | Dashboard lance `go test ./...` six fois (RunAll + un RunUseCaseTests par UC) alors qu'une seule exécution -v suffit | 0/3 |
| A-244 | Suggestion | `internal/service/verdict.go:319` | join est un alias trivial de strings.Join(values, ", ") et n'est pas utilisé uniformément | 0/3 |
| A-239 | Suggestion | `internal/service/verdict.go:322` | sortedSubjectIDs et store.sortedKeys réinventent slices.Sorted(maps.Keys(m)) | 0/3 |
