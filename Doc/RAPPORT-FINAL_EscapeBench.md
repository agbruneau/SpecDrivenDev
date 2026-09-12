# EscapeBench — verdicts consolidés des treize hypothèses

**Révision du 2026-09-12 :** H-006 passe d'infirmée à **confirmée**. Son premier verdict était un artefact du banc, non une observation ; le défaut est corrigé et le rejeu a été mené. La section *Errata* dit ce qui s'est passé et sur quoi la correction repose. Les douze autres verdicts sont inchangés, et un harnais de non-régression les rejoue désormais sur les campagnes archivées.

Clôture du 2026-09-10. Le banc a éprouvé treize affirmations de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026). Chaque critère de réfutation a été gelé avant les mesures qui le jugent, et chaque verdict est produit par un évaluateur qui applique ce texte à la lettre.

## Les treize verdicts

| Hypothèse | Page | Ce que le livre affirme | Verdict |
|---|---|---|---|
| H-001 | 245, 253 | Copier une petite structure peut coûter moins cher que passer un pointeur | **infirmée** |
| H-002 | 253 | La valeur reste préférable de un à trois mots machine | **infirmée** |
| H-003 | 256 | Passer de la valeur au pointeur double les allocations | confirmée |
| H-004 | 254 | Un accès dispersé coûte dix à deux cents fois un accès séquentiel | **infirmée** |
| H-005 | 114 | La préallocation est six fois plus rapide et prend un cinquième de la mémoire | confirmée |
| H-006 | 238-242 | Les quatre causes d'échappement énumérées couvrent les cas | confirmée *(verdict corrigé le 2026-09-12 ; infirmée à tort le 2026-09-10, voir Errata)* |
| H-007 | 253 | Même affirmation que H-002, sur une disposition assignable aux registres | confirmée |
| H-008 | 254 | Même affirmation que H-004, sur une chaîne de pointeurs dépendante | confirmée |
| H-009 | 241 | La règle des conteneurs se généralise de la map à la tranche et à la structure | confirmée |
| H-010 | 256 | Même affirmation que H-003, sur une base d'allocation qui varie | **infirmée** |
| H-011 | 114 | Même affirmation que H-005, aux tolérances du livre | **infirmée** |
| H-012 | 253 | Même affirmation que H-007, sur un plancher de bruit apparié et mesuré | confirmée |
| H-013 | 254 | Même affirmation que H-008, sous attestation de quiétude de la machine | confirmée |

Sept infirmations, six confirmations. Les verdicts viennent de deux campagnes menées en série sur la même machine et dans la même soirée: `C-2026-09-10-11` pour douze hypothèses, 541 sujets et 1 h 08; `C-2026-09-10-12` pour H-012, 82 sujets et 11 min. Aucun sujet en échec. L'occupation médiane des cœurs hors du sujet vaut 5,8 % et 5,5 %, sous le seuil de 12 % que C-010 annonce.

Deux hypothèses ne peuvent pas cohabiter dans une même campagne, et c'est voulu. H-012 exige cinq réplicats de chaque paire; H-007 indexe ses comparaisons par taille et n'en retiendrait qu'une, tirée de l'ordre du fichier. La campagne refuse donc la combinaison au lieu de rendre un verdict que cet ordre aurait dicté.

## Les quatre infirmations qui portent

**H-004 tombe de très loin.** Le livre annonce un rapport de dix à deux cents entre un accès dispersé et un accès séquentiel. Mesuré sur des jeux de travail de 32 et 128 Mio, le rapport vaut 1,9 et 2,8. L'écart n'est pas marginal, il est d'un ordre de grandeur. La raison est que les deux parcours mesurent un débit et non une latence: leurs chargements sont indépendants et se recouvrent, si bien que le préchargeur matériel absorbe l'essentiel de la dispersion. C'est ce que H-008 corrige en mesurant une chaîne dépendante, où chaque adresse est lue à l'accès précédent, et H-008 est confirmée avec un rapport de 169,9. La même affirmation du livre est donc fausse sur le sujet de mesure qu'on lui applique d'ordinaire et vraie sur celui qui l'éprouve réellement.

**H-010 établit que le doublement n'est pas une propriété du mode de passage.** Le livre écrit qu'un simple changement de valeur en pointeur double le nombre d'allocations. Sur les 120 paires éligibles, les 60 qui allouent une charge par instance doublent, et les 60 qui en allouent deux passent de 2 à 3, de 4 à 6, de 8 à 12 et de 32 à 48. Le rapport vaut le nombre de charges plus un, divisé par le nombre de charges. Le doublement n'apparaît que dans le cas particulier où la fonction n'alloue rien d'autre que la valeur retournée. H-003, qui pose le seuil à un facteur d'au moins deux sur une base nulle, reste confirmée: c'est la formulation forte qui tombe, pas la faible.

