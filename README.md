# Prospection : Cadrage et Développement IA avec Claude Code

**Statut** : premier projet (EscapeBench) clos le 2026-09-10; sept autres au stade du cadrage.

## Résumé

Les ouvrages de génie logiciel transmettent des règles de performance chiffrées que l'on applique souvent sans les vérifier. Ce dépôt en fait des objets d'étude. Il repère dans *Building Enterprise Projects with Go* (Shahsavan, 2026) les affirmations assez précises pour qu'une mesure puisse les contredire, puis construit avec un agent de codage, Claude Code (Marco, 2026), des bancs d'essai qui les mettent à l'épreuve. Le développement suit le *Spec-Driven Development* (Martinelli, 2026) : chaque affirmation devient une hypothèse dont le critère de réfutation est écrit et gelé avant la première mesure. Le premier banc, EscapeBench, porte sur la gestion de la mémoire en Go. Sur treize hypothèses, sept sont infirmées et six confirmées. Les infirmations les plus instructives ne montrent pas que le livre se trompe, mais qu'il décrit un cas particulier sans le dire : la règle des « un à trois mots machine » dépend de la forme d'une structure autant que de sa taille; le rapport de 10 à 200 entre cache et mémoire vaut pour une latence, pas pour un débit; et le passage au pointeur ne double les allocations que si la fonction n'alloue rien d'autre.

**Mots-clés** : Go, analyse d'échappement, micro-benchmark, réfutabilité, *Spec-Driven Development*, agents de codage, Claude Code.

## 1. Introduction

### 1.1 Problème

Un livre de génie logiciel affirme que copier une petite structure coûte moins cher que transmettre son adresse, qu'un défaut de cache coûte de 10 à 200 fois un succès, ou que préallouer une tranche (*slice*) rend son remplissage six fois plus rapide (Shahsavan, 2026, p. 114, 253–254). Ces règles guident des choix de conception, mais leur domaine de validité reste implicite : quelle machine, quelle version du compilateur, quelle forme de données? Les éprouver demande un banc de mesure rigoureux, long à construire à la main.

### 1.2 Pari méthodologique

Un agent de codage abaisse assez ce coût pour rendre la vérification systématique, à une condition : l'encadrer. Un agent qui écrit à la fois le banc, les mesures et leur interprétation peut ajuster l'une à l'autre sans que personne ne le remarque. Le dépôt associe donc trois ouvrages : le premier fournit les affirmations, le deuxième un processus qui fige ce qui sera jugé avant que les données existent, le troisième les réglages qui contraignent l'agent.

### 1.3 Questions de recherche

- **QR1**. Les affirmations chiffrées du livre sur la mémoire en Go résistent-elles à une mesure dont le critère de jugement est fixé d'avance?
- **QR2**. Un processus piloté par la spécification permet-il de conduire une telle étude avec un agent de codage sans rompre la traçabilité entre l'affirmation, le critère, le code et le verdict?

La section 4 répond à QR1 par des mesures. QR2 n'est pas mesurée : le dépôt en fournit un cas documenté, discuté à la section 5.

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

## 4. Résultats

### 4.1 Conditions

Les treize verdicts viennent de deux campagnes menées en série sur la même machine, le 2026-09-10 : `C-2026-09-10-11` (douze hypothèses, 541 sujets, 1 h 08) et `C-2026-09-10-12` (H-012, 82 sujets, 11 min). Machine : Intel Core Ultra 9 275HX, 24 cœurs logiques, cache L1 de données de 48 Kio, dernier niveau de cache de 36 Mio, Windows sur amd64, Go 1.27.0. Aucun sujet en échec; occupation médiane des cœurs extérieurs au sujet de 5,8 % et 5,5 %.

### 4.2 Verdicts

