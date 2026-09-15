# Prospection : Cadrage et Développement IA avec Claude Code

**Statut** : deux projets clos, EscapeBench (P1) le 2026-09-10, avec des verdicts révisés le 2026-09-12 après un audit du code, et LeakLab (P3) le 2026-09-13; six autres projets au stade du cadrage.

## Résumé

Les ouvrages de génie logiciel transmettent des règles de performance chiffrées que l'on applique souvent sans les vérifier. Ce dépôt en fait des objets d'étude. Il repère dans *Building Enterprise Projects with Go* (Shahsavan, 2026) les affirmations assez précises pour qu'une mesure puisse les contredire, puis construit avec un agent de codage, Claude Code (Marco, 2026), des bancs d'essai qui les mettent à l'épreuve. Le développement suit le *Spec-Driven Development* (Martinelli, 2026) : chaque affirmation devient une hypothèse dont le critère de réfutation est écrit et gelé avant la première mesure. Le premier banc, EscapeBench, porte sur la gestion de la mémoire en Go. Sur treize hypothèses, cinq sont infirmées et huit confirmées. Les infirmations les plus instructives ne montrent pas que le livre se trompe, mais qu'il décrit un cas particulier sans le dire : la règle des « un à trois mots machine » dépend de la forme d'une structure autant que de sa taille; le rapport de 10 à 200 entre cache et mémoire vaut pour une latence, pas pour un débit; et le passage au pointeur ne double les allocations que si la fonction n'alloue rien d'autre. Un audit du code mené après la clôture a en outre montré qu'un verdict publié, l'infirmation de H-006, était un artefact du banc : corrigé et rejoué, il passe à *confirmée*. Le second banc, LeakLab, éprouve ce que le livre dit des fuites de goroutines, des interblocages et des outils censés les révéler. Sur quatorze hypothèses, huit sont infirmées : un test ordinaire et le détecteur de courses ne voient aucune des onze fuites du corpus, `testing/synctest` en manque trois, et deux interblocages que le livre dit fatals bloquent dix minutes sous `go test`.

**Mots-clés** : Go, analyse d'échappement, micro-benchmark, concurrence, fuites de goroutines, réfutabilité, *Spec-Driven Development*, agents de codage, Claude Code.

## 1. Introduction

### 1.1 Problème

Un livre de génie logiciel affirme que copier une petite structure coûte moins cher que transmettre son adresse, qu'un défaut de cache coûte de 10 à 200 fois un succès, ou que préallouer une tranche (*slice*) rend son remplissage six fois plus rapide (Shahsavan, 2026, p. 114, 253–254). Ces règles guident des choix de conception, mais leur domaine de validité reste implicite : quelle machine, quelle version du compilateur, quelle forme de données? Les éprouver demande un banc de mesure rigoureux, long à construire à la main.

### 1.2 Pari méthodologique

Un agent de codage abaisse assez ce coût pour rendre la vérification systématique, à une condition : l'encadrer. Un agent qui écrit à la fois le banc, les mesures et leur interprétation peut ajuster l'une à l'autre sans que personne ne le remarque. Le dépôt associe donc trois ouvrages : le premier fournit les affirmations, le deuxième un processus qui fige ce qui sera jugé avant que les données existent, le troisième les réglages qui contraignent l'agent.

### 1.3 Questions de recherche

- **QR1**. Les affirmations du livre sur la mémoire et la concurrence en Go résistent-elles à une mesure dont le critère de jugement est fixé d'avance?
- **QR2**. Un processus piloté par la spécification permet-il de conduire une telle étude avec un agent de codage sans rompre la traçabilité entre l'affirmation, le critère, le code et le verdict?

Les sections 4 (EscapeBench, la mémoire) et 8 (LeakLab, la concurrence) répondent à QR1 par des mesures. QR2 n'est pas mesurée : le dépôt en fournit deux cas documentés, discutés aux sections 5.3 et 8.5.

## 2. Notions préalables

**Pile et tas.** Un programme range ses données dans deux zones. La *pile* est le brouillon d'une fonction : la place s'y réserve à l'entrée et se libère d'un bloc à la sortie, pour un coût quasi nul. Le *tas* accueille les données qui doivent survivre à la fonction qui les a créées : chaque réservation y coûte, et le ramasse-miettes (*garbage collector*) doit ensuite retrouver et libérer ce qui ne sert plus.

**Analyse d'échappement.** En Go, le programmeur ne choisit pas entre pile et tas : le compilateur décide. Il laisse une variable sur la pile, sauf s'il ne peut pas prouver qu'elle cesse de servir à la sortie de la fonction; on dit alors qu'elle *échappe*, et elle part sur le tas. Shahsavan (2026, p. 238–242) en donne quatre causes courantes : retourner son adresse, la capturer dans une fermeture (*closure*), l'envoyer sur un canal, la ranger dans une *map*, une tranche ou une structure. L'option `go build -gcflags=-m` affiche ces décisions.

**Valeur ou pointeur.** Transmettre une structure *par valeur* en copie tous les octets; la transmettre *par pointeur* ne passe qu'une adresse de huit octets, mais impose un détour pour lire les données et peut forcer la structure sur le tas. D'où la règle du livre (p. 253) : préférer la valeur pour un type petit, typiquement d'un à trois mots machine, soit 8 à 24 octets sur une machine 64 bits.

