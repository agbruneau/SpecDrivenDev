# QR2 mesurée après coup : sessions, agents et jetons

**Date :** 2026-09-22 · **Lot 4** du plan d'implantation de l'évaluation (constats E-05, E-04, E-27) · **Décision :** D-58

QR2 demande si un processus piloté par la spécification permet de conduire l'étude avec un agent sans rompre la traçabilité. Le dépôt n'avait publié aucune mesure du processus. Ce document en donne une, tirée après coup des transcriptions locales de Claude Code, avec ses trous. Il décrit un cas; il ne compare rien à rien, faute de groupe témoin.

## 1. Source et méthode

- **Source.** Les 51 transcriptions principales de `~/.claude/projects/C--Users-agbru-OneDrive-Documents-GitHub-Prospection/`, et les 2 183 transcriptions de sous-agents rangées à côté (1 781 pour les sessions du projet, dont 1 775 horodatées) (`<session>/subagents/`, dont `subagents/workflows/wf_*`). Les fichiers d'exécution de workflow (`<session>/workflows/wf_*.json`) donnent le nom de chaque revue et son décompte final d'agents.
- **Script.** [`Doc/outils/qr2-sessions.sh`](outils/qr2-sessions.sh), bash et `jq`, rejouable sur ses propres transcriptions. Il ne lit que des horodatages, des noms d'outils, des compteurs de jetons et des noms de modèle; il n'écrit aucun texte de message. Appel utilisé ici :
  `bash Doc/outils/qr2-sessions.sh ~/.claude/projects/C--Users-agbru-OneDrive-Documents-GitHub-Prospection /tmp/qr2`
  Exécuté deux fois le 2026-09-22 (environ 2 min 40 s par exécution); les agrégats du projet sont identiques d'une exécution à l'autre, seule la session en cours a grandi.
- **Définitions.**
  - *Tour* : un message humain, commande slash incluse; les notifications de tâches et les retours d'outils sont exclus.
  - *Temps actif* : somme des écarts de moins de 15 minutes entre deux enregistrements successifs d'une transcription principale (`QR2_IDLE_MIN`). C'est une estimation : le seuil est un choix, pas une mesure.
  - *Jetons d'entrée nouveaux* : `input_tokens + cache_creation_input_tokens`. La *lecture de cache* (`cache_read_input_tokens`) est comptée à part. Chaque réponse n'est comptée qu'une fois, dédoublonnée par identifiant de message.
- **Phases.** Chaque événement est classé par son horodatage (UTC) dans une fenêtre tirée de l'historique git. Les bornes sont écrites dans le script, et `QR2_PHASES` permet de les changer.

| Phase | Fenêtre UTC | Ce qui la borne |
|---|---|---|
| Cadrage | 2026-09-08 00:00 → 2026-09-10 13:54 | choix du projet, revue avant lancement; premier commit de code (`UC-001..UC-005`, 09:54 HAE) |
| EscapeBench | 2026-09-10 13:54 → 2026-09-11 16:00 | construction, campagnes, revues, clôture, restructuration du 2026-09-11 |
| Audit | 2026-09-11 16:00 → 2026-09-13 09:00 | lancement de l'audit du code, `Doc/AUDIT.md` |
| LeakLab | 2026-09-13 09:00 → 2026-09-14 00:00 | noyau, corpus, campagnes, clôture de LeakLab |
| Dépôt final | 2026-09-14 00:00 → 2026-09-15 10:00 | CI ajoutée puis retirée, décisions D-50 à D-53 |
| *Après l'évaluation* | depuis 2026-09-15 10:00 | évaluation académique, purge D-54, implantation : **comptée à part** |
| *Hors projet* | avant 2026-09-08 | 38 sessions d'autres travaux hébergés dans le même dossier : **exclues** |

## 2. Mesures (vérifiées, transcriptions locales)

### 2.1 Par phase

| Phase | Sessions | Tours | Appels d'outils | Temps actif | Sous-agents | Entrée nouvelle | Lecture de cache | Sortie |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cadrage | 6 | 18 | 420 | 167 min | 0 | 2,18 M | 62,0 M | 0,59 M |
| EscapeBench | 3 | 25 | 596 | 279 min | 398 | 24,45 M | 570,2 M | 3,05 M |
| Audit | 1 | 4 | 107 | 40 min | 1 377 | 22,48 M | 292,4 M | 2,54 M |
| LeakLab | 1 | 4 | 336 | 94 min | 0 | 0,57 M | 60,9 M | 0,35 M |
| Dépôt final | 2 | 4 | 138 | 40 min | 0 | 0,26 M | 17,9 M | 0,10 M |
| **Projet** | **11** | **55** | **1 597** | **620 min** | **1 775** | **49,95 M** | **1 003,3 M** | **6,62 M** |
| *Après l'évaluation (à part)* | 2 | 7 | 111 | 48 min | 6 | 1,44 M | 43,4 M | 0,36 M |

