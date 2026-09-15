# Évaluation académique du dépôt Prospection

**Objet évalué** : dépôt `agbruneau/Prospection`, branche `main`, commit `1aeaa50` (2026-09-14), arbre de travail propre.
**Date** : 2026-09-15.
**Évaluateur** : Claude Fable 5.1, agent de codage, sur mandat de l'auteur du dépôt. L'évaluation repose sur la lecture intégrale de la documentation (README, `Doc/`, `Revue/`, `Campagnes/`, `docs/` des deux bancs), la lecture partielle du code et l'exécution locale des contrôles de l'annexe A (poste Windows 11, go1.27.0). Elle tient lieu d'évaluation à blanc : elle ne remplace pas un jury, et l'évaluateur n'est pas indépendant de l'objet, puisque les bancs ont été construits par un agent de la même famille de modèles.
**Régime** : production. Chaque fait est marqué *vérifié* (lu ou exécuté pendant l'évaluation) ou *supposé* (déduit, ou cité de mémoire).

---

## 1. Gabarit d'évaluation

### 1.1 Nature du travail

Le dépôt contient trois choses de nature différente, qu'un même gabarit doit pouvoir juger :

1. **Deux études empiriques de réfutation** (EscapeBench, LeakLab) : des affirmations chiffrées d'un ouvrage de génie logiciel sont converties en hypothèses à critère gelé, puis éprouvées par des bancs de mesure. C'est une *recherche d'ingénierie* (conception et évaluation d'un artefact) doublée d'un *benchmarking* (mesure systématique sous protocole).
2. **Une étude de cas méthodologique** (QR2) : la conduite de ces études par un agent de codage sous *Spec-Driven Development*, sans groupe témoin, documentée par journaux de décisions, revues et audit.
3. **Des documents de cadrage** : cartographie des affirmations réfutables du livre, grille de sélection de huit projets, guide de méthode.

L'équivalent universitaire le plus proche est un **projet de recherche de deuxième cycle en génie logiciel** (rapport de projet ou mémoire à volet expérimental), avec artefact logiciel joint. C'est à cette aune que le travail est jugé, et non comme un simple livrable logiciel ni comme un article soumis à une conférence.

### 1.2 Gabarits considérés