| Hyp. | Affirmation du livre (page) | Mesure | Verdict |
|---|---|---|---|
| H-001 | Copier une petite structure peut coûter moins cher qu'un pointeur (245, 253) | Disposition en tableau : pointeur plus rapide à 8, 16 et 24 octets (écarts de 0,06, 0,05 et 0,67 ns) | **infirmée** |
| H-002 | La valeur reste préférable d'un à trois mots machine (253) | Disposition en tableau : bascule vers le pointeur dès 8 octets sans champ pointeur, dès 24 octets avec | **infirmée** |
| H-003 | Passer au pointeur double les allocations (256) | 0 → 1 allocation sur les 20 paires où la valeur n'alloue rien | confirmée |
| H-004 | Un défaut de cache coûte de 10 à 200 fois un succès (254) | Parcours dispersé contre séquentiel : ×1,9 à 32 Mio, ×2,8 à 128 Mio | **infirmée** |
| H-005 | Préallocation environ 6 fois plus rapide, un cinquième de la mémoire (114), avec une tolérance doublée | Temps ×0,225, mémoire ×0,196 | confirmée |
| H-006 | Les quatre causes d'échappement couvrent les cas observés (238–242) | 120 cellules sur 380 échappent hors des quatre causes | **infirmée** |
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

**Une liste d'exemples, pas une classification (H-006).** Sur 380 cellules qui échappent, 120 le font pour une raison absente des quatre causes : une charge allouée par la fonction part sur le tas alors que la structure mesurée reste sur la pile. Le livre présentait ses causes comme les plus courantes, sans prétendre à l'exhaustivité; la mesure lui donne raison de cette prudence.

**La marge décide, pas la mesure (H-005, H-011).** La préallocation tient deux des trois chiffres de la page 114 : un cinquième de la mémoire et une seule allocation au lieu de 27. Le gain en temps, ×4,45, passe la tolérance large de H-005 mais manque celle que le livre s'accorde lui-même (×4,8).

**Réponse à QR1.** Partielle. Les règles du livre tiennent dans les conditions où il les illustre; ses chiffres et ses généralisations tombent dès qu'on sort de ces conditions ou qu'on les lit à la lettre.

## 5. Discussion

### 5.1 Portée des confirmations

Une confirmation vaut ce que valait la possibilité d'infirmer. H-008 et H-013 ne pouvaient pas l'être sur cette machine : leur critère infirme si la latence hors cache tombe sous 100 ns, et la mémoire de la machine en sert 137. Seule une machine plus rapide, par exemple arm64, pourrait les contredire. H-012 lit la règle de la page 253 sans l'exception qu'autorise le « typiquement » du livre, car la lecture tolérante rendait l'infirmation arithmétiquement inatteignable. Elle écarte aussi la cellule de 8 octets à champ pointeur, dont l'unique champ est le pointeur lui-même. Jugée, cette cellule aurait infirmé l'hypothèse dans ses cinq réplicats : son retrait, qui favorise le livre, a été déclaré avant la mesure pour cette raison.

### 5.2 Ce que le banc a appris sur lui-même

Les revues contradictoires ont trouvé quatre défauts de construction. Le gabarit de H-010 rendait le doublement inévitable, jusqu'à ce que le nombre de charges allouées devienne une dimension mesurée. Le témoin nul de H-007 comparait deux bras au code machine identique et sous-estimait donc le bruit d'une vraie paire; les réplicats de H-012 le mesurent sur la paire réelle. La branche du rapport de H-008 ne se distinguait pas d'une mémoire encombrée par d'autres processus; l'attestation de quiétude de H-013 y répond. Les trois ont été corrigés par des hypothèses successeurs, jamais par la retouche d'un critère gelé. Le quatrième, une confusion du classificateur entre stockage dans un conteneur et retour d'adresse par le champ d'une structure, n'influe sur aucun verdict et reste épinglé par un test.

### 5.3 Le développement assisté par IA (QR2)

Le dépôt n'offre pas de groupe témoin : les observations suivantes décrivent un cas, elles ne mesurent pas un effet.

