# Planification d'exécution : implantation de l'évaluation académique

**Source** : [`EVALUATION-ACADEMIQUE_2026-09-15.md`](EVALUATION-ACADEMIQUE_2026-09-15.md), 34 constats (E-01 à E-34) et 12 recommandations (R1 à R12).
**État de départ** : `main` au commit `1eddb85` (2026-09-15, D-54), arbre propre, `origin/main` identique.
**Date** : 2026-09-15.
**But** : porter le dépôt de B+ (77) en zone A- (80 et plus) sans toucher à un critère gelé, à un fichier de résultats ni à un gabarit du harnais hors du lot prévu à cet effet.

Le format reprend celui de la planification des correctifs de [`AUDIT.md`](AUDIT.md) : des lots ordonnés par dépendance, chacun avec ses constats, ses actions, son exécutant et sa barre de sortie. Deux différences. Une partie des lots ne peut être exécutée que par le chercheur (réécriture d'historique publique, revue humaine, matériel, comptes externes, validation de références). Et la plupart des lots sont documentaires : ils changent ce que le dépôt dit, pas ce qu'il mesure.

---

## 0. Ce que la vérification préalable a établi

Faits vérifiés le 2026-09-15 sur le commit `1eddb85`, qui conditionnent le plan.

| Fait | Conséquence |
|---|---|
| La purge D-54 a réécrit `main` et `audit-md` (66 commits), mais `refs/original/refs/heads/main` et `refs/original/refs/heads/audit-md` (sauvegardes de `git filter-branch`) retiennent encore les anciens commits; le reflog compte 65 entrées; les trois paquets (87 Mo) contiennent toujours les objets des PDF. Sur GitHub, `refs/pull/1/head` pointe sur `818b566`, l'ancien historique, et la branche `claude/audit-md-implementation-cctjkk` existe encore (réécrite). | E-02 n'est pas clos : lot 0. |
| L'empreinte gelée d'EscapeBench couvre l'identifiant, l'énoncé et le critère de chaque hypothèse (`internal/adapters/specs/specs.go`, ligne 105); la colonne « source » du tableau n'y entre pas. Celle de LeakLab ne couvre que le critère. | La prose des « deux écarts » de H-012 et H-013, logée dans la colonne source, peut être déplacée sans invalider une campagne (E-20). Le texte des critères, lui, ne bouge pas. |
| 51 transcriptions locales de sessions Claude Code existent pour ce projet (`~/.claude/projects/C--Users-agbru-OneDrive-Documents-GitHub-Prospection/`, 144 Mo, du 2026-09-09 au 2026-09-15). Les sessions en nuage (commits signés « Claude » du 2026-09-12) n'y sont pas. | QR2 peut être mesurée après coup, avec des trous nommés (E-05); les prompts donnés aux sous-agents des revues y sont récupérables (E-04, E-27). |
| WSL2 porte go1.22.2, qui sait télécharger une chaîne d'outils par `GOTOOLCHAIN=go1.27.0`. | Le rejeu Linux ne demande aucune installation (E-03, E-13). Il tourne sur le même processeur virtualisé : il éprouve l'effet du système, pas celui de la machine. |
| Les matrices locales existent, dont `M-57477f022103` avec les cellules `Size0008FieldsPtr_LOCAL_*`; `go tool objdump` est disponible. | Le désassemblage demandé par E-18 est faisable sans nouvelle campagne. |

---

## 1. Règles qui gouvernent toute l'implantation

Elles reprennent celles de l'audit et en ajoutent trois propres à un travail déjà publié.

1. **La spécification d'abord.** Tout changement de comportement observable commence par une révision datée dans `docs/`, puis le code, dans le même commit.
2. **Un critère gelé ne se retouche pas.** Le test d'intégration de D-47 recalcule l'empreinte de chaque campagne archivée : il est la garde, et il doit rester au vert après chaque lot qui touche `docs/requirements.md`.
3. **Jamais d'écriture sous `results/`, `matrices/` ni `docs/dashboard.md`.** Les campagnes de rejeu (lot 8) produisent leurs fichiers par le binaire, lancé par le chercheur, jamais par un agent.
4. **Les gabarits du harnais sont à part.** Seul le lot 9 y touche, en une fois, avant la première campagne arm64 (D-50).
5. **Un chiffre publié ne change que par erratum daté.** Le recomptage des verdicts (lot 2) ajoute une lecture; il ne remplace ni un verdict ni un rationale.
6. **Ce qui est supposé reste marqué supposé** jusqu'à vérification en source primaire : aucune référence ne passe dans le README sans DOI ou URL consultée (lot 3).
7. **Une décision par écart.** Chaque lot qui tranche quelque chose que la spécification ou une décision antérieure laissait ouvert ajoute une entrée `D-##` dans `DECISION.md` ou `DECISION_LeakLab.md`; la numérotation reprend à D-55 et D-18.
8. **Messages de commit** : `UC-###:` ou `H-###:` pour le code des bancs, `LeakLab:` pour ce qui touche tout LeakLab, `Évaluation : lot N — …` pour les lots documentaires transversaux (E-22).
9. **Barre de sortie commune.** Depuis `EscapeBench/` : `go vet ./...`, `go test -race -shuffle=on -count=1 ./...`, `go test -race -shuffle=on -count=1 -tags=integration_test -timeout 300s ./...`, `gofmt -l ./cmd ./internal` vide, `bash .claude/hooks/selftest.sh`. Depuis `LeakLab/` : `go vet ./...`, `go test -race -shuffle=on -count=1 ./...`, `cd lab && go test -count=1 ./corpus`. Un lot documentaire n'est pas dispensé : une retouche de `requirements.md` peut déplacer une empreinte.

---

## 2. Décisions à prendre par le chercheur avant de lancer

Aucun lot ne les prend à sa place. Chaque réponse devient une entrée de décision.

| # | Question | Options | Lot concerné |
|---|---|---|---|
| Q1 | Réécrire l'historique une seconde fois pour retirer les commits sans rapport avec le projet (psaumes du 2026-08-30 et 2026-08-31, rapports agentiques du 2026-09-06 et 2026-09-07) ? | Oui, en même temps que l'achèvement de la purge (un seul `push --force`); ou non, et E-31 reste consigné comme accepté. | 0 |
| Q2 | Vérification hors poste : rétablir une CI minimale (revient sur D-53) ou tenir un journal de rejeu manuel daté ? | CI ou journal. Le plan suppose le journal, conforme à D-53. | 8 |
| Q3 | Règle `testing/synctest` d'EscapeBench : l'appliquer ou l'amender ? | Amender, en exemptant les tests qui mesurent des compteurs réels du système (recommandé : une bulle ne simule pas le temps processeur d'un processus enfant); ou réécrire les tests temporels. | 5 |
| Q4 | Publier des agrégats tirés des transcriptions personnelles de sessions (nombre de sessions, durées, jetons, prompts des sous-agents) ? | Oui, agrégats et prompts seulement, jamais les transcriptions; ou non, et E-05 reste ouvert. | 4 |
| Q5 | Étiquetage : `v1.0` sur l'état purgé (fin du lot 0) puis `v1.1` en fin de plan, avec DOI Zenodo sur `v1.1` ? | Oui; ou une seule étiquette en fin de plan. | 0, 11 |
| Q6 | Matériel arm64 : disponible, et quand ? | Conditionne le lot 9 et la seconde moitié de E-03. | 9 |
| Q7 | Revue humaine à froid des huit cas d'utilisation (D-01 des deux journaux) : le chercheur la fait lui-même, ou la délègue à un tiers ? | Un tiers rétablit la lettre de l'AIUP; le chercheur seul en rétablit l'esprit. | 1 |

