# Dossier de revue humaine à froid des cas d'utilisation — 2026-09-22

**Destinataire :** le tiers chargé de la revue (décision Q7 du chercheur, [`Doc/PLAN-IMPLANTATION_EVALUATION_2026-09-15.md`](../Doc/PLAN-IMPLANTATION_EVALUATION_2026-09-15.md), lot 1).
**Préparé par :** un agent (Claude Code), le 2026-09-22. L'agent n'a fait **aucune** revue au sens de l'AIUP et n'a rien approuvé. La section 5 est une pré-revue d'agent : une liste de constats *candidats* que le tiers retient ou rejette.
**Objet :** huit cas d'utilisation (UC) passés à `Approved` puis `Deployed` sans la relecture humaine que l'AIUP réserve à l'humain (D-01 de [`Doc/DECISION.md`](../Doc/DECISION.md) et de [`Doc/DECISION_LeakLab.md`](../Doc/DECISION_LeakLab.md); constat E-16 de l'évaluation académique). La revue a lieu après coup; elle ne rouvre aucun verdict publié.

---

## 1. Ce qu'on attend du tiers

Pour chacun des huit UC : lire à froid le fichier, ses exigences liées et ses entités (section 4), remplir le formulaire (section 3), puis dire, pour chaque constat candidat de la section 5, s'il est retenu ou rejeté, et ajouter ses propres constats. Le tiers ne modifie aucun fichier de spécification : la consignation (section 6) se fait sur sa réponse écrite.

Temps estimé (supposé, non mesuré) : 45 à 90 minutes par UC d'EscapeBench, 30 à 45 par UC de LeakLab.

## 2. D'où viennent les instruments

