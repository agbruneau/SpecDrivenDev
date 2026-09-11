# Campagne de référence C-2026-09-10-1 — résultats et lecture

**Date :** 2026-09-10 · **Matrice :** `M-823d8b5af441` · **Banc :** EscapeBench (P1)
**Provenance :** go1.27.0, windows/amd64, Intel Core Ultra 9 275HX, L3 36 Mo

## 1. Résultat

Les six hypothèses ont un verdict, produit par le seul critère gelé avant la mesure. Ces verdicts sont enregistrés et ne changent pas. Mais trois d'entre eux ne disent pas ce qu'ils ont l'air de dire, et un audit contradictoire de 172 agents, dont j'ai vérifié chaque affirmation chiffrée, en donne la raison.

| Hypothèse | Verdict | Ce que la campagne établit réellement |
|---|---|---|
| H-001 | REFUTED | Repose sur une paire au niveau du bruit et une paire qui est un artefact de disposition. Ne tranche pas l'affirmation du livre. |
| H-002 | REFUTED | Le point de bascule mesuré est celui de la sortie du passage en registres de Go, pas celui de la taille. Une structure idiomatique ne bascule pas. |
| H-003 | CONFIRMED | Vrai par construction du harnais : 0 → 1 allocation sur les 22 paires, aucun facteur 2 jamais calculé. |
| H-004 | REFUTED | La sonde mesure du débit, le livre parle de latence. De plus le jeu de travail de 32 Mo tient dans le L3 de la machine. |
| H-005 | CONFIRMED | **Le seul résultat propre.** Le volet mémoire du livre est reproduit à 2 % près ; son volet temps ne l'est pas. |
| H-006 | CONFIRMED | Aucune cause hors-livre dans un corpus qui, par construction, ne pouvait pas en produire. Un contre-exemple existe et a été démontré. |

En une phrase : le banc fonctionne, la métrologie est saine, la discipline de gel a tenu, et c'est la conception de trois sujets de mesure sur six qui limite la portée des verdicts.

## 2. Ce que la campagne a produit

| | |
|---|---|
| Génération de la matrice, 230 paquets Go compilés | 23 s |
| Classification d'échappement, 220 cellules | 22 s |
| Campagne de mesure, 230 sujets | 29 min 29 s |
| Sujets mesurés / en échec | 230 / 0 |
| Répétitions par sujet | 20, `-benchtime=250ms`, `-cpu=1`, un processus `go test` par sujet |
| Durée contre la cible NFR-005 | 29 min contre 60 |

Le décompte de la matrice est exactement celui de BR-001-4 : 22 TypeSpec, 220 Cell, 10 Probe. Sur les 220 cellules, 110 échappent : 22 par retour de pointeur, 44 par capture de closure, 22 par envoi sur canal, 22 par stockage dans une map. Aucune erreur de compilation.

**NFR-002 est soldée.** Une seconde classification sur la même toolchain donne zéro cellule divergente sur 220. C'était la seule exigence non fonctionnelle jamais exercée.

## 3. Les trois défauts de construction, avec leurs contre-épreuves

### 3.1 H-002 mesure la frontière du passage en registres, pas celle de la taille

Le générateur donne à tout type la forme `Tag uint64` suivi de `Fill [n-1]uint64`. Or la convention d'appel de Go ne passe en registres qu'un type dont tous les champs le sont, et **un tableau de longueur supérieure à 1 ne l'est pas**. À 8 et 16 octets le remplissage vaut `[0]` et `[1]`, donc registres. À partir de 24 octets il vaut `[2]`, donc mémoire. La falaise du bras valeur tombe exactement là.

Contre-épreuve, mêmes drapeaux, 30 répétitions, médianes en ns/op :

| Disposition du type | Taille | Valeur | Pointeur | Gagnant |
|---|---|---|---|---|
| `Tag uint64; Fill [1]uint64` | 16 o | 0,808 | 0,783 | égalité |
| `Tag uint64; Fill [2]uint64` | 24 o | 1,056 | 0,495 | pointeur |
| `F0, F1, F2 uint64` | 24 o | 0,442 | 0,741 | **valeur** |
| `F0, F1, F2, F3 uint64` | 32 o | 0,487 | 0,770 | **valeur** |
| `Tag uint64; Fill [3]uint64` | 32 o | 1,215 | 0,479 | pointeur |

À taille identique, le verdict s'inverse selon la seule disposition des champs. Avec une structure écrite comme on en écrit, le passage par valeur reste gagnant à 32 octets, au-delà des trois mots du livre. La matrice ne contient qu'une forme de structure, donc H-002 a éprouvé cette forme, pas la taille.

### 3.2 H-004 mesure du débit, sur un jeu de travail qui tient en cache

La sonde de parcours dispersé émet des chargements **indépendants** : elle lit `ptrs[j].V` en avançant séquentiellement dans une tranche de pointeurs. Le processeur garde des dizaines d'accès en vol. Le livre, lui, oppose un succès de cache à un défaut de cache, c'est-à-dire deux latences.