| Gabarit | Ce qu'il apporte | Ce qui manque seul |
|---|---|---|
| Grille de mémoire de maîtrise (problématique, état de l'art, méthode, résultats, discussion, forme) | Couverture complète d'un travail de recherche, familière à un jury | Trop générique pour juger la rigueur d'un banc de mesure ou la qualité d'un artefact logiciel |
| ACM SIGSOFT Empirical Standards (Ralph et coll., 2020), volets *Engineering Research* et *Benchmarking* | Attributs essentiels et souhaitables propres à la recherche d'ingénierie et au benchmarking; suppléments *Open Science* et *Registered Reports* | Pas de pondération ni de note; ne juge pas la rédaction ni l'éthique |
| Badges d'artefact ACM (*Available*, *Functional*, *Reusable*, *Results Reproduced*) | Échelle reconnue pour la reproductibilité | Ne couvre que l'artefact |
| Revue de code contre les règles internes du projet (`CLAUDE.md`) | Mesure la cohérence entre les règles annoncées et le code livré | Interne au projet, sans valeur académique propre |

### 1.3 Grille retenue

Grille hybride : dix critères pondérés sur 100, dont les descripteurs sont empruntés aux quatre sources ci-dessus. Le poids reflète ce qui distingue un travail de recherche d'un simple projet logiciel : la méthode et les résultats pèsent la moitié de la note.

| # | Critère | Poids | Ce que l'évaluateur cherche | Source |
|---|---|---|---|---|
| C1 | Problématique, questions de recherche, objectifs | 8 | Questions explicites, réfutables, motivées; périmètre justifié; choix du sujet argumenté | Mémoire |
| C2 | État de l'art et positionnement | 10 | Travaux antérieurs sur le domaine (benchmarking, analyse d'échappement, détection de fuites, développement assisté par IA); ce que l'étude ajoute ou réplique | Mémoire; ES *Engineering* (comparaison à l'existant) |
| C3 | Méthodologie et rigueur expérimentale | 20 | Protocole écrit avant les données, opérationnalisation défendue, contrôle du bruit, isolation, statistiques adaptées, témoins, vérité terrain indépendante | ES *Benchmarking*; *Registered Reports* |
| C4 | Réalisation technique et qualité logicielle | 15 | Architecture, tests, couverture, analyse statique, garde-fous, hygiène du dépôt, respect des règles annoncées | ES *Engineering*; revue interne |
| C5 | Résultats, analyse et interprétation | 15 | Verdicts lisibles, distinction entre l'affirmation et son opérationnalisation, mécanismes expliqués, chiffres d'en-tête fidèles | Mémoire; ES *Benchmarking* |
| C6 | Validité et limites | 8 | Menaces à la validité interne, externe, de construit; ce qu'une confirmation établit et n'établit pas | ES (*threats to validity*) |
| C7 | Reproductibilité, traçabilité, science ouverte | 10 | Résultats archivés, provenance, identifiants déterministes, rejeu vérifié, procédure, licence, dépôt citable | Badges ACM; *Open Science* |
| C8 | Contribution méthodologique (QR2) | 6 | Ce que le cas établit sur le développement par agent sous spécification; mesures; honnêteté sur les écarts au processus | ES *Case Study* (attributs de rapport d'expérience) |
| C9 | Rédaction et communication | 4 | Structure, clarté, concision, cohérence entre documents, langue | Mémoire |
| C10 | Éthique, intégrité, transparence | 4 | Déclaration du rôle de l'IA, errata, droits d'auteur, absence de biais de présentation | Mémoire; *Open Science* |

### 1.4 Échelle

Chaque critère est noté sur son poids selon cinq niveaux : *insuffisant* (moins de 40 % du poids), *passable* (40 à 59 %), *bien* (60 à 74 %), *très bien* (75 à 89 %), *excellent* (90 % et plus). La note globale sur 100 est convertie en lettre selon une échelle indicative d'université québécoise : A+ 90 et plus, A 85 à 89, A- 80 à 84, B+ 77 à 79, B 73 à 76, B- 70 à 72, C+ 65 à 69, C 60 à 64, échec sous 60.

### 1.5 Ce que la grille ne juge pas

- La justesse des affirmations du livre source, ni la valeur de ce livre.
- La pertinence des six projets non réalisés (P2, P4 à P8), au-delà de la qualité de leur cadrage.
- Le coût du travail (temps, jetons, argent), que le dépôt ne consigne pas.

---

## 2. Synthèse

**Note globale : 77 / 100, soit B+.**

Le dépôt est un travail de recherche d'ingénierie d'une rigueur expérimentale rare pour le genre : critères de réfutation gelés et vérifiés par empreinte, hypothèses successeurs plutôt que retouches, témoins de sensibilité, attestation de quiétude de la machine, vérité terrain indépendante des détecteurs jugés, harnais de non-régression qui rejoue les verdicts archivés. Tout ce que le dépôt affirme sur son propre état a été vérifié et tient : suites de tests au vert sous détecteur de course, hooks fonctionnels, oracle du corpus au vert, résultats archivés et rejouables. La culture de l'erratum (H-006 dans EscapeBench, H-008 et H-009 dans LeakLab) est exemplaire.

Deux faiblesses le tiennent hors de la plage A. La première est académique : il n'y a **aucun état de l'art scientifique**. Les seules références sont trois ouvrages de praticiens; la littérature sur la métrologie des micro-benchmarks, sur la détection des fuites de goroutines et sur le développement assisté par agents est absente, alors que plusieurs des défauts découverts en cours de route (passage en registres, débit contre latence, plancher de bruit inter-binaires) y sont documentés depuis longtemps. La seconde touche l'intégrité du dépôt : **les trois ouvrages sous droit d'auteur ont été committés en PDF dans un dépôt public**, retirés de l'arbre, mais toujours présents dans l'historique.

S'y ajoutent des limites de validité externe (une machine hybride, Windows, une soirée, aucun rejeu sous Linux ni arm64), des chiffres d'en-tête qui comptent comme infirmations des verdicts que le rapport lui-même attribue à un défaut de mesure ou à une lecture stricte, et une question de recherche méthodologique (QR2) sans aucune mesure.

| Critère | Poids | Note | Niveau |
|---|---|---|---|
| C1 Problématique | 8 | 7 | très bien |
| C2 État de l'art | 10 | 4 | passable |
| C3 Méthodologie | 20 | 16 | très bien |
| C4 Réalisation technique | 15 | 12 | très bien |
| C5 Résultats et analyse | 15 | 12,5 | très bien |
| C6 Validité et limites | 8 | 7 | très bien |
| C7 Reproductibilité | 10 | 8 | très bien |
| C8 Contribution méthodologique | 6 | 4 | bien |
| C9 Rédaction | 4 | 3,5 | très bien |
| C10 Éthique et intégrité | 4 | 3 | très bien |
| **Total** | **100** | **77** | **B+** |

---

## 3. Évaluation par critère

### C1. Problématique, questions de recherche, objectifs (7 / 8)

**Établi (vérifié).** Le README pose deux questions : QR1, les affirmations du livre résistent-elles à une mesure au critère fixé d'avance; QR2, un processus piloté par la spécification permet-il de conduire l'étude avec un agent sans rompre la traçabilité. Le choix du sujet vient d'une grille à cinq critères sur huit projets (`Doc/Projets-candidats…`, §5), et l'ordre P1 puis P3 est justifié par la vitesse de la boucle de rétroaction et la réutilisation du vocabulaire. Chaque banc a une vision avec objectifs et hors-périmètre explicites.

**Analyse.** QR1 est réfutable et bien découpée en hypothèses numérotées, chacune rattachée à une page. Le document de cadrage marque ses degrés de certitude (*Confirmé*, *Probable*, *Hypothèse*, *À vérifier*), pratique rare et bienvenue. QR2 est posée comme une question mais n'est pas opérationnalisée : le README le reconnaît (« QR2 n'est pas mesurée »). Une question de recherche qu'on ne peut ni confirmer ni infirmer devrait être reformulée en objectif descriptif (« documenter un cas »), ce que le texte fait de facto sans le dire dans l'énoncé.

**Retenu.** Problématique claire et motivée; QR2 mal calibrée.

### C2. État de l'art et positionnement (4 / 10)

**Établi (vérifié).** Les références du README et des documents de cadrage se limitent à trois ouvrages (Shahsavan 2026, Martinelli 2026, Marco 2026), aux notes de version de Go et à quelques pages web. Aucun article scientifique n'est cité. Le document de cadrage écrit lui-même que « P1 réplique surtout des résultats connus de la littérature Go » (§9) sans citer cette littérature.

**Analyse.** C'est la lacune académique majeure. Trois domaines auraient dû être couverts :