**H-006 tombe sur un cas que le livre ne prévoyait pas.** Sur 380 cellules qui échappent, 120 le font pour une raison étrangère aux quatre causes énumérées: la charge allouée par le profil de retour s'échappe alors que la structure mesurée, elle, reste sur la pile. Le catalogue notait déjà que le livre ne revendique pas l'exhaustivité, parlant d'un des cas les plus courants. Cette infirmation le confirme par la mesure.

> **Erratum du 2026-09-12 — ce paragraphe est faux, et le rejeu l'a établi.** L'audit du code a démontré que les 120 cellules comptées « hors des quatre causes » n'ont jamais été classées : le classificateur d'échappement rendait `OTHER` dès que le compilateur nomme l'expression d'allocation plutôt qu'une variable. La sortie réelle du compilateur pour ces cellules est « `new(payload) escapes to heap` », que `escapedIdentifier` ne savait pas lire ; `classifyDiagnostic` rendait alors `OTHER` sans consulter l'arbre syntaxique. Or le corps de `produceValueAlloc` affecte cette allocation à `p` puis exécute `return t, p` : la charge échappe **par retour de pointeur**, la première des quatre causes du livre, et non par une cause étrangère. Le verdict `REFUTED` publié pour H-006 est un artefact du classificateur, pas une observation. Le défaut est corrigé, et le rejeu mené le 2026-09-12 a rendu **`CONFIRMED`** : les 380 cellules qui échappent se classent toutes dans les quatre causes du livre, `OTHER` valant zéro contre 120 auparavant. Voir `audit.md` (constat A-072) et `Doc/DECISION.md` (D-39, D-48).

**H-011 tombe de peu, et pas là où on l'attendrait.** La préallocation tient deux des trois chiffres de la page 114: la mémoire vaut un cinquième et le compte d'allocations tombe de 27 à exactement un. C'est le gain en temps qui manque, à 4,45 fois pour un plancher de 4,8. H-005, qui accorde une tolérance deux fois plus large sur la même mesure, reste confirmée. La différence entre les deux verdicts est la largeur de la marge, pas la mesure.

## Errata

**2026-09-12 — H-006 passe d'infirmée à confirmée.** Le verdict `REFUTED` publié le 2026-09-10 était un artefact du classificateur d'échappement et non une observation : les 120 cellules comptées « hors des quatre causes du livre » échappent par retour de pointeur, la première des quatre. C'est le résultat le plus important de l'audit — une affirmation du livre avait été déclarée fausse sur la foi d'un défaut du banc qui la mesure.

**Ce que le rejeu a produit.** La matrice `M-b44a93baae51` a été régénérée depuis les paramètres consignés à `LANCEMENT.md` : même identifiant, 532 cellules, même empreinte de harnais `551ce66b…` que les campagnes publiées. UC-002 y a été rejoué sous `go1.27.0`, la version de Go des campagnes. Le décompte passe de `RETURN_POINTER 140 · OTHER 120` à `RETURN_POINTER 260 · OTHER 0` ; `NONE`, `CLOSURE_CAPTURE`, `CHANNEL_SEND` et `CONTAINER_STORE` sont inchangés au cas près. Comparé cellule par cellule au rapport archivé, l'écart est d'**une seule transition, `OTHER` vers `RETURN_POINTER`, sur exactement 120 cellules**, toutes du profil `RETURNED_ALLOCATING` en mode valeur ; les 412 autres gardent leur catégorie, et le drapeau d'échappement comme le statut sont identiques sur les 532.

UC-005 a ensuite été rejoué sur les deux campagnes qui portaient un H-006 infirmé, `C-2026-09-10-7` et `C-2026-09-10-11`. Sur chacune, **une seule hypothèse change — H-006, de `REFUTED` à `CONFIRMED` — et les onze autres sont strictement identiques, verdict et motif**. Le tableau de bord a été régénéré par le binaire.

**La réserve qu'il faut lire avec ce verdict.** Les campagnes ont été mesurées sur `windows/amd64` ; le rejeu a eu lieu sur `linux/amd64`, la version de Go étant la même. Le nouveau rapport d'échappement porte donc une provenance que la règle de sélection de UC-005 n'apparie pas à celle des campagnes, et il a été désigné explicitement par `--escape` ; les fichiers de verdicts le citent. Ce qui autorise cette désignation n'est pas une commodité mais une mesure : sur 532 cellules, 412 rendent la même catégorie sous `windows/go1.27.0` et sous `linux/go1.27.0`, et les 120 qui diffèrent sont exactement celles que le correctif vise. Le système d'exploitation est donc sans effet sur ce corpus. Un rejeu sur le poste de référence lèverait la réserve tout à fait ; rien dans les données n'incite à en attendre autre chose.

**Ce qui n'a pas été rejoué.** La matrice `M-8f03757ac206` porte elle aussi 80 cellules `OTHER`, mais ses paramètres ne sont consignés nulle part et n'ont pas été retrouvés. Aucune des campagnes qui l'emploient ne gèle H-006 : le verdict publié n'en dépend pas.

## Ce que les confirmations valent, et ce qu'elles ne valent pas