Temps par élément de 64 octets :

| Jeu de travail | Séquentiel | Dispersé | Rapport |
|---|---|---|---|
| 256 Ko | 0,333 ns | 0,431 ns | ×1,29 |
| 4 Mo | 0,734 ns | 0,973 ns | ×1,33 |
| 32 Mo | 1,924 ns | 3,537 ns | ×1,84 |
| 128 Mo | 2,558 ns | 6,048 ns | ×2,36 |

Deux choses se lisent ici. D'abord 3,5 ns par accès prétendument manquant, quand une latence DRAM vaut 60 à 100 ns : une vingtaine d'accès sont donc servis en parallèle, ce qui est la signature d'une mesure de débit. Ensuite le rapport croît doucement au lieu de sauter d'un ordre de grandeur au franchissement du cache, ce qu'aucune mesure de latence ne ferait.

S'y ajoute une erreur de prémisse. Le critère nomme son seuil « au-delà de tout cache L2 courant » et le fixe à 32 Mo. **Le L3 de la machine de mesure fait 36 Mo.** Le point à 32 Mo est donc résident en cache. Seul le point à 128 Mo touche vraiment la mémoire, et il donne ×2,36.

Le verdict REFUTED porte sur l'opérationnalisation retenue, pas sur l'affirmation de la page 254.

### 3.3 H-006 a été confirmée par un corpus qui ne pouvait pas la réfuter

Les cinq profils de durée de vie de la matrice sont les quatre causes du livre plus le profil local. Les tailles sont bornées à 4096 octets par BR-001-4. Le critère nomme lui-même trois contre-exemples possibles, dont « taille excédant la limite de pile » : aucun n'est atteignable dans ce corpus.

Contre-épreuve : une structure de 131 080 octets, en profil local et mode pointeur — un profil qui ne produit aucun échappement à toute taille de la matrice — fait écrire au compilateur `moved to heap: t`. Le classificateur du banc la range en `OTHER`, hors du livre. C'est exactement le contre-exemple que le critère nomme, obtenu en dépassant la seule borne de taille.

La bonne nouvelle est que **le classificateur n'est pas circulaire** : il ne déduit rien du profil de la cellule, il lit la sortie du compilateur et l'usage syntaxique de la variable, et il produit bien `OTHER` quand la cause sort de la liste. L'instrument est juste ; c'est l'échantillon qui était borné.

## 4. La marge de H-001

Le critère exige deux tailles réfutantes parmi les trois éligibles. Il en obtient exactement deux, sans aucune marge.

- **16 octets** : écart de −0,041 ns, borne haute de l'intervalle à −0,0097 ns, soit trois centièmes de cycle. Les deux distributions de vingt répétitions se recouvrent presque entièrement.
- **24 octets** : écart de −0,689 ns, distributions disjointes. Solide, mais c'est précisément la paire que la section 3.1 identifie comme un artefact de disposition.

Une mesure du plancher de bruit rend la première paire indéfendable. Le bras pointeur exécute le même travail à toutes les tailles, puisqu'il ne transmet qu'un mot. Sa médiane va pourtant de 0,4297 à 0,8392 ns selon la taille du type, **un facteur 1,95 sans aucun lien avec la taille**. Ce décalage d'environ 0,4 ns entre binaires vaut dix fois l'écart qui fait basculer H-001.

Le bootstrap ne peut pas le voir : il rééchantillonne les vingt répétitions d'un même sujet, issues d'un seul binaire dans un seul processus. Tout ce qui sépare systématiquement les deux bras — disposition du code, alignement, cœur d'exécution — a un effectif de un et ne contribue pas à l'intervalle. La machine est de surcroît hybride, avec cœurs performance et cœurs efficacité, sans épinglage d'affinité et sans trace de ce choix dans la provenance.

## 5. Ce que la campagne établit solidement

- **H-005, volet mémoire.** La préallocation divise la mémoire par 5,1 quand le livre annonce un cinquième : reproduit à 2 % près. Les allocations passent de 27 à 1, quand le livre écrit 28 à 1. En revanche le gain en temps vaut ×4,4 là où le livre annonce environ ×6, soit 37 % d'écart. Le seuil gelé, à la moitié, est trois fois plus lâche que l'affirmation citée : « CONFIRMED » ne veut pas dire « le ×6 est vérifié ».
- **Métrologie saine.** Dispersion intra-sujet médiane de 2,3 %, quatre-vingt-dixième centile à 4,7 %. Aucune dérive thermique sur les vingt-neuf minutes : la médiane des cinq dernières répétitions est inférieure de 0,36 % à celle des cinq premières, et plus de sujets accélèrent qu'ils ne ralentissent.
- **Reproductibilité des verdicts d'échappement.** Zéro divergence sur 220 cellules entre deux exécutions.
- **La discipline de gel a tenu.** Les critères ont été figés à la création de la campagne et n'ont pas bougé ; c'est ce qui rend ces verdicts opposables, y compris ceux qui me déplaisent.