**Cache, latence et débit.** Le processeur garde près de lui des copies des données récentes, dans des caches de plus en plus grands et lents (L1 à L3). Sur la machine de mesure, une lecture servie par le cache L1 coûte moins d'une nanoseconde; une lecture servie par la mémoire principale, plus d'une centaine. Mais quand les adresses à lire sont connues d'avance, le processeur lance plusieurs lectures à la fois et en masque l'attente : on mesure alors un *débit*. Quand chaque adresse dépend de la lecture précédente, comme en suivant une chaîne de pointeurs, il doit attendre chaque réponse : on mesure une *latence*. Cette distinction décide de deux verdicts.

**Réfutabilité et critère gelé.** Une affirmation n'est vérifiable que si l'on sait dire d'avance quel résultat la contredirait. Chaque hypothèse porte donc un *critère de réfutation* écrit, puis gelé avant toute mesure, comme un protocole préenregistré : une fois les données connues, le seuil ne peut plus bouger. Le verdict est *confirmée*, *infirmée* ou *non concluante*. Une confirmation signifie seulement que la mesure n'a pas contredit l'affirmation, pas qu'elle la prouve.

## 3. Méthode

### 3.1 Choix du sujet

Le document [`Doc/Projets-candidats…`](Doc/Projets-candidats_Building-Enterprise-Projects-with-Go.md) recense une quarantaine d'affirmations réfutables dans le livre, en tire huit projets (P1 à P8) et les note sur cinq critères : réfutabilité, place du concept dans le livre, légèreté de l'infrastructure, adéquation à une boucle agentique « modifier, mesurer, comparer », réutilisation du banc. Trois projets se détachent, à 22 ou 23 points sur 25. EscapeBench (P1) passe en premier : sa boucle de rétroaction se compte en secondes, sans autre infrastructure que `go test`, et le projet suivant (P3) réutilise son vocabulaire.

### 3.2 Processus de développement

Le banc est construit selon l'*AI Unified Process* (AIUP) de Martinelli (2026). Le dossier `EscapeBench/docs/` fait autorité sur le code; il contient une vision, un catalogue d'exigences, un modèle d'entités et cinq cas d'utilisation : générer la matrice de sujets (UC-001), classer l'échappement (UC-002), exécuter une campagne de mesure (UC-003), comparer valeur et pointeur (UC-004), produire les verdicts (UC-005). Le dépôt ajoute au catalogue un quatrième type d'entrée, l'hypothèse `H-###`, qui relie une page du livre, un énoncé réfutable et son critère gelé. Tout changement de comportement commence dans la spécification. Chaque cas d'utilisation progresse de `Draft` à `Review`, `Approved`, `Implemented`, `Verified` puis `Deployed`, ce dernier statut signifiant ici « campagne exécutée et rapport publié ».

L'agent est encadré selon Marco (2026). Un fichier `CLAUDE.md` court porte les règles de construction. Six *skills* (`/spec-review`, `/implement`, `/go-test`, `/spec-coverage`, `/bench`, `/refute`) et deux sous-agents de revue (spécification, code) découpent le travail. Quatre *hooks* le contrôlent : chemins protégés, `gofmt` et `go vet` après chaque édition, forme des cas d'utilisation, suite de tests en fin de tour. Toute demande à l'agent nomme un identifiant (`UC-###` ou `H-###`) : « améliore » ou « optimise » ne sont pas des instructions recevables.

### 3.3 Dispositif expérimental

- **Sujets.** Le banc génère des types de structures Go de 8 à 4 096 octets, avec ou sans champ pointeur, sous deux dispositions : un champ suivi d'un tableau de remplissage, ou uniquement des champs nommés. Chaque type est placé dans un profil de durée de vie (locale, retournée, capturée par une fermeture, envoyée sur un canal, rangée dans une *map*, une tranche ou une structure) et mesuré dans les deux modes, par valeur et par pointeur. La matrice de référence compte 22 types, 220 cellules et 10 sondes.
- **Sondes.** Des programmes indépendants des types générés mesurent le parcours séquentiel ou dispersé d'un jeu de travail, une chaîne de pointeurs dépendante, et le remplissage d'une tranche avec ou sans préallocation.
- **Isolement.** Chaque sujet forme un paquet Go distinct, mesuré par son propre processus `go test` : 20 répétitions de 250 ms (`-count 20 -benchtime 250ms`, `-cpu 1` pour les cellules).
- **Classification.** Le banc lit la sortie de `-gcflags=-m` et analyse syntaxiquement (`go/ast`) l'usage réel de la variable. Il ne déduit jamais la cause du profil de la cellule, sans quoi l'hypothèse sur l'exhaustivité des causes (H-006) serait circulaire.
- **Provenance.** Chaque résultat consigne la version de Go, le système, l'architecture, le processeur, la taille des caches et la date; un dossier de résultats n'est jamais réécrit.
- **Harnais figé.** Une empreinte SHA-256 des gabarits de mesure accompagne chaque campagne; modifier le harnais invalide les campagnes antérieures.

### 3.4 Analyse statistique