Les appels d'outils ne comptent que la session principale; les jetons additionnent la session principale et ses sous-agents. Deux sessions chevauchent deux phases (`ef21fc00` : cadrage et EscapeBench; `2f7ab848` : audit et dépôt final), d'où 11 sessions distinctes pour 13 lignes de session. La ligne « après l'évaluation » est un instantané du 2026-09-22 vers 10 h 15 UTC : la session d'implantation était encore ouverte.

Dans les 11 sessions du projet, **les jetons de sortie se répartissent ainsi** : 1,87 M pour l'agent principal (28 %), 4,01 M pour les trois revues par agents (61 %) et 0,75 M pour les trois workflows de rédaction contradictoire (11 %).

Sessions actives par jour UTC : 2026-09-08 : 2 · 09-09 : 2 · 09-10 : 3 · 09-11 : 2 · 09-12 : 1 · 09-13 : 1 · 09-14 : 2.

Modèles relevés : `claude-opus-5` dans dix des onze sessions; `claude-fable-5-1` dans cinq (cadrage, EscapeBench, audit), seul dans une session du cadrage; `claude-sonnet-5` dans une session du cadrage.

### 2.2 Par revue ou rédaction par agents

Les six exécutions de workflow du projet. Les prompts, rôles et règles d'accord sont archivés dans [`Revue/PROMPTS-REVUES_2026-09-10.md`](../Revue/PROMPTS-REVUES_2026-09-10.md).