- *Métrologie des micro-benchmarks.* Les pièges rencontrés et corrigés au fil des campagnes sont connus : biais de disposition mémoire et d'alignement entre binaires, nécessité de répliquer au niveau du processus et du binaire et non seulement de l'itération, séparation débit/latence, contrôle de la charge concurrente. La littérature sur l'évaluation statistiquement rigoureuse des performances (Georges, Buytaert et Eeckhout 2007; Mytkowicz et coll. 2009; Kalibera et Jones 2013; Hoefler et Belli 2015) les décrit et propose des remèdes (bootstrap multi-niveaux, randomisation de la disposition) que H-012 redécouvre partiellement. *Supposé* : ces références sont citées de mémoire, non consultées pendant l'évaluation.
- *Détection des fuites de goroutines.* Le profil `goroutineleak` jugé par H-013 de LeakLab découle de travaux publiés sur la détection de fuites par le ramasse-miettes (Saioc et coll., vers 2024 et 2025, *supposé*); les outils hors bibliothèque standard écartés par C-002 (`goleak`, analyses statiques) ont eux aussi une littérature. Sans ce positionnement, la « matrice de détectabilité » ne peut pas être comparée à ce qui est déjà connu.
- *Développement assisté par agents et préenregistrement.* QR2 touche deux champs actifs : l'évaluation des agents de codage et la pré-déclaration des hypothèses (Nosek et coll. 2018 sur le préenregistrement, *supposé*). Le dépôt applique un préenregistrement sans jamais le nommer ni le situer.

Le contraste est net : le travail est méthodologiquement au niveau de cette littérature, mais il ne le sait pas, et un lecteur ne peut pas dire ce qui est réplication, ce qui est confirmation indépendante et ce qui est neuf. Ce que l'étude apporte de propre (l'effet de disposition sur le passage en registres, le rôle de la minuterie d'alarme de `go test` dans `checkdead`, l'angle mort du profil sur les petits mutex) n'est pas mis en valeur faute de comparaison.

**Retenu.** Absence d'état de l'art scientifique; contribution non positionnée.

### C3. Méthodologie et rigueur expérimentale (16 / 20)

**Établi (vérifié).**

- *Gel des critères.* L'empreinte SHA-256 du texte des critères est enregistrée à la création de chaque campagne et revérifiée à chaque production de verdicts (EscapeBench BR-003-5, LeakLab C-008). Un test d'intégration recalcule l'empreinte de chaque campagne archivée contre le catalogue courant (D-47); il passe. Dans LeakLab, le gel est daté par un commit antérieur au code (`6ad40aa` puis `4ff35fd`).
- *Successeurs.* Sept hypothèses successeurs dans EscapeBench (H-007 à H-013), une dans LeakLab (H-014); aucun critère gelé n'a été retouché.
- *Témoins.* Témoin nul et témoin de sensibilité (H-007, H-012), témoin d'ordonnancement des bandes de cache (H-013), témoin qui échoue et bras témoins par sonde (LeakLab, D-11).
- *Contrôle de l'environnement.* Un processus `go test` par sujet; attestation de quiétude sur l'arbre de processus (C-010), démontrée par une campagne délibérément contaminée puis par une contamination accidentelle détectée (D-33, D-36); réplicats séparés par une passe complète de la matrice (C-009); délai `-test.timeout` transmis aux binaires après le défaut de la première campagne LeakLab (D-13).
- *Statistiques.* Différence des médianes sur 20 répétitions, intervalle à 95 % par bootstrap percentile à 2 000 rééchantillonnages, graine dérivée de la paire; `significant` si et seulement si l'intervalle exclut zéro (BR-004-2).
- *Vérité terrain.* L'oracle de LeakLab lit les piles de goroutines et non les détecteurs qu'il sert à juger (D-08); il passe sur 29 cas.
- *Classification non circulaire.* Le classificateur d'échappement ne lit jamais le profil de la cellule (D-06).

**Analyse.** Les forces sont réelles et peu communes. Les réserves suivantes tiennent la note sous le niveau excellent :