Une confirmation n'est pas une preuve. Trois d'entre elles portent une réserve écrite au catalogue avant les mesures.

**H-007 et H-012** lisent la règle de la page 253 sur une disposition à champs nommés. H-012 l'abandonne sans exception là où le livre écrit « typically », parce que la lecture tolérante rend l'infirmation arithmétiquement inatteignable: trois mots nommés tiennent dans les registres d'argument sur les deux architectures du catalogue, le bras valeur y gagne structurellement, et au plus une taille peut donc basculer. H-012 écarte aussi du jugement la taille de huit octets à champ pointeur, dont le type n'a qu'un champ qui est le pointeur lui-même. Cette cellule franchit le seuil dans ses cinq réplicats: jugée, elle aurait infirmé. Le retrait était déclaré avant la mesure, précisément parce qu'il pousse le verdict du côté du livre.

**H-008** est confirmée par un corpus qui ne pouvait pas la contredire sur ce matériel. Au repos, la latence non résidente vaut 137 nanosecondes contre un seuil d'infirmation de cent, et le rapport 169,9 contre un plafond de deux cents. Aucune des deux branches n'est atteignable. Il faudrait un boîtier dont la mémoire principale sert un accès dépendant sous cent nanosecondes.

**H-013** partage cette réserve mais ajoute ce qui manquait à H-008: son verdict est adossé à une mesure de la charge de la machine extérieure aux sondes. Ce n'est pas une précaution théorique. Une campagne de cette soirée a été mesurée sans que je m'en aperçoive avec cinq cœurs occupés par des processus oubliés d'une contre-épreuve précédente. Les onze autres hypothèses ont rendu leur verdict sans broncher; H-013 a rendu non concluant et nommé la cause. La campagne a été refaite au propre. C'est la seule preuve qui vaille pour une garde: qu'elle morde contre son auteur.

## Ce que le banc a appris sur lui-même

Trois défauts de construction ont été trouvés par des revues contradictoires, chacun corrigé par une hypothèse successeur plutôt que par une retouche d'un critère gelé.

Le premier est que H-010 ne pouvait pas être infirmée avant qu'on rende mesurable le nombre de charges allouées par instance: le doublement était une identité du gabarit, pas une propriété du mode de passage. Le second est que le plancher de bruit de H-007 était mesuré sur un témoin dont les deux bras exécutent le même code machine, vérifié par comparaison des sources générées et par désassemblage; H-012 le mesure sur cinq réplicats de la paire réelle, séparés chacun par une passe complète de la matrice, et la dispersion relevée tombe d'un facteur six à huit. Le troisième est que la branche du rapport de H-008 est indissociable d'un artefact de contention; H-013 y répond par l'attestation de quiétude.

Un quatrième défaut a été trouvé et non corrigé, faute de pouvoir l'être sans une capacité que le banc n'a pas: le classificateur d'échappement attribue un stockage dans un conteneur à une écriture d'adresse dans le champ d'une structure ensuite retournée, parce que ses alias ne traversent pas les champs. Aucun verdict n'en dépend aujourd'hui, et un test épingle le comportement pour que son évolution ne passe pas inaperçue.

## État du dépôt à la clôture

| Contrôle | Résultat |
|---|---|
| Cas d'utilisation | les cinq à `Deployed` |
| Hypothèses au catalogue | treize, toutes avec un évaluateur et un verdict |
| Contraintes de capacité | C-008, C-009 et C-010 satisfaites |
| Campagnes archivées | six, dont une contaminée conservée comme témoin |
| Analyse statique et mise en forme | propres |
| Suite de tests | au vert sous détecteur de course et ordre aléatoire |
| Couverture de statements | 93,6 % |
| Contrôle des hooks et des spécifications | sans constatation |

La matrice de référence n'a jamais bougé: `M-823d8b5af441`, 220 cellules et 10 sondes, et les six verdicts de la première campagne sont inchangés après trois extensions du modèle. Les trois dimensions ajoutées gardent leur valeur par défaut hors de la représentation canonique, de sorte qu'une demande antérieure produit toujours le même identifiant.

## Ce qui reste ouvert

- **Un boîtier plus rapide.** Sur la machine du catalogue, ni H-008 ni H-013 ne peuvent être infirmées: leur latence non résidente au repos dépasse de vingt-sept à trente-sept pour cent le seuil du livre. Une architecture arm64, que la contrainte d'architectures souhaite déjà, trancherait.
- **Le désassemblage des deux bras d'une paire.** C'est la capacité qui expliquerait pourquoi, à huit octets avec champ pointeur, le bras pointeur exécute un chargement de plus et va pourtant plus vite. Sans elle, cette cellule reste consignée et non jugée.
- **La contention mémoire elle-même.** L'attestation de quiétude mesure l'occupation processeur, indicateur nécessaire et non suffisant. Borner directement la bande passante demanderait les compteurs de performance du processeur, hors de la bibliothèque standard et donc hors des contraintes de dépendances du banc.