| Workflow | Début (UTC) | Agents | Durée | Entrée nouvelle | Lecture de cache | Sortie | Constats |
|---|---|---:|---:|---:|---:|---:|---|
| `audit-campagne-escapebench` (« 172 agents ») | 09-10 14:33 | 172 | 44 min | 8,23 M | 117,7 M | 0,85 M | 19 retenus sur 54 |
| `rediger-h007-h011` | 09-10 15:55 | 80 | 16 min | 3,09 M | 27,3 M | 0,20 M | 5 hypothèses rédigées |
| `revue-c008` | 09-10 16:44 | 113 | 34 min | 6,43 M | 93,7 M | 0,72 M | 7 retenus sur 35 |
| `rediger-h012-h013` | 09-10 17:35 | 20 | 48 min | 2,82 M | 59,5 M | 0,32 M | 1 faille bloquante restante |
| `h012-h013-passe2` | 09-10 18:28 | 13 | 39 min | 2,33 M | 26,5 M | 0,23 M | version finale |
| `audit-escapebench` | 09-11 16:18 | 581 (1 377 transcriptions) | voir 2.3 | 21,49 M | 261,4 M | 2,44 M | 119 retenus sur 187 (fichier d'exécution) |

### 2.3 Ce que les transcriptions révèlent en plus

- **L'audit a été interrompu par les limites d'usage.** Sur 581 agents, 189 ont échoué : 168 sur la limite de dépense mensuelle, 21 sur la limite de session. Il y a eu quatre reprises, le 2026-09-12 entre 11 h 34 et 11 h 58 UTC. La seconde ronde n'a pas eu lieu, parce que le critique de complétude a lui-même échoué. Les 1 378 fichiers de transcription de sous-agents (1 377 horodatés) comprennent les tentatives de chaque reprise : 921 ne contiennent aucune réponse du modèle, 303 portent `claude-fable-5-1` et 154 `claude-opus-5`, alors que le fichier d'exécution déclare `claude-opus-5` pour les 581 agents. La durée de la dernière reprise est de 21 minutes; celle des précédentes n'est pas mesurable.
- **Les décomptes de l'audit ne concordent pas.** Le fichier d'exécution donne 187 constats après dédoublonnage et 119 retenus. `Doc/AUDIT.md` écrit 190 items, dont 102 confirmés, 4 rejetés et 84 non vérifiés. L'écart n'est pas expliqué ici.
- **Deux revues ont perdu des voix sur un refus du modèle.** Il s'agit d'un vérificateur de la revue « 172 agents » et de deux de `revue-c008`, rejetés par les garde-fous du modèle. La règle d'accord a alors porté sur les votes restants.
- **Aucun des *skills* ni des deux sous-agents de revue du banc n'a été invoqué dans les sessions locales du projet** : 0 appel `Agent`, 0 appel `Skill`, aucune commande `/spec-review` ni `/implement`. Les seules commandes slash sont `/model`, `/goal` et `workflow-authoring`. Toutes les revues par agents passent par l'outil `Workflow`.
- **Le projet compte 55 tours humains**, pour 1 597 appels d'outils de l'agent principal et 1 775 transcriptions de sous-agents.

## 3. Trous nommés

1. **Sessions en nuage du 2026-09-12.** Neuf commits signés « Claude », de 16 h 55 à 20 h 21 HAE, n'ont aucune transcription locale : l'implantation des lots de l'audit et l'erratum de H-006. La « contre-vérification indépendante » à huit agents que décrit `Doc/AUDIT.md` n'a pas non plus de trace locale. Ce travail est absent des mesures. Qu'il se soit déroulé en nuage est *supposé*, d'après l'auteur des commits.
2. **Avant le 2026-09-09.** Les deux sessions locales du 2026-09-08 sont comptées dans le cadrage. Les 38 sessions antérieures portent sur d'autres travaux et sont exclues. Rien n'indique qu'elles aient servi au projet, mais seule leur première demande a été lue pour les classer. Trois commits du 2026-09-10, entre 6 h 19 et 7 h 43 HAE (« Create Projets-candidats… », « Livables de base », « Pré SDD »), précèdent toute session locale de ce jour-là. Ils ont été faits hors de Claude Code local, *supposément* par l'interface de GitHub.
3. **Sessions lancées depuis `EscapeBench/`.** Deux dossiers de projet existent à ce nom. Ils ne contiennent que des scripts de workflow des sessions `ef21fc00` et `2f7ab848`, dont le répertoire courant avait changé, et aucune transcription (vérifié). Une session supprimée avant le 2026-09-22 ne laisserait pas de trace : son absence ne se prouve pas.
4. **Intégrité des transcriptions.** 46 des 51 fichiers portent la même date de modification (2026-09-17 16:34). La cause n'est pas établie. Les mesures reposent sur les horodatages internes, pas sur les dates de fichier, et le contenu des transcriptions n'est pas vérifiable par un tiers, puisqu'il n'est pas publié (D-58).
5. **Six transcriptions de sous-agents du projet** n'ont aucun horodatage : une de l'audit, cinq de la session EscapeBench. Elles ne sont pas comptées (1 775 sur 1 781 fichiers).
6. **Coût monétaire.** Il n'est pas calculé : il faudrait appliquer une grille de prix, ce qui serait une déduction et non une mesure.

## 4. Ce que les chiffres permettent de dire

| Question | Réponse | Statut |
|---|---|---|
| Coût d'une hypothèse, EscapeBench | 13 hypothèses; 21 min actives et 235 k jetons de sortie par hypothèse sur la phase EscapeBench seule; 34 min et 280 k avec le cadrage | déduit (division d'agrégats mesurés; le cadrage sert aussi le choix entre huit projets) |
| Coût d'une hypothèse, LeakLab | 14 hypothèses; 6,7 min actives et 25 k jetons de sortie par hypothèse, sans aucun sous-agent | déduit; borne haute, car la session a aussi révisé le README |
| Coût d'une revue | « 172 agents » : 0,85 M jetons de sortie, 44 min, soit 15,8 k par constat examiné; `revue-c008` : 0,72 M, 34 min, 20,4 k par constat; audit : 2,44 M, 13,1 k par constat | mesuré (agrégats); le ratio par constat est déduit |
| Part du travail en revue, en jetons | 61 % des jetons de sortie du projet vont aux trois revues, 11 % aux rédactions contradictoires, 28 % à l'agent principal | mesuré |
| Part du temps en revue | Sur les 659 min de la session EscapeBench (`ef21fc00`, de 12 h 35 à 23 h 34 UTC), les workflows ont tourné 181 min (27 %), dont 78 min de revue proprement dite (12 %) et 103 min de rédaction contradictoire. La durée de l'audit n'est pas mesurable (voir 2.3). | déduit (durées de workflow mesurées, rapportées au temps de mur de la session) |
| Effort humain | 55 tours humains sur tout le projet, et 4 par phase après EscapeBench | mesuré; le temps de lecture humain hors session n'est pas mesuré |

**Ce que les chiffres ne disent pas.** Ils ne disent pas si le processus a *préservé la traçabilité* : c'est la question de QR2, et elle reste qualitative (README §5.3). Ils ne disent pas non plus ce qu'aurait coûté la même étude sans spécification ni revues : il n'y a pas de groupe témoin. Enfin, l'écart entre EscapeBench et LeakLab (un facteur de 10 environ par hypothèse) ne s'attribue pas à une seule cause. LeakLab a réutilisé le vocabulaire et l'outillage, et n'a eu aucune revue par agents : la comparaison est confondue.