| Instrument | Source dans le dépôt | Référence citée |
|---|---|---|
| Trois tests d'exécutabilité | [`EscapeBench/.claude/agents/spec-reviewer.md`](../EscapeBench/.claude/agents/spec-reviewer.md), ligne 23 (vérification 1 de dix) | Martinelli (2026), *Spec-Driven Development*, ch. 3–4 |
| Neuf autres vérifications de forme (facultatives, mais c'est la grille qu'a appliquée l'agent en section 5) | même fichier, lignes 24–32 | même source |
| Cinq questions de l'ordre de revue | [`EscapeBench/.claude/agents/code-reviewer.md`](../EscapeBench/.claude/agents/code-reviewer.md), lignes 20–25 | SDD, p. 76 |
| Relecture « par quelqu'un qui ne l'a pas écrite » | [`EscapeBench/LANCEMENT.md`](../EscapeBench/LANCEMENT.md), §3, étape 3 (ligne 55) | SDD, p. 151 (*Adaptation*) |
| Transition `Reviewed → Approved` réservée à l'humain | [`EscapeBench/CLAUDE.md`](../EscapeBench/CLAUDE.md), ligne 8 | — |
| Contrôle de forme automatique (`spec-lint`) | [`EscapeBench/.claude/hooks/spec-lint.sh`](../EscapeBench/.claude/hooks/spec-lint.sh) | SDD, p. 51 (mots vagues) |

Deux remarques. Les cinq questions ont été écrites pour la revue de conformité du code : elles demandent de confronter le UC au code et aux tests (section 4 donne les emplacements). LeakLab n'a ni sous-agent `spec-reviewer` ni hook `spec-lint` (vérifié : `LeakLab/.claude/` ne contient que `hooks/go-check.sh`, `hooks/guard-paths.sh` et les skills `implement` et `refute`); on lui applique les instruments d'EscapeBench par analogie, les deux bancs suivant le même gabarit de UC.

## 3. Formulaire (un par UC)

Copier ce bloc huit fois dans la réponse écrite.

```
UC : <banc>/<UC-00n> — <titre>
Relu par : <nom>            Date : 2026-__-__        Durée : __ min

A. Trois tests d'exécutabilité (Martinelli; spec-reviewer.md l. 23)
  A1. Deux lecteurs imagineraient-ils des issues différentes ?   oui / non
      Lignes fautives : ...
  A2. Un testeur peut-il dériver des critères d'acceptation ?     oui / non
      Lignes fautives : ...
  A3. Un agent devrait-il inventer une règle ?                    oui / non
      Lignes fautives : ...

B. Cinq questions de l'ordre de revue (SDD p. 76; code-reviewer.md l. 20-25)
  B1. La spécification est-elle correcte et à jour par rapport au code ?
  B2. Le comportement implémenté est-il celui décrit ?
  B3. L'implémentation respecte-t-elle CLAUDE.md (layout, models sans dépendance,
      service sans I/O, context.Context en premier, %w, noms des tests) ?
  B4. Chaque précondition est-elle vérifiée, chaque étape observable produite,
      chaque flux alternatif branché, chaque postcondition de succès et d'échec
      garantie, chaque règle BR- appliquée ?
  B5. Les tests couvrent-ils le scénario principal, chaque flux alternatif et
      chaque règle, sous un nom TestUC###_<flux ou règle> ?

C. Constats candidats de l'agent (section 5)
  <id> : retenu / rejeté — motif en une ligne
  ...

D. Constats propres au relecteur
  <n>. <fichier:ligne> — <problème> — comportement ou rédaction ?

E. Bilan : n constats au total, dont m retenus. Critère gelé touché : oui / non.
```

« Critère gelé touché » : oui si un constat retenu vise le texte des colonnes « Énoncé réfutable » ou « Critère de réfutation » de `requirements.md`. Un tel constat ne se corrige pas; il appelle une hypothèse successeur (section 6).

## 4. Fichiers à lire, par UC

Pour chaque UC : le fichier du UC en entier; dans `docs/requirements.md`, les lignes des identifiants de **Linked Requirements** et **Linked Hypotheses**; dans `docs/entity-model.md`, les sections des **Entities**. Numéros de ligne relevés le 2026-09-22.

Repères communs. EscapeBench : exigences FR/NFR aux lignes 9–25, contraintes C-001 à C-010 aux lignes 31–40 de [`EscapeBench/docs/requirements.md`](../EscapeBench/docs/requirements.md); hypothèses dans le tableau qui suit. Dans [`EscapeBench/docs/entity-model.md`](../EscapeBench/docs/entity-model.md), une section `## <Entité>` par entité (fichier en cours de révision le 2026-09-22 : chercher le titre plutôt qu'un numéro de ligne). LeakLab : FR/NFR l. 9–25, C-001 à C-008 l. 31–38, lecture d'une cellule l. 87 de [`LeakLab/docs/requirements.md`](../LeakLab/docs/requirements.md); entités de [`LeakLab/docs/entity-model.md`](../LeakLab/docs/entity-model.md) : Case l. 16, Detector 37, Run 41, Provenance 56, Observation 65, Cell 78, ProbeResult 82, Hypothesis 93, Verdict 102.

| UC | Fichier | Exigences liées | Entités | Code et tests (pour B1–B5) |
|---|---|---|---|---|
| EscapeBench UC-001 | [`UC-001-generer-matrice.md`](../EscapeBench/docs/use-cases/UC-001-generer-matrice.md) | FR-001, NFR-002, NFR-004, C-001, C-004, C-005, C-007, C-008, C-009; H-007 à H-010, H-012, H-013 | Matrix, TypeSpec, LifetimeProfile, Cell, Probe | `internal/service/matrix.go`; 17 tests `TestUC001_*` dans `internal/{service,models}/matrix_test.go`, `adapters/cli/cli_test.go`, `adapters/specs/specs_test.go` |
| EscapeBench UC-002 | [`UC-002-classer-echappement.md`](../EscapeBench/docs/use-cases/UC-002-classer-echappement.md) | FR-002, NFR-001, NFR-002, C-001, C-003, C-005, C-009; H-006, H-009 | Matrix, Cell, EscapeVerdict, Provenance | `internal/service/escape.go`, `internal/adapters/escape/`; 11 tests `TestUC002_*` (dont `escape_integration_test.go`) |
| EscapeBench UC-003 | [`UC-003-executer-campagne.md`](../EscapeBench/docs/use-cases/UC-003-executer-campagne.md) | FR-003, FR-006, NFR-001, NFR-003 à NFR-005, C-001 à C-003, C-005, C-006, C-008 à C-010; H-001 à H-005, H-007, H-008, H-010 à H-013 | Matrix, Cell, Probe, Campaign, Measurement, Provenance | `internal/service/campaign.go`, `internal/adapters/{gotool,store}/`; 32 tests `TestUC003_*` |
| EscapeBench UC-004 | [`UC-004-comparer-valeur-pointeur.md`](../EscapeBench/docs/use-cases/UC-004-comparer-valeur-pointeur.md) | FR-004, NFR-003, NFR-004, C-002, C-008, C-009; H-001, H-002, H-007, H-010, H-012 | Campaign, Cell, Measurement, Comparison | `internal/service/compare.go`; 11 tests `TestUC004_*` |
| EscapeBench UC-005 | [`UC-005-produire-verdicts.md`](../EscapeBench/docs/use-cases/UC-005-produire-verdicts.md) | FR-005, FR-007, NFR-001, NFR-004; H-001 à H-013 | Hypothesis, Verdict, Campaign, Matrix, Cell, TypeSpec, Comparison, Measurement, Probe, EscapeVerdict | `internal/service/{verdict,criteria*}.go`, `internal/adapters/{specs,dashboard}/`; 54 tests `TestUC005_*` |
| LeakLab UC-001 | [`UC-001-executer-campagne.md`](../LeakLab/docs/use-cases/UC-001-executer-campagne.md) | FR-001 à FR-004, NFR-001 à NFR-005, C-001, C-003 à C-008; H-001 à H-014 | Case, Detector, Run, Provenance, Observation, ProbeResult, Hypothesis | `internal/campaign/`, `lab/corpus/`; 13 tests `TestUC001_*` |
| LeakLab UC-002 | [`UC-002-produire-verdicts.md`](../LeakLab/docs/use-cases/UC-002-produire-verdicts.md) | FR-005, FR-006, NFR-004, C-008; H-001 à H-014 | Run, Observation, Cell, ProbeResult, Case, Hypothesis, Verdict | `internal/verdict/`; 4 tests `TestUC002_*` et `TestH001` à `TestH014` |
| LeakLab UC-003 | [`UC-003-analyser-ctxvet.md`](../LeakLab/docs/use-cases/UC-003-analyser-ctxvet.md) | FR-003, FR-007, C-002 | Observation | `internal/ctxvet/`, `cmd/leaklab/main.go`; 4 tests `TestUC003_*` |

Chemins de code relatifs au banc. Décomptes de tests obtenus par `grep -rh "func TestUC00n_"` sur les `*_test.go` du banc.

## 5. Pré-revue par agent — constats candidats

**Ceci n'est pas une revue.** C'est la grille de `spec-reviewer.md` (vérifications 1 à 10, notées V1 à V10) appliquée par un agent, pour que le tiers ne parte pas de zéro. Chaque constat porte un identifiant, un emplacement `fichier:ligne` et l'origine de l'affirmation : **vérifié** (lu dans le fichier cité, ou obtenu par une commande nommée) ou **supposé** (déduit, non contrôlé dans le code). Le tiers peut tout rejeter. L'agent ne rend aucun verdict `APPROVE` ou `REVISE`.

Deux constats vérifiés par commande : le hook `spec-lint` d'EscapeBench, appliqué aux huit fichiers le 2026-09-22 (entrée JSON `{"tool_input":{"file_path":"docs/use-cases/<fichier>"}}`, `CLAUDE_PROJECT_DIR` sur le banc), ne signale rien sur les cinq UC d'EscapeBench et signale P-L1-01 et P-L3-02.

### EscapeBench UC-001 — Générer la matrice

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-E1-01 | `UC-001-generer-matrice.md:23-24`, `:85` | V7, A3 | La note de revue (l. 85) dit que le nombre de réplicats « vaut un ou cinq », « l'étape 2 refusant toute autre valeur » ; ni l'étape 1 ni l'étape 2 ne nomment ce paramètre. Une règle de validation vit dans les notes : un agent qui implanterait depuis le flux devrait l'inventer. | vérifié |
| P-E1-02 | `:25` | V2 | L'étape 3 décrit la représentation canonique (dimensions de C-008 omises à leur valeur par défaut) : détail interne ou comportement observable (stabilité de l'identifiant) ? | vérifié (lecture); qualification au tiers |
| P-E1-03 | `:19` | A1, V5 | Précondition « aucune campagne n'est au statut `RUNNING` » : aucun flux ne dit ce que fait le système si elle est fausse, ni comment elle est constatée. | vérifié (lecture); comportement du code non contrôlé |
| P-E1-04 | `:12` | V8 | Les hypothèses liées omettent H-011 alors que la ligne dit le UC « préalable à toutes les autres hypothèses » : la liste est-elle celle des hypothèses dont les sujets sont générés ici, ou une liste incomplète ? | vérifié |
| P-E1-05 | `:77-87` | lisibilité | Notes non chronologiques (clôture du 2026-09-10 en tête, révisions du même jour ensuite) et deux lignes vides (l. 80-81). | vérifié |

### EscapeBench UC-002 — Classer l'échappement

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-E2-01 | `UC-002-classer-echappement.md:19`, `:24`, `:33-38` | V5 | « Le harnais est intact » est à la fois une précondition et une validation du flux (étape 2, A1). SDD veut une précondition comme ligne de départ, pas comme validation. | vérifié |
| P-E2-02 | `:48` | A1 | A3 compare « un fichier de verdicts » existant : s'il y en a plusieurs pour la même toolchain, lequel ? Deux lecteurs peuvent choisir le plus récent ou tous. | vérifié (lecture); comportement du code non contrôlé |
| P-E2-03 | `:52`, `:54-61` | A2, V5 | Quand A3 « signale une violation de NFR-002 », l'issue est-elle un succès ou un échec (code de sortie, postcondition) ? Aucune postcondition ne le dit. | vérifié |
| P-E2-04 | `:26-27`, `:69`, `:79` | A2, A3 | Le passage d'une raison du compilateur à une catégorie est renvoyé à l'implémentation (l. 79) ; BR-002-2 renvoie aux « quatre causes de BEPG p. 238–242 ». Un testeur ne peut dériver l'attendu d'une raison donnée sans le livre ni le code. D-39 (`Doc/DECISION.md`, l. 105) documente une mauvaise classification née exactement de ce vide. | vérifié |

### EscapeBench UC-003 — Exécuter une campagne

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-E3-01 | `UC-003-executer-campagne.md:53`, `:79-80` ; `internal/service/campaign.go:304-308` | B1, V5 | A3 dit la Campaign `COMPLETED` « si au moins un sujet a été mesuré » et se tait sur le cas contraire ; le code la passe à `ABORTED` avec la raison « aucun sujet mesuré ». La postcondition d'échec ne cite que A1 et A2. La spécification est en retard sur le code. | vérifié (UC et code lus) |
| P-E3-02 | `:25`, `:41-42` | V6 | L'étape 2 vérifie l'empreinte du harnais ; aucun flux alternatif n'y est branché (A2 ne se déclenche qu'à l'étape 7). | vérifié |
| P-E3-03 | `:106`, `:16-20` | V7, A3 | Le refus d'une campagne qui porte H-007 sur une matrice à réplicats est dit « une précondition », mais il n'apparaît ni dans les préconditions ni dans un flux : il ne vit que dans une note. | vérifié |
| P-E3-04 | `:104`, `:28` | V7, B4 | La mesure de quiétude de C-010 (deux relevés de temps processeur par mesure, fraction consignée) est un comportement observable (champ `Measurement.quietudeOccupancy`) décrit seulement en note ; l'étape 5 n'en dit rien. | vérifié |
| P-E3-05 | `:79-80`, `:59`, `:65-70` | V5 | La postcondition d'échec ne couvre ni la reprise refusée (A4, la Campaign reste `RUNNING`) ni l'interruption (A5). | vérifié |
| P-E3-06 | `:20`, `:91` | V5, V6 | « Aucune autre campagne n'est en cours » est une précondition que le système contrôle (verrou `results/.campaign-lock`, BR-003-3) ; aucun flux ne décrit le refus au démarrage quand le verrou est tenu. | vérifié (lecture); comportement du code non contrôlé |
| P-E3-07 | `:7` | B1 | Acteur « (ou Pipeline CI) » alors que la CI est retirée (D-53). D-53 garde ce libellé exprès, comme « usage prévu » : le tiers dit si ce motif le convainc. | vérifié |
| P-E3-08 | `:12`, `:114`, `:49` | lisibilité, V6 | Un paragraphe de révisions est logé dans l'Overview, entre deux champs d'en-tête. La note l. 114 exclut les pannes techniques des flux alors qu'A3 compte « compilation, panic, délai dépassé » : la frontière entre variante métier et panne technique n'est pas nette. | vérifié |

### EscapeBench UC-004 — Comparer valeur et pointeur

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-E4-01 | `UC-004-comparer-valeur-pointeur.md:17`, `:23`, `:32-36` | V5 | « Campaign `COMPLETED` » est une précondition et la validation d'A1. | vérifié |
| P-E4-02 | `:23` | V6 | L'étape 2 vérifie l'empreinte du harnais contre la Matrix ; aucun flux alternatif pour un écart. | vérifié |
| P-E4-03 | `:25`, `:72`, `:86` | A1, A2 | La méthode de l'intervalle de confiance à 95 % est renvoyée à l'implémentation (l. 86), alors que `significant` (BR-004-2) et le point de bascule en dépendent : deux implantations conformes peuvent rendre des `significant` différents. | vérifié (lecture) |
| P-E4-04 | `:26`, `:39-43` | A1 | Point de bascule : « toutes les tailles supérieures mesurées du couple ». Une taille dont la paire est exclue par A2 compte-t-elle comme mesurée ? | vérifié (lecture); comportement du code non contrôlé |
| P-E4-05 | `:77-88` | lisibilité | Notes non chronologiques ; la révision du 2026-09-12 (l. 82) requalifie celle du 2026-09-10 (l. 84), placée après elle. | vérifié |

### EscapeBench UC-005 — Produire les verdicts

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-E5-01 | `UC-005-produire-verdicts.md:18` | B1, V10 | La précondition n'énumère les résultats requis que pour H-001 à H-006 ; H-007 à H-013 (réplicats, quiétude) n'y figurent pas, alors que la ligne 12 les lie. | vérifié |
| P-E5-02 | `:19`, `:24`, `:34-39` | V5 | La correspondance des empreintes est une précondition et la validation d'A1. | vérifié |
| P-E5-03 | `:29` | V4, V6 | L'étape 7 combine régénération, lecture des statuts et exécution des tests, et ouvre un second point d'entrée (« la régénération seule est aussi disponible ») qui n'a ni flux ni postconditions. | vérifié |
| P-E5-04 | `:34`, `:60` ; `docs/entity-model.md` (Campaign, `hypothesesDigest`) | A1 | A1 s'intitule « Critère modifié », mais l'empreinte couvre énoncé et critère (modèle d'entités; plan d'implantation §0 : `specs.go`, l. 105). BR-005-1 parle du « critère enregistré au démarrage » alors que seule son empreinte l'est ; l'évaluation est faite par le code. | vérifié (UC, modèle, plan); `specs.go` non relu |
| P-E5-05 | `:74` | V7 | « doit rester évaluable mécaniquement » : une règle de rédaction du catalogue logée en note. | vérifié |

### LeakLab UC-001 — Exécuter une campagne

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-L1-01 | `UC-001-executer-campagne.md:23` | V3 | « au besoin » : mot vague, signalé par `spec-lint` d'EscapeBench. | vérifié (commande) |
| P-L1-02 | `:51-54` | V6 | A4 ne se termine pas sur un état explicite (« aucun fichier n'est écrit »), contrairement à A1–A3. | vérifié |
| P-L1-03 | `:27` | V4 | L'étape 5 lance, tue, classe et consigne en une phrase ; le délai « au-delà du délai » ne dit pas que les binaires gardent le délai par défaut de `go test` (C-004, précision du 2026-09-13, cause de D-13). | vérifié |
| P-L1-04 | `:83`, `:25`, `:34-38` | V7, A2 | L'oracle n'exécute pas les cas `D` et exécute les cas `R` sans `-race` : règle portée par une note, dont dépend le déclencheur d'A1. | vérifié |
| P-L1-05 | `:12`, `:81-84` | B1 | H-014 a été ajoutée après la première campagne (D-14) et C-004 précisée (D-13) ; le UC n'en porte aucune révision datée, seulement une mention dans la note de clôture. | vérifié |

### LeakLab UC-002 — Produire les verdicts

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-L2-01 | `UC-002-produire-verdicts.md:23` ; `requirements.md:87` | A1, A3 | « Issue majoritaire » : définie comme « plus de la moitié des répétitions » ; le cas sans majorité (5 répétitions réparties 2-2-1) n'est défini ni dans le UC ni dans l'exigence. | vérifié (lecture); comportement du code non contrôlé |
| P-L2-02 | `:53` | V2 | « chaque évaluateur code le texte de son critère » décrit l'implémentation plutôt qu'un comportement observable, et ne se teste pas tel quel. | vérifié |
| P-L2-03 | `:25`, `:45-48` | A2 | Deux fichiers écrits à l'étape 5 ; si le second échoue, la postcondition d'échec (« `results/` est inchangé ») tient-elle ? | supposé (non contrôlé dans `internal/verdict/`) |

### LeakLab UC-003 — Analyser un paquet avec ctxvet

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-L3-01 | `UC-003-analyser-ctxvet.md:7` | B1 | « Pipeline CI en acteur secondaire » : même situation que P-E3-07 (D-53 garde le libellé exprès). | vérifié |
| P-L3-02 | `:33-35` | V5 | Pas de postcondition `**Failure:**` (A1 sort en code 2) ; signalé par `spec-lint`. | vérifié (commande) |
| P-L3-03 | `:11`, `:13` | V8 | Entité `Observation` et FR-003 : employé seul, `ctxvet` écrit des diagnostics sur la sortie standard ; l'Observation et l'attribution aux cas relèvent de LeakLab UC-001 (étape 6, BR-001-3). | vérifié |
| P-L3-04 | `:23` | A2 | La forme de `<appel>` dans le diagnostic n'est pas fixée (alias local ou chemin d'import, BR-003-2) : un testeur ne peut écrire la chaîne attendue. | vérifié (lecture) |
| P-L3-05 | `:17`, `:22` | A1 | Un répertoire qui ne contient que des `_test.go` satisfait la précondition, puis rien n'est analysé : code 0 ou erreur ? | supposé |

### Transversal

| Id | Emplacement | Grille | Constat candidat | Origine |
|---|---|---|---|---|
| P-X-01 | `LeakLab/.claude/` | processus | LeakLab n'a ni `spec-reviewer` ni `spec-lint` : ses UC n'ont jamais passé de contrôle de forme automatique, ce que confirment P-L1-01 et P-L3-02. | vérifié |
| P-X-02 | les huit notes « Clôture » | V2, B1 | Les notes de clôture d'EscapeBench (UC-001:79, UC-002:76, UC-003:101, UC-004:79, UC-005:70) sont identiques et justifient `Deployed` par des éléments communs aux cinq UC ; elles ne disent pas ce qui établit chacun. | vérifié |

## 6. Procédure de consignation

À faire par le chercheur (ou par un agent, sur la réponse écrite du tiers), après réception des huit formulaires.

1. **Note datée.** Dans chaque fichier de UC, sous `## Notes de revue`, ajouter en dernière puce :
   `- Revue humaine à froid du 2026-__-__ par <nom> : n constats, dont m retenus (<identifiants>). Formulaire : Revue/<fichier de la réponse>.`
   Emplacements : EscapeBench UC-001 l. 77, UC-002 l. 74, UC-003 l. 99, UC-004 l. 77, UC-005 l. 68 ; LeakLab UC-001 l. 81, UC-002 l. 61, UC-003 l. 54 (en-têtes de section au 2026-09-22).
2. **Statut.** `**Status:**` reste `Deployed`. La revue ne rouvre pas la chaîne de statuts et ne retire aucun verdict publié.
3. **Constats retenus.** Aucun ne se corrige dans la note ni dans le code directement :
   - un constat qui change un comportement devient une **révision de spécification datée** dans `docs/`, prise par les lots 5 ou 6 du plan, code et spécification dans le même commit (règle 1 du plan);
   - un constat de rédaction seule (lisibilité, ordre des notes, mot vague) va au lot 10;
   - un constat qui vise le texte d'un **critère gelé** n'est **jamais** une retouche : il devient une hypothèse successeur au numéro neuf (lot 9), ou il est consigné comme limite.
4. **Commits.** Un commit par banc, jamais EscapeBench et LeakLab ensemble (plan, §6), préfixe `Évaluation : lot 1 — …`. La réponse du tiers est versée sous `Revue/`.
5. **Barre de sortie.** Les notes ne touchent pas `requirements.md`, donc aucune empreinte ; la barre commune du plan (§1, règle 9) passe quand même, test D-47 compris.
6. **Décisions.** Compléter D-56 (`Doc/DECISION.md`) et D-18 (`Doc/DECISION_LeakLab.md`), puis le README §5.3, avec le décompte réel.
