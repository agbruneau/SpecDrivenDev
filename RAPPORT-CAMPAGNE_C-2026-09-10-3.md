# Campagne C-2026-09-10-3 — première épreuve de H-007 à H-013

Matrice `M-8f03757ac206`, 252 cellules et 4 sondes, 256 sujets mesurés sans un seul échec, 20 répétitions par sujet, 32 minutes 34. Sept critères gelés à la création de la campagne.

Deux campagnes ont tourné sur cette matrice. La première, `C-2026-09-10-2`, a mesuré pendant que les agents de la rédaction contradictoire compilaient et saturaient la mémoire sur la même machine. Elle est conservée comme témoin de campagne chargée. La seconde, ci-dessous, a tourné seule.

## Verdicts

| Hypothèse | Verdict | Ce que la mesure dit |
|---|---|---|
| H-007 | confirmée | Aucune série ne donne l'avantage au pointeur sur plus d'une des trois petites tailles |
| H-008 | confirmée | Latence non résidente 132,85 ns, rapport 169,5 fois, dans la plage annoncée |
| H-009 | confirmée | Les trois conteneurs font échapper la locale, à toutes les tailles |
| H-010 | **infirmée** | La moitié des paires ne doublent pas |
| H-011 | **infirmée** | Gain en temps de 4,04 fois, sous le plancher de 4,8 |
| H-012 | non concluante | Aucun évaluateur, C-009 non satisfaite |
| H-013 | non concluante | Aucun évaluateur, C-010 non satisfaite |

## H-010, le résultat de cette campagne

Le livre écrit qu'un simple changement de passage par valeur en passage par pointeur double le nombre d'allocations. Jusqu'à cette campagne, le banc ne pouvait que le confirmer: son gabarit allouait une charge par instance côté valeur et cette même charge plus la valeur retournée côté pointeur, donc un rapport de deux par arithmétique et non par propriété du mode de passage.

La dimension du nombre de charges par instance, ajoutée après la revue du même jour, change cela. Sur les quatre-vingts paires éligibles:

| Charges par instance | Paires | Allocations valeur → pointeur | Rapport |
|---|---|---|---|
| une | 40 | 1 → 2, 2 → 4, 4 → 8, 16 → 32 | 2,00 |
| deux | 40 | 2 → 3, 4 → 6, 8 → 12, 32 → 48 | 1,50 |

Le doublement est une propriété du nombre de charges que la fonction alloue, jamais du mode de passage. L'affirmation de la page 256 ne vaut que dans le cas particulier où la fonction n'alloue rien d'autre que la valeur retournée. C'est une infirmation nette, obtenue sur un corpus qui pouvait aussi bien confirmer.

## H-011, une infirmation de peu

La préallocation tient deux des trois chiffres de la page 114. La mémoire vaut un cinquième de la version sans préallocation, mesurée à 5,11 fois, dans l'intervalle que le critère accorde. Le compte d'allocations tombe de 27 à exactement 1, ce que le livre annonce. Le gain en temps, lui, vaut 4,04 fois pour un plancher de 4,8, soit 16 pour cent sous la tolérance que le critère tirait du mot « about ». H-005, qui posait une tolérance deux fois plus large sur la même mesure, reste confirmée. La différence entre les deux verdicts est la largeur de la marge, pas la mesure.

## H-007, et pourquoi H-012 existe

La série des structures à champs nommés en profil local, telle que la campagne propre la mesure:

| Série | Taille | Écart pointeur moins valeur | Intervalle | Médiane valeur |
|---|---|---|---|---|
| sans champ pointeur | 8 | +0,058 | [+0,017, +0,104] | 0,831 |
| sans champ pointeur | 16 | −0,075 | [−0,097, −0,042] | 0,879 |
| sans champ pointeur | 24 | +0,382 | [+0,349, +0,405] | 0,430 |
| avec champ pointeur | 8 | −0,221 | [−0,261, −0,158] | 0,822 |
| avec champ pointeur | 16 | +0,106 | [+0,085, +0,153] | 0,825 |
| avec champ pointeur | 24 | +0,324 | [+0,305, +0,333] | 0,424 |

Les six écarts sont significatifs. À vingt-quatre octets la valeur gagne largement dans les deux séries: trois mots nommés tiennent dans les registres d'argument entiers et le bras pointeur paie une indirection. C'est un fait de convention d'appel, pas un aléa.

Le témoin de sensibilité fonctionne. À cent vingt-huit octets le pointeur gagne d'une nanoseconde, à mille vingt-quatre octets de six et demie, dans les deux séries. Le banc voit donc un écart réel quand il y en a un.

Reste que le plancher de bruit de H-007 vaut 0,044 et 0,037 nanoseconde sur cette campagne, mesuré sur un témoin dont les deux bras exécutent le même code machine. Les écarts à trancher aux petites tailles valent de 0,058 à 0,221. La marge est mince, et c'est exactement ce que H-012 corrige en appariant le plancher à la taille et en le mesurant sur des réplicats de la paire réelle.

## Ce que la comparaison des deux campagnes apprend

Les cinq verdicts communs sont identiques d'une campagne à l'autre. La contamination n'a donc renversé aucune conclusion. Elle a en revanche déplacé les chiffres:

| Grandeur | Campagne chargée | Campagne propre | Écart |
|---|---|---|---|
| Sonde non résidente | 136,70 ns | 132,85 ns | −2,8 % |
| Sonde résidente | 0,786 ns | 0,784 ns | −0,3 % |
| Ajout sans préallocation | 234 686 ns | 229 378 ns | −2,3 % |
| Écart absolu médian sur les 252 cellules | | | 4,04 % |
| Neuvième décile | | | 9,19 % |
| Maximum | | | 51,98 % |

Le sens de l'écart n'est pas systématique: la campagne propre est plus lente sur quatre-vingt-six cellules sur deux cent cinquante-deux. Ce n'est donc pas une dégradation uniforme mais une gigue, et son ampleur médiane de quatre pour cent est du même ordre que les effets que H-007 et H-012 doivent trancher, où soixante millièmes de nanoseconde sur neuf dixièmes valent six pour cent. Une campagne dont la quiétude n'est pas attestée ne permet donc pas de trancher à cette échelle, ce qui est l'argument de C-010.

## Ce qui reste

- **C-009 et C-010 ne sont pas satisfaites.** H-012 et H-013 rendent non concluant tant qu'elles ne le sont pas. Le mécanisme est celui de C-008, qui a nommé les capacités de H-007 à H-010 avant qu'elles soient construites.
- **Le verdict de H-008 se lit avec réserve.** Il est rendu par un corpus qui ne pouvait pas le contredire: la latence non résidente vaut 132,85 nanosecondes au repos contre un seuil d'infirmation de cent, et le rapport 169,5 contre un plafond de deux cents. Aucune des deux branches n'est atteignable sur cette machine. C'est la lecture qu'impose la section des limites du catalogue.
- **La matrice de référence n'est pas touchée.** `M-823d8b5af441` compte toujours 220 cellules et 10 sondes, et les six verdicts de `C-2026-09-10-1` sont inchangés.