- **Le gel a empêché la rationalisation après coup.** Les critères défectueux ont été remplacés par sept hypothèses successeurs plutôt que réécrits, et l'empreinte des critères bloque tout verdict si leur texte change.
- **Les pouvoirs sont séparés.** Un *hook* interdit à l'agent d'écrire les résultats, les matrices et le tableau de bord : seul le binaire du banc les produit, et le harnais est verrouillé pendant une campagne.
- **La garde a mordu contre son auteur.** Une campagne a été mesurée sous charge parce que des processus d'une contre-épreuve précédente n'avaient pas été arrêtés, une erreur de l'agent. Onze hypothèses ont rendu leur verdict sans rien voir; H-013 a rendu *non concluante* en nommant la cause. La campagne a été refaite, et la version contaminée conservée comme témoin.
- **Un écart au processus est assumé.** L'AIUP réserve le passage d'un cas d'utilisation au statut `Approved` à une décision humaine. L'agent l'a franchi lui-même, en vertu du mandat de construire sans intervention (décision D-01, [`Doc/DECISION.md`](Doc/DECISION.md)) : la revue humaine prévue par Martinelli (2026) n'a pas eu lieu.

**Réponse à QR2.** Le cas montre qu'un agent peut conduire l'étude de bout en bout en gardant chaque verdict rattaché à un critère gelé, à une page du livre et à un code identifié. Il ne montre pas que l'agent aurait détecté seul ses défauts de construction : ce sont les revues contradictoires et les gardes du banc qui les ont révélés.

## 6. Limites et menaces à la validité

- **Validité externe.** Micro-benchmarks sur des types synthétiques, une seule machine, une seule soirée, Windows sur amd64 et Go 1.27.0, alors que le livre se réfère à Go 1.25. L'analyse d'échappement varie d'une version du compilateur à l'autre (Shahsavan, 2026, p. 242) : les résultats sont datés par construction.
- **Validité de construit.** Un verdict juge une opérationnalisation de l'affirmation, pas sa prose : H-004 et H-008 donnent deux verdicts opposés sur la même page du livre.
- **Validité interne.** L'attestation de quiétude mesure l'occupation des processeurs, indicateur nécessaire mais non suffisant de l'encombrement de la mémoire. Mesurer la bande passante demanderait les compteurs de performance du processeur, hors de la bibliothèque standard à laquelle le banc se limite.
- **Revue humaine.** Voir l'écart au processus décrit en 5.3.

## 7. Travaux futurs

- Rejouer H-008 et H-013 sur une machine dont la mémoire sert un accès dépendant en moins de 100 ns, par exemple arm64.
- Désassembler les deux bras d'une paire pour expliquer pourquoi, à 8 octets avec champ pointeur, le bras pointeur exécute une lecture de plus et va pourtant plus vite.
- Borner directement l'encombrement de la mémoire par les compteurs de performance du processeur.
- Conduire les projets suivants selon la séquence recommandée, P3 (LeakLab, détectabilité des anti-patrons de concurrence) puis P4 (HexaGuard, règle de dépendance hexagonale exécutable), en partant d'`EscapeBench/` comme gabarit.

## 8. Reproduire les résultats

Prérequis : Go 1.25 ou plus récent; aucune dépendance hors bibliothèque standard. Chaîne complète, depuis `EscapeBench/` :

```bash
go vet ./... && go test -race -shuffle=on -count=1 ./...
go run ./cmd/escapebench matrix --reference
go run ./cmd/escapebench escape --matrix <matrixId>
go run ./cmd/escapebench campaign --matrix <matrixId> --count 20
go run ./cmd/escapebench compare --campaign <campaignId>
go run ./cmd/escapebench verdict --campaign <campaignId>
```

La matrice de référence demande environ 29 minutes sur la machine décrite en 4.1. H-012 exige une matrice à cinq réplicats et une campagne qui nomme ses hypothèses (`--hypotheses`). Les classifications d'échappement, les mesures de campagne et les verdicts publiés sont archivés dans [`EscapeBench/results/`](EscapeBench/results/); la procédure détaillée est dans [`EscapeBench/LANCEMENT.md`](EscapeBench/LANCEMENT.md).

## 9. Organisation du dépôt

```
Prospection/
├── Book/          ouvrages de référence (PDF)
├── Doc/           cadrage, méthode, décisions et rapport final
├── Campagnes/     rapports des campagnes de mesure intermédiaires
├── Revue/         revues du dépôt et revues contradictoires
└── EscapeBench/   le banc (P1) : spécification, code Go, réglages de l'agent, résultats
```

