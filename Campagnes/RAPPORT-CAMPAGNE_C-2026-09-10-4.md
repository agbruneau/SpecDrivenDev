# Campagne C-2026-09-10-4 — première épreuve de H-012

Matrice `M-57477f022103`, 80 cellules et 2 sondes, 82 sujets mesurés sans un seul échec, 20 répétitions par sujet, 11 minutes 16. Trois critères gelés à la création: H-005, H-011 et H-012.

C'est la première campagne à porter des réplicats. Quatre tailles, deux séries, deux modes de passage et cinq réplicats, en disposition à champs nommés et en profil local uniquement.

## Verdicts

| Hypothèse | Verdict |
|---|---|
| H-005 | confirmée |
| H-011 | infirmée |
| H-012 | confirmée |

## H-012, cellule par cellule

Le plancher de bruit est l'étendue des trois écarts de rang médian des cinq réplicats. La barrière relative vaut un dixième du plus grand des deux bras. Le seuil est le plus grand des deux.

| Cellule | Plancher | Barrière | Seuil | Réplicats franchissant |
|---|---|---|---|---|
| 8 octets sans champ pointeur | 0,0121 | 0,0802 | 0,0802 | 0 sur 5 |
| 16 octets sans champ pointeur | 0,0337 | 0,0902 | 0,0902 | 0 sur 5 |
| 24 octets sans champ pointeur | 0,0032 | 0,0771 | 0,0771 | 0 sur 5 |
| 16 octets avec champ pointeur | 0,0190 | 0,0834 | 0,0834 | 0 sur 5 |
| 24 octets avec champ pointeur | 0,0145 | 0,0783 | 0,0783 | 0 sur 5 |
| 8 octets avec champ pointeur, **écartée du jugement** | 0,0359 | 0,0799 | 0,0799 | **5 sur 5** |

Les cinq cellules jugées sont résolues: le plancher mesuré y reste bien sous la barrière postulée, donc c'est la barrière qui fixe le seuil et non le bruit. Aucune ne donne l'avantage au pointeur. H-012 est confirmée.

## Trois choses que cette campagne établit

**Les réplicats font ce qu'on attendait d'eux.** La contre-épreuve du même jour, faite sur trois exécutions consécutives, relevait à seize octets des planchers de 0,21 à 0,26 nanoseconde, très au-dessus de la barrière de 0,096. La campagne, avec cinq réplicats séparés chacun par une passe complète de la matrice, relève 0,034. La dispersion tombe d'un facteur six ou huit. Ce n'est pas le nombre de réplicats qui produit cela mais leur séparation: cinq processus espacés de plusieurs minutes échantillonnent une classe de bruit que cinq exécutions d'affilée ne voient pas.

**La confirmation doit se lire avec la limite déclarée.** La cellule de huit octets à champ pointeur franchit le seuil dans ses cinq réplicats. Si elle était jugée, H-012 serait infirmée. Elle est écartée parce que son type n'a qu'un champ et que ce champ est un pointeur: les deux bras y passent un pointeur et la règle de la page 253 n'y compare rien. Ce retrait était annoncé dans la ligne gelée et dans la section des limites, avant la mesure, précisément parce qu'il pousse le verdict du côté du livre. Il reste que le banc mesure là un écart net et inexpliqué: le bras pointeur exécute un chargement de plus que le bras valeur et va pourtant plus vite de deux dixièmes de nanoseconde sur huit dixièmes. Consigner le désassemblage des deux bras trancherait, et aucune contrainte ne le demande.

**Le critère ne bascule pas sur un écart sous le bruit.** À seize octets sans champ pointeur, les cinq intervalles hauts sont négatifs, de moins 0,006 à moins 0,031: le pointeur y devance légèrement la valeur, dans les cinq réplicats. Aucun n'atteint le seuil de 0,090. C'est exactement ce que H-007 ne savait pas faire, son plancher étant mesuré sur un témoin dont les deux bras exécutaient le même code machine.

## H-011, confirmée dans son infirmation

Le gain en temps de la préallocation vaut 4,31 fois ici, contre 4,04 sur la campagne précédente, pour un plancher de 4,8. Deux campagnes indépendantes, sur deux matrices différentes, donnent le même verdict. La mémoire et le compte d'allocations tiennent les chiffres de la page 114 dans les deux cas. H-005, qui accorde une tolérance deux fois plus large sur la même mesure, reste confirmée.

## Ce qui reste

- **C-010 n'est pas satisfaite**, donc H-013 reste sans évaluateur et rend non concluant. Il y faut aussi une machine dont la mémoire principale sert un accès dépendant sous cent nanosecondes.
- **H-007 n'est pas évaluable sur une matrice à réplicats** et la campagne la refuse. Son verdict acquis sur les campagnes précédentes reste valide.
- **Le point de bascule d'une série répliquée n'est pas publié**, le balayage descendant dépendant alors de l'ordre du fichier.
