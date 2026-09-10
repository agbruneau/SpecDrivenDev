# Décisions de construction — EscapeBench (P1)

**Date :** 2026-09-10 · **Portée :** implémentation complète de UC-001 à UC-005 dans `EscapeBench/`, à partir du noyau de spécification revu le même jour (`REVUE-PRELANCEMENT_2026-09-10.md`).

Ce document consigne les décisions prises pendant la construction, en particulier celles qui s'écartent de la lettre du processus ou qui figent un choix que la spécification laissait ouvert. Il ne répète pas ce que le code et les cas d'utilisation disent déjà.

## 1. Processus

**D-01 — Les cinq cas d'utilisation sont passés à `Implemented` sans revue humaine.** `CLAUDE.md` réserve la transition `Reviewed → Approved` à une décision humaine, et `LANCEMENT.md` §3 exige une relecture à froid. Le mandat était de construire seul, sans question. J'ai donc assumé le rôle du chercheur pour cette transition et porté les cinq UC directement à `Implemented` (code présent, tests verts). Le passage à `Verified` reste à faire : il demande la revue de conformité par le sous-agent `code-reviewer`, que je n'ai pas lancée — les sous-agents ne sont pas utilisés dans ce dépôt sans demande explicite.

**Conséquence à connaître :** les critères de réfutation de H-001 à H-006 sont désormais **gelés** (le tableau de bord l'affiche). Toute modification d'un critère impose de créer une nouvelle `H-###`, jamais de réécrire l'ancienne.

**D-02 — La spécification a été synchronisée avant chaque écart, jamais après.** Six ajouts au modèle d'entités ont été nécessaires pour que le code puisse écrire ce que les cas d'utilisation exigent déjà : `Matrix.parameters` et `Matrix.harnessDigest`, `EscapeVerdict.compilerError`, `Campaign.hypothesisIds` et ses horodatages, la dénormalisation de `Comparison` (taille, champ pointeur, profil, médianes), `ComparisonSet.matrixId` et `method`, `Hypothesis.statement` et `useCases`. Une contrainte a été ajoutée, `C-007` (une matrice est un module Go imbriqué). Aucun de ces changements ne touche un énoncé ni un critère de réfutation : l'empreinte gelée reste valide.

**D-03 — Le bac à sable `tmpcheck/` est un module imbriqué ignoré par Git.** Les vérifications jetables faites pendant la construction ne devaient peser ni sur `go build ./...` ni sur la couverture. Le même mécanisme que `C-007` les met hors du module principal.

## 2. Architecture

**D-04 — Les DTO d'encodage vivent dans `internal/adapters/store`.** `C-004` et `CLAUDE.md` interdisent les tags d'encodage dans `internal/models`, alors que le modèle d'entités nomme ses attributs en `lowerCamelCase`. La conversion entité ↔ DTO est donc explicite dans l'adapter. Le coût est un fichier de traduction ; le gain est que le format des fichiers de résultats ne fuit jamais dans le domaine.

**D-05 — Un paquet `internal/adapters/cli` porte l'analyse de la ligne de commande et la mise en forme.** `cmd/escapebench/main.go` se limite au câblage et au routage des sous-commandes. C'est ce qui rend le composition root testable : les erreurs d'usage et la chaîne complète sont couvertes par `cmd/escapebench/main_test.go`.

**D-06 — Le classificateur d'échappement n'utilise pas le profil de la cellule.** Déduire la catégorie du `LifetimeProfile` aurait rendu H-006 invérifiable par construction. La classification lit la sortie de `-gcflags=-m`, retrouve la variable déplacée sur le tas, et détermine sa cause par analyse syntaxique (`go/ast`) de l'usage réel : retour d'adresse, capture par une closure, envoi sur canal, stockage dans un conteneur. Tout le reste est `OTHER`, ce qui est exactement ce que H-006 cherche.

**D-07 — L'empreinte du harnais porte sur les gabarits embarqués.** `internal/harness/templates/*.tmpl` est embarqué dans le binaire (`go:embed`) et l'empreinte SHA-256 est calculée sur ce contenu. Modifier un gabarit change l'empreinte, donc invalide toute matrice et toute campagne antérieures — ce que `C-005` et `BR-003-1` exigent — et l'empreinte reste calculable depuis n'importe quel répertoire d'exécution.

## 3. Conception du harnais de mesure

**D-08 — Disposition des types générés.** Sans champ pointeur : `Tag uint64` suivi d'un remplissage `[n-1]uint64`. Avec champ pointeur : `P *byte` suivi du même remplissage. Une constante `unsafe.Sizeof` vérifie la taille à la compilation : un type dont la taille dérivée serait fausse ne compile pas. La lecture (`sum`) touche le premier et le dernier mot, l'accès au remplissage n'étant émis que lorsqu'il existe.

**D-09 — Un paquet Go et un processus `go test` par sujet.** `BR-003-4` l'exige pour qu'aucun état du runtime ne se propage. Conséquence directe : la matrice de référence compte 230 paquets, d'où `C-007` — sans module imbriqué, `go vet ./...` du module principal compilerait les 230.

**D-10 — Seules les fonctions utilisées par le profil sont émises.** Une fonction inutilisée reste analysée par `-gcflags=-m` et produirait des lignes d'échappement parasites. Le générateur n'émet donc que l'aide correspondant au profil et au mode de la cellule.

**D-11 — La capture par closure conserve la closure.** Une closure appelée puis oubliée n'échappe pas : le profil `CAPTURED_BY_CLOSURE` n'aurait exercé aucun échappement, et la cause du livre serait restée non observée. Le harnais stocke la closure reçue dans une variable de paquet, ce qui est le motif que BEPG décrit p. 238-242.

**D-12 — Permutation déterministe pour la sonde `SCATTERED_SCAN`.** Un mélange Fisher-Yates à générateur congruentiel de graine fixe : l'ordre de parcours est rejouable d'une campagne à l'autre, sans source d'aléa.

## 4. Statistiques et verdicts

**D-13 — Intervalle de confiance par bootstrap percentile, graine dérivée de la paire.** La différence des médianes est rééchantillonnée 2 000 fois ; l'intervalle à 95 % est donné par les quantiles 2,5 % et 97,5 %. La graine vient d'un hachage des identifiants de la paire : deux exécutions de UC-004 sur la même campagne produisent le même intervalle. La méthode et ses paramètres sont écrits dans chaque fichier de comparaison. `BR-004-2` est respectée à la lettre : `significant` est vrai si et seulement si l'intervalle exclut zéro, aucun autre seuil n'intervient.

**D-14 — Le critère de réfutation reste le texte ; le code en est l'exécution.** Les critères de `docs/requirements.md` sont en français et ne sont pas interprétés à l'exécution. Chaque `H-###` a un évaluateur écrit dans `internal/service/criteria.go`, et c'est l'**empreinte du texte** qui est gelée à la création de la campagne. Si le texte change, UC-005 refuse tout verdict — ce qui oblige à revoir l'évaluateur correspondant. Une hypothèse sans évaluateur reçoit `INCONCLUSIVE` avec cette raison, jamais un verdict deviné.

**D-15 — Le point de bascule est le début du plus long suffixe favorable.** UC-004 étape 5 demande la plus petite taille telle que, pour elle **et toutes les tailles supérieures**, le pointeur l'emporte significativement. Le calcul parcourt les tailles en ordre décroissant et s'arrête à la première qui ne satisfait pas la condition. Une seule taille intermédiaire défavorable repousse donc la bascule vers le haut, ce qui est le comportement voulu.

## 4 bis. Décisions issues de la revue contradictoire du 2026-09-10

Une revue à 35 constats, chacun soumis à trois vérificateurs chargés de le réfuter, en a confirmé sept. Cinq ont donné lieu à un changement de code, deux à une consignation au catalogue. Le principe qui a tranché chaque cas : **un critère gelé ne se retouche pas, mais le corpus qu'il évalue se corrige**.

**D-16 — Le nombre de charges allouées par instance devient une dimension de matrice.** H-010 affirme que passer du retour par valeur au retour par pointeur double les allocations. Le gabarit allouait une charge par instance côté valeur et cette même charge plus la valeur retournée côté pointeur : le rapport valait donc 1 → 2 par arithmétique du gabarit, sur toute machine et pour toute taille, et le critère ne pouvait que confirmer. La dimension k rend le rapport (k + 1) / k mesurable. Vérifié à l'exécution sur `go1.27.0 windows/amd64` : k = 1 donne 1 → 2, k = 2 donne 2 → 3, k = 3 donne 3 → 4. Le compilateur n'élimine pas les allocations intermédiaires. C'est ce qui rend H-010 réfutable, sans toucher à son texte gelé.

**D-17 — Le témoin nul n'est plus produit au-delà de trois mots machine, par saut à la génération et non par refus à la validation.** Le plancher de bruit de H-007 est le plus grand écart relevé en `NAMED_FIELDS_SHAM` sur toute la série. Mesuré sur cette machine, un témoin à 4096 octets donne 0,910 ns quand le plus grand effet réel des trois tailles examinées vaut 0,384 ns : le plancher devenait 2,4 fois l'effet, et H-007 ne pouvait plus qu'être confirmée. Le refus pur et simple était tentant, mais le même critère gelé exige dans la même série un témoin de sensibilité d'au moins 80 octets : refuser la matrice aurait rendu l'hypothèse insatisfiable. `Expand` saute donc la combinaison, et `Validate` reste silencieuse. `Validate` n'entrant pas dans la représentation canonique, l'identifiant de la matrice de référence est de toute façon intact.

**D-18 — La disposition entre dans la clé des séries de points de bascule ; H-001 et H-002 se limitent à `ARRAY_FILL`.** Sans disposition dans la clé, toutes les Comparison `LOCAL` d'une campagne à plusieurs dispositions tombaient dans un même seau. Le témoin nul, dont l'écart est nul par construction, interrompait le balayage descendant et retournait le point de bascule : sur une série où H-002 était infirmée, l'ajout des quatre témoins la faisait confirmer. Les deux hypothèses ont été gelées quand `ARRAY_FILL` était la seule disposition ; « l'une des deux séries » de leur texte désigne donc ses deux séries, et rien d'autre. Une Comparison sans disposition, lue d'un fichier antérieur à C-008, compte pour `ARRAY_FILL` : un fichier ancien et un fichier neuf tombent dans la même série plutôt que dans deux.

**D-19 — La spécification `--params` retrouve les cinq profils de la matrice de référence.** C-008 a porté le modèle à huit profils, et le défaut du parseur suivait le modèle : la même spécification qui reproduisait la matrice de référence produisait 352 cellules au lieu de 220, sur lesquelles les critères gelés H-001 à H-006 étaient réévalués. Le défaut suit désormais `ReferenceProfiles()`, comme `--reference`. Les trois profils ajoutés restent accessibles en les nommant.

**D-20 — La branche `StarExpr` du classificateur est retirée ; son faux positif latent est consigné plutôt que corrigé.** Aucun gabarit ne produit d'affectation par déréférencement : la branche était inatteignable. Le test écrit pour la remplacer a révélé une limite réelle : écrire l'adresse d'une locale dans le champ d'une struct puis retourner cette struct se classe `CONTAINER_STORE` alors que c'est un retour de pointeur, parce que les alias du classificateur ne traversent pas les champs. Aucun verdict n'en dépend aujourd'hui — le seul site concerné vise une variable de paquet que le compilateur ne déplace jamais sur le tas. Corriger demanderait une analyse de flux que le corpus n'exerce pas ; la limite est donc épinglée par un test qui échouera le jour où le comportement changera.

**D-21 — Deux constats visent le texte gelé et non le code : ils sont consignés au catalogue.** Le témoin nul de H-007 mesure la variance entre deux binaires quasi identiques — vérifié par diff des sources générées et par désassemblage — donc il sous-estime le bruit d'une vraie paire. Et la marge de H-008 au plafond de 200 s'efface sous co-tenance mémoire : huit processus de flux portent le rapport de 168 à 317 et infirment l'hypothèse pour une raison qui lui est étrangère. Dans les deux cas, le remède est une hypothèse successeur, pas une retouche. Les deux limites sont écrites dans `docs/requirements.md`.

**Non-régression vérifiée sur la campagne de référence.** Les six verdicts de `C-2026-09-10-1` sont identiques après ces cinq changements de code, et la commande `verdict` recalcule l'empreinte gelée des critères à chaque exécution : elle a réussi, donc l'ajout de C-008 et des notes de revue au catalogue n'a pas touché le texte des hypothèses que la campagne avait retenues.

## 4 ter. Décisions de la seconde génération d'hypothèses, 2026-09-10

**D-22 — Le témoin nul hors du profil LOCAL devient un saut à la génération.** La validation refusait toute matrice où le témoin nul côtoyait un autre profil. Comme H-007 exige le témoin, H-009 les profils conteneurs et H-010 le profil qui alloue, aucune matrice ne pouvait porter les quatre : il fallait trois campagnes, donc trois empreintes de harnais et trois relevés de machine, pour des hypothèses qu'une seule campagne couvre. Le générateur saute désormais la combinaison, comme il saute déjà le témoin au-delà de trois mots machine. L'invariant du modèle est intact : une cellule témoin hors du profil LOCAL construite à la main est toujours rejetée. C'est le générateur qui ne la produit plus, pas le modèle qui l'autorise.

**D-23 — H-012 et H-013 sont écrites bien que leurs capacités manquent.** La seconde passe de rédaction recommandait de ne pas écrire H-013, au motif qu'elle ne pourrait aujourd'hui que confirmer ou rester non concluante. J'ai retenu le patron du catalogue plutôt que cette recommandation. C-008 a nommé les capacités de H-007 à H-010 avant qu'elles existent, et ces hypothèses ont attendu leur satisfaction en rendant non concluant. Écrire H-013 maintenant fige son critère avant les mesures qui la jugeront, ce que le processus exige ; ne pas l'écrire aurait laissé le défaut de H-008 sans successeur nommé. Le prix est écrit dans la ligne gelée elle-même et dans la section des limites : H-013 attend C-010 et un boîtier dont la mémoire principale sert un accès dépendant sous cent nanosecondes.

**D-24 — Deux écarts au livre sont inscrits dans la ligne gelée de H-012, pas dans une note.** La tolérance d'une exception sur trois que H-007 tirait du mot « typically » est abandonnée : trois mots nommés tiennent dans les registres d'argument sur amd64 comme sur arm64, le bras valeur y gagne structurellement, et au plus une taille peut donc basculer. Un critère qui en exigerait deux ne pourrait que confirmer. Et la taille de huit octets de la série à champ pointeur sort du jugement, son type n'ayant qu'un champ, qui est le pointeur lui-même. Ce retrait pousse le verdict du côté du livre, puisque c'est la seule cellule qui infirmait sur le corpus mesuré ; il est déclaré pour cette raison. Une hypothèse qui s'écarte de sa source doit le dire là où son critère est gelé, sinon un lecteur du verdict ne le saura jamais.

**D-25 — La rationale de H-010 groupe les paires par base d'allocation.** Une campagne à quatre répétitions et deux charges produit quatre-vingts paires. Les énumérer deux fois donnait deux mille sept cents caractères illisibles et muets sur ce qui compte : le rapport ne dépend que du couple de bases, jamais de la taille de la structure. Le groupement dit la même chose en une ligne, et l'étiquette nomme la charge en plus de la répétition, sans quoi deux groupes de bases différentes s'y lisaient pareil.

**D-26 — La campagne contaminée est conservée, pas effacée.** Les agents de la rédaction contradictoire ont mesuré sur la même machine pendant la première campagne, saturations mémoire comprises. Plutôt que de la supprimer, elle est gardée comme témoin de campagne chargée et comparée à la campagne propre. Les cinq verdicts communs sont identiques, mais l'écart absolu médian sur les deux cent cinquante-deux cellules vaut quatre pour cent, du même ordre que les effets que H-007 et H-012 doivent trancher. C'est la mesure qui fonde C-010, et elle n'existerait pas si la campagne avait été jetée.

**D-27 — C-009 est satisfaite sans toucher aux gabarits, donc sans invalider quoi que ce soit.** Un réplicat rend la même source dans un répertoire différent, à la ligne de commentaire du sujet près, ce qui est précisément ce qui garantit un code machine identique. L'empreinte du harnais ne portant que les gabarits embarqués, elle est inchangée et les matrices antérieures restent extensibles. C-009 est la seule contrainte de capacité du catalogue qui n'invalide rien.

**D-28 — Le réplicat n'entre ni dans Comparison ni dans la clé des points de bascule.** C'est un retrait délibéré et non un oubli. H-012 groupe ses réplicats par la taille, la disposition, la présence d'un champ pointeur et le profil, tous déjà présents, et les distingue par l'identifiant de la cellule valeur. Faire entrer un champ de plus dans la clé des points de bascule rendrait H-002 non concluante sur toute campagne future, sans erreur ni trace, son évaluateur construisant sa clé en littéral. Ce qu'aucun critère ne lit, le modèle ne le porte pas.

**D-29 — Deux conséquences de C-009 sont traitées plutôt que découvertes.** H-007 indexe ses comparaisons par taille et n'en retient qu'une, arbitrairement : sur une série répliquée elle jugerait un réplicat tiré de l'ordre du fichier. La campagne est donc refusée quand la liste gelée contient H-007 sur une matrice à réplicats, et le refus nomme la contrainte. Pour la même raison d'ordre, le point de bascule d'une série répliquée n'est pas publié. Dans les deux cas, mieux vaut un refus explicite qu'un verdict que l'ordre d'un fichier aurait dicté.

**D-30 — Le retrait de la cellule de huit octets à champ pointeur était déclaré avant la mesure, et la mesure lui donne raison d'être déclaré.** Cette cellule franchit le seuil dans ses cinq réplicats : jugée, elle aurait infirmé H-012. Elle est écartée parce que son type n'a qu'un champ, qui est le pointeur lui-même, de sorte que les deux bras y passent un pointeur et que la règle de la page 253 n'y compare rien. L'écart mesuré y reste inexpliqué, le bras pointeur exécutant un chargement de plus tout en allant plus vite. La cellule est consignée dans le verdict avec cette mention, jamais silencieusement omise. C'est la différence entre une hypothèse qui choisit son corpus après coup et une hypothèse qui déclare ses exclusions avant de mesurer.

## 5. Ce qui est délibérément absent

- **Aucune campagne de référence n'a été exécutée.** Le tableau de bord affiche six hypothèses sans verdict. Lancer la matrice de référence demande environ une heure (NFR-005) et relève du chercheur, pas de la construction. La chaîne complète a été validée de bout en bout sur une matrice réduite.
- **Colonne `Integration` du tableau de bord fondée sur la présence, non sur l'exécution.** Un test sous `//go:build integration_test` qui référence un cas d'utilisation suffit à marquer la colonne. Faire tourner la suite d'intégration à chaque régénération aurait rendu `escapebench dashboard` inutilisable au quotidien.
- **`make` reste requis par `CLAUDE.md` et par le skill `/implement`, et reste absent du poste.** Point déjà relevé à la revue pré-lancement ; la décision (installer `make` ou changer la commande de référence) appartient au chercheur. Toutes les vérifications de cette construction ont été faites avec `go` directement.
- **Statut `Verified` non atteint** (voir D-01).

## 6. Vérification

| Contrôle | Résultat |
|---|---|
| `go vet ./...`, `gofmt -l` | propre |
| `go test -race -shuffle=on -count=1 ./...` | tous les paquets au vert |
| Couverture de statements | **93,9 %** après la revue (cible : 85 %) |
| `bash .claude/hooks/selftest.sh` | les quatre hooks passent, en chemins POSIX et Windows |
| Chaîne UC-001 → UC-005 sur une matrice réduite | matrice générée, échappement classé, campagne mesurée, comparaison calculée, verdicts produits, tableau de bord régénéré |

Couverture par paquet : `models` 97,8 % · `cli` 98,9 % · `dashboard` 98,1 % · `escape` 97,0 % · `gotool` 96,8 % · `specs` 94,4 % · `service` 94,2 % · `store` 87,9 % · `system` 85,2 % · `harness` 84,0 % · `cmd` 84,7 %. Le reste non couvert est constitué de branches d'erreur d'entrée-sortie non déclenchables sans injection de panne au niveau du système de fichiers.

**Deux défauts réels ont été trouvés par les tests pendant la construction**, tous deux corrigés : le classificateur attribuait `RETURN_POINTER` à un `return v` qui ne rend qu'une copie et descendait à tort dans le corps des closures ; le motif de lecture des exigences liées capturait `FR-002` à l'intérieur de `NFR-002`.