## 6. Ce que le critère ne regarde pas

Les évaluateurs ne consomment qu'une partie des données mesurées, et le reste va parfois dans l'autre sens.

- Sur les **110 paires** comparées, **80 donnent l'avantage à la valeur**, dont 74 significativement ; seules 30 le donnent au pointeur. H-001 et H-002 sont réfutées sur la tranche étroite du profil local.
- Les **88 paires hors profil local** sont mesurées, complètes, et n'entrent dans aucun évaluateur.
- **Deux points de bascule sur dix séries** ont été observés. Les huit autres ne basculent jamais jusqu'à 4096 octets. Ni le verdict ni le tableau de bord ne le disent.
- Les **quatre sondes de parcours sous le seuil** sont mesurées et ignorées, alors que celle de 256 Ko donne le témoin qui tranche la conception de la sonde : un rapport de ×1,29 sur un jeu entièrement résident mesure le surcoût d'indirection pur.

## 7. Hypothèses successeurs — écrites au catalogue

Les critères de H-001 à H-006 sont gelés et le resteront. Ce que la campagne apprend est écrit en cinq hypothèses nouvelles dans `docs/requirements.md`, et la contrainte `C-008` y nomme les capacités de mesure qu'elles exigent. La campagne écoulée a été relue après cet ajout : son empreinte gelée valide toujours et ses six verdicts sont inchangés, `Campaign.hypothesesDigest` ne portant que sur les hypothèses qu'elle a elle-même retenues.

| Nouvelle | Objet | Ce qu'elle corrige | Mesurable |
|---|---|---|---|
| H-007 | Point de bascule sur des structures à champs nommés, passables en registres, avec plancher de bruit mesuré par témoin nul | Sépare l'effet de taille de l'effet de disposition (§3.1) | après C-008 |
| H-008 | Latence par accès sur chaîne de pointeurs dépendante, bandes définies par les tailles de cache relevées de la machine | Mesure une latence, sur un jeu réellement hors cache (§3.2) | après C-008 |
| H-009 | La généralisation de la quatrième cause aux tranches et aux structs, que le livre affirme réellement | Rend la question réfutable, et cesse de prêter au livre une exhaustivité qu'il ne revendique pas (§3.3) | après C-008 |
| H-010 | Doublement des allocations sur une base non nulle qui varie, avec quorum d'étalement | Éprouve le « doublement » du livre, que 0 → 1 n'éprouve pas | après C-008 |
| H-011 | Les trois chiffres de la page 114 aux tolérances du livre, non concluante sur la présente campagne | Teste l'affirmation du livre plutôt qu'une borne trois fois plus lâche | immédiatement |

Trois précautions traversent ces énoncés, chacune tirée d'un échec de cette campagne. Un critère ne bascule plus sur un écart inférieur au bruit, H-007 mesurant son plancher au lieu de le postuler. Un critère exige un témoin, de sensibilité ou de spécificité, sans quoi un verdict ne distingue pas un banc muet d'une affirmation vraie. Enfin H-011 se déclare non concluante sur `C-2026-09-10-1` : un critère écrit après les données qu'il évalue n'éprouve rien.

Restent à faire, hors catalogue : consigner dans la provenance les tailles de cache, la taille de page, `GOMAXPROCS` et le type de cœur ; épingler l'affinité processeur ; faire porter le rééchantillonnage sur plusieurs binaires et non sur les seules répétitions d'un binaire ; archiver le `b.N` de chaque répétition.

## 8. Limites de validité externe

Une machine, un système, une architecture, une version de Go, une campagne. `C-006` demande explicitement `arm64` pour H-002 et H-004, et cette architecture n'a pas été mesurée. Aucun des six verdicts ne devrait être cité sans sa provenance.

## 9. Traçabilité

| Artefact | Chemin |
|---|---|
| Matrice | régénérable à l'identique par `escapebench matrix --reference`, identifiant déterministe |
| Verdicts d'échappement | `EscapeBench/results/escape/M-823d8b5af441/` — deux fichiers, la comparaison NFR-002 |
| Campagne et mesures | `EscapeBench/results/campaigns/C-2026-09-10-1/` — 230 fichiers |
| Comparaison | `EscapeBench/results/campaigns/C-2026-09-10-1/comparison-20260910T142948Z.json` |
| Verdicts par hypothèse | `EscapeBench/results/verdicts/C-2026-09-10-1-20260910T143001Z.json` |
| Tableau de bord | `EscapeBench/docs/dashboard.md` |

Une réserve d'archivage relevée par l'audit et confirmée : les fichiers de mesure conservent les vingt valeurs par sujet mais ni le `b.N` de chaque répétition ni la sortie brute de `go test`. Pour un verdict qui bascule sur 0,04 ns, c'est insuffisant pour recalculer ou contester une médiane. À corriger avant la campagne suivante.