1. *Environnement de mesure.* Une seule machine, un portable à cœurs hybrides (performance et efficacité) sans épinglage d'affinité, sous Windows, en une soirée. Le rapport de la campagne de référence le signale (§4) mais le rapport final et le README ne le reprennent pas dans leurs limites. Ni la version du système, ni le plan d'alimentation, ni l'état du turbo ne figurent dans la provenance. Pour des effets de quelques centièmes de nanoseconde, c'est la première menace, et elle n'est traitée qu'indirectement par la quiétude et les réplicats.
2. *Bootstrap intra-binaire.* L'intervalle de confiance rééchantillonne les 20 répétitions d'un même processus; tout effet systématique entre les deux bras (alignement, cœur d'exécution) a un effectif de un et n'entre pas dans l'intervalle. Le rapport de campagne 1 (§4) le démontre : la médiane du bras pointeur varie d'un facteur 1,95 selon la taille du type alors qu'il fait le même travail. Seule H-012 répond, sur cinq cellules. H-001, H-002 et H-007 restent jugées sur des intervalles qui sous-estiment la variance.
3. *Comparaisons multiples.* Treize hypothèses, des dizaines de paires par série, aucune discussion du risque de faux positifs par multiplicité. Les critères exigent souvent deux tailles concordantes, ce qui atténue le risque sans le nommer.
4. *Préenregistrement séquentiel.* Les successeurs sont écrits après lecture des données de la campagne précédente. Le dépôt en tire la bonne conséquence (H-011 non concluante sur la campagne qui a précédé sa rédaction; réserve écrite sur H-014, dont le critère connaît l'ordre de grandeur du témoin), mais le mot juste, « préenregistrement séquentiel informé », n'est pas prononcé, et le lecteur doit reconstituer lui-même ce que chaque critère savait au moment de son gel.
5. *Détecteurs construits par l'auteur.* Dans LeakLab, NUMGOROUTINE est une construction du banc (K = 10 exécutions, seuil K/2, attente d'une seconde). H-012 confirme donc que l'outil du banc voit les fuites, plus que la pratique décrite par le livre. De même, H-002 opérationnalise « signaler » par la présence des mots `leak` ou `goroutine` dans la sortie, ce qui est défendable mais étroit.
6. *Revues « contradictoires ».* Les trente-cinq constats à trois vérificateurs, l'audit à dix-huit lecteurs et « 172 agents » du rapport de campagne 1 sont des revues par agents de modèle de langage. Le README parle de « sous-agents » sans dire qu'aucune revue humaine indépendante n'a eu lieu, ce que seul D-01 consigne. Une revue par agents est utile; elle n'est pas une revue par les pairs, et sa méthode (prompts, modèle, critères d'accord) n'est pas décrite.
7. *Répétitions de LeakLab.* Cinq répétitions par cellule, issue majoritaire. Suffisant pour des issues discrètes stables (aucune cellule instable observée), mais faible pour les sondes à grandeurs continues (H-006, H-009, H-011, H-014), rapportées par médiane de cinq valeurs.

**Retenu.** Protocole de très haute tenue, limité par l'environnement unique, l'estimation de la variance et l'absence de cadre statistique nommé.

### C4. Réalisation technique et qualité logicielle (12 / 15)

**Établi (vérifié).**

| Contrôle | EscapeBench | LeakLab |
|---|---|---|
| `go vet ./...` | propre | propre |
| `go test -race -shuffle=on -count=1 ./...` | 12 paquets au vert | 6 paquets au vert |
| Suite d'intégration (`-tags=integration_test`) | au vert | s. o. |
| `gofmt -l` | vide | vide |
| Couverture par paquet | cmd 77,9 %; store 83,1 %; dashboard 89,6 %; harness 90,4 %; system 90,8 %; cli 92,0 %; specs 93,1 %; service 93,6 %; gotool 93,7 %; escape 96,3 %; models 96,6 % | cmd 34,3 %; campaign 49,4 %; results 55,6 %; spec 88,8 %; ctxvet 91,8 %; verdict 98,6 % |
| Oracle du corpus | s. o. | au vert, 29 cas |
| `selftest.sh` des hooks | tous les contrôles passent | s. o. |
| Fonctions de test | 293, dont 125 nommées `TestUC###_` | 45, dont 21 `TestUC###_` et 14 `TestH###` |
| Volume Go (tests compris) | 19 669 lignes | 3 843 lignes |
| Dépendances hors bibliothèque standard | aucune | aucune |

Architecture hexagonale respectée dans EscapeBench (`models` sans tags, `service` sans I/O, ports côté consommateur); layout plat assumé dans LeakLab (D-05), avec l'argument du livre lui-même contre les interfaces à implémentation unique. Écritures atomiques, résultats immuables, identifiants de matrice déterministes verrouillés par test. Hooks à échec fermé (sans `jq`, la garde refuse), couverts par un contrôle qui exerce les chemins POSIX et Windows. L'audit du 2026-09-12 a démontré trois défauts bloquants et vingt majeurs, tous corrigés sauf le lot 9, consigné comme dette (D-50).

**Analyse.** Les réserves :

1. *Règles internes non suivies.* `CLAUDE.md` d'EscapeBench impose `testing/synctest` « pour tout comportement temporel ou concurrent, jamais de `time.Sleep` ». Aucun fichier de test d'EscapeBench n'importe `synctest`, et `quietude_test.go` (ligne 96) appelle `time.Sleep`. La règle est donc lettre morte, ce qu'un audit à dix-huit lecteurs n'a pas relevé.
2. *Couverture de LeakLab.* Trois paquets du module principal sont sous 60 %, dont `campaign`, qui porte l'orchestration des processus. Le journal de décisions de LeakLab ne rapporte aucune couverture, alors qu'EscapeBench en fait un indicateur de clôture.
3. *Intégration continue retirée* (D-53, à la demande du chercheur). Il ne reste aucune vérification hors du poste Windows. Or la seule exécution de la CI avait trouvé un défaut invisible sous Windows. Les cas d'utilisation continuent de nommer un acteur « Pipeline CI » qui n'existe plus.
4. *Hygiène du dépôt.* Messages de commit hors convention (« Huge commit », « 16k Lignes », « resstructuration des répertoires »); deux répertoires `.github/workflows` vides laissés en place; branche distante `claude/audit-md-implementation-cctjkk` non supprimée après fusion; historique réutilisé d'un dépôt antérieur sans rapport (commits de 2026-08-30 et 2026-09-06).
5. *Cohérence entre bancs.* LeakLab nomme un test par hypothèse (`TestH001` à `TestH014`); EscapeBench n'a aucun `TestH###`, ses évaluateurs étant testés sous `TestUC005_*`. Ce n'est pas une faute, mais les deux conventions coexistent sans être documentées.

**Retenu.** Logiciel solide, vérifiable et vérifié; écarts aux règles annoncées et couverture inégale.

### C5. Résultats, analyse et interprétation (12,5 / 15)

**Établi (vérifié).** Vingt-sept verdicts (13 + 14), chacun avec rationale, fichiers cités et campagne. Les lectures du rapport final distinguent systématiquement l'affirmation du livre de son opérationnalisation : H-004 (débit) contre H-008 (latence); H-003 (0 vers 1 allocation) contre H-010 (base variable); H-005 (tolérance large) contre H-011 (tolérance du livre). Les mécanismes avancés sont vérifiés ou marqués comme non vérifiés : passage en registres démontré par contre-épreuve à taille égale; `checkdead` cité avec ses lignes dans `proc.go`; allocateur *tiny* explicitement « non vérifié dans le code ».

**Analyse.**

1. *Chiffres d'en-tête.* Le résumé annonce « cinq infirmées, huit confirmées » et « huit infirmées sur quatorze ». Or le rapport de LeakLab attribue lui-même l'infirmation de H-009 à une sonde défectueuse; H-004 et H-011 d'EscapeBench infirment une opérationnalisation que le rapport juge inadéquate ou une tolérance de lecture. Un lecteur du seul résumé retient un score de réfutation que le corps du texte ne soutient pas. La forme honnête serait de compter à part les infirmations d'artefact et les infirmations de lecture stricte.
2. *Confirmations creuses.* H-008 et H-013 ne pouvaient pas être infirmées sur cette machine (latence de 137 ns contre un seuil de 100); H-003 est vraie par construction du harnais (0 vers 1); H-006 juge un corpus qui ne peut produire que les quatre causes du livre. Le rapport le dit (§5.1 du README, section « ce que les confirmations valent »), ce qui est à son honneur, mais ces verdicts restent comptés au même titre que les autres.
3. *Exclusion favorable au livre.* H-012 écarte la cellule de 8 octets à champ pointeur, seule cellule qui infirmait, au motif que son unique champ est le pointeur. La décision est déclarée avant mesure et motivée; elle reste une exclusion post-hoc par rapport à H-007, dont H-012 est la successeur, et l'écart mesuré y reste inexpliqué. Le désassemblage des deux bras, nommé comme travail futur, aurait dû précéder la clôture.
4. *Interprétation solide ailleurs.* La lecture de H-005 et H-008 de LeakLab (la bulle `synctest` n'attend que ce que sa documentation nomme; l'interblocage est fatal pour un programme, pas sous `go test`) est précise, sourcée et utile au praticien. La recommandation d'intégration continue tirée de la matrice est la contribution la plus directement transférable du dépôt.

**Retenu.** Analyse fine et honnête dans le corps; en-têtes trop favorables au récit de la réfutation.

### C6. Validité et limites (7 / 8)

**Établi (vérifié).** Le README (§6) traite validité externe, de construit et interne, la provenance du rejeu de H-006 et l'absence de revue humaine. Le catalogue d'EscapeBench porte une section « ce que H-012 et H-013 renoncent à éprouver », écrite avant les mesures. LeakLab nomme ses limites (corpus synthétique, oracle muet sur les courses et l'accessibilité, détecteurs idéalisés).

**Analyse.** Il manque, dans les limites finales : les cœurs hybrides sans affinité et le gouverneur de fréquence (présents seulement dans le rapport de campagne 1); la résolution de l'horloge Windows, qui rend H-006 de LeakLab une borne plutôt qu'une mesure (D-15 le dit, le README non); le risque de multiplicité; et la nature automatisée des revues comme menace à la validité de conclusion.

**Retenu.** Limites franches et bien hiérarchisées; quatre omissions.

### C7. Reproductibilité, traçabilité, science ouverte (8 / 10)

**Établi (vérifié).** Résultats archivés dans Git (dix campagnes EscapeBench, deux campagnes LeakLab, verdicts, classifications), jamais réécrits. Identifiants de matrice déterministes, paramètres des deux campagnes de clôture consignés dans `LANCEMENT.md` et vérifiés comme redonnant les mêmes identifiants et la même empreinte de harnais. Harnais de non-régression qui rejoue les évaluateurs sur les verdicts archivés : exécuté pendant l'évaluation, au vert. Procédure de rejeu complète pour les deux bancs. Licences MIT (code) et CC BY 4.0 (prose et résultats). Provenance par campagne : version de Go, système, architecture, processeur, tailles de cache, `GOMAXPROCS`, date.

Sur l'échelle des badges ACM (*supposé* quant au libellé exact) : *Available* acquis; *Functional* acquis (chaîne exécutable et testée); *Reusable* acquis pour EscapeBench (procédure d'extraction, skills, paramètres) et partiel pour LeakLab; *Results Reproduced* non établi, aucune campagne n'ayant été rejouée par un tiers ni sur une autre machine.

**Analyse.** Les manques : pas d'étiquette de version ni de DOI pour l'état évalué; sortie brute de `go test` et valeurs de `b.N` non archivées (reconnu dès la campagne 1, jamais corrigé); paramètres de la matrice `M-8f03757ac206` perdus, quatre verdicts non rejouables (reconnu); provenance sans version du système ni état d'alimentation; aucun rejeu sous Linux malgré une chaîne d'outils disponible en téléchargement; CI retirée. Les matrices ne sont pas versionnées, ce qui est défendable (sources générées, reconstruction prouvée par identifiant) et bien argumenté (D-42).

**Retenu.** Traçabilité exemplaire à l'intérieur du dépôt; reproduction externe non tentée.

### C8. Contribution méthodologique, QR2 (4 / 6)

**Établi (vérifié).** Le README (§5.3, §8.5) rapporte le cas sans le survendre : le gel a empêché la rationalisation après coup; les pouvoirs sont séparés par des hooks; la garde a mordu contre son auteur; un verdict faux a été publié puis corrigé par erratum; l'approbation des cas d'utilisation, réservée à l'humain par le processus, a été assumée par l'agent (D-01). La leçon de H-006 a été transférée à LeakLab (oracle indépendant), et LeakLab n'a publié aucun verdict faux.

**Analyse.** Le cas est documenté avec une honnêteté qui vaut à elle seule une bonne part de la note. Mais rien n'est mesuré : ni le nombre de sessions, ni la durée, ni les jetons, ni le coût, ni le nombre de constats par type de revue. Les transcriptions et les prompts ne sont pas archivés; le processus agentique n'est donc pas reproductible, à la différence des mesures. L'écart D-01 retire au cas une part de ce qu'il voulait montrer : sans revue humaine, le processus étudié n'est plus l'AIUP mais une variante entièrement automatisée, ce que le titre du dépôt ne dit pas. Enfin, l'audit qui a trouvé le verdict faux est lui-même conduit par agents, si bien que la conclusion « l'agent n'aurait pas détecté seul ses défauts » demande une nuance : ce sont d'autres agents, autrement instruits, qui les ont trouvés.

**Retenu.** Rapport d'expérience honnête et instructif; aucune donnée, processus non reproductible.

### C9. Rédaction et communication (3,5 / 4)

**Établi (vérifié).** Prose française soignée, terminologie constante (entités reprises telles quelles des spécifications au code), README structuré comme un article (résumé, questions, notions, méthode, résultats, discussion, limites, travaux futurs, reproduction). Les documents se citent mutuellement avec des liens exacts.

**Analyse.** Trois réserves de forme : les critères de H-012 et H-013 occupent chacun environ trois mille caractères dans une cellule de tableau, ce qui nuit à la lecture et masque des degrés de liberté qu'un critère court exposerait; les notes de clôture sont recopiées mot pour mot dans les cinq cas d'utilisation d'EscapeBench; `Doc/AUDIT.md` (144 Ko, 75 constats sans contre-vérification) est difficile à parcourir. La mention de « 172 agents » dans le rapport de campagne 1 n'est expliquée nulle part.

**Retenu.** Excellente tenue, quelques lourdeurs.

### C10. Éthique, intégrité, transparence (3 / 4)

**Établi (vérifié).** Le rôle de l'agent est déclaré : sept commits signés « Claude » le 2026-09-12, décisions D-01 dans les deux journaux, README explicite sur l'absence de revue humaine. Les errata sont datés et motivés plutôt que substitués en silence. Les campagnes contaminées ou non conformes sont conservées et étiquetées. Les licences séparent code et prose.

**Analyse.** Un manquement sérieux : les PDF des trois ouvrages (Shahsavan, Martinelli, Marco) ont été committés dans le dépôt public les 2026-09-08 et 2026-09-10, retirés de l'arbre les 2026-09-09 et 2026-09-13, et sont toujours dans l'historique (`git log --all -- '*.pdf'`). Le README affirme que « les ouvrages ne sont pas distribués avec le dépôt », ce qui est vrai de l'arbre et faux du dépôt. Pour un travail académique, c'est une faute de diffusion d'œuvres protégées, indépendante de l'intention. Un second point, mineur : les revues par agents sont désignées par des termes (« revue contradictoire », « audit », « vérificateurs indépendants ») que le lecteur académique associe à des humains.

**Retenu.** Transparence remarquable sur le processus; manquement au droit d'auteur dans l'historique.

---

## 4. Conformité aux standards empiriques

Résumé de mémoire des attributs des ACM SIGSOFT Empirical Standards (*supposé* quant à la formulation exacte). *S* satisfait, *P* partiel, *A* absent.

| Attribut | État | Renvoi |
|---|---|---|
| **Engineering Research** : décrit l'artefact et sa conception | S | `docs/` des deux bancs, modèle d'entités, cas d'utilisation |
| Justifie le besoin de l'artefact | S | README §1, cadrage §4 et §5 |
| Évalue l'artefact sur des scénarios réalistes | P | Types synthétiques et corpus minimal, assumés; aucune base de code réelle |
| Compare à l'existant | A | Aucun outil ni étude antérieure comparés (C2) |
| Discute hypothèses et limites | S | README §6, catalogue, rapports finaux |
| Artefact disponible | S | Dépôt public, licences |
| **Benchmarking** : justifie le choix des sujets et des métriques | S | BR-001-4, C-003, C-006, C-007 |
| Décrit l'environnement en détail | P | Provenance sans version du système, sans affinité ni fréquence |
| Répétitions et statistiques adaptées | P | 20 répétitions et bootstrap (EscapeBench); 5 répétitions (LeakLab); variance inter-binaire partielle |
| Contrôle des facteurs de confusion | P | Quiétude, isolation; pas d'affinité, machine hybride |
| Rapporte les résultats bruts | P | Vingt valeurs par sujet archivées; `b.N` et sortie brute absents |
| Menaces à la validité | S | README §6 |
| **Registered Reports** : hypothèses et critères fixés avant les données | S | Empreintes gelées, commit antérieur au code (LeakLab) |
| Distingue analyses préenregistrées et exploratoires | P | Contre-épreuves hors campagne signalées; successeurs informés non nommés comme tels |
| **Open Science** : données, code et procédure ouverts | S | Dépôt, `LANCEMENT.md`, README §9 |
| Dépôt citable et figé | A | Ni étiquette, ni release, ni DOI |

---

## 5. Registre des constats

Tous les constats, y compris mineurs. Gravité : **M** majeur (pèse sur la note d'un critère d'au moins un niveau), **m** modéré, **f** faible. Chaque constat est *vérifié* sauf mention.

| # | G | Critère | Constat |
|---|---|---|---|
| E-01 | M | C2 | Aucune référence scientifique; contribution non positionnée par rapport aux travaux sur la métrologie des benchmarks, la détection de fuites et le développement par agents. |
| E-02 | M | C10 | Trois ouvrages sous droit d'auteur committés en PDF dans le dépôt public (commits `22b73ac`, `00a8d21`, `a144f68`, `5c559e3`), retirés de l'arbre (`f83f1f6`, `4c025f3`), présents dans l'historique. Le README affirme qu'ils ne sont pas distribués. |
| E-03 | M | C3, C6 | Une seule machine (portable à cœurs hybrides, sans affinité), un système (Windows), une soirée; aucun rejeu sous Linux ni arm64 malgré la contrainte C-006. |
| E-04 | M | C3, C8 | Revues, audit et vérificateurs sont des agents de modèle de langage; leur méthode n'est pas décrite et le README ne le dit pas explicitement. |
| E-05 | M | C8 | QR2 sans aucune mesure (sessions, durée, jetons, coût); transcriptions et prompts non archivés. |
| E-06 | m | C5 | Décomptes d'en-tête : H-009 (LeakLab) infirmée par une sonde défectueuse, H-004 et H-011 (EscapeBench) infirmées sur une opérationnalisation ou une tolérance, comptées comme les autres. |
| E-07 | m | C5 | Confirmations non infirmables sur ce matériel (H-008, H-013) ou par construction (H-003, H-006), reconnues mais comptées à égalité. |
| E-08 | m | C3 | Intervalle de confiance rééchantillonné à l'intérieur d'un seul processus par bras; variance inter-binaire non captée hors H-012. |
| E-09 | m | C3 | Aucune considération des comparaisons multiples. |
| E-10 | m | C3 | Préenregistrement séquentiel informé par les données antérieures, non nommé; H-014 écrite en connaissant l'ordre de grandeur du témoin (réserve consignée). |
| E-11 | m | C4 | Couverture LeakLab : `cmd` 34,3 %, `campaign` 49,4 %, `results` 55,6 %; non rapportée dans le journal de décisions. |
| E-12 | m | C4 | Règle `testing/synctest` de `CLAUDE.md` jamais appliquée dans EscapeBench; `time.Sleep` dans `internal/adapters/gotool/quietude_test.go:96`. |
| E-13 | m | C4, C7 | CI retirée (D-53) : plus aucune vérification hors du poste; acteur « Pipeline CI » toujours nommé dans les cas d'utilisation. |
| E-14 | m | C7 | Provenance sans version du système d'exploitation, plan d'alimentation, état du turbo ni type de cœur. |
| E-15 | m | C3, C5 | NUMGOROUTINE est un détecteur construit par le banc (K = 10, seuil K/2, attente 1 s); H-012 de LeakLab juge cet outil plus que la pratique du livre. |
| E-16 | m | C8 | Approbation des cas d'utilisation par l'agent (D-01) : le processus étudié n'est plus l'AIUP tel que décrit par Martinelli. |
| E-17 | m | C7 | Paramètres de `M-8f03757ac206` perdus; quatre verdicts archivés non rejouables (reconnu, D-42). |
| E-18 | m | C5 | H-012 écarte la seule cellule qui infirmait; motivée et déclarée, mais l'écart reste inexpliqué faute de désassemblage. |
| E-19 | m | C7 | Sortie brute de `go test` et `b.N` par répétition non archivés (reconnu dès la campagne 1). |
| E-20 | f | C9 | Critères de H-012 et H-013 d'environ trois mille caractères dans une cellule de tableau. |
| E-21 | f | C9 | Note de clôture recopiée mot pour mot dans les cinq UC d'EscapeBench. |
| E-22 | f | C4 | Messages de commit hors convention : « Huge commit », « 16k Lignes », « resstructuration des répertoires », « Update audit.md ». |
| E-23 | f | C4 | Répertoires `.github/workflows/` vides à la racine et dans `EscapeBench/`. |
| E-24 | f | C4 | Branche distante `claude/audit-md-implementation-cctjkk` conservée après fusion. |
| E-25 | f | C7 | Aucune étiquette, release ni DOI pour l'état évalué. |
| E-26 | f | C9 | `Doc/AUDIT.md` (144 Ko) : 75 constats sans contre-vérification, navigation difficile. |
| E-27 | f | C9 | « 172 agents » (rapport de campagne 1) sans description. |
| E-28 | f | C4 | Hooks dépendants de `bash` et `jq` sous Windows; documenté, mais fragile pour un tiers. |
| E-29 | f | C7 | Identifiant de processeur générique dans la provenance de LeakLab (« Intel64 Family 6 Model 198 ») là où EscapeBench consigne le modèle commercial. |
| E-30 | f | C6 | Résolution d'horloge Windows : H-006 de LeakLab borne la durée sans la mesurer (D-15); absent des limites du README. |
| E-31 | f | C4 | Historique du dépôt réutilisé d'un projet sans rapport (commits du 2026-08-30 au 2026-09-06). |
| E-32 | f | C4 | Conventions de nommage des tests d'hypothèses différentes entre les deux bancs, non documentées. |
| E-33 | f | C1 | Le README annonce « six autres projets au stade du cadrage »; seules des fiches existent, sans noyau de spécification. |
| E-34 | f | C6 | Le résumé ne dit pas que les infirmations de H-004 et H-011 portent sur une lecture stricte; il faut lire §4.3 et §5.1. |

---

## 6. Recommandations

Par ordre de gain sur la note.

1. **Ajouter un état de l'art** (C2, E-01) : une section de deux à trois pages sur la métrologie des micro-benchmarks, la détection des fuites de goroutines et le préenregistrement, puis une section « ce que l'étude réplique, confirme ou ajoute ». C'est le seul changement qui, seul, porterait la note en zone A-.
2. **Purger les PDF de l'historique** (C10, E-02) avec `git filter-repo`, puis pousser de force et le consigner dans une décision. Cette opération réécrit un historique public : à décider par l'auteur, pas par un agent.
3. **Rejouer sous Linux, puis sur arm64** (C3, C7, E-03), en consignant version du système, affinité et gouverneur de fréquence, et en épinglant les sujets sur un type de cœur.
4. **Recompter les verdicts** (C5, E-06, E-07) : présenter les infirmations d'artefact, de lecture stricte et de fond sur trois lignes, dans le résumé.
5. **Nommer et décrire les revues par agents** (C3, C8, E-04) : modèle, prompts, règle d'accord, et ce qu'elles ne remplacent pas.
6. **Mesurer QR2** (C8, E-05) : sessions, durée, jetons, constats par source; archiver les prompts des skills et les transcriptions des revues.
7. **Rétablir une vérification hors poste** (C4, C7, E-13), ou à défaut documenter un rejeu manuel sous Linux avec sa date et son commit.
8. **Aligner les règles et le code** (C4, E-12) : soit appliquer `synctest` aux tests temporels, soit retirer la règle.
9. **Relever la couverture de LeakLab** (C4, E-11) sur `campaign`, `results` et `cmd`, et la rapporter.
10. **Figer et citer** (C7, E-25) : étiquette `v1.0-final`, release GitHub, dépôt Zenodo avec DOI.
11. **Archiver la sortie brute** (C7, E-19) de `go test` par sujet, ou au moins `b.N` par répétition.
12. **Hygiène** (E-22 à E-24, E-31) : supprimer la branche distante et les répertoires vides; pour l'avenir, tenir la convention de message de commit du projet.

---

## 7. Note finale

**77 / 100, B+.** Travail de recherche d'ingénierie dont la rigueur expérimentale, la traçabilité et l'honnêteté des errata sont de niveau A; note tenue en B+ par l'absence d'état de l'art, la diffusion d'œuvres protégées dans l'historique public, l'environnement de mesure unique et une question méthodologique restée sans mesure. Les recommandations 1 à 4 suffiraient à porter l'ensemble en zone A-.

---

## Annexe A. Contrôles exécutés pendant l'évaluation

Tous exécutés le 2026-09-15 sur le commit `1aeaa50`, poste Windows 11, `go1.27.0 windows/amd64`, `jq-1.8.2`.

| Contrôle | Résultat |
|---|---|
| `git log`, `git ls-files`, `git log --all -- '*.pdf'` | 68 commits du 2026-08-30 au 2026-09-14; 2 256 fichiers suivis, dont 205 hors répertoires de résultats; deux auteurs (André-Guy Bruneau, Claude); trois PDF d'ouvrages ajoutés puis retirés, présents dans l'historique |
| `EscapeBench` : `go vet ./...` puis `go test -race -shuffle=on -count=1 -cover ./...` | propre; 12 paquets au vert; couverture au tableau de C4 |
| `EscapeBench` : `go test -race -shuffle=on -count=1 -tags=integration_test -timeout 300s ./...` | au vert (harnais de non-régression des verdicts archivés) |
| `EscapeBench` : `bash .claude/hooks/selftest.sh` | « hooks : tous les contrôles passent » |
| `LeakLab` : `go vet ./...` puis `go test -race -shuffle=on -count=1 -cover ./...` | propre; 6 paquets au vert; couverture au tableau de C4 |
| `LeakLab/lab` : `go test -count=1 ./corpus` | oracle au vert |
| `gofmt -l` sur les deux bancs | vide |
| Grep des règles internes | aucun `panic(` hors tests dans EscapeBench; aucun import de `synctest` dans ses tests; un `time.Sleep` dans un test; `-test.timeout=10m0s` transmis par LeakLab (`campaign.go:34`) |
| Décompte des tests | EscapeBench 293 fonctions (125 `TestUC###_`), LeakLab 45 (21 `TestUC###_`, 14 `TestH###`) |
| Lignes Go | EscapeBench 19 669 (hors matrices générées), LeakLab 3 843 |
| Cohérence des décomptes du README | 5 + 8 = 13 et 8 + 6 = 14 vérifiés contre les tableaux; 960 + 64 + 75 = 1 099 mesures LeakLab |

Non exécuté : les campagnes de mesure elles-mêmes (29 minutes à 1 h 08 pour EscapeBench, 9 minutes pour LeakLab). L'évaluation se fie aux fichiers archivés et au harnais de non-régression qui les rejoue.

## Annexe B. Références

Références internes au dépôt : lues intégralement pendant l'évaluation.

Références externes : citées de mémoire, non consultées pendant l'évaluation; à vérifier avant tout usage dans le dépôt.

- Ralph, P. et coll. (2020). *Empirical Standards for Software Engineering Research*. ACM SIGSOFT. arXiv:2010.03525.
- Georges, A., Buytaert, D., Eeckhout, L. (2007). *Statistically Rigorous Java Performance Evaluation*. OOPSLA.
- Mytkowicz, T., Diwan, A., Hauswirth, M., Sweeney, P. F. (2009). *Producing Wrong Data Without Doing Anything Obviously Wrong!* ASPLOS.
- Kalibera, T., Jones, R. (2013). *Rigorous Benchmarking in Reasonable Time*. ISMM.
- Hoefler, T., Belli, R. (2015). *Scientific Benchmarking of Parallel Computing Systems*. SC.
- Nosek, B. A. et coll. (2018). *The Preregistration Revolution*. PNAS.
- Saioc, G.-V. et coll. (vers 2024 et 2025). Travaux sur la détection de fuites de goroutines par analyse dynamique et par le ramasse-miettes, à l'origine du profil `goroutineleak`.
- ACM. *Artifact Review and Badging*, version 1.1.