---

## 3. Lots

Exécutant : **C** chercheur seul, **A** agent seul, **A→C** l'agent prépare et le chercheur exécute ou valide. Taille : S (une session courte), M (une session), L (plusieurs sessions ou une attente longue).

### Lot 0 — Achever la purge, nettoyer les références, figer l'état

**Exécutant : C.** Taille : S. Sans préalable; conditionne tout, parce qu'aucune étiquette ni aucune citation ne doit pointer sur un historique qui va encore bouger. Constats : E-02 (suite), E-23, E-24, E-31 (si Q1 = oui), E-25 (première moitié).

Actions, dans l'ordre :

1. Si Q1 = oui : réécrire une dernière fois pour retirer les six commits sans rapport (`76d5c45` à `7314bfe` et leurs retraits), avec `git filter-repo` de préférence à `filter-branch` (il nettoie lui-même les sauvegardes). Sinon, passer au point 2.
2. Supprimer les sauvegardes locales et purger les objets :
   ```bash
   git for-each-ref --format='%(refname)' refs/original/ | xargs -n1 git update-ref -d
   git reflog expire --expire=now --all
   git gc --prune=now --aggressive
   git log --all --diff-filter=A --name-only -- '*.pdf'
   ```
   La dernière commande ne doit plus lister que les PDF de l'auteur (ou rien, si Q1 = oui).