Pour chaque paire valeur/pointeur, le banc calcule l'écart des médianes sur les 20 répétitions et son intervalle de confiance à 95 % par *bootstrap* percentile : les mesures sont rééchantillonnées 2 000 fois, avec une graine dérivée de la paire, de sorte que le calcul est rejouable. Un écart est *significatif* si et seulement si l'intervalle exclut zéro. Les effets étudiés étant souvent inférieurs à la nanoseconde, les hypothèses de seconde génération exigent en plus de franchir un plancher de bruit mesuré : d'abord sur un témoin nul dont les deux bras exécutent le même code (H-007), puis sur cinq réplicats de la paire réelle, séparés chacun par une passe complète de la matrice (H-012).

### 3.5 Garde-fous de validité

- **Gel vérifié.** L'empreinte du texte des critères est enregistrée à la création d'une campagne; si le texte change, aucun verdict n'est produit. Chaque critère a un évaluateur écrit dans le code; une hypothèse sans évaluateur reçoit *non concluante*, jamais un verdict deviné.
- **Successeurs plutôt que retouches.** Un critère défectueux ne se corrige pas : on écrit une hypothèse successeur. H-007 à H-013 reprennent ainsi les affirmations de H-001 à H-006 sur des sujets de mesure corrigés.
- **Témoins de sensibilité.** Un critère exige de montrer que le banc sait détecter un effet réel, faute de quoi il rend *non concluante*.
- **Attestation de quiétude.** Autour de chaque mesure, le banc relève l'occupation des cœurs extérieurs au sujet mesuré; au-delà de 12 %, l'hypothèse de latence H-013 refuse de trancher.
- **Revues contradictoires.** Des sous-agents sont chargés de réfuter chaque constat d'une revue. Celle des capacités ajoutées pour la seconde génération d'hypothèses a soumis trente-cinq constats à trois vérificateurs chacun et en a retenu sept.
- **Audit du code et non-régression.** Après la clôture, un audit a confronté le code aux spécifications : dix-huit lectures parallèles, puis trois vérificateurs indépendants par constat ([`Doc/AUDIT.md`](Doc/AUDIT.md)). Un harnais de non-régression rejoue les évaluateurs sur 67 verdicts archivés, et un test vérifie que l'empreinte des critères gelés de chaque campagne n'a pas bougé.

## 4. Résultats

### 4.1 Conditions

Les treize verdicts viennent de deux campagnes menées en série sur la même machine, le 2026-09-10 : `C-2026-09-10-11` (douze hypothèses, 541 sujets, 1 h 08) et `C-2026-09-10-12` (H-012, 82 sujets, 11 min). Machine : Intel Core Ultra 9 275HX, 24 cœurs logiques, cache L1 de données de 48 Kio, dernier niveau de cache de 36 Mio, Windows sur amd64, Go 1.27.0. Aucun sujet en échec; occupation médiane des cœurs extérieurs au sujet de 5,8 % et 5,5 %. Le verdict de H-006 vient du rejeu du 2026-09-12 : classification d'échappement refaite après correction du classificateur (Go 1.27.0, linux/amd64), puis verdicts reproduits sur `C-2026-09-10-11`, dont les onze autres sont restés identiques.

### 4.2 Verdicts

| Hyp. | Affirmation du livre (page) | Mesure | Verdict |
|---|---|---|---|
| H-001 | Copier une petite structure peut coûter moins cher qu'un pointeur (245, 253) | Disposition en tableau : pointeur plus rapide à 8, 16 et 24 octets (écarts de 0,06, 0,05 et 0,67 ns) | **infirmée** |
| H-002 | La valeur reste préférable d'un à trois mots machine (253) | Disposition en tableau : bascule vers le pointeur dès 8 octets sans champ pointeur, dès 24 octets avec | **infirmée** |
| H-003 | Passer au pointeur double les allocations (256) | 0 → 1 allocation sur les 20 paires où la valeur n'alloue rien | confirmée |
| H-004 | Un défaut de cache coûte de 10 à 200 fois un succès (254) | Parcours dispersé contre séquentiel : ×1,9 à 32 Mio, ×2,8 à 128 Mio | **infirmée** |
| H-005 | Préallocation environ 6 fois plus rapide, un cinquième de la mémoire (114), avec une tolérance doublée | Temps ×0,225, mémoire ×0,196 | confirmée |
| H-006 | Les quatre causes d'échappement couvrent les cas observés (238–242) | Les 380 cellules qui échappent se rangent toutes dans les quatre causes (verdict corrigé le 2026-09-12) | confirmée |
| H-007 | Comme H-002, sur des champs nommés | Pointeur en avance, au-delà du bruit, sur au plus une des trois petites tailles par série | confirmée |
| H-008 | Comme H-004, sur une chaîne de pointeurs dépendante | 0,81 ns en L1, 137 ns hors cache, rapport ×169,9 | confirmée |
| H-009 | La règle du conteneur vaut pour la *map*, la tranche et la structure (241) | La locale échappe dans les trois conteneurs, à toutes les tailles mesurées | confirmée |
| H-010 | Comme H-003, quand la valeur alloue déjà | 60 paires sur 120 ne doublent pas : 2 → 3, 8 → 12, 32 → 48 allocations | **infirmée** |
| H-011 | Comme H-005, aux tolérances du livre (au moins ×4,8) | Temps ×4,45, mémoire ×5,11, allocations 27 → 1 | **infirmée** |
| H-012 | Comme H-007, plancher de bruit mesuré sur cinq réplicats | Aucune des cinq cellules jugées ne donne l'avantage au pointeur | confirmée |
| H-013 | Comme H-008, machine attestée au repos | 137 ns hors cache, rapport ×169,9, occupation sous le seuil de 12 % | confirmée |

