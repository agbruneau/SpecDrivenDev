# Campagnes C-2026-09-10-5 et 6 — première épreuve de H-013

Matrice `M-abb3d708d0e1`, deux cellules et trois sondes de poursuite de pointeurs, une par bande de résidence. Cinq sujets, 20 répétitions chacun, 46 secondes.

Deux campagnes ont tourné sur la même matrice, à quelques minutes d'intervalle. La première sur une machine laissée au repos. La seconde avec quatre processus de charge lancés exprès. Elles servent ensemble de démonstration de C-010.

## Verdicts

| Campagne | État de la machine | Verdict de H-013 |
|---|---|---|
| C-2026-09-10-5 | au repos | **confirmée** |
| C-2026-09-10-6 | quatre cœurs occupés | **non concluante** |

## La campagne au repos

| Bande | Paramètre | Latence par accès | Occupation relevée |
|---|---|---|---|
| résidente | 16 Kio | 0,78 ns | 6,1 % |
| intermédiaire | 256 Kio | 2,99 ns | 6,1 % |
| non résidente | 256 Mio | 134,45 ns | 6,2 % |

Le rapport vaut 171,6 fois. La latence non résidente dépasse les cent nanosecondes qu'annonce la page 254, le rapport reste sous le plafond de deux cents, et les trois fractions d'occupation sont bien sous le seuil de douze pour cent que C-010 annonce. H-013 est confirmée.

Le témoin d'ordonnancement fonctionne: les trois bandes se rangent dans l'ordre strict attendu, ce qui exclut un banc muet dont les trois sondes se vaudraient.

## La campagne sous charge, qui est la vraie démonstration

Quatre processus occupant chacun un cœur ont tourné pendant toute la mesure. L'occupation relevée passe à vingt-trois pour cent, ce qui correspond exactement au bruit de fond du poste augmenté de quatre vingt-quatrièmes. Le verdict devient non concluant, et le message nomme la sonde, la fraction et le seuil.

Sans C-010, cette campagne aurait rendu un verdict. C'est précisément le défaut que la revue avait établi sur H-008: la contre-épreuve y avait produit trois fausses infirmations entre 166 et 172 nanosecondes, pour une valeur de repos de 133, sans qu'aucun seuil sur les sondes elles-mêmes ne puisse les distinguer d'une hiérarchie mémoire lente.

## Comment la fraction est calculée, et pourquoi l'arbre de processus compte

La fenêtre de chaque mesure est encadrée de deux relevés des temps processeur de la machine, tous cœurs confondus. Le travail de la campagne en est retranché, puis le reste est rapporté à la capacité de la machine sur la même fenêtre.

Le piège est le mot « campagne ». Un `go test` compile le sujet, lie le binaire, puis l'exécute, chaque étape dans un processus enfant. Retrancher le seul temps du processus direct laissait tout ce travail légitime dans la fraction, qui montait alors de dix à douze pour cent sur une campagne parfaitement saine. La garde aurait refusé toutes les campagnes. Windows agrège l'arbre par un Job Object, dont le compteur inclut les processus déjà terminés; les autres plateformes se rabattent sur le processus et ses enfants attendus. Avec cette agrégation, la même campagne saine relève de six à huit pour cent.

Un second piège vaut d'être noté parce qu'il inverserait le verdict: sur Windows, le temps noyau que rend l'interface système inclut le temps d'inactivité. Le confondre ferait passer une machine chargée pour une machine au repos. La fonction qui en dérive le relevé est pure et éprouvée sur des valeurs construites à la main.

## Ce que la garde détecte, et ce qu'elle ne détecte pas

Le seuil de douze pour cent a été fixé avant toute mesure de H-013, sur deux relevés faits le même jour: au repos et sans campagne, l'occupation de ce poste vaut de 4,2 à 7,5 pour cent; pendant une campagne, une fois l'arbre retranché, de 6,1 à 8,4; un cœur occupé de plus en ajoute 4,2, la machine en comptant vingt-quatre.

La garde refuse donc de façon fiable deux cœurs occupés et plus, et de façon intermittente un seul, le bruit de fond d'un poste de travail ordinaire étant du même ordre. Sur une machine dédiée, dont le repos vaut une fraction de pour cent, le même seuil devient bien plus sensible.

Surtout, l'occupation processeur est un indicateur nécessaire et non suffisant de la contention mémoire. Un processus qui sature la bande passante occupe un cœur à plein et se voit donc dans la fraction, mais la fraction ne mesure pas la bande passante elle-même. Une hypothèse qui voudrait la borner directement demanderait les compteurs de performance du processeur, hors de la bibliothèque standard et donc hors des contraintes de dépendances du banc.

## Ce que ce verdict vaut

H-013 est confirmée sur une machine dont la quiétude est attestée. Deux réserves l'accompagnent, toutes deux écrites au catalogue avant la mesure.

La première est que sur ce matériel, aucune des deux branches d'infirmation n'est atteignable au repos: la latence non résidente vaut 134 nanosecondes contre un seuil de cent, et le rapport 171,6 contre un plafond de deux cents. Il faudrait un boîtier dont la mémoire principale sert un accès dépendant sous cent nanosecondes pour que la première branche redevienne en jeu.

La seconde est que la confirmation porte sur une lecture des deux ancres chiffrées de la page 254, un succès de cache sous dix nanosecondes et un défaut au-delà de cent, et non sur la plage de dix à deux cents fois prise isolément.

Le verdict de H-008 sur les mêmes sondes reste ce qu'il était: rendu sans garantie que la machine ait été au repos. C'est exactement ce que H-013 existe pour corriger.