3. Supprimer la branche distante `claude/audit-md-implementation-cctjkk` et la branche locale `audit-md` : leur contenu est fusionné dans `main` depuis le 2026-09-13.
4. Demander au soutien de GitHub la suppression de `refs/pull/1/head` et l'exécution d'un ramassage côté serveur; consigner la date de la demande et de la réponse.
5. Cloner à neuf dans un répertoire vide et y rejouer le point 2, dernière commande : c'est la seule preuve que la purge est complète du point de vue d'un tiers.
6. Retirer les deux répertoires vides `.github/workflows/` (racine et `EscapeBench/`).
7. Si Q5 = oui : étiquette annotée `v1.0` sur le commit qui clôt ce lot, avec pour message l'état évalué (« dépôt final, historique purgé, évalué B+ le 2026-09-15 »).
8. Décision D-55 : ce qui a été retiré, ce qui reste (PDF de l'auteur), ce que GitHub a confirmé, et la mise à jour de l'entrée E-02 de l'évaluation (« corrigé » devient « clos » avec la date du clone de vérification).

**Sortie du lot.** Le clone neuf ne contient aucun objet des trois ouvrages; `git ls-remote origin` ne montre plus que `main` (et `v1.0`); D-55 écrite.

### Lot 1 — Revue humaine à froid des cas d'utilisation

**Exécutant : C** (ou un tiers, Q7). Taille : M. Sans préalable technique; à faire avant tout lot qui révise une spécification (5, 6, 10), parce que l'AIUP réserve l'approbation à l'humain et que D-01 des deux journaux dit que cette revue « reste la première chose à faire ». Constat : E-16.

Actions :

1. Pour chacun des huit cas d'utilisation (`EscapeBench/docs/use-cases/UC-001` à `UC-005`, `LeakLab/docs/use-cases/UC-001` à `UC-003`), lire à froid le fichier, ses exigences liées et ses entités, puis répondre par écrit aux trois tests d'exécutabilité de Martinelli (deux lecteurs, un testeur, une règle à inventer) et aux cinq questions de l'ordre de revue. Le sous-agent `spec-reviewer` peut être lancé d'abord (`/spec-review UC-00n`) pour fournir une première liste; la revue humaine doit dire ce qu'elle en retient et ce qu'elle rejette.
2. Consigner dans les « Notes de revue » de chaque fichier une entrée datée « Revue humaine à froid du 2026-09-xx : n constats, dont m retenus »; le statut reste `Deployed`.
3. Les constats retenus qui changent un comportement deviennent des révisions de spécification prises par les lots 5 et 6, jamais des retouches de critère.
4. Décision D-56 (EscapeBench) et D-18 (LeakLab) : la revue a eu lieu après coup; ce qu'elle a trouvé; en quoi le processus étudié par QR2 s'en trouve requalifié (« AIUP entièrement automatisé, puis revu »).
5. Mettre à jour le README §5.3 (« Un écart au processus est assumé ») pour dire que la revue a été faite après clôture et ce qu'elle a donné.

**Sortie du lot.** Huit fichiers de UC portent une entrée de revue humaine; deux décisions écrites; README §5.3 à jour.

### Lot 2 — Rédaction : décomptes, limites, revues par agents

**Exécutant : A.** Taille : M. Sans préalable. Constats : E-04, E-06, E-07, E-08, E-09, E-10, E-15, E-27, E-30, E-33, E-34. Recommandations : R4, R5 (volet texte).

Fichiers : `README.md`, `Doc/RAPPORT-FINAL_EscapeBench.md`, `Doc/RAPPORT-FINAL_LeakLab.md`, `Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md`, `Doc/Guide-implementation_AIUP-Claude-Code.md`.

Actions :

1. **Recompter les verdicts** (E-06, E-07, E-34). Dans le résumé du README et dans les deux rapports finaux, remplacer les décomptes à deux nombres par trois lignes : infirmations de fond, infirmations de lecture stricte ou d'opérationnalisation (EscapeBench H-004, H-011; LeakLab H-004, H-008), infirmations d'artefact du banc (LeakLab H-009); et, pour les confirmations, séparer celles qui pouvaient être infirmées sur ce matériel de celles qui ne le pouvaient pas (H-008, H-013) ou qui tiennent par construction du corpus (H-003, H-006). Ajouter aux tableaux §4.2 et §8.3 une colonne « portée » à trois valeurs. Les verdicts et rationales ne changent pas; une note datée en tête de chaque tableau dit que la colonne est une lecture ajoutée le 2026-09-xx.
2. **Compléter les limites** (E-08, E-09, E-10, E-15, E-30). README §6 : cœurs hybrides sans affinité, plan d'alimentation et turbo non contrôlés; intervalle de confiance intra-processus, variance inter-binaire captée seulement par H-012; absence de correction pour comparaisons multiples et ce qui l'atténue (critères à deux tailles concordantes); « préenregistrement séquentiel informé » nommé comme tel, avec la liste de ce que chaque critère successeur savait à son gel; NUMGOROUTINE comme construction du banc; résolution de l'horloge Windows pour H-006 de LeakLab. Chaque point renvoie à la décision ou au rapport de campagne qui le documente déjà.
3. **Nommer les revues par agents** (E-04, E-27). README §3.5, remplacer « Revues contradictoires » par « Revues contradictoires par agents » et dire en une phrase qu'aucune revue humaine indépendante n'a eu lieu avant la clôture (renvoi au lot 1). Dans `Campagnes/RAPPORT-CAMPAGNE_C-2026-09-10-1.md`, remplacer « un audit contradictoire de 172 agents » par ce que le lot 4 aura récupéré (nombre de sous-agents, rôles, règle d'accord) ou, à défaut, par « une revue par sous-agents dont la méthode n'a pas été archivée ». Dans le guide, §7, ajouter ce qu'une revue par agents ne remplace pas.
4. **README, statut** (E-33) : « six autres projets au stade du cadrage » devient « six autres projets dont seules les fiches de cadrage existent ».
5. Relire les deux rapports finaux pour que leurs décomptes suivent la même présentation que le README; erratum daté s'il y a lieu (règle 5).

**Sortie du lot.** Un lecteur du seul résumé obtient les mêmes proportions qu'un lecteur du corps; aucun rationale modifié; barre de sortie au vert (le README n'entre dans aucune empreinte, mais la règle 9 s'applique).

### Lot 3 — État de l'art et positionnement

**Exécutant : A→C.** Taille : L. Dépend du lot 2 (même fichier README; le lot 2 passe en premier). Constat : E-01. Recommandation : R1. C'est le lot au plus fort gain : il porte à lui seul C2 de 4 à 8 ou plus, soit la zone A-.

Actions :

1. **Vérifier les candidats** en source primaire (DOI ou page d'éditeur), un par un, avant toute rédaction. Liste de départ, tirée de l'annexe B de l'évaluation et complétée; tout y est *supposé* jusqu'à vérification :
   - Métrologie des benchmarks : Georges, Buytaert et Eeckhout (OOPSLA 2007), évaluation statistiquement rigoureuse; Mytkowicz, Diwan, Hauswirth et Sweeney (ASPLOS 2009), biais de disposition mémoire; Curtsinger et Berger (ASPLOS 2013, *Stabilizer*), randomisation de la disposition; Kalibera et Jones (ISMM 2013), bootstrap multi-niveaux et nombre de répétitions par niveau; Hoefler et Belli (SC 2015), règles de rapport.
   - Concurrence en Go : Tu et coll. (ASPLOS 2019), étude des bogues de concurrence réels en Go; travaux de Saioc et coll. (vers 2024 et 2025) sur la détection de fuites de goroutines par analyse dynamique puis par le ramasse-miettes, à l'origine du profil `goroutineleak`; travaux de vérification statique des canaux (Lange et coll., POPL 2017, à confirmer).
   - Préenregistrement et standards : Nosek et coll. (PNAS 2018); Ralph et coll. (ACM SIGSOFT Empirical Standards, 2020); politique de badges d'artefact de l'ACM.
   - Agents de codage : au moins un banc d'évaluation publié (SWE-bench, ICLR 2024) et, si elle existe en source citable, une description publiée du *Spec-Driven Development* autre que l'ouvrage de Martinelli.
   Toute référence non retrouvée est retirée; une référence trouvée est citée avec DOI.
2. **Rédiger** une section « Travaux connexes » dans le README (entre §2 et §3, ou en §1.4), de deux à trois pages, en trois paragraphes qui suivent les trois domaines, chacun terminé par ce que le dépôt en applique ou en ignore.
3. **Positionner** : un tableau « ce que l'étude réplique, confirme indépendamment ou ajoute », une ligne par résultat porteur : effet de disposition et passage en registres (H-001, H-002, H-007, H-012), débit contre latence (H-004, H-008), doublement comme cas particulier (H-010), `checkdead` et la minuterie de `go test` (LeakLab H-008), angle mort du profil sur les petits mutex (H-013), détecteurs du livre et fuites (H-002, H-004, H-005). Pour chaque ligne : « connu dans la littérature (réf.) », « connu des praticiens sans source citable » ou « non retrouvé ».
4. Reprendre dans `Doc/Projets-candidats…`, §9, la phrase « P1 réplique surtout des résultats connus de la littérature Go » en y mettant les références.
5. **Validation par le chercheur** : chaque référence ouverte et lue, au moins le résumé et la section citée; les marqueurs *supposé* disparaissent alors du texte.
6. Décision D-57 : l'état de l'art est écrit après les mesures; il ne les a pas guidées; ce qu'il aurait changé s'il avait été écrit avant (au minimum : H-007 aurait exigé des réplicats dès sa rédaction).

**Sortie du lot.** Aucune référence non vérifiée dans le README; tableau de positionnement complet; D-57 écrite.

### Lot 4 — Mesurer QR2 après coup et archiver les prompts des revues

**Exécutant : A→C.** Taille : M. Sans préalable; suppose Q4 = oui. Constats : E-05, E-04 (volet méthode), E-27. Recommandation : R6.

Actions :

1. **Extraire des agrégats** des 51 transcriptions locales, sans en publier le contenu : nombre de sessions par jour, durée de chaque session (premier et dernier horodatage), nombre de tours, nombre d'appels d'outils par type, nombre de sous-agents lancés (`Agent`), jetons d'entrée et de sortie quand les enregistrements les portent, modèle. Un script court (`jq` ou PowerShell), archivé sous `Doc/outils/qr2-sessions.sh` avec sa commande d'appel, pour que le calcul soit rejouable par quiconque possède ses propres transcriptions.
2. **Récupérer les prompts des revues** : dans les sessions du 2026-09-10 (revue contradictoire de C-008, rédaction contradictoire, « 172 agents ») et du 2026-09-12 (audit, si la session était locale), extraire les prompts donnés aux sous-agents et la règle d'accord appliquée. Les archiver sous `Revue/PROMPTS-REVUES_2026-09-10.md` avec, pour chaque revue : nombre d'agents, rôles, modèle, règle de rétention.
3. **Écrire** `Doc/QR2-MESURES.md` : tableau des agrégats par phase (cadrage, EscapeBench, audit, LeakLab, dépôt final), trous nommés (sessions en nuage du 2026-09-12; toute session antérieure au 2026-09-09; sessions lancées depuis `EscapeBench/` si elles ont existé et ont été purgées), et ce que les chiffres permettent de dire : coût d'une hypothèse, coût d'une revue, part du temps passé en revue.
4. Mettre à jour le README §5.3 : QR2 passe de « non mesurée » à « mesurée après coup, avec les limites de `Doc/QR2-MESURES.md` »; §3.5 renvoie aux prompts archivés.
5. Décision D-58 : ce qui est publié, ce qui ne l'est pas, pourquoi.

**Sortie du lot.** Une table de mesures datée, un script rejouable, les prompts des revues archivés, le README à jour.

### Lot 5 — Règles internes, couverture et provenance de LeakLab

**Exécutant : A.** Taille : M. Dépend du lot 1 pour les révisions de spécification. Constats : E-11, E-12, E-22 (pour l'avenir), E-29, E-32. Recommandations : R8, R9.

Actions :

1. **Règle `synctest`** (E-12, selon Q3). Option recommandée : amender `EscapeBench/CLAUDE.md` et le skill `/go-test` pour dire « `testing/synctest` pour tout délai, ticker ou goroutine simulables; les tests qui lisent des compteurs réels du système (temps processeur d'un arbre de processus, `quietude_test.go`) en sont exemptés et le disent en commentaire ». Ajouter ce commentaire à `internal/adapters/gotool/quietude_test.go`, ligne 96. Option alternative : réécrire les tests temporels sous `synctest`, en laissant le `time.Sleep` de la mesure réelle avec justification. Décision D-59.
2. **Conventions de test** (E-32) : une ligne dans chaque `CLAUDE.md` : EscapeBench nomme ses tests d'évaluateur `TestUC005_H###_…`; LeakLab nomme les siens `TestH###_…`. Aucun renommage.
3. **Couverture de LeakLab** (E-11). Lister les fonctions non couvertes (`go test -coverprofile` puis `go tool cover -func`) de `internal/campaign`, `internal/results` et `cmd/leaklab`; écrire des tests sur la fixture existante (`internal/campaign/testdata/fixture`) et sur des campagnes synthétiques pour `classify.go`, `static.go`, `probes.go`, l'écriture exclusive de `results.go` et le routage de `main.go`. Cible : chaque paquet du module principal à 75 % ou plus. Rapporter la couverture par paquet dans `DECISION_LeakLab.md`, §5, comme EscapeBench le fait.
4. **Nom du processeur** (E-29). Spécification d'abord : `LeakLab/docs/entity-model.md`, attribut `cpu` de `Provenance`, « nom commercial du processeur; repli sur l'identifiant que le système expose ». Puis reprendre dans `internal/campaign/campaign.go` (`cpuName`) la lecture du registre de `EscapeBench/internal/adapters/system/cpumodel_windows.go` et de `/proc/cpuinfo` (`model name`) sous Linux. Les campagnes archivées restent valides : aucun critère ne lit ce champ.
5. **Messages de commit** (E-22) : ajouter la règle 8 de ce plan au README, section « Conventions ».

**Sortie du lot.** `CLAUDE.md` et code cohérents; couverture de LeakLab rapportée et à 75 % ou plus par paquet; barre de sortie au vert.

### Lot 6 — Provenance étendue et archivage brut des mesures

**Exécutant : A.** Taille : M. Dépend des lots 1 et 5 (mêmes fichiers de spécification). Doit précéder le lot 8, pour que les campagnes de rejeu portent les nouveaux champs. Constats : E-14, E-19. Recommandation : R11.

Aucun gabarit du harnais n'est touché : l'empreinte du harnais reste `551ce66b…` et les campagnes restent comparables. Tous les champs ajoutés sont facultatifs, comme ceux de C-008 : un fichier antérieur reste valide, et le harnais de non-régression doit le prouver.

Actions :

1. **Spécification** (EscapeBench) : `docs/entity-model.md`, `Provenance` reçoit `osVersion` (version et build du système), `powerPlan` (plan d'alimentation sous Windows, gouverneur sous Linux, vide ailleurs), `cpuAffinity` (masque ou « non épinglé »), `coreTypes` (nombre de cœurs performance et efficacité quand la topologie les distingue, déjà relevée par `topology_*.go`); `Measurement` reçoit `iterations` (le `b.N` de chaque répétition, exactement `count` valeurs) et `rawOutputFile` (chemin de la sortie brute de `go test`, sous `results/campaigns/<id>/raw/`). `UC-003`, étapes 5 et 6, et BR-003-3 : la sortie brute est un fichier créé, jamais réécrit. Révision datée, note de revue.
2. **Code** : `internal/adapters/system` (lecture des nouveaux champs par plateforme, vide si indisponible), `internal/adapters/gotool` (capture de `b.N` depuis la ligne de résultat et conservation de la sortie brute), `internal/adapters/store` (DTO et écriture), `internal/models` (validation : `iterations` vide ou de longueur `count`). Tests par flux; le harnais de non-régression relit les dix campagnes archivées sans les champs.
3. **LeakLab** : `Provenance` reçoit `osVersion` seulement (entity-model, puis `campaign.go`).
4. Décision D-60 : pourquoi ces champs et pas d'autres; pourquoi `powerPlan` est consigné et non imposé (le banc constate, il ne configure pas).

**Sortie du lot.** Une campagne de fumée locale (matrice réduite, `--count 20`) produit des Measurement avec `iterations` et un fichier brut; les dix campagnes archivées se relisent; barre de sortie au vert.

### Lot 7 — Contre-épreuves hors campagne

**Exécutant : A.** Taille : S. Sans préalable; parallélisable. Constats : E-17, E-18. Ne produit rien sous `results/` : les sorties vont sous `Campagnes/`.

Actions :

1. **Désassemblage de la cellule écartée par H-012** (E-18). Depuis `EscapeBench/matrices/M-57477f022103/`, compiler les deux bras de `Size0008FieldsPtr_LOCAL` (`go test -c` dans chaque sujet), puis `go tool objdump -s 'consumeValue|consumePointer|Run'` sur chacun; comparer les corps de boucle et expliquer, si possible, pourquoi le bras pointeur exécute un chargement de plus et va plus vite (alignement de la boucle, dépendance de stockage, fusion d'instructions). Écrire `Campagnes/CONTRE-EPREUVE_H-012-8-octets.md` avec les deux listings, la lecture et ce qui reste inexpliqué. Renvoi depuis le rapport final, section « Ce qui reste ouvert », et depuis le README §7.
2. **Recherche des paramètres de `M-8f03757ac206`** (E-17). Balayer les combinaisons plausibles au 2026-09-10 entre C-008 et C-009 (tailles parmi celles de la matrice de référence et de C-2026-09-10-11, disposition `ARRAY_FILL`, `NAMED_FIELDS`, `NAMED_FIELDS_SHAM`, profils à huit, répétitions et charges, quatre sondes) en calculant `MatrixID()` jusqu'à retrouver l'identifiant, avec 252 cellules et 4 sondes comme filtre. Budget : une heure de calcul. Si trouvé : consigner les paramètres dans `LANCEMENT.md` §5 et vérifier que le harnais de non-régression rejoue désormais les quatre verdicts de H-009 et H-010 (67 devient 71). Sinon : consigner le balayage fait et fermer E-17 comme accepté.
3. Décision D-61 : résultats des deux contre-épreuves.

**Sortie du lot.** Un rapport de contre-épreuve; E-17 résolu ou clos avec le périmètre du balayage.

### Lot 8 — Rejeu sous Linux et journal de vérification hors poste

**Exécutant : A→C** (l'agent prépare et rédige; le chercheur lance les campagnes). Taille : L (environ 1 h 30 de mesures plus la rédaction). Dépend du lot 6. Constats : E-03 (première moitié), E-13. Recommandations : R3 (Linux), R7.

Actions :

1. **Journal de vérification hors poste** (E-13, selon Q2). Créer `Doc/VERIFICATION-HORS-POSTE.md` : une ligne par exécution (date, commit, système, version de Go, commandes, résultat). Première entrée : sous WSL2, depuis chaque banc, `GOTOOLCHAIN=go1.27.0 go vet ./... && GOTOOLCHAIN=go1.27.0 go test -race -shuffle=on -count=1 ./...`, plus la suite d'intégration d'EscapeBench et l'oracle de LeakLab. Ajouter dans `UC-003` (EscapeBench) et `UC-001` (LeakLab) une phrase : l'acteur « Pipeline CI » désigne un usage prévu; depuis D-53, il est tenu par ce journal.
2. **Campagne LeakLab sous WSL2** (9 min) : `GOTOOLCHAIN=go1.27.0 go run ./cmd/leaklab run`, puis `verdict`. Le binaire produit `results/runs/R-2026-09-xx-1.json` et ses verdicts. Comparer à `R-2026-09-13-2` cellule par cellule; rapport `Campagnes/RAPPORT-CAMPAGNE_R-<date>-linux.md`; README §8 et rapport final : verdicts par plateforme.
3. **Campagnes EscapeBench sous WSL2** (1 h 08 puis 11 min) : les deux commandes de `LANCEMENT.md` §5, en série, machine au repos, après avoir vérifié que l'attestation de quiétude fonctionne sous Linux (D-31 : repli sur le processus et ses enfants attendus; si la fraction dépasse 12 %, H-013 rendra non concluante, ce qui est le comportement voulu et doit être rapporté tel quel). Puis `compare` et `verdict`. Rapport `Campagnes/RAPPORT-CAMPAGNE_C-<date>-linux.md` : treize verdicts contre treize; écart médian des cellules; H-006 rejouée sur le poste de référence sous Linux natif lève-t-elle la réserve de D-48 (même système que le rejeu du 2026-09-12, même machine que les campagnes) : le dire.
4. **Limites** : WSL2 sur le même processeur hybride virtualisé mesure l'effet du système et de la chaîne d'exécution, pas celui d'une autre machine. README §6 le dit; E-03 reste ouvert pour sa moitié « autre machine, autre architecture » jusqu'au lot 9.
5. Décision D-62 (EscapeBench) et D-19 (LeakLab) : conditions du rejeu, écarts observés, verdicts qui changent s'il y en a (erratum, jamais substitution).

**Sortie du lot.** Deux rapports de campagne Linux, le journal de vérification avec sa première entrée, README §4, §6, §8 à jour.

### Lot 9 — Lot 9 de l'audit, série arm64, hypothèses successeurs

**Exécutant : A→C.** Taille : L. Dépend des lots 6 et 8 et de Q6 (matériel). Constats : E-03 (seconde moitié), E-08, E-09 (par successeurs). Recommandation : R3 (arm64).

C'est le seul lot qui change l'empreinte du harnais (D-50). Il ouvre une nouvelle série de comparaison : les campagnes Windows et Linux des lots précédents restent valides et comparables entre elles, pas avec celles-ci.

Actions :

1. Appliquer le lot 9 de l'audit (A-073, A-081, volet code de A-246) en une fois, spécification d'abord (C-005 : ce que l'empreinte couvre), puis les gabarits. Décision D-63 : nouvelle empreinte, date, série ouverte.
2. Écrire, **avant toute mesure arm64**, les hypothèses successeurs qui répondent à E-08 et E-09 : H-014 (successeur de H-001) et H-015 (successeur de H-002) sur des réplicats de processus séparés comme H-012, avec un plancher de bruit mesuré par cellule; H-016 (successeur de H-012) avec une règle explicite de multiplicité (par exemple : infirmée si au moins une cellule franchit son seuil dans quatre réplicats sur cinq, non concluante si exactement trois). Chaque critère cite ce qu'il sait des campagnes antérieures (règle du préenregistrement séquentiel informé, lot 2). Empreintes gelées à la première campagne arm64.
3. Campagnes arm64 : matrice de référence, puis les deux campagnes de clôture, puis LeakLab. Rapports comme au lot 8. H-008 et H-013 deviennent infirmables si la mémoire de la machine sert un accès dépendant sous 100 ns : le dire dans le rapport, dans un sens ou dans l'autre.
4. README : §4, §6, §7, §8; rapports finaux : verdicts par architecture.

**Sortie du lot.** Une série arm64 complète avec ses rapports; E-03 clos; E-08 et E-09 traités par successeurs.

### Lot 10 — Lisibilité des spécifications et de l'audit

**Exécutant : A.** Taille : S. Dépend du lot 1 (les UC sont touchés). Constats : E-20, E-21, E-26.

Actions :

1. **H-012 et H-013** (E-20). Dans `EscapeBench/docs/requirements.md`, ramener la colonne « Source » de chaque ligne à sa citation (« p. 253 — … », « p. 254 — … ») et déplacer la prose des « deux écarts » dans la section « Ce que H-012 et H-013 renoncent à éprouver », qui la porte déjà en partie. **Le texte des colonnes « Énoncé réfutable » et « Critère de réfutation » ne change pas d'un octet.** Vérification obligatoire : le test d'intégration de D-47 au vert, et `escapebench verdict --campaign C-2026-09-10-12` qui accepte encore l'empreinte (exécuté par le chercheur ou en lecture seule : il écrit un fichier de verdicts, donc à lancer à la main, ou remplacer par le test). Ajouter sous le tableau un « résumé lisible, non normatif » de chaque critère long, en cinq lignes, avec la mention que seul le tableau fait foi.
2. **Notes de clôture** (E-21). Dans les cinq UC d'EscapeBench, remplacer le paragraphe de clôture recopié par une ligne : « Clôture du 2026-09-10 : statut `Deployed`; ce qui l'établit est consigné en D-37. » Le hook `spec-lint` doit rester au vert (sections obligatoires intactes).
3. **AUDIT.md** (E-26). Ajouter en tête une table des matières avec les décomptes par gravité et par état (implanté, dette assumée, sans contre-vérification), et, dans la section des 75 constats sans contre-vérification, une colonne « état » à trois valeurs (implanté au fil de l'eau, ouvert, sans objet depuis D-53). Aucun constat n'est supprimé.

**Sortie du lot.** Empreintes inchangées (test D-47), `spec-lint` au vert sur les huit UC, `AUDIT.md` navigable.

### Lot 11 — Figer et citer

**Exécutant : C.** Taille : S. Dernier; dépend de tout ce qui précède, sauf le lot 9 s'il attend le matériel. Constat : E-25. Recommandation : R10.

Actions :

1. Ajouter `CITATION.cff` à la racine (auteur, titre, date, version, licence, URL, DOI une fois obtenu) et une section « Citer ce dépôt » dans le README.
2. Étiquette annotée `v1.1` (ou `v1.0` si Q5 = non) sur le commit qui clôt le dernier lot exécuté; release GitHub avec, pour notes, la liste des lots implantés et le renvoi à l'évaluation.
3. Activer l'intégration Zenodo du dépôt avant de publier la release, pour que le DOI soit frappé sur elle; reporter le DOI dans `CITATION.cff` et le README (un commit de plus, qui ne change pas l'étiquette).
4. Mettre à jour l'évaluation : une section « Suivi » en fin de `EVALUATION-ACADEMIQUE_2026-09-15.md`, une ligne par constat avec son état (clos, accepté, ouvert) et le lot ou la décision qui le clôt. La note n'est pas recalculée par l'auteur : c'est le rôle d'une seconde évaluation.

**Sortie du lot.** Un DOI résout vers la release; `CITATION.cff` valide; l'évaluation porte son suivi.

---

## 4. Ordre d'exécution et parallélisme

```
Lot 0 (C)  ──┬──> Lot 1 (C) ──┬──> Lot 5 (A) ──> Lot 6 (A) ──> Lot 8 (A→C) ──> Lot 9 (A→C, si Q6)
             │                └──> Lot 10 (A)                                       │
             ├──> Lot 2 (A) ──> Lot 3 (A→C)                                          │
             ├──> Lot 4 (A→C)                                                        │
             └──> Lot 7 (A)                                                          │
                                                                    Lot 11 (C) <─────┘ (ou après le lot 8 si le lot 9 attend)
```

- Le lot 0 est seul en tête : rien ne s'étiquette ni ne se cite sur un historique non stabilisé.
- Les lots 2, 4 et 7 démarrent dès la fin du lot 0, en parallèle, sur des fichiers disjoints. Le lot 3 suit le lot 2 (même README).
- Le lot 1 précède 5, 6 et 10 parce qu'ils révisent des spécifications.
- Les lots 5 puis 6 sont en série (mêmes fichiers d'EscapeBench); le lot 6 précède le lot 8.
- Le lot 9 attend Q6; le lot 11 ne l'attend pas si le matériel n'est pas disponible dans le mois : on étiquette sans arm64 et on rouvre une version ensuite.

Temps de chercheur, hors attente des campagnes : lot 0 une heure; lot 1 une demi-journée; lot 3 validation des références deux heures; lot 8 lancement et surveillance deux heures; lot 11 une heure. Temps d'agent : une session par lot, deux pour les lots 3 et 6.

---

## 5. Traçabilité

### Constats → lots

| Constat | Lot | Issue attendue |
|---|---|---|
| E-01 | 3 | clos |
| E-02 | 0 | clos après clone de vérification |
| E-03 | 8 (Linux), 9 (arm64) | clos au lot 9; partiel après le lot 8 |
| E-04 | 2 (texte), 4 (méthode) | clos |
| E-05 | 4 | clos avec trous nommés |
| E-06 | 2 | clos |
| E-07 | 2 | clos |
| E-08 | 2 (limite), 9 (successeurs) | clos au lot 9 |
| E-09 | 2 (limite), 9 (successeurs) | clos au lot 9 |
| E-10 | 2 | clos |
| E-11 | 5 | clos |
| E-12 | 5 | clos (D-59) |
| E-13 | 8 | clos par journal (Q2) |
| E-14 | 6 | clos pour les campagnes futures |
| E-15 | 2 | clos par documentation |
| E-16 | 1 | clos après coup (D-56, D-18) |
| E-17 | 7 | clos ou accepté |
| E-18 | 7 | clos ou réduit à ce qui reste inexpliqué |
| E-19 | 6 | clos pour les campagnes futures |
| E-20 | 10 | clos |
| E-21 | 10 | clos |
| E-22 | 5 (règle), 0 (rien à réécrire) | accepté pour le passé |
| E-23 | 0 | clos |
| E-24 | 0 | clos |
| E-25 | 0, 11 | clos |
| E-26 | 10 | clos |
| E-27 | 2, 4 | clos |
| E-28 | aucun | accepté (garde à échec fermé, documentée) |
| E-29 | 5 | clos |
| E-30 | 2 | clos |
| E-31 | 0 | clos si Q1 = oui, accepté sinon |
| E-32 | 5 | clos |
| E-33 | 2 | clos |
| E-34 | 2 | clos |

### Recommandations → lots

| R | Lot |
|---|---|
| R1 état de l'art | 3 |
| R2 purge | 0 |
| R3 Linux, arm64 | 8, 9 |
| R4 recompter | 2 |
| R5 revues par agents | 2, 4 |
| R6 mesurer QR2 | 4 |
| R7 vérification hors poste | 8 |
| R8 règles et code | 5 |
| R9 couverture LeakLab | 5 |
| R10 figer et citer | 0, 11 |
| R11 sortie brute | 6 |
| R12 hygiène | 0, 5 |

### Décisions à écrire

| Décision | Lot | Objet |
|---|---|---|
| D-55 | 0 | Achèvement de la purge, refs supprimées, demande à GitHub, étiquette `v1.0` |
| D-56, D-18 (LeakLab) | 1 | Revue humaine après coup, requalification du processus étudié |
| D-57 | 3 | État de l'art écrit après les mesures |
| D-58 | 4 | Agrégats publiés, transcriptions non publiées |
| D-59 | 5 | Règle `synctest` amendée ou appliquée |
| D-60 | 6 | Champs de provenance ajoutés, archivage brut |
| D-61 | 7 | Contre-épreuves : désassemblage, paramètres de `M-8f03757ac206` |
| D-62, D-19 (LeakLab) | 8 | Rejeu Linux, écarts, verdicts par plateforme |
| D-63 | 9 | Nouvelle empreinte du harnais, série arm64, successeurs H-014 à H-016 |

---

## 6. Conduite en mode agent

Pour chaque lot confié à un agent : une session neuve, un lot, un commit par fichier de spécification touché plus un commit de code, jamais un commit qui mêle une révision de `docs/` d'EscapeBench et une de LeakLab.

**Ce qu'il faut donner à l'agent.** Le numéro du lot et son texte ci-dessus, les constats concernés tels qu'écrits dans l'évaluation, les règles de la section 1, et la réponse aux questions de la section 2 qui touchent le lot. Ne pas donner l'évaluation entière : l'agent doit lire le dépôt, pas un résumé du dépôt.

**Vérification de chaque lot.** Sur le diff complet, trois lectures distinctes avant le commit : un relecteur de conformité (la spécification dit-elle ce que le code fait, et rien de plus), un chercheur de régression (barre de sortie, harnais de non-régression, empreintes), un sceptique chargé de trouver ce que le lot a cassé ailleurs (liens du README, décomptes, tableau de bord). Ces trois lectures sont des revues par agents; elles se déclarent comme telles dans le message de commit, et la revue humaine du lot 1 ne s'y substitue pas.

**Règle d'arrêt.** Un lot s'arrête quand sa barre de sortie est verte, que les empreintes gelées des douze campagnes archivées sont inchangées (test D-47), et que les 67 verdicts rejoués (71 si le lot 7 retrouve `M-8f03757ac206`) sont identiques. Les seuls verdicts nouveaux admis sont ceux des campagnes des lots 8 et 9, produits par le binaire.

**Ce qu'aucun agent ne fait.** Pousser de force, supprimer une branche distante, écrire à GitHub, lancer une campagne de mesure, valider une référence bibliographique, approuver un cas d'utilisation, publier une release.