### 4.3 Lecture

**La forme compte autant que la taille (H-001, H-002, H-007, H-012).** La convention d'appel de Go ne passe une structure dans les registres du processeur que si tous ses champs s'y prêtent, et un tableau de plus d'un élément ne s'y prête pas. Une structure de 24 octets faite d'un champ et d'un tableau de deux mots passe donc par la mémoire, alors que la même taille en trois champs nommés passe en registres. La contre-épreuve de la première campagne l'illustre ([`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md), §3.1) : à 24 octets, le pointeur l'emporte sur la disposition en tableau (0,50 ns contre 1,06), la valeur sur les champs nommés (0,44 ns contre 0,74). La règle du livre tient donc pour les structures telles qu'on les écrit d'ordinaire, ce que confirment H-007 et H-012; elle échoue sur la disposition en tableau. Les écarts de H-001 à 8 et 16 octets (0,06 et 0,05 ns) sont du même ordre que le plancher de bruit mesuré pour H-007 (0,06 à 0,09 ns), plancher que le critère de H-001 n'exigeait pas de franchir.

**Latence ou débit (H-004, H-008, H-013).** Le parcours dispersé de H-004 émet des lectures indépendantes, que le processeur recouvre : le rapport ne vaut que 1,9 à 2,8. La chaîne dépendante de H-008 empêche ce recouvrement et fait apparaître la latence réelle de la mémoire, 137 ns, soit 169,9 fois celle du cache L1. L'affirmation du livre est fausse sur la mesure qu'on lui applique d'ordinaire et vraie sur celle qui l'éprouve réellement.

**Le doublement est un cas particulier (H-003, H-010).** Si une fonction alloue déjà *k* objets par appel, retourner un pointeur en ajoute un : le rapport vaut (*k* + 1) / *k* et n'atteint 2 que pour *k* = 1. H-003, qui compte le passage de 0 à 1 allocation comme un doublement, reste confirmée; c'est la formulation générale du livre qui tombe.

**Un verdict faux, puis corrigé (H-006).** Publiée le 2026-09-10, l'infirmation de H-006 reposait sur 120 cellules qui semblaient échapper hors des quatre causes. L'audit du code a montré qu'elles n'avaient jamais été classées : quand le compilateur nomme l'expression allouée (`new(payload) escapes to heap`) plutôt qu'une variable, le classificateur concluait à une autre cause sans lire le code. Or cette allocation est rangée dans une variable que la fonction retourne : c'est un retour de pointeur, la première des quatre causes. Une fois le classificateur corrigé, les 380 cellules qui échappent se rangent dans les quatre causes et le verdict passe à *confirmée* ([`Doc/RAPPORT-FINAL_EscapeBench.md`](Doc/RAPPORT-FINAL_EscapeBench.md), *Errata*).

**La marge décide, pas la mesure (H-005, H-011).** La préallocation tient deux des trois chiffres de la page 114 : un cinquième de la mémoire et une seule allocation au lieu de 27. Le gain en temps, ×4,45, passe la tolérance large de H-005 mais manque celle que le livre s'accorde lui-même (×4,8).

**Réponse à QR1 (mémoire).** Partielle. Les règles du livre tiennent dans les conditions où il les illustre; ses chiffres et ses généralisations tombent dès qu'on sort de ces conditions ou qu'on les lit à la lettre.

## 5. Discussion

### 5.1 Portée des confirmations

Une confirmation vaut ce que valait la possibilité d'infirmer. H-008 et H-013 ne pouvaient pas l'être sur cette machine : leur critère infirme si la latence hors cache tombe sous 100 ns, et la mémoire de la machine en sert 137. Seule une machine plus rapide, par exemple arm64, pourrait les contredire. H-012 lit la règle de la page 253 sans l'exception qu'autorise le « typiquement » du livre, car la lecture tolérante rendait l'infirmation arithmétiquement inatteignable. Elle écarte aussi la cellule de 8 octets à champ pointeur, dont l'unique champ est le pointeur lui-même. Jugée, cette cellule aurait infirmé l'hypothèse dans ses cinq réplicats : son retrait, qui favorise le livre, a été déclaré avant la mesure pour cette raison. H-006, enfin, ne juge que les situations que le banc sait produire : hormis la variable locale, ses profils de durée de vie reproduisent chacun l'une des quatre causes du livre. La confirmation dit qu'aucune cinquième cause n'est apparue dans ce corpus, pas que la liste est exhaustive, ce que le livre ne prétend d'ailleurs pas.

### 5.2 Ce que le banc a appris sur lui-même

Les revues contradictoires ont trouvé quatre défauts de construction. Le gabarit de H-010 rendait le doublement inévitable, jusqu'à ce que le nombre de charges allouées devienne une dimension mesurée. Le témoin nul de H-007 comparait deux bras au code machine identique et sous-estimait donc le bruit d'une vraie paire; les réplicats de H-012 le mesurent sur la paire réelle. La branche du rapport de H-008 ne se distinguait pas d'une mémoire encombrée par d'autres processus; l'attestation de quiétude de H-013 y répond. Les trois ont été corrigés par des hypothèses successeurs, jamais par la retouche d'un critère gelé. Le quatrième, une confusion du classificateur entre stockage dans un conteneur et retour d'adresse par le champ d'une structure, n'influe sur aucun verdict et reste épinglé par un test.