| Document | Rôle |
|---|---|
| [`Doc/RAPPORT-FINAL_EscapeBench.md`](Doc/RAPPORT-FINAL_EscapeBench.md) | Verdicts consolidés, portée des confirmations, défauts de construction, questions ouvertes. **À lire en premier.** |
| [`Doc/Projets-candidats_Building-Enterprise-Projects-with-Go.md`](Doc/Projets-candidats_Building-Enterprise-Projects-with-Go.md) | Cartographie des affirmations réfutables, grille d'évaluation, fiches P1 à P8, séquence recommandée |
| [`Doc/Guide-implementation_AIUP-Claude-Code.md`](Doc/Guide-implementation_AIUP-Claude-Code.md) | Méthode : AIUP adapté aux bancs de réfutation, réglages Claude Code, cycle de travail par cas d'utilisation |
| [`Doc/DECISION.md`](Doc/DECISION.md) | Journal des décisions D-01 à D-38 : écarts assumés, conception du harnais, statistiques, clôture |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md) | Campagne de référence : premiers verdicts (H-001 à H-006) et audit contradictoire |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-3.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-3.md) | Première épreuve de la seconde génération (H-007 à H-013) |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-4.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-4.md) | Première épreuve de H-012 sur une matrice à réplicats |
| [`Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-5.md`](Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-5.md) | Première épreuve de H-013 et démonstration de l'attestation de quiétude |
| [`Revue/REVUE-PRELANCEMENT_2026-09-10.md`](Revue/REVUE-PRELANCEMENT_2026-09-10.md) | Revue du cadrage avant le développement |
| [`Revue/REVUE-C008_2026-09-10.md`](Revue/REVUE-C008_2026-09-10.md) | Revue contradictoire des capacités ajoutées pour la seconde génération d'hypothèses |
| [`EscapeBench/`](EscapeBench/) | Le banc : spécification (`docs/`), code Go (`cmd/`, `internal/`), réglages de l'agent (`CLAUDE.md`, `.claude/`), résultats (`results/`), procédure (`LANCEMENT.md`) |

Les campagnes finales `C-2026-09-10-11` et `C-2026-09-10-12`, dont viennent les treize verdicts, n'ont pas de rapport propre : elles sont consolidées dans le rapport final.

Ordre de lecture suggéré : le rapport final, puis [`EscapeBench/docs/`](EscapeBench/docs/) dans l'ordre AIUP (vision, exigences, modèle d'entités, cas d'utilisation). Pour cadrer un nouveau projet : `Doc/Projets-candidats…` §1–5 et §7, puis `Doc/Guide-implementation…` §1–7.

Conventions : prose en français, identifiants et code en anglais; pages citées = folios imprimés; marqueurs épistémiques *Confirmé*, *Probable*, *Hypothèse*, *À vérifier* et *Adaptation* dans les documents de cadrage.

## Références

Les trois ouvrages sont versionnés dans [`Book/`](Book/).

Marco, E. (2026). *Agentic Coding with Claude Code: The everyday developer's guide to agentic coding with Claude Code*. Packt Publishing. Première publication en mars 2026 (© 2025). ISBN 978-1-80602-259-5. Fichier : [`Book/Agentic_Coding_with_Claude_Code.pdf`](Book/Agentic_Coding_with_Claude_Code.pdf).

Martinelli, S. (2026). *Spec-Driven Development: From Specs to Code with AI Agents*. Apress (Apress Pocket Guides). ISBN 979-8-8688-2850-8 (imprimé), 979-8-8688-2851-5 (numérique). https://doi.org/10.1007/979-8-8688-2851-5. Fichier : [`Book/Spec-Driven Development.pdf`](Book/Spec-Driven%20Development.pdf).

Shahsavan, S. (2026). *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*. Apress. ISBN 979-8-8688-2369-5 (imprimé), 979-8-8688-2370-1 (numérique). https://doi.org/10.1007/979-8-8688-2370-1. Fichier : [`Book/Building_Enterprise_Projects_with_Go.pdf`](Book/Building_Enterprise_Projects_with_Go.pdf).