Le défaut le plus grave n'a été trouvé qu'après la clôture, par l'audit du code : l'erreur du classificateur qui a produit le verdict faux de H-006 (voir 4.3). L'audit a retenu trois défauts bloquants, dont deux empêchaient de reprendre une campagne interrompue, et vingt majeurs. Tous les lots de correctifs ont été appliqués sauf un, le lot 9, consigné comme dette assumée parce qu'il rendrait les campagnes archivées incomparables (D-50), et le harnais de non-régression établit que les douze autres verdicts publiés n'ont pas bougé.

### 5.3 Le développement assisté par IA (QR2)

Le dépôt n'offre pas de groupe témoin : les observations suivantes décrivent un cas, elles ne mesurent pas un effet.

- **Le gel a empêché la rationalisation après coup.** Les critères défectueux ont été remplacés par sept hypothèses successeurs plutôt que réécrits, et l'empreinte des critères bloque tout verdict si leur texte change.
- **Les pouvoirs sont séparés.** Un *hook* interdit à l'agent d'écrire les résultats, les matrices et le tableau de bord : seul le binaire du banc les produit, et le harnais est verrouillé pendant une campagne.
- **La garde a mordu contre son auteur.** Une campagne a été mesurée sous charge parce que des processus d'une contre-épreuve précédente n'avaient pas été arrêtés, une erreur de l'agent. Onze hypothèses ont rendu leur verdict sans rien voir; H-013 a rendu *non concluante* en nommant la cause. La campagne a été refaite, et la version contaminée conservée comme témoin.
- **Un verdict faux a été publié, puis corrigé par erratum.** Ni les revues contradictoires ni les gardes du banc n'avaient vu l'erreur du classificateur à l'origine de l'infirmation de H-006; l'audit du code l'a trouvée deux jours après la publication. La correction est datée et motivée dans le rapport final plutôt que substituée en silence, et le rejeu, mené sous Linux alors que les campagnes l'avaient été sous Windows, consigne pourquoi cet écart de provenance est sans effet (décision D-48).
- **Un écart au processus est assumé.** L'AIUP réserve le passage d'un cas d'utilisation au statut `Approved` à une décision humaine. L'agent l'a franchi lui-même, en vertu du mandat de construire sans intervention (décision D-01, [`Doc/DECISION.md`](Doc/DECISION.md)) : la revue humaine prévue par Martinelli (2026) n'a pas eu lieu.

**Réponse à QR2.** Le cas montre qu'un agent peut conduire l'étude de bout en bout en gardant chaque verdict rattaché à un critère gelé, à une page du livre et à un code identifié. Il ne montre pas que l'agent aurait détecté seul ses défauts de construction : ce sont les revues contradictoires, les gardes du banc et un audit du code postérieur à la clôture qui les ont révélés. Pour H-006, la traçabilité a permis de corriger un verdict faux, pas d'éviter sa publication.

## 6. Limites et menaces à la validité

- **Validité externe.** Micro-benchmarks sur des types synthétiques, une seule machine, une seule soirée, Windows sur amd64 et Go 1.27.0, alors que le livre se réfère à Go 1.25. L'analyse d'échappement varie d'une version du compilateur à l'autre (Shahsavan, 2026, p. 242) : les résultats sont datés par construction.
- **Validité de construit.** Un verdict juge une opérationnalisation de l'affirmation, pas sa prose : H-004 et H-008 donnent deux verdicts opposés sur la même page du livre.
- **Validité interne.** L'attestation de quiétude mesure l'occupation des processeurs, indicateur nécessaire mais non suffisant de l'encombrement de la mémoire. Mesurer la bande passante demanderait les compteurs de performance du processeur, hors de la bibliothèque standard à laquelle le banc se limite.
- **Provenance du rejeu de H-006.** La classification corrigée a été refaite sous linux/amd64, alors que les campagnes l'avaient été sous windows/amd64, avec la même version de Go. Sur 532 cellules, 412 rendent la même catégorie sur les deux systèmes et les 120 autres sont exactement celles que vise le correctif (D-48); un rejeu sur le poste de référence lèverait la réserve.
- **Revue humaine.** Voir l'écart au processus décrit en 5.3.
- **LeakLab.** Ses limites propres (corpus synthétique, poste Windows unique, oracle muet sur les courses et l'accessibilité des primitives) sont décrites en 8.5.

## 7. Travaux futurs

- Rejouer H-008 et H-013 sur une machine dont la mémoire sert un accès dépendant en moins de 100 ns, par exemple arm64.
- Désassembler les deux bras d'une paire pour expliquer pourquoi, à 8 octets avec champ pointeur, le bras pointeur exécute une lecture de plus et va pourtant plus vite.
- Borner directement l'encombrement de la mémoire par les compteurs de performance du processeur.
- Rejouer la classification d'échappement de H-006 sur le poste de référence Windows.
- Appliquer le lot 9 de l'audit, qui touche les gabarits du harnais, avant la première campagne arm64, qui ouvrira de toute façon une nouvelle série de comparaison (D-50).
- LeakLab : rejouer la campagne sous Linux et sur arm64; vérifier dans le runtime l'angle mort du profil `goroutineleak` sur les petits mutex, et l'étendre à `sync.RWMutex`; mesurer le compte de goroutines dans une suite réelle, sous `t.Parallel`.
- Conduire le projet suivant de la séquence recommandée, P4 (HexaGuard, règle de dépendance hexagonale exécutable), P3 étant réalisé (section 8).

## 8. Second projet : LeakLab (P3)

### 8.1 Question

Pour chaque anti-patron de concurrence du chapitre 20 — fuite de goroutine, interblocage de canal, contexte non annulé, I/O qui ignore le contexte — et pour les courses de données, quel mécanisme de détection le révèle, avec quels faux négatifs et quels faux positifs? Le livre fait des promesses sur ces outils : le cadre de test signalerait les goroutines fuitées (p. 217), `-race` en CI attraperait les bogues de concurrence (p. 232), `synctest` paniquerait sur toute goroutine restée bloquée (p. 291–292), la hausse de `runtime.NumGoroutine()` signalerait une fuite (p. 293). Il chiffre ou qualifie aussi des coûts : un délai de 500 ms testé instantanément (p. 291), la mémoire et les goroutines d'un `cancel()` oublié (p. 567), la croissance de la mémoire d'un `time.After` en boucle (p. 276).

### 8.2 Méthode

Le processus est celui d'EscapeBench (section 3), avec trois différences tirées de ses défauts.

- **Un corpus à vérité terrain indépendante.** 32 cas minimaux, dont 16 fautifs (11 fuites, 2 interblocages, 1 contexte non annulé, 2 courses), leurs 15 corrections et un témoin qui échoue. La vérité terrain est vérifiée par un oracle qui lit les piles de goroutines, et non par l'un des détecteurs qu'elle sert à juger : c'est la leçon du verdict faux de H-006.
- **Huit détecteurs, un processus par observation.** Test ordinaire, test sous `-race`, bulle `synctest`, compte de goroutines, profil `goroutineleak` (Go 1.27), programme, `go vet` et `ctxvet`, un analyseur écrit pour le banc. Chaque cas passe sous chaque détecteur dynamique cinq fois, dans un processus neuf tué au bout de 5 s. Les critères lisent l'issue majoritaire.
- **Un témoin par hypothèse** : un cas qui doit échouer, un bras hors bulle, un parent de contexte que `context` surveille par une goroutine, des minuteries gardées exprès. Sans lui, un verdict ne distingue pas un détecteur muet d'une affirmation vraie.

Les critères ont été committés avant la première ligne de code ([`LeakLab/docs/requirements.md`](LeakLab/docs/requirements.md)).

### 8.3 Verdicts

Campagne de référence `R-2026-09-13-2` : go1.27.0, windows/amd64, 1 099 mesures en 9 min, aucune cellule dont les répétitions divergent.

| Hyp. | Affirmation (page) | Mesure | Verdict |
|---|---|---|---|
| H-001 | Une fuite passe les tests en silence (561) | Test ordinaire vert sur les 11 fuites | confirmée |
| H-002 | Le cadre de test signale les goroutines fuitées (217) | Aucun signalement sur 55 exécutions | **infirmée** |
| H-003 | `-race` rapporte les courses (232) | 2 courses sur 2, aucun faux positif | confirmée |
| H-004 | `-race` en CI attrape les bogues de concurrence (232) | 0 défaut autre qu'une course sur 14 | **infirmée** |
| H-005 | `synctest` panique sur une goroutine restée bloquée (289–292) | 8 fuites sur 11 ; bloque sur mutex, canal global, réseau | **infirmée** |
| H-006 | Le délai de 500 ms se teste instantanément (291) | Sous la résolution de l'horloge, contre 500,6 ms hors bulle | confirmée |
| H-007 | Les deux interblocages finissent en erreur fatale (564) | Vrai pour un programme | confirmée |
| H-008 | Même affirmation, sous `go test` | Les deux bloquent : l'alarme de 10 min empêche la détection | **infirmée** |
| H-009 | Un `cancel()` oublié retient de la mémoire jusqu'à l'échéance (567) | Sonde défectueuse (voir H-014) | **infirmée** |
| H-010 | Un `cancel()` oublié coûte des goroutines (567) | 0 avec un parent standard | **infirmée** |
| H-011 | `time.After` en boucle fait croître la mémoire (276) | +0,01 octet par itération | **infirmée** |
| H-012 | La hausse de `NumGoroutine` signale une fuite (293) | 11 fuites sur 11, aucun faux positif | confirmée |
| H-013 | Le profil `goroutineleak` voit les primitives inaccessibles (notes Go 1.26/1.27) | Manque le mutex de 8 octets | **infirmée** |
| H-014 | Successeur de H-009, résidu net du coût d'expiration | 275 à 319 octets retenus, rendus après l'échéance | confirmée |

### 8.4 Lecture

Le livre recommande deux outils qui ne voient pas les fuites, le test et `-race`, et le seul détecteur dynamique qui les a toutes vues est le plus rudimentaire, le compte de goroutines d'un scénario répété. Les infirmations suivent le motif d'EscapeBench : le livre décrit un cas particulier sans le dire. `synctest` ne panique que si la goroutine est *durablement* bloquée, ce qui exclut les mutex et le réseau. L'interblocage est fatal pour un programme, pas sous `go test`, dont la minuterie d'alarme empêche le runtime de le déclarer (`checkdead`). Oublier `cancel()` coûte de la mémoire, et des goroutines seulement avec un parent que `context` ne reconnaît pas. Le conseil sur `time.After` date d'avant Go 1.23. Le rapport en tire une recommandation pour l'intégration continue : `go vet` explicite, dont `go test` omet l'analyseur `lostcancel` ; contrôle de goroutines dans les tests ; `-timeout` court.

**Réponse à QR1 (concurrence).** Partielle aussi : huit infirmations sur quatorze, dont cinq portent sur ce que le livre prête aux mécanismes de détection (H-002, H-004, H-005, H-008, H-013).

### 8.5 Ce que le banc a appris sur lui-même

La première campagne, `R-2026-09-13-1`, a produit un verdict faux (H-008) et une infirmation tirée d'une sonde défectueuse (H-009). Relus avant la rédaction du rapport, les résultats contraires à l'attente ont révélé deux défauts de construction. Le banc lançait les binaires de test sans le délai que `go test` leur transmet : sans alarme, le runtime déclarait les interblocages, H-008 était confirmée à tort, et `synctest` semblait voir deux fuites qu'il ne voit pas. La sonde de H-009 comptait, comme mémoire retenue par les contextes, ce que le runtime garde après l'expiration de toute minuterie. Le premier défaut a été corrigé et la campagne refaite ; le second a donné une hypothèse successeur, H-014, sans retouche du critère gelé. Contrairement à H-006 dans EscapeBench, aucun verdict faux n'a atteint le rapport : la campagne fautive est archivée comme telle, et quatre contre-épreuves hors campagne ont tranché les doutes restants. Cela montre que l'agent peut appliquer la leçon d'un projet au suivant, pas qu'il n'a plus besoin de vérification extérieure : le banc n'a eu aucune revue humaine (D-01).

Limites propres à LeakLab : corpus synthétique, un seul poste Windows, pas de campagne sous Linux, et une vérité terrain que l'oracle ne vérifie pas pour les courses ni pour l'accessibilité des primitives. Détails : [`Doc/RAPPORT-FINAL_LeakLab.md`](Doc/RAPPORT-FINAL_LeakLab.md), décisions D-01 à D-17 : [`Doc/DECISION_LeakLab.md`](Doc/DECISION_LeakLab.md).

## 9. Reproduire les résultats

### 9.1 EscapeBench

Prérequis : Go 1.25 ou plus récent; aucune dépendance hors bibliothèque standard. Chaîne complète, depuis `EscapeBench/` :

```bash
go vet ./... && go test -race -shuffle=on -count=1 ./...
go run ./cmd/escapebench matrix --reference
go run ./cmd/escapebench escape --matrix <matrixId>
go run ./cmd/escapebench campaign --matrix <matrixId> --count 20
go run ./cmd/escapebench compare --campaign <campaignId>
go run ./cmd/escapebench verdict --campaign <campaignId>
```

La matrice de référence demande environ 29 minutes sur la machine décrite en 4.1. Le harnais de non-régression, qui rejoue les évaluateurs sur les verdicts archivés sans rien mesurer, s'exécute en moins d'une minute :

```bash
go test -race -shuffle=on -count=1 -tags=integration_test ./...
```

H-012 exige une matrice à cinq réplicats et une campagne qui nomme ses hypothèses (`--hypotheses`). Les classifications d'échappement, les mesures de campagne et les verdicts publiés sont archivés dans [`EscapeBench/results/`](EscapeBench/results/); la procédure détaillée est dans [`EscapeBench/LANCEMENT.md`](EscapeBench/LANCEMENT.md).

### 9.2 LeakLab

Prérequis : Go 1.27.0 ou plus récent, sans dépendance hors bibliothèque standard. Depuis `LeakLab/`, les tests, puis une campagne (environ 9 minutes), puis ses verdicts :

```bash
go vet ./... && go test -race -shuffle=on -count=1 ./...
```

```bash
go run ./cmd/leaklab run
```

```bash
go run ./cmd/leaklab verdict -run <runId>
```

Le binaire refuse la campagne si le catalogue du corpus diverge de la spécification, si l'oracle contredit la vérité terrain ou si un pilote ne compile pas. Les campagnes et les verdicts, matrice comprise, sont archivés dans [`LeakLab/results/`](LeakLab/results/).

## 10. Organisation du dépôt

```
Prospection/
├── Doc/           cadrage, méthode, décisions, audit du code et rapports finaux
├── Campagnes/     rapports des campagnes de mesure intermédiaires d'EscapeBench
├── Revue/         revues du dépôt et revues contradictoires
├── EscapeBench/   le premier banc (P1) : spécification, code Go, réglages de l'agent, résultats
└── LeakLab/       le second banc (P3) : spécification, corpus, code Go, réglages de l'agent, résultats
```

| Document | Rôle |
|---|---|
| [`Doc/RAPPORT-FINAL_EscapeBench.md`](Doc/RAPPORT-FINAL_EscapeBench.md) | Verdicts consolidés, portée des confirmations, défauts de construction, questions ouvertes. **À lire en premier.** |
| [`Doc/Projets-candidats_Building-Enterprise-Projects-with-Go.md`](Doc/Projets-candidats_Building-Enterprise-Projects-with-Go.md) | Cartographie des affirmations réfutables, grille d'évaluation, fiches P1 à P8, séquence recommandée |
| [`Doc/Guide-implementation_AIUP-Claude-Code.md`](Doc/Guide-implementation_AIUP-Claude-Code.md) | Méthode : AIUP adapté aux bancs de réfutation, réglages Claude Code, cycle de travail par cas d'utilisation |
| [`Doc/DECISION.md`](Doc/DECISION.md) | Journal des décisions D-01 à D-54 : écarts assumés, conception du harnais, statistiques, clôture, correctifs de l'audit, dépôt final |
| [`Doc/AUDIT.md`](Doc/AUDIT.md) | Audit du code du 2026-09-12 : constats vérifiés, lots de correctifs, état d'implantation et décisions sur les points restants |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md) | Campagne de référence : premiers verdicts (H-001 à H-006) et audit contradictoire |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-3.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-3.md) | Première épreuve de la seconde génération (H-007 à H-013) |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-4.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-4.md) | Première épreuve de H-012 sur une matrice à réplicats |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-5.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-5.md) | Première épreuve de H-013 et démonstration de l'attestation de quiétude |
| [`Revue/REVUE-PRELANCEMENT_2026-09-10.md`](Revue/REVUE-PRELANCEMENT_2026-09-10.md) | Revue du cadrage avant le développement |
| [`Revue/REVUE-C008_2026-09-10.md`](Revue/REVUE-C008_2026-09-10.md) | Revue contradictoire des capacités ajoutées pour la seconde génération d'hypothèses |
| [`EscapeBench/`](EscapeBench/) | Le banc : spécification (`docs/`), code Go (`cmd/`, `internal/`), réglages de l'agent (`CLAUDE.md`, `.claude/`), résultats (`results/`), procédure (`LANCEMENT.md`) |
| [`Doc/RAPPORT-FINAL_LeakLab.md`](Doc/RAPPORT-FINAL_LeakLab.md) | LeakLab : verdicts des quatorze hypothèses, matrice de détectabilité, recommandation pour l'intégration continue, défauts de construction |
| [`Doc/DECISION_LeakLab.md`](Doc/DECISION_LeakLab.md) | Journal des décisions de LeakLab, D-01 à D-17 : gel, architecture, oracle, détecteurs, décisions prises après la première campagne |
| [`LeakLab/`](LeakLab/) | Le second banc : spécification (`docs/`), corpus et pilotes (`lab/`), code Go (`cmd/`, `internal/`), réglages de l'agent (`CLAUDE.md`, `.claude/`), résultats (`results/`) |

Les campagnes finales `C-2026-09-10-11` et `C-2026-09-10-12`, dont viennent les treize verdicts d'EscapeBench, n'ont pas de rapport propre : elles sont consolidées dans le rapport final.

Ordre de lecture suggéré : le rapport final d'EscapeBench, puis [`EscapeBench/docs/`](EscapeBench/docs/) dans l'ordre AIUP (vision, exigences, modèle d'entités, cas d'utilisation) ; ensuite le rapport final de LeakLab et [`LeakLab/docs/`](LeakLab/docs/). Pour cadrer un nouveau projet : `Doc/Projets-candidats…` §1–5 et §7, puis `Doc/Guide-implementation…` §1–7.

Conventions : prose en français, identifiants et code en anglais; pages citées = folios imprimés; marqueurs épistémiques *Confirmé*, *Probable*, *Hypothèse*, *À vérifier* et *Adaptation* dans les documents de cadrage.

## Licence

Le code Go, les scripts et la configuration sont distribués sous licence MIT ([`LICENSE`](LICENSE)). La prose (spécifications, rapports, décisions, revues) et les résultats de mesure sont distribués sous licence [Creative Commons Attribution 4.0 International](https://creativecommons.org/licenses/by/4.0/deed.fr) (CC BY 4.0). Les extraits cités des ouvrages restent la propriété de leurs auteurs et éditeurs.

## Références

Les ouvrages ne sont pas distribués avec le dépôt; les liens mènent à la notice de l'éditeur.

Marco, E. (2026). *Agentic Coding with Claude Code: The everyday developer's guide to agentic coding with Claude Code*. Packt Publishing. Première publication en mars 2026 (© 2025). ISBN 978-1-80602-259-5 (imprimé), 978-1-80602-258-8 (numérique). https://www.packtpub.com/en-us/product/agentic-coding-with-claude-code-9781806022588.

Martinelli, S. (2026). *Spec-Driven Development: From Specs to Code with AI Agents*. Apress (Apress Pocket Guides). ISBN 979-8-8688-2850-8 (imprimé), 979-8-8688-2851-5 (numérique). https://doi.org/10.1007/979-8-8688-2851-5.

Shahsavan, S. (2026). *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*. Apress. ISBN 979-8-8688-2369-5 (imprimé), 979-8-8688-2370-1 (numérique). https://doi.org/10.1007/979-8-8688-2370-1.
