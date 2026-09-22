# Prompts des revues par agents, 2026-09-10 au 2026-09-12

**Archivé le :** 2026-09-22 (lot 4 du plan d'implantation de l'évaluation; décision D-58) · **Source :** les scripts des six workflows et leurs fichiers d'exécution, conservés dans les transcriptions locales de Claude Code (voir [`Doc/QR2-MESURES.md`](../Doc/QR2-MESURES.md)).

Les revues contradictoires, les rédactions contradictoires d'hypothèses et l'audit du code n'ont pas été faits par des humains. Des sous-agents d'un modèle de langage les ont menés, lancés par l'outil `Workflow` de Claude Code, qui exécute un script JavaScript d'orchestration. Ce document archive, pour chaque exécution, le nombre d'agents, leurs rôles, le modèle, la règle d'accord et les prompts donnés aux sous-agents.

**Ce qui est publié et ce qui ne l'est pas (D-58).** Sont publiés les gabarits de prompts, tels qu'écrits dans les scripts, et les fragments d'orchestration qui les assemblent et appliquent la règle d'accord. Les schémas de sortie sont omis. Ne sont publiés ni les réponses des sous-agents, ni les conversations, ni aucun autre contenu des transcriptions. Les chemins locaux sont remplacés par `<DEPOT>` (racine du dépôt) et `<SCRATCHPAD>` (répertoire temporaire de la session). Aucun autre texte n'est modifié. Les accents manquants sont d'origine : cinq scripts sur six ont été écrits sans diacritiques. Les expressions `${...}` sont des substitutions faites à l'exécution : identifiant du constat, dimension, texte d'une proposition précédente.

**Décompte.** Le nombre d'agents vient du champ `agentCount` du fichier d'exécution de chaque workflow; la ventilation par phase, du journal de progression du même fichier. Les jetons viennent des transcriptions de sous-agents ([`Doc/outils/qr2-sessions.sh`](../Doc/outils/qr2-sessions.sh)). Tous les agents déclarent le modèle `claude-opus-5`, sauf dans l'audit du code (voir §6).

**Ce que ces revues ne remplacent pas.** Aucune n'est une revue humaine. Les agents d'une même revue partagent le modèle, le contexte commun et souvent la même formulation de la question. Leur indépendance est donc celle de tirages distincts d'un même modèle, pas celle de relecteurs distincts. La revue humaine prévue par le processus n'a pas eu lieu (README §5.3; lot 1 du plan).

| § | Workflow | Objet | Agents | Règle d'accord |
|---|---|---|---:|---|
| 1 | `audit-campagne-escapebench` | Audit contradictoire des six verdicts de C-2026-09-10-1, les « 172 agents » du rapport de campagne | 172 | constat retenu si au moins 2 votes valides sur 3 le maintiennent |
| 2 | `rediger-h007-h011` | Rédaction de H-007 à H-011 par panel de juges | 80 | synthèse à partir du brouillon le mieux noté (moyenne de 4 juges, sur 50) |
| 3 | `revue-c008` | Revue contradictoire de la satisfaction de C-008 ([`REVUE-C008_2026-09-10.md`](REVUE-C008_2026-09-10.md)) | 113 | comme au §1 |
| 4 | `rediger-h012-h013` | Rédaction de H-012, H-013 et C-009, puis attaque adversariale | 20 | toute faille bloquante ou majeure corrigée, ou écartée par écrit; aucun vote |
| 5 | `h012-h013-passe2` | Seconde passe sur H-012 et H-013 | 13 | comme au §4 |
| 6 | `audit-escapebench` | Audit du code EscapeBench ([`Doc/AUDIT.md`](../Doc/AUDIT.md)) | 581 | au moins 2 vérificateurs sur 3 maintiennent le constat; sévérité majoritaire |

**Non archivées, faute de trace locale :** la « contre-vérification indépendante » à huit agents du 2026-09-12 que décrit `Doc/AUDIT.md`, et l'implantation des lots de l'audit. Les deux ont eu lieu dans des sessions dont aucune transcription n'est sur le poste (*supposé* : sessions en nuage, d'après les commits signés « Claude »). La revue avant lancement ([`REVUE-PRELANCEMENT_2026-09-10.md`](REVUE-PRELANCEMENT_2026-09-10.md)) n'a lancé aucun sous-agent : l'agent principal l'a faite seul (vérifié : aucun appel `Agent` ni `Workflow` dans les sessions du 2026-09-08 au 2026-09-10 avant 14 h 32 UTC). Les deux sous-agents de revue et les six *skills* du banc sont versionnés sous `EscapeBench/.claude/`, mais aucun n'a été invoqué dans les sessions locales (voir `Doc/QR2-MESURES.md` §2.3).

---

## 1. `audit-campagne-escapebench` : les « 172 agents »

- **Exécution :** `wf_1780a1be-9a4`, session `ef21fc00`, du 2026-09-10 14 h 33 à 15 h 17 UTC (44 min).
- **Agents : 172** = 9 auditeurs (un par hypothèse H-001 à H-006, plus trois dimensions transversales) + 162 vérificateurs (54 constats × 3 lentilles; 161 ont répondu, 1 a été bloqué par les garde-fous du modèle) + 1 critique de complétude.
- **Rôles :** l'auditeur rend au plus 6 constats chiffrés. Chaque vérificateur cherche à *réfuter* un constat sous l'une de trois lentilles : exactitude des chiffres, pertinence, contre-explication.
- **Règle d'accord :** un constat survit si au moins 2 votes valides le maintiennent (`kept >= 2`); en cas de doute, le vérificateur vote la réfutation. Un vote manquant compte comme absent, pas comme réfutation.
- **Résultat (journal d'exécution) :** 19 constats confirmés sur 54 examinés.
- **Jetons :** 8,23 M en entrée nouvelle, 117,7 M en lecture de cache, 0,85 M en sortie.

```js
const ROOT = '<DEPOT>/EscapeBench'

const CONTEXTE = `
Projet : EscapeBench, banc de refutation Go qui eprouve des affirmations du livre
"Building Enterprise Projects with Go" (Shahsavan, Apress 2026). Racine du projet : ${ROOT}

Campagne executee : C-2026-09-10-1, matrice de reference M-823d8b5af441.
Provenance : go1.27.0 windows/amd64, Intel Core (Family 6 Model 198), 24 coeurs logiques.
230 sujets mesures (220 Cell + 10 Probe), 0 echec, 20 repetitions par sujet,
-benchtime=250ms, -cpu=1, un processus "go test" distinct par sujet.

Fichiers a lire (chemins absolus) :
- ${ROOT}/docs/requirements.md            (enonces et criteres de refutation geles, table H-001..H-006)
- ${ROOT}/docs/use-cases/                 (les cinq cas d'utilisation)
- ${ROOT}/internal/service/criteria.go    (un evaluateur par hypothese)
- ${ROOT}/internal/service/compare.go     (bootstrap percentile, points de bascule)
- ${ROOT}/internal/harness/templates/cell.go.tmpl   (gabarit des cellules)
- ${ROOT}/internal/harness/templates/probe.go.tmpl  (gabarit des sondes)
- ${ROOT}/results/verdicts/C-2026-09-10-1-20260910T143001Z.json
- ${ROOT}/results/campaigns/C-2026-09-10-1/campaign.json
- ${ROOT}/results/campaigns/C-2026-09-10-1/comparison-20260910T143000Z.json
- ${ROOT}/results/campaigns/C-2026-09-10-1/measurements/*.json  (230 fichiers, un par sujet)
- ${ROOT}/results/escape/M-823d8b5af441/20260910T135930Z.json

Verdicts produits :
  H-001 REFUTED  H-002 REFUTED  H-003 CONFIRMED
  H-004 REFUTED  H-005 CONFIRMED  H-006 CONFIRMED

Mesures notables deja relevees (a verifier, ne pas croire sur parole) :
- LOCAL sans champ pointeur, medianes ns/op : 8 o valeur 0.8313 / pointeur 0.8414 ;
  16 o valeur 0.8590 / pointeur 0.8176 ; 24 o valeur 1.1760 / pointeur 0.4858.
- Sondes de parcours, medianes ns/op : SEQUENTIAL 32 MiB 1009458, SCATTERED 32 MiB 1855756 ;
  SEQUENTIAL 128 MiB 5366698, SCATTERED 128 MiB 12690805.
- Sondes append n=100000 : PREALLOC 53519 ns / 802816 B / 1 alloc ;
  GROW 233486 ns / 4101375 B / 27 allocs.

Regle du projet : un critere de refutation est gele avant la mesure et ne se modifie jamais.
Un verdict enregistre reste ce qu'il est. Ton travail n'est PAS de changer le verdict, mais de
dire si sa LECTURE est solide : le banc mesure-t-il bien ce dont parle l'hypothese, la mesure
est-elle fiable, le critere ecrit correspond-il a ce que le code evalue.
`

const DIMENSIONS = [
  {
    key: 'H-001',
    prompt: `Audite le verdict REFUTED de H-001.
H-001 : "Pour des structures de taille <= 24 octets sans champ pointeur, en profil LOCAL, le passage
par valeur a un ns/op inferieur ou egal au passage par pointeur." Critere : infirmee si au moins deux
tailles <= 24 octets ont significant=true et ciHigh < 0.
Le verdict repose sur des ecarts de 0.04 ns/op (16 o) et 0.69 ns/op (24 o).
Questions : un ecart de 0.04 ns/op a ~3.5 GHz est-il physiquement interpretable ? Le bootstrap sur
20 repetitions tres reproductibles produit-il des intervalles trompeusement etroits ? Compare la
dispersion intra-sujet (min, max des 20 repetitions) a l'ecart entre sujets. Cherche des anomalies
de comportement entre tailles voisines qui trahiraient un artefact de disposition de code.`,
  },
  {
    key: 'H-002',
    prompt: `Audite le verdict REFUTED de H-002.
H-002 : "Le point de bascule valeur -> pointeur est superieur a 24 octets pour le profil LOCAL."
Critere : infirmee si le point de bascule de l'une des deux series LOCAL est <= 24 octets.
Points de bascule observes : sans champ pointeur 16 octets, avec champ pointeur 24 octets.
Questions : la definition du point de bascule (plus petite taille telle que, pour elle et toutes les
tailles superieures, la comparaison est significative et le delta negatif) est-elle robuste quand les
ecarts sont sub-nanoseconde ? Le point de bascule a 16 octets tient-il a une seule comparaison
fragile ? Que devient-il si l'on exige un ecart pratique minimal ? Verifie le calcul dans compare.go
contre l'etape 5 de UC-004.`,
  },
  {
    key: 'H-003',
    prompt: `Audite le verdict CONFIRMED de H-003.
H-003 : "Pour le profil RETURNED, passer du retour par valeur au retour par pointeur multiplie
allocs/op par un facteur >= 2 sur au moins une taille." Critere : infirmee si le rapport des medianes
reste < 2 partout ; une paire dont la mediane valeur est 0 compte comme rapport >= 2 si la mediane
pointeur est >= 1.
Observe : 22 paires sur 22, toutes 0 -> 1 alloc/op.
Questions : l'affirmation du livre parle de DOUBLER le nombre d'allocations ; 0 -> 1 est-il un
doublement ? Le verdict CONFIRMED repose-t-il entierement sur la regle speciale du critere ? La
regle speciale a-t-elle ete ecrite avant ou apres avoir vu des mesures ? Verifie dans l'historique
git si le critere de H-003 a change. Le harnais RETURNED mesure-t-il ce dont parle le livre p. 256
(profilage memoire d'une struct passee par valeur vs pointeur) ?`,
  },
  {
    key: 'H-004',
    prompt: `Audite le verdict REFUTED de H-004. C'est la dimension la plus suspecte.
H-004 : "Le rapport ns/op entre parcours disperse et parcours sequentiel d'un meme jeu de travail se
situe dans [10, 200] lorsque le jeu de travail excede le cache L2." Critere : infirmee si, pour toutes
les sondes >= 32 MiB, le rapport est < 10 ou > 200.
Observe : x1.8 a 32 MiB, x2.4 a 128 MiB.
Le livre (p. 254) parle de LATENCE : "cache hit < 10 ns, miss > 100 ns, roughly 10x to 200x".
Lis le gabarit ${ROOT}/internal/harness/templates/probe.go.tmpl. La sonde SCATTERED_SCAN parcourt
une tranche de pointeurs melangee : "for j := range ptrs { s += ptrs[j].V }".
Questions decisives : les chargements sont-ils dependants (poursuite de pointeurs serialisee) ou
independants ? Si independants, combien d'acces peuvent etre en vol simultanement, et la sonde
mesure-t-elle alors un DEBIT plutot qu'une LATENCE ? Calcule le temps par element dans les deux cas
(taille d'element 64 octets) et compare-le a une latence DRAM plausible de 60 a 100 ns. Conclus :
le verdict REFUTED porte-t-il sur l'affirmation du livre ou sur la conception de la sonde ?`,
  },
  {
    key: 'H-005',
    prompt: `Audite le verdict CONFIRMED de H-005.
H-005 : "Pour n = 100 000 entiers, l'append avec preallocation a un ns/op <= 1/4 et un B/op <= 1/4
de la version sans preallocation." Critere : infirmee si l'un des deux rapports depasse 1/2.
Observe : ns/op x0.229, B/op x0.196, allocs 1 contre 27.
Le livre (p. 114) annonce environ 6x plus rapide, un cinquieme de la memoire, 28 allocations -> 1.
Questions : les valeurs observees (4.4x plus rapide, 5.1x moins de memoire, 27 -> 1 allocations)
concordent-elles avec le livre ? Le critere a 1/4 avec seuil de refutation a 1/2 est-il si laxiste
que presque toute mesure le confirmerait ? La sonde mesure-t-elle bien ce dont parle le livre ?
Verifie B/op : 802816 octets pour 100000 int de 8 octets, est-ce coherent ? et 4101375 pour la
croissance geometrique ?`,
  },
  {
    key: 'H-006',
    prompt: `Audite le verdict CONFIRMED de H-006. Cherche un vice de circularite.
H-006 : "Toute cellule dont le compilateur rapporte un echappement se classe dans l'une des quatre
causes du livre." Critere : infirmee si au moins une cellule echappe pour une raison hors des quatre
categories.
Observe : 110 cellules echappent, 22 RETURN_POINTER, 44 CLOSURE_CAPTURE, 22 CHANNEL_SEND,
22 CONTAINER_STORE, 0 OTHER.
Question decisive : la matrice est construite a partir de cinq profils de duree de vie qui sont
exactement les quatre causes du livre plus LOCAL. Un tel corpus PEUT-IL seulement refuter H-006 ?
Lis ${ROOT}/internal/harness/templates/cell.go.tmpl et le modele d'entites pour verifier.
Examine aussi le classificateur ${ROOT}/internal/adapters/escape/escape.go : attribue-t-il la
categorie a partir du profil de la cellule (ce qui serait circulaire) ou de la sortie du compilateur ?
Quelle experience refuterait reellement H-006 ?`,
  },
  {
    key: 'qualite-mesure',
    prompt: `Audite la qualite metrologique de la campagne, toutes hypotheses confondues.
Examine les 230 fichiers de ${ROOT}/results/campaigns/C-2026-09-10-1/measurements/ (echantillonne
intelligemment si necessaire, et dis ce que tu as echantillonne).
Questions : 20 repetitions suffisent-elles ? Quelle est la dispersion intra-sujet typique (coefficient
de variation) ? Y a-t-il des sujets a dispersion anormale, des valeurs aberrantes, des derives dans
l'ordre des repetitions ? -benchtime=250ms donne-t-il assez d'iterations pour les sujets les plus
rapides (sub-nanoseconde) et pour les plus lents (parcours de 128 MiB) ? Le drapeau -cpu=1 est-il
approprie aux sondes comme aux cellules ? La machine a 24 coeurs logiques et la campagne a dure
29 minutes : y a-t-il trace d'interference thermique ou de frequence variable entre le debut et la
fin ? La provenance enregistree suffit-elle a rejouer la campagne (NFR-001) ?`,
  },
  {
    key: 'validite-harnais',
    prompt: `Audite la validite de construction du harnais : chaque sujet mesure-t-il ce dont parle
l'hypothese qu'il sert ?
Lis ${ROOT}/internal/harness/templates/cell.go.tmpl et probe.go.tmpl en entier.
Questions : la disposition des types (Tag uint64 + remplissage, ou P *byte + remplissage) fait-elle
que le cout de copie croit bien avec la taille annoncee ? La methode sum() lit le premier et le
dernier mot : est-ce representatif d'un usage reel, et introduit-elle un biais entre variantes avec et
sans champ pointeur (l'une lit un uint64, l'autre convertit un pointeur en uintptr) ? Les fonctions
marquees go:noinline ajoutent un cout d'appel fixe : quelle part des mesures sub-nanoseconde
represente-t-il, et masque-t-il l'effet etudie ? Le profil LOCAL/POINTER passe l'adresse d'une
variable creee AVANT la boucle, alors que LOCAL/VALUE copie cette meme variable a chaque appel :
la comparaison est-elle equitable ? Cherche tout autre biais systematique.`,
  },
  {
    key: 'fidelite-evaluateurs',
    prompt: `Audite la fidelite des evaluateurs au texte gele des criteres.
Compare mot a mot la colonne "Critere de refutation" de la table des hypotheses de
${ROOT}/docs/requirements.md avec les fonctions evaluateH001 a evaluateH006 de
${ROOT}/internal/service/criteria.go.
Pour chaque hypothese : le code implemente-t-il exactement la condition ecrite, ni plus ni moins ?
Attention particuliere aux comparaisons de bornes (<= contre <, >= contre >), au quantificateur
(au moins deux / pour toutes / sur au moins une), au perimetre (profil, presence d'un champ
pointeur, borne de taille), et au traitement des cas limites (mediane nulle, absence de donnees).
Verifie aussi que l'empreinte gelee couvre bien l'enonce ET le critere, et qu'aucune modification
du texte depuis la creation de la campagne n'aurait pu passer inapercue. Signale toute divergence,
meme mineure.`,
  },
]

phase('Audit')
log(`${DIMENSIONS.length} auditeurs sur la campagne C-2026-09-10-1`)

const results = await pipeline(
  DIMENSIONS,
  (d) =>
    agent(
      `${CONTEXTE}\n\n=== TA DIMENSION : ${d.key} ===\n${d.prompt}\n\n` +
        `Lis reellement les fichiers avant d'affirmer quoi que ce soit. Chiffre chaque constat.\n` +
        `Rends au plus 6 constats, du plus grave au plus benin. Si tu dois couper, mets truncated=true.\n` +
        `N'invente aucun chiffre : si tu ne peux pas verifier, ne le rapporte pas.`,
      { label: `audit:${d.key}`, phase: 'Audit', schema: FINDINGS_SCHEMA, effort: 'high' }
    ),
  (audit, d) => {
    if (!audit || !audit.findings || audit.findings.length === 0) return []
    if (audit.truncated) log(`${d.key} : liste tronquee par l'auditeur`)
    const LENSES = [
      'exactitude des chiffres : les valeurs citees existent-elles vraiment dans les fichiers, et le calcul est-il juste',
      'pertinence : le constat change-t-il quelque chose a la lecture du verdict, ou enfonce-t-il une porte ouverte',
      'contre-explication : existe-t-il une explication plus simple ou plus probable que celle avancee',
    ]
    return parallel(
      audit.findings.map((f) => () =>
        parallel(
          LENSES.map((lens) => () =>
            agent(
              `${CONTEXTE}\n\n=== CONSTAT A REFUTER (dimension ${d.key}) ===\n` +
                `Affirmation : ${f.claim}\nGravite annoncee : ${f.severity}\n` +
                `Preuves avancees : ${f.evidence}\nImpact annonce : ${f.impact}\n\n` +
                `Ton angle : ${lens}.\n` +
                `Verifie dans les fichiers du projet. Cherche activement a REFUTER ce constat.\n` +
                `En cas de doute, mets refuted=true. Si le constat tient mais est mal formule, ` +
                `mets refuted=false et propose une reformulation exacte dans "correction".`,
              { label: `verif:${d.key}`, phase: 'Verify', schema: VERDICT_SCHEMA, effort: 'high' }
            )
          )
        ).then((votes) => {
          const valid = votes.filter(Boolean)
          const kept = valid.filter((v) => !v.refuted).length
          const corrections = valid.map((v) => v.correction).filter((c) => c && c.length > 0)
          return {
            dimension: d.key,
            finding: f,
            survives: valid.length > 0 && kept >= 2,
            votesKept: kept,
            votesTotal: valid.length,
            objections: valid.filter((v) => v.refuted).map((v) => v.reasoning),
            corrections,
          }
        })
      )
    )
  }
)

const all = results.filter(Boolean).flat()
const confirmed = all.filter((r) => r && r.survives)
const rejected = all.filter((r) => r && !r.survives)
log(`${confirmed.length} constats confirmes sur ${all.length} examines`)

phase('Critique')
const inventaire = confirmed
  .map((r) => `[${r.dimension}][${r.finding.severity}][${r.finding.impact}] ${r.finding.claim}`)
  .join('\n')

const critique = await agent(
  `${CONTEXTE}\n\n=== CONSTATS CONFIRMES PAR L'AUDIT ===\n${inventaire || '(aucun)'}\n\n` +
    `Tu es le critique de completude. Ta question : qu'est-ce qui N'A PAS ete examine ?\n` +
    `Cherche : une hypothese dont un aspect du critere n'a ete teste par personne ; un fichier de\n` +
    `resultats que personne n'a ouvert ; une affirmation du livre couverte par la matrice mais\n` +
    `absente des six hypotheses ; une menace a la validite externe (une seule machine, une seule\n` +
    `version de Go, une seule architecture) ; une confusion entre signification statistique et\n` +
    `signification pratique ; un angle mort de la conception du banc lui-meme.\n` +
    `Rends tes constats dans le meme format, du plus grave au plus benin.`,
  { label: 'critique-completude', phase: 'Critique', schema: FINDINGS_SCHEMA, effort: 'high' }
)

```

## 2. `rediger-h007-h011` : rédaction par panel de juges

- **Exécution :** `wf_ea547a4a-6e2`, session `ef21fc00`, du 2026-09-10 15 h 55 à 16 h 11 UTC (16 min).
- **Agents : 80** = 15 auteurs (5 hypothèses × 3 angles : minimaliste, adversaire, fidélité au livre) + 60 juges (15 brouillons × 4 lentilles : réfutabilité, mécanisabilité, fidélité, robustesse au bruit) + 5 synthétiseurs.
- **Règle d'accord :** aucun vote. Chaque juge note cinq axes de 0 à 10, et la note d'un brouillon est la moyenne des juges (sur 50). Le synthétiseur part du brouillon le mieux noté, y greffe les idées des autres et doit corriger tous les défauts relevés.
- **Résultat :** une ligne de catalogue par hypothèse, H-007 à H-011.
- **Jetons :** 3,09 M en entrée nouvelle, 27,3 M en lecture de cache, 0,20 M en sortie.

```js
const ROOT = '<DEPOT>/EscapeBench'

const CONTEXTE = `
Projet EscapeBench (racine ${ROOT}) : banc de refutation Go qui eprouve des affirmations du livre
"Building Enterprise Projects with Go" (Shahsavan, Apress 2026, abrege BEPG).

Le catalogue docs/requirements.md contient une table d'hypotheses a cinq colonnes :
  | ID | Source (BEPG) | Enonce refutable | Critere de refutation | UC lies |

Regles absolues du projet :
- Un critere de refutation est GELE avant la mesure et ne se modifie JAMAIS. Une erreur de
  formulation est definitive : elle oblige a creer une nouvelle H-### et a rejeter l'ancienne.
- Le critere doit etre evaluable MECANIQUEMENT, par comparaisons numeriques sur des champs du
  modele d'entites : Comparison (deltaNsPerOp, ciLow, ciHigh, significant, sizeBytes,
  hasPointerField, lifetimeProfile, medianValueNsPerOp, medianPointerNsPerOp),
  ComparisonSet.tippingPoints (cle : profil x presence d'un champ pointeur ; valeur : taille en
  octets ou "non observe"), Measurement (nsPerOp, bytesPerOp, allocsPerOp, chacune une liste de
  count valeurs ; medianes calculees par le banc), EscapeVerdict (escapes, category parmi
  RETURN_POINTER / CLOSURE_CAPTURE / CHANNEL_SEND / CONTAINER_STORE / OTHER / NONE, status,
  compilerReason).
- Le critere enonce ce qui INFIRME. Le banc rend REFUTED si le critere se declenche, CONFIRMED
  sinon, INCONCLUSIVE si les donnees ne permettent pas de l'evaluer.
- L'enonce refutable doit etre une affirmation, pas une question, et doit pouvoir etre faux.
- Une hypothese ne doit pas etre redondante avec H-001 a H-006.

Etat actuel du banc (capacites existantes) :
- LifetimeProfile : LOCAL, RETURNED, CAPTURED_BY_CLOSURE, SENT_ON_CHANNEL, STORED_IN_MAP.
- ProbeKind : SEQUENTIAL_SCAN, SCATTERED_SCAN, APPEND_PREALLOC, APPEND_GROW.
- TypeSpec : une seule disposition, "Tag uint64" suivi de "Fill [n-1]uint64" ; tailles bornees a
  [8, 4096] octets, multiples de 8 ; variante avec ou sans champ pointeur.
- Provenance : goVersion, goos, goarch, cpuModel, capturedAt. AUCUNE taille de cache consignee.
- ComparisonSet.tippingPoints est indexe par (lifetimeProfile, hasPointerField) uniquement.

=== CE QUE LA PREMIERE CAMPAGNE A APPRIS (campagne C-2026-09-10-1, 230 sujets, verifie) ===

Verdicts obtenus : H-001 REFUTED, H-002 REFUTED, H-003 CONFIRMED, H-004 REFUTED,
H-005 CONFIRMED, H-006 CONFIRMED. Trois defauts de CONSTRUCTION ont ete demontres :

(1) La disposition "Tag + Fill [n]uint64" sort le type du passage en registres de Go des 24 octets,
    car un tableau de longueur > 1 n'est pas assignable a registres. Contre-epreuve mesuree, en
    ns/op medians : a 24 octets, "Tag + Fill [2]uint64" donne valeur 1.056 / pointeur 0.495 ;
    trois champs uint64 nommes donnent valeur 0.442 / pointeur 0.741. Meme taille, verdict inverse.
    A 32 octets, quatre champs nommes donnent encore l'avantage a la valeur (0.487 contre 0.770).
    Donc H-002 a mesure une frontiere d'ABI, pas une frontiere de taille.

(2) La sonde SCATTERED_SCAN parcourt une tranche de pointeurs melangee, donc emet des chargements
    INDEPENDANTS : elle mesure un debit, pas une latence, alors que BEPG p. 254 oppose deux
    latences. Temps par element de 64 octets mesures : 256 Ko 0.333 / 0.431 (x1.29) ;
    4 Mo 0.734 / 0.973 (x1.33) ; 32 Mo 1.924 / 3.537 (x1.84) ; 128 Mo 2.558 / 6.048 (x2.36).
    De plus le critere de H-004 traite 32 MiB comme "au-dela de tout cache L2" alors que le cache
    de dernier niveau de la machine de mesure fait 36 Mo : ce point est resident en cache.

(3) H-006 a ete confirmee par un corpus qui ne pouvait pas la refuter : les cinq profils SONT les
    quatre causes du livre plus LOCAL, et les tailles s'arretent a 4096 octets. Contre-epreuve :
    une structure de 131080 octets en profil LOCAL, mode POINTER, produit "moved to heap: t" et le
    classificateur la range correctement en OTHER (hors livre). Le classificateur n'est donc PAS
    circulaire ; c'est l'echantillon qui etait borne. La limite d'allocation de pile de Go vaut
    128 Ko pour les variables implicites.

(4) H-003 est confirmee par construction : les 22 paires RETURNED passent de 0 a 1 allocation ;
    aucun facteur 2 n'a jamais ete calcule, seule la clause d'exception "mediane valeur nulle"
    porte le verdict. Le livre parle pourtant de DOUBLER un nombre d'allocations, ce qui suppose
    une base non nulle.

(5) H-005 est le seul resultat propre : memoire x0.196 (livre : un cinquieme), allocations 27 -> 1
    (livre : 28 -> 1), mais temps x0.229, soit 4.4 fois plus rapide quand le livre annonce environ
    6 fois. Le seuil gele de H-005 (1/2) est trois fois plus lache que l'affirmation citee.
`

const CIBLES = [
  {
    id: 'H-007',
    objet: `Reprendre la question de H-002 (la regle "1 a 3 mots machine" de BEPG p. 253) sur une
disposition de type qui ne fausse pas la mesure : des structures a champs nommes, assignables a
registres. Corrige le defaut (1).`,
  },
  {
    id: 'H-008',
    objet: `Reprendre la question de H-004 (BEPG p. 254, "roughly 10x to 200x" entre succes et defaut
de cache) avec une sonde qui mesure une LATENCE : chaine de pointeurs dependante, ou chaque acces
depend du precedent. Et un seuil de jeu de travail defini par rapport au cache reellement present
sur la machine, pas par une constante. Corrige le defaut (2).`,
  },
  {
    id: 'H-009',
    objet: `Rendre refutable la question de H-006 (BEPG p. 238-242, les quatre causes d'echappement) :
formuler l'hypothese de facon qu'un corpus elargi puisse produire le contre-exemple, et definir le
corpus minimal qui rend le test valide. Corrige le defaut (3).`,
  },
  {
    id: 'H-010',
    objet: `Reprendre la question de H-003 (BEPG p. 256, "doubles the number of allocations") dans la
configuration dont parle le livre : une base d'allocation NON NULLE du cote valeur, pour qu'un
facteur 2 soit reellement calculable. Corrige le defaut (4).`,
  },
  {
    id: 'H-011',
    objet: `Eprouver les chiffres exacts que BEPG p. 114 annonce pour la preallocation (environ 6 fois
plus rapide, un cinquieme de la memoire, 28 allocations reduites a 1), la ou H-005 s'etait donne un
seuil trois fois plus lache. Corrige le defaut (5). Note d'integrite : la campagne C-2026-09-10-1
contient deja des donnees qui portent sur cette question ; le brouillon doit dire si le critere doit
etre evalue sur une nouvelle campagne.`,
  },
]

const ANGLES = [
  {
    key: 'minimal',
    consigne: `Angle MINIMALISTE. Ecris le critere le plus simple qui puisse etre faux. Prefere une
seule comparaison numerique sur un seul champ. Methode : demande-toi quelle est la plus petite
observation qui suffirait a infirmer, et n'ecris rien de plus. Mefie-toi des criteres a clauses
multiples : chaque clause est une occasion de figer une erreur.`,
  },
  {
    key: 'adversaire',
    consigne: `Angle ADVERSAIRE. Ecris d'abord la facon dont un lecteur de mauvaise foi pourrait
declarer l'hypothese confirmee sans rien prouver, ou infirmee par un artefact ; puis redige un
critere qui ferme ces portes. Prevois explicitement les cas limites : mediane nulle, donnees
absentes, corpus incapable de produire un contre-exemple, ecart statistiquement significatif mais
physiquement negligeable. Le banc a deja rendu un verdict qui basculait sur trois centiemes de
cycle : ne laisse pas ce piege se reproduire.`,
  },
  {
    key: 'fidelite',
    consigne: `Angle FIDELITE AU LIVRE. Pars du texte de BEPG et de rien d'autre. Le PDF est
disponible : ${ROOT}/../Building_Enterprise_Projects_with_Go.pdf (les folios imprimes sont decales
de +3 a +10 pages par rapport a l'index du PDF). Cite l'affirmation aussi exactement que possible,
puis redige un enonce qui dit ni plus ni moins que le livre. Signale si le livre affirme moins fort
que ce que l'hypothese lui prete, ou s'il ne borne pas ce que l'hypothese borne.`,
  },
]

phase('Brouillons')
log(`${CIBLES.length} hypotheses, ${ANGLES.length} auteurs chacune, ${ANGLES.length * 4} juges par hypothese`)

const retenues = await pipeline(
  CIBLES,
  (cible) =>
    parallel(
      ANGLES.map((angle) => () =>
        agent(
          `${CONTEXTE}\n\n=== TA MISSION : rediger ${cible.id} ===\n${cible.objet}\n\n` +
            `${angle.consigne}\n\n` +
            `Lis les fichiers du projet dont tu as besoin avant d'ecrire, en particulier\n` +
            `${ROOT}/docs/requirements.md pour le style des lignes existantes et pour ne pas\n` +
            `dupliquer H-001 a H-006, et ${ROOT}/internal/models/models.go pour les noms de champs.\n` +
            `Rends UNE ligne de catalogue complete, prete a etre collee.`,
          { label: `brouillon:${cible.id}/${angle.key}`, phase: 'Brouillons', schema: DRAFT_SCHEMA, effort: 'high' }
        )
      )
    ),
  (drafts, cible) => {
    const valides = (drafts || []).filter(Boolean)
    if (valides.length === 0) return null
    return parallel(
      valides.map((d, i) => () =>
        parallel(
          [
            'Tu juges la REFUTABILITE. Un resultat plausible peut-il declencher ce critere ? Le corpus necessaire est-il atteignable ? Une hypothese que rien ne peut infirmer vaut zero.',
            'Tu juges la MECANISABILITE. Traduis mentalement le critere en code Go sur les champs nommes du modele. Toute ambiguite qui obligerait le programmeur a choisir est une faute grave, car le texte est gele et le code devra le suivre.',
            'Tu juges la FIDELITE. Compare a ce que BEPG affirme reellement. Une hypothese qui prete au livre plus qu il ne dit produira un verdict sans valeur. Le PDF est lisible.',
            'Tu juges la ROBUSTESSE AU BRUIT ET AUX ARTEFACTS. Ce critere pourrait-il basculer sur un ecart sous le plancher de bruit du banc, ou sur un artefact de disposition, de cache ou de coeur ? Impose-t-il un effet minimal ?',
          ].map((lentille) => () =>
            agent(
              `${CONTEXTE}\n\n=== BROUILLON A JUGER (${cible.id}, auteur ${valides.length > i ? i : i}) ===\n` +
                `Source : ${d.source}\nEnonce : ${d.statement}\nCritere : ${d.criterion}\n` +
                `UC lies : ${d.useCases}\nPrerequis : ${d.prerequisite}\nJustification : ${d.rationale}\n\n` +
                `${lentille}\n\nNote de 0 a 10 sur chaque axe et nomme le defaut le plus grave.`,
              { label: `juge:${cible.id}`, phase: 'Jugement', schema: SCORE_SCHEMA, effort: 'high' }
            )
          )
        ).then((notes) => {
          const v = notes.filter(Boolean)
          const somme = v.reduce(
            (acc, n) => acc + n.falsifiability + n.mechanical + n.fidelity + n.nonRedundancy + n.robustness,
            0
          )
          return {
            draft: d,
            score: v.length ? somme / v.length : 0,
            defauts: v.map((n) => n.fatalFlaw).filter((f) => f && f.length > 0),
            verdicts: v.map((n) => n.verdict),
          }
        })
      )
    ).then((notees) => ({ id: cible.id, objet: cible.objet, candidats: notees.filter(Boolean) }))
  }
)

phase('Synthese')
const finales = await parallel(
  retenues.filter(Boolean).map((r) => () => {
    const classes = [...r.candidats].sort((a, b) => b.score - a.score)
    const expose = classes
      .map(
        (c, i) =>
          `--- CANDIDAT ${i + 1} (score ${c.score.toFixed(1)}/50)\n` +
          `Source : ${c.draft.source}\nEnonce : ${c.draft.statement}\nCritere : ${c.draft.criterion}\n` +
          `UC : ${c.draft.useCases}\nPrerequis : ${c.draft.prerequisite}\n` +
          `Defauts releves par les juges : ${c.defauts.join(' | ') || 'aucun'}`
      )
      .join('\n\n')
    return agent(
      `${CONTEXTE}\n\n=== SYNTHESE DE ${r.id} ===\n${r.objet}\n\n` +
        `Trois auteurs ont redige, quatre juges ont note chacun. Voici les candidats classes :\n\n${expose}\n\n` +
        `Produis la version FINALE de ${r.id}. Pars du candidat le mieux note, mais greffe les\n` +
        `meilleures idees des autres et corrige TOUS les defauts releves par les juges. Le texte que\n` +
        `tu produis sera gele et ne pourra plus jamais changer : relis-le comme si une erreur te\n` +
        `condamnait a creer une hypothese de remplacement. Verifie une derniere fois que le critere\n` +
        `se calcule sur des champs nommes du modele et qu'un resultat plausible peut le declencher.`,
      { label: `synthese:${r.id}`, phase: 'Synthese', schema: DRAFT_SCHEMA, effort: 'high' }
    ).then((finale) => ({
      id: r.id,
      finale,
      meilleurScore: classes[0] ? classes[0].score : 0,
      defautsCorriges: [...new Set(classes.flatMap((c) => c.defauts))],
    }))
  })
)

```

## 3. `revue-c008` : revue contradictoire de C-008

- **Exécution :** `wf_c2302683-471`, session `ef21fc00`, du 2026-09-10 16 h 44 à 17 h 18 UTC (34 min). Le rapport publié est [`REVUE-C008_2026-09-10.md`](REVUE-C008_2026-09-10.md).
- **Agents : 113** = 7 réviseurs (un par capacité ou risque) + 105 vérificateurs (35 constats × 3 lentilles : exactitude, conséquence, contre-explication; 2 bloqués par les garde-fous du modèle) + 1 critique de complétude.
- **Règle d'accord :** la même qu'au §1; au plus 5 constats par réviseur. Les réviseurs pouvaient compiler et mesurer dans un répertoire temporaire, jamais dans le dépôt.
- **Résultat (journal) :** 7 constats confirmés sur 35 examinés. Le rapport ajoute deux vérifications refaites à la main, hors agents.
- **Jetons :** 6,43 M en entrée nouvelle, 93,7 M en lecture de cache, 0,72 M en sortie.

```js
const ROOT = '<DEPOT>/EscapeBench'

const CONTEXTE = `
Projet EscapeBench (racine ${ROOT}) : banc de refutation Go qui eprouve des affirmations du livre
"Building Enterprise Projects with Go" (Shahsavan, Apress 2026, abrege BEPG).

La contrainte C-008 de ${ROOT}/docs/requirements.md vient d'etre satisfaite. Elle exige six
capacites, toutes destinees a rendre mesurables les hypotheses H-007 a H-010 sans repeter les
defauts de construction de la premiere campagne :

(a) une dimension de disposition sur TypeSpec : ARRAY_FILL (l'ancienne, "Tag uint64" + "Fill
    [n-1]uint64"), NAMED_FIELDS (wordCount champs uint64 declares un a un, sans tableau, donc
    assignable aux registres de l'ABI Go), NAMED_FIELDS_SHAM (temoin nul) ;
(b) le temoin nul NAMED_FIELDS_SHAM : ses DEUX cellules de paire executent le corps du mode VALUE,
    donc son delta vrai est nul par construction et il mesure l'ecart entre deux binaires ;
(c) le genre de Probe POINTER_CHASE : anneau de noeuds de 64 octets chaines en permutation, une
    iteration de b.N valant un seul acces dont l'adresse a ete lue a l'acces precedent ;
(d) les profils STORED_IN_SLICE et STORED_IN_STRUCT, calques sur STORED_IN_MAP ;
(e) le profil RETURNED_ALLOCATING : chaque instance produite s'accompagne d'une charge allouee,
    avec une dimension Repeat (instances par operation) pour faire varier la base ;
(f) dans la Provenance : l1DataCacheBytes, lastLevelCacheBytes, pageSizeBytes, gomaxprocs,
    detectes sur la machine et jamais saisis.

Fichiers a lire (chemins absolus) :
- ${ROOT}/docs/requirements.md            (C-008 et le texte GELE de H-007 a H-011)
- ${ROOT}/docs/entity-model.md            (modele synchronise)
- ${ROOT}/docs/use-cases/UC-001-generer-matrice.md
- ${ROOT}/internal/models/models.go       (Layout, profils, ProbeKind, Provenance, Cell.Repeat)
- ${ROOT}/internal/models/matrix.go       (Canonical, Expand, ValuePointerPairs)
- ${ROOT}/internal/harness/harness.go     (newCellData, layoutOf)
- ${ROOT}/internal/harness/templates/cell.go.tmpl
- ${ROOT}/internal/harness/templates/probe.go.tmpl
- ${ROOT}/internal/adapters/system/topology.go et topology_windows.go
- ${ROOT}/internal/adapters/store/dto.go
- ${ROOT}/internal/service/criteria_c008.go   (evaluateH007 a evaluateH010)
- ${ROOT}/internal/service/criteria.go        (evaluateH001 a H-006, H-011)

=== MESURES DEJA OBTENUES SUR LES NOUVEAUX SUJETS (a verifier, ne pas croire sur parole) ===
Machine : Intel Core Ultra 9 275HX, L1 de donnees 48 Ko, dernier niveau 36 Mo, page 4 Ko,
GOMAXPROCS 24, go1.27.0 windows/amd64. Campagne de controle, 20 repetitions, -benchtime=50ms,
-cpu=1, medianes :
  probe/POINTER_CHASE/16384        0.781 ns par acces      (resident en L1)
  probe/POINTER_CHASE/150994944  131.550 ns par acces      (au-dela du dernier niveau)
  rapport 168, donc dans [10, 200] et au-dessus de 100 ns
  Size0024Ptr/RETURNED_ALLOCATING/VALUE       1 alloc/op    POINTER   2 allocs/op
  Size0024Ptr/RETURNED_ALLOCATING_R2/VALUE    2 allocs/op   POINTER   4 allocs/op
  Size0024Ptr/RETURNED_ALLOCATING_R4/VALUE    4 allocs/op   POINTER   8 allocs/op
  Size0024Ptr/RETURNED_ALLOCATING_R16/VALUE  16 allocs/op   POINTER  32 allocs/op
  STORED_IN_MAP/POINTER, STORED_IN_SLICE/POINTER, STORED_IN_STRUCT/POINTER : echappent tous,
  classes CONTAINER_STORE ; leurs homologues VALUE n'echappent pas.
Controle d'ABI mesure precedemment, 30 repetitions : a 24 octets, "Tag + Fill [2]uint64" donne
valeur 1.056 / pointeur 0.495 ns, tandis que trois champs uint64 nommes donnent valeur 0.442 /
pointeur 0.741. Meme taille, verdict inverse.

=== CE QUI DOIT ETRE PRESERVE ===
- L'identifiant de la matrice de reference doit rester M-823d8b5af441 et son decompte 220 Cell +
  10 Probe (BR-001-4), pour que la campagne archivee C-2026-09-10-1 garde sa matrice.
- Les fichiers de resultats anterieurs a C-008 doivent rester lisibles et valides.
- L'empreinte du harnais a change, ce que C-008 annonce : les matrices anterieures ne peuvent plus
  etre etendues, et c'est le comportement voulu.
- Les criteres de H-001 a H-011 sont GELES : le code doit les appliquer a la lettre, jamais les
  reinterpreter.

Ton travail : trouver ce qui rendrait un futur verdict FAUX ou ININTERPRETABLE. Chiffre chaque
constat et cite le fichier et la ligne.
`

const DIMENSIONS = [
  {
    key: 'disposition',
    prompt: `Audite la disposition NAMED_FIELDS. Question centrale : le type produit est-il reellement
assignable aux registres de l'ABI Go a TOUTES les tailles que H-007 evalue, et cesse-t-il de l'etre
au-dela ? Verifie la regle exacte de l'ABI Go (cmd/compile/abi-internal.md) : nombre de registres
d'argument entiers sur amd64 et sur arm64, traitement des tableaux, des structs imbriquees.
La variante a champ pointeur remplace le dernier uint64 par un *uint64 : cela change-t-il
l'assignabilite ? Le seuil de 80 octets retenu comme temoin de sensibilite est-il le bon sur amd64,
et l'est-il sur arm64, ou l'ABI compte 8 registres d'argument ? Verifie enfin que sum() lit un
travail comparable entre les deux dispositions : la variante pointeur lit une adresse convertie en
uintptr, l'autre un champ entier.`,
  },
  {
    key: 'temoin-nul',
    prompt: `Audite le temoin nul NAMED_FIELDS_SHAM, sur lequel repose tout le plancher de bruit de
H-007. Lis le gabarit et newCellData. Questions : les deux cellules de la paire produisent-elles
reellement des binaires DIFFERENTS executant un travail IDENTIQUE ? Si les deux fichiers source
etaient octet pour octet identiques, le compilateur produirait-il le meme code, et le plancher
mesurerait-il alors autre chose que ce qu'il pretend ? Le nom du type differe-t-il entre les deux
cellules de la paire, ou seulement le repertoire du paquet ? Le plancher est defini comme le plus
grand |deltaNsPerOp| des comparaisons sham : trois tailles seulement, est-ce un estimateur suffisant
d'un maximum ? Et si le sham mesurait un plancher ANORMALEMENT eleve, H-007 deviendrait-elle
inrefutable ?`,
  },
  {
    key: 'chaine-dependante',
    prompt: `Audite la sonde POINTER_CHASE, qui doit mesurer une LATENCE la ou la sonde de H-004
mesurait un debit. Lis probe.go.tmpl.
Questions decisives : le compilateur ou le processeur peuvent-ils recouvrir les acces malgre la
dependance ? La boucle "for i := 0; i < n; i++ { p = p.next }" peut-elle etre deroulee de facon a
lancer plusieurs chargements en vol ? Le chainage par permutation forme-t-il bien un cycle UNIQUE
couvrant tous les noeuds, ou peut-il produire plusieurs cycles disjoints, auquel cas le jeu de
travail effectif serait plus petit que le jeu annonce ? Verifie l'algorithme exact du gabarit.
Le curseur conserve entre repetitions introduit-il un biais ? Les 131.55 ns mesures a 144 Mo sont-ils
compatibles avec une latence DRAM plus un parcours de table de pages, et le rapport de 168 est-il
assez loin de la borne 200 pour qu'un artefact de TLB ne fasse pas basculer le verdict ?`,
  },
  {
    key: 'conteneurs',
    prompt: `Audite les profils STORED_IN_SLICE et STORED_IN_STRUCT, sur lesquels porte H-009.
Lis les corps correspondants dans cell.go.tmpl et compare-les mot a mot a STORED_IN_MAP.
Questions : les trois conteneurs sont-ils reellement comparables ? Une map est allouee sur le tas
par construction ; une tranche de un element et un new(holder) locaux le sont-ils ? Si le conteneur
lui-meme n'echappe pas, l'echappement observe de la locale vient-il du conteneur ou d'autre chose ?
Le livre parle d'un conteneur "already heap-allocated" : la forme retenue est-elle fidele ?
Verifie aussi que le classificateur d'echappement attribue bien CONTAINER_STORE dans les trois cas,
et pourquoi il a fallu ajouter SelectorExpr et StarExpr a isContainerTarget : cette extension
peut-elle produire des faux positifs sur d'autres profils, par exemple en classant CONTAINER_STORE
une affectation "t.Tag = ..." ?`,
  },
  {
    key: 'allocations',
    prompt: `Audite le profil RETURNED_ALLOCATING et la dimension Repeat, sur lesquels porte H-010.
Lis le gabarit et la mesure obtenue : base R du cote valeur, 2R du cote pointeur, pour R dans
{1, 2, 4, 16}.
Questions : le doublement observe vient-il du mode de passage ou de la construction du harnais ?
Le cote pointeur alloue la charge ET la valeur retournee : est-ce le meme phenomene que celui dont
parle BEPG p. 256 ? La struct payload fait 24 octets, au-dessus du seuil de l'allocateur tiny de Go
(16 octets) : verifie que ce choix est deliberement documente et qu'une charge plus petite fausserait
le compte. La garde constantAllocs exige un compte identique sur les 20 repetitions : est-ce
realiste, et que se passe-t-il si le ramasse-miettes se declenche pendant une repetition ?
H-010 exige un etalement de facteur quatre entre bases : {1,2,4,16} le fournit-il, et le fournirait-il
encore si une paire tombait en FAILED ?`,
  },
  {
    key: 'detection-caches',
    prompt: `Audite la detection des tailles de cache, dont depend entierement H-008.
Lis topology.go, topology_windows.go et topology_linux.go.
Questions : la disposition du SYSTEM_LOGICAL_PROCESSOR_INFORMATION supposee (32 octets, relation a
l'offset 8, CACHE_DESCRIPTOR a l'offset 16 avec Level a 16, Size a 20, Type a 24) est-elle exacte sur
Windows x64 ? Verifie la documentation Microsoft. Le premier appel a GetLogicalProcessorInformation
avec un pointeur nul est-il legitime, et le code traite-t-il correctement ERROR_INSUFFICIENT_BUFFER ?
Sur une machine hybride a coeurs performance et efficacite, retenir le PLUS GRAND L1 de donnees
est-il le bon choix quand la campagne n'epingle aucune affinite et peut donc s'executer sur un coeur
efficacite au L1 plus petit ? Les bandes de H-008 (resident <= L1d/2, non resident >= 4 x dernier
niveau) restent-elles correctes si la detection rend une valeur agregee plutot que par coeur ?`,
  },
  {
    key: 'compatibilite-et-evaluateurs',
    prompt: `Audite deux choses a la fois.
D'abord la compatibilite : l'identifiant de la matrice de reference doit rester M-823d8b5af441.
Lis Canonical() dans matrix.go et verifie que les dimensions de C-008 n'y entrent que si elles
different de leur valeur par defaut. Cherche tout chemin par lequel une demande equivalente
produirait un identifiant different, ou par lequel deux demandes differentes produiraient le meme.
Verifie aussi que les DTO relisent une disposition absente comme ARRAY_FILL et un repeat absent
comme 1, et que Cell.Validate accepte encore les cellules relues d'un fichier anterieur.
Ensuite les evaluateurs : compare mot a mot le texte GELE de H-007, H-008, H-009 et H-010 dans
docs/requirements.md avec evaluateH007 a evaluateH010 dans criteria_c008.go. Le code applique-t-il
exactement la condition ecrite, ni plus ni moins ? Attention aux bornes (<= contre <), aux
quantificateurs, a l'ordre entre gardes d'evaluabilite et infirmation, et au fait que H-010 dit
"profil RETURNED" alors que le profil implemente s'appelle RETURNED_ALLOCATING : cette lecture par
famille est-elle defendable ou est-ce une reinterpretation d'un critere gele ?`,
  },
]

phase('Revue')
log(`${DIMENSIONS.length} dimensions sur la satisfaction de C-008`)

const results = await pipeline(
  DIMENSIONS,
  (d) =>
    agent(
      `${CONTEXTE}\n\n=== TA DIMENSION : ${d.key} ===\n${d.prompt}\n\n` +
        `Lis reellement les fichiers avant d'affirmer quoi que ce soit. Tu peux compiler et mesurer :\n` +
        `ecris tes essais dans un repertoire temporaire a toi, JAMAIS dans ${ROOT}.\n` +
        `Rends au plus 5 constats, du plus grave au plus benin. N'invente aucun chiffre.`,
      { label: `revue:${d.key}`, phase: 'Revue', schema: FINDINGS_SCHEMA, effort: 'high' }
    ),
  (review, d) => {
    if (!review || !review.findings || review.findings.length === 0) return []
    if (review.truncated) log(`${d.key} : liste tronquee`)
    const LENSES = [
      'exactitude : les faits cites existent-ils dans le code, et le raisonnement technique tient-il',
      'consequence : ce constat changerait-il un verdict, ou est-ce une remarque de style',
      'contre-explication : le code traite-t-il deja le cas, ailleurs, ou par une garde que le constat a manquee',
    ]
    return parallel(
      review.findings.map((f) => () =>
        parallel(
          LENSES.map((lens) => () =>
            agent(
              `${CONTEXTE}\n\n=== CONSTAT A REFUTER (dimension ${d.key}) ===\n` +
                `Affirmation : ${f.claim}\nGravite : ${f.severity}\nPreuves : ${f.evidence}\n` +
                `Impact annonce : ${f.impact}\n\nTon angle : ${lens}.\n` +
                `Verifie dans les fichiers. Cherche activement a REFUTER. En cas de doute, refuted=true.`,
              { label: `verif:${d.key}`, phase: 'Verify', schema: VERDICT_SCHEMA, effort: 'high' }
            )
          )
        ).then((votes) => {
          const valid = votes.filter(Boolean)
          const kept = valid.filter((v) => !v.refuted).length
          return {
            dimension: d.key,
            finding: f,
            survives: valid.length > 0 && kept >= 2,
            votes: `${kept}/${valid.length}`,
            objections: valid.filter((v) => v.refuted).map((v) => v.reasoning),
            corrections: valid.map((v) => v.correction).filter((c) => c && c.length > 0),
          }
        })
      )
    )
  }
)

const all = results.filter(Boolean).flat()
const confirmed = all.filter((r) => r && r.survives)
log(`${confirmed.length} constats confirmes sur ${all.length} examines`)

phase('Critique')
const inventaire = confirmed
  .map((r) => `[${r.dimension}][${r.finding.severity}] ${r.finding.claim}`)
  .join('\n')
const critique = await agent(
  `${CONTEXTE}\n\n=== CONSTATS CONFIRMES ===\n${inventaire || '(aucun)'}\n\n` +
    `Tu es le critique de completude. Qu'est-ce qui N'A PAS ete examine ?\n` +
    `Cherche : une capacite de C-008 dont un aspect n'a ete teste par personne ; un chemin de code\n` +
    `nouveau sans test ; une interaction entre deux capacites ; une regression possible sur les\n` +
    `hypotheses H-001 a H-006 dont les verdicts sont archives ; un risque de portabilite (arm64,\n` +
    `linux) ; un cas ou une future campagne rendrait un verdict que personne ne saurait interpreter.`,
  { label: 'critique-completude', phase: 'Critique', schema: FINDINGS_SCHEMA, effort: 'high' }
)

```

## 4. `rediger-h012-h013` : rédaction puis attaque

- **Exécution :** `wf_59f89d56-7b2`, session `ef21fc00`, du 2026-09-10 17 h 35 à 18 h 24 UTC (48 min).
- **Agents : 20** = 4 rédacteurs (angles : minimal, rigueur, fidélité, adversarial) + 12 attaquants (4 propositions × 3 lentilles : falsifiabilité, satisfaisabilité, forme et processus) + 1 synthétiseur + 3 contre-épreuves.
- **Règle d'accord :** aucun vote. La synthèse doit corriger chaque faille bloquante ou majeure, ou l'écarter avec une raison écrite. La contre-épreuve compte les failles bloquantes restantes.
- **Résultat (journal) :** 15 failles bloquantes relevées sur les 4 propositions; 1 faille bloquante restante après la contre-épreuve, d'où la seconde passe (§5).
- **Jetons :** 2,82 M en entrée nouvelle, 59,5 M en lecture de cache, 0,32 M en sortie.

```js
const RACINE = '<DEPOT>/EscapeBench'

const CONTEXTE = [
  'Projet EscapeBench (racine ' + RACINE + ') : banc de refutation Go qui eprouve des affirmations du livre',
  '"Building Enterprise Projects with Go" (Shahsavan, Apress 2026, abrege BEPG). Processus AI Unified Process :',
  'docs/ fait autorite, les criteres de refutation sont GELES et ne se modifient JAMAIS ; on cree une nouvelle H-### a la place.',
  '',
  'LIS TOI-MEME les sources avant de repondre. Ne deduis rien, verifie :',
  '- docs/requirements.md : le catalogue complet (contraintes C-001 a C-008, hypotheses H-001 a H-011,',
  '  et la section "Limites relevees le 2026-09-10 par la revue contradictoire de C-008" qui motive ce travail).',
  '- docs/entity-model.md : ce que le modele sait representer (TypeSpec, Layout, LifetimeProfile, Cell, Probe,',
  '  Provenance, Campaign, Measurement, Comparison, ComparisonSet).',
  '- internal/models/models.go et internal/models/matrix.go : ce que le banc sait reellement produire.',
  '- internal/service/criteria.go et internal/service/criteria_c008.go : comment un critere est execute.',
  '- internal/harness/templates/cell.go.tmpl et probe.go.tmpl : les corps mesures.',
  '- REVUE-C008_2026-09-10.md a la racine du depot parent (' + RACINE + '/..) : le rapport de la revue.',
  '',
  'TACHE : rediger H-012 et H-013, deux hypotheses successeurs.',
  'H-012 succede a H-007. H-013 succede a H-008. Elles ne remplacent ni ne corrigent leurs ainees,',
  'dont les criteres restent geles et les verdicts acquis. Elles reprennent la meme affirmation du livre',
  'sur un sujet de mesure qui ne porte plus le defaut demontre par la revue.',
  '',
  'Les deux defauts a neutraliser, tels que la revue les a etablis et verifies :',
  '(a) H-007 : le temoin nul NAMED_FIELDS_SHAM fait executer aux deux cellules de la paire le meme corps.',
  '    Les deux sources generees ne different que par deux lignes de commentaire et le desassemblage donne',
  '    le meme code machine. Le plancher mesure donc la variance entre deux processus pour un code identique',
  '    (de l ordre du centieme de nanoseconde), alors que la dispersion d une meme paire reelle entre deux',
  '    executions vaut six a sept centiemes. Le temoin sous-estime le bruit contre lequel il protege.',
  '    De plus le plancher de H-007 est le maximum sur TOUTE la serie, sans appariement de taille.',
  '(b) H-008 : le rapport mesure vaut environ 168 pour un plafond de 200, soit 19 pour cent de marge.',
  '    Avec huit processus de flux memoire sur les autres coeurs, la sonde non residente passe de 130,7 a',
  '    274,4 ns pendant que la sonde residente ne bouge que de 10 pour cent : le rapport monte a 317 et',
  '    infirme H-008 pour une raison etrangere a l hypothese. Le critere gele de H-008 exige EXACTEMENT une',
  '    Probe POINTER_CHASE par bande, ce qui interdit d y ajouter une sonde de controle.',
  '',
  'CONTRAINTES DE FORME, imperatives :',
  '- Chaque hypothese est UNE ligne de tableau Markdown a cinq cellules, dans cet ordre :',
  '  ID, Source (BEPG), Enonce refutable, Critere de refutation, UC lies.',
  '- Le parseur (internal/adapters/specs/specs.go, splitRow) coupe sur le caractere barre verticale :',
  '  AUCUNE barre verticale ne doit apparaitre dans le texte d une cellule, sous aucune forme.',
  '- Redige en francais canadien. Les termes techniques, noms de commandes et identifiants de code restent',
  '  en anglais. Les citations de BEPG restent en anglais entre guillemets.',
  '- Les UC lies se choisissent parmi UC-001 a UC-005 selon ce que l hypothese exige reellement.',
  '',
  'CONTRAINTES DE FOND, tirees des trois precautions que le catalogue s impose deja et des lecons de la revue :',
  '1. Un critere ne bascule jamais sur un ecart inferieur au bruit, et le plancher de bruit doit etre',
  '   homogene a la comparaison qu il borne.',
  '2. Un critere exige un temoin de sensibilite ou de specificite, faute de quoi un verdict ne distingue pas',
  '   un banc muet d une affirmation vraie.',
  '3. Un critere ne doit pas etre une identite arithmetique du gabarit : verifie qu au moins une mesure',
  '   physiquement possible peut l INFIRMER. C est le defaut qui rendait H-010 inrefutable.',
  '4. Un critere dit explicitement quand il est NON CONCLUANT, et se declare non concluant sur toute campagne',
  '   dont les mesures sont anterieures a sa redaction.',
  '5. Si l hypothese exige du banc une capacite qu il n a pas, NE PAS faire semblant : nommer la capacite',
  '   manquante. Le catalogue a un precedent exact, la contrainte C-008, qui a nomme les capacites exigees par',
  '   H-007 a H-010 avant qu elles soient construites. Propose alors une contrainte C-009 sur ce modele.',
  '',
  'Reponds uniquement par l objet structure demande.',
].join('\n')

const ANGLES = [
  {
    cle: 'minimal',
    prompt: 'ANGLE : le moins de capacites nouvelles possible. Cherche d abord si le corpus et le banc actuels '
      + 'suffisent deja, en changeant seulement le TEXTE du critere. Exemple a eprouver : le plancher de bruit '
      + 'de H-007 est le maximum sur toute la serie ; un critere qui dirait plutot le temoin de MEME sizeBytes '
      + 'n exige aucune capacite nouvelle, puisque le temoin nul existe deja aux tailles 8, 16 et 24. '
      + 'Ne propose une capacite nouvelle que si tu demontres qu aucune formulation ne s en passe.',
  },
  {
    cle: 'rigueur',
    prompt: 'ANGLE : rigueur statistique et metrologique. Le plancher doit etre homogene a la comparaison qu il '
      + 'borne. Exige un temoin de sensibilite ET un temoin de specificite. Pense a la dispersion inter-executions, '
      + 'pas seulement intra-execution. Pour H-013, souviens-toi que la campagne mesure sujet par sujet, avec les '
      + 'count repetitions consecutives (internal/service/campaign.go, boucle sur matrix.SubjectIDs) : une '
      + 'perturbation stable sur toute la campagne est invisible a la dispersion intra-sujet.',
  },
  {
    cle: 'fidelite',
    prompt: 'ANGLE : fidelite a ce que BEPG affirme reellement. Relis les pages citees telles que le catalogue les '
      + 'rapporte. Une hypothese successeur doit eprouver la MEME affirmation du livre, pas une affirmation voisine '
      + 'plus commode. Verifie que les tolerances viennent des mots du livre (about, roughly, typically) et non d un '
      + 'confort de mesure. Signale si l affirmation, bien lue, ne dit pas ce que l ainee lui faisait dire.',
  },
  {
    cle: 'adversarial',
    prompt: 'ANGLE : redige en trichant contre toi-meme. Pour chaque formulation que tu envisages, demande-toi '
      + 'comment un banc de mauvaise foi, ou un gabarit maladroit, la rendrait automatiquement vraie ou '
      + 'automatiquement fausse, puis ferme la porte dans le texte du critere. Passe en revue les trois pieges '
      + 'deja rencontres par ce projet : identite arithmetique du gabarit (H-010), corpus incapable de refuter '
      + '(H-006), plancher de bruit non borne (H-007). Assure-toi qu aucun ne se reproduit.',
  },
]

const LENTILLES = [
  {
    cle: 'falsifiabilite',
    prompt: 'LENTILLE : falsifiabilite. Ta seule question : existe-t-il une mesure physiquement possible qui '
      + 'INFIRME ce critere ? Et une qui le CONFIRME ? Si l une des deux branches est fermee par construction, '
      + 'c est bloquant. Cherche l identite arithmetique cachee, la garde de non-conclusion qui avale tous les cas, '
      + 'le seuil que le materiel ne peut pas franchir. Chiffre quand tu peux.',
  },
  {
    cle: 'satisfaisabilite',
    prompt: 'LENTILLE : satisfaisabilite par le banc. Le corpus que ce critere exige peut-il exister ? Verifie dans '
      + 'internal/models/matrix.go (Expand, Validate) et internal/harness/templates/ que chaque sujet nomme est '
      + 'produisible, et dans internal/models/models.go que chaque grandeur lue existe au modele. Verifie aussi que '
      + 'les exigences du critere ne se contredisent pas entre elles, comme la restriction du temoin nul aux petites '
      + 'tailles contredirait un temoin de sensibilite a 80 octets si elle etait un refus et non un saut. '
      + 'Si une capacite manque, dis exactement laquelle et si la C-009 proposee la couvre.',
  },
  {
    cle: 'forme-et-processus',
    prompt: 'LENTILLE : forme et processus AIUP. Verifie que chaque ligne proposee se parse : cinq cellules, aucune '
      + 'barre verticale dans le texte. Verifie que l ajout ne touche ni l enonce ni le critere de H-001 a H-011, '
      + 'donc que l empreinte gelee de la campagne C-2026-09-10-1 survit (internal/adapters/specs/specs.go, Digest). '
      + 'Verifie la coherence des UC lies avec ce que le critere exige vraiment, et que les identifiants ne sont pas '
      + 'reutilises. Signale toute prose du critere qui serait ambigue a l execution, car un evaluateur Go devra '
      + 'l appliquer a la lettre.',
  },
]

phase('Rediger')
log('4 redacteurs independants, puis 3 lentilles adversariales chacun')

const attaquees = await pipeline(
  ANGLES,
  (a) => agent(CONTEXTE + '\n\n' + a.prompt, {
    label: 'redige:' + a.cle,
    phase: 'Rediger',
    schema: SCHEMA_PROPOSITION,
  }),
  (proposition, angle) => {
    if (!proposition) return null
    const rendu = JSON.stringify(proposition, null, 2)
    return parallel(LENTILLES.map((l) => () => agent(
      CONTEXTE + '\n\n' + l.prompt
      + '\n\nPROPOSITION A ATTAQUER (angle ' + angle.cle + ') :\n' + rendu
      + '\n\nTa tache est de la REFUTER, pas de l approuver. Ne rapporte une faille que si tu peux la prouver '
      + 'en citant un fichier et une ligne, ou en executant quelque chose. Si tu ne trouves rien de bloquant, '
      + 'dis-le franchement et rapporte quand meme les failles mineures.',
      { label: 'attaque:' + angle.cle + ':' + l.cle, phase: 'Attaquer', schema: SCHEMA_ATTAQUE },
    ))).then((verdicts) => ({ angle: angle.cle, proposition, verdicts: verdicts.filter(Boolean) }))
  },
)

const retenues = attaquees.filter(Boolean)
const bloquants = retenues.flatMap((r) => r.verdicts.flatMap((v) => v.failles.filter((f) => f.gravite === 'bloquant')))
log(retenues.length + ' propositions attaquees, ' + bloquants.length + ' failles bloquantes relevees')

phase('Synthetiser')
const dossier = retenues.map((r) => 'PROPOSITION ' + r.angle + '\n'
  + JSON.stringify(r.proposition, null, 2) + '\n'
  + 'ATTAQUES SUBIES :\n' + JSON.stringify(r.verdicts, null, 2)).join('\n\n' + '='.repeat(80) + '\n\n')


const synthese = await agent(
  CONTEXTE
  + '\n\nQuatre redacteurs ont propose, et chaque proposition a subi trois attaques adversariales.'
  + '\nVoici le dossier complet.\n\n' + dossier
  + '\n\nProduis LA version retenue de H-012, de H-013 et, si elle est necessaire, de C-009.'
  + '\nPrends la meilleure formulation de chaque proposition et greffe les bonnes idees des autres.'
  + '\nToute faille bloquante ou majeure doit etre soit corrigee dans le texte, soit ecartee avec une raison ecrite.'
  + '\nRelis une derniere fois : cinq cellules, aucune barre verticale, francais canadien, criteres qui nomment'
  + '\nexplicitement leurs conditions de non-conclusion, et une branche d infirmation reellement atteignable.'
  + '\nEcris aussi le paragraphe de revision a inserer dans docs/requirements.md avant le tableau des hypotheses,'
  + '\ndans le meme ton que les paragraphes de revision qui s y trouvent deja.',
  { label: 'synthese', phase: 'Synthetiser', schema: SCHEMA_SYNTHESE, effort: 'high' },
)

phase('Contre-epreuve')
const rendu = JSON.stringify(synthese, null, 2)
const CONTRE = [
  'Tente de demontrer que H-012 telle que retenue ne peut pas etre infirmee par une mesure reelle sur cette machine, '
  + 'ou au contraire qu elle sera infirmee quoi qu il arrive. Construis le scenario chiffre.',
  'Tente de demontrer que H-013 telle que retenue ne peut pas etre infirmee, ou qu elle sera non concluante en '
  + 'pratique sur toute campagne realiste. Construis le scenario chiffre.',
  'Verifie que les deux lignes se parsent, que le catalogue reste coherent, que l empreinte gelee de la campagne '
  + 'C-2026-09-10-1 survit, et que C-009 telle que redigee couvre exactement les capacites que les deux criteres '
  + 'exigent, ni plus ni moins. Signale tout ecart, meme mineur.',
]

const contre = (await parallel(CONTRE.map((c, i) => () => agent(
  CONTEXTE + '\n\nVERSION RETENUE :\n' + rendu + '\n\n' + c
  + '\n\nSois adversarial : ton but est de la faire tomber. Prouve chaque affirmation en citant un fichier et une '
  + 'ligne, ou en executant quelque chose. Si elle tient, dis-le et rapporte les failles mineures qui restent.',
  { label: 'contre-epreuve:' + (i + 1), phase: 'Contre-epreuve', schema: SCHEMA_ATTAQUE, effort: 'high' },
)))).filter(Boolean)

const bloquantsFinaux = contre.flatMap((c) => c.failles.filter((f) => f.gravite === 'bloquant'))
log('contre-epreuve : ' + bloquantsFinaux.length + ' failles bloquantes restantes')

```

## 5. `h012-h013-passe2` : seconde passe

- **Exécution :** `wf_6a4d196d-654`, session `ef21fc00`, du 2026-09-10 18 h 28 à 19 h 07 UTC (39 min). Le dossier de la première passe (`<SCRATCHPAD>/dossier-h012-h013-passe1.md`), que lisaient les agents, n'est pas conservé.
- **Agents : 13** = 3 réparateurs (stratégies : renoncer, estimateur, domaine de machine) + 9 épreuves (3 réparations × 3 lentilles : atteignabilité, ambiguïté, direction de l'erreur) + 1 arrêt de la version finale.
- **Règle d'accord :** comme au §4. De plus, une hypothèse qui ne peut que confirmer sur toute machine disponible ne doit pas être gelée.
- **Résultat (journal) :** 7 failles bloquantes restantes après les épreuves, tranchées par l'agent d'arrêt.
- **Jetons :** 2,33 M en entrée nouvelle, 26,5 M en lecture de cache, 0,23 M en sortie.

```js
const RACINE = '<DEPOT>/EscapeBench'
const DOSSIER = '<SCRATCHPAD>/dossier-h012-h013-passe1.md'

const CONTEXTE = [
  'Projet EscapeBench (racine ' + RACINE + ') : banc de refutation Go qui eprouve des affirmations du livre',
  '"Building Enterprise Projects with Go" (Shahsavan, Apress 2026, abrege BEPG). Les criteres de refutation sont',
  'GELES et ne se modifient jamais une fois ecrits ; on cree une nouvelle H-### a la place. C est pourquoi il faut',
  'les mettre a l epreuve AVANT de les ecrire, et c est l objet de ce travail.',
  '',
  'LIS D ABORD, en entier : ' + DOSSIER,
  'Ce fichier contient la version de H-012 et H-013 issue d une premiere passe, la contrainte de capacite C-009',
  'proposee, et surtout la CONTRE-EPREUVE, qui a mesure sur la machine reelle et laisse des failles ouvertes.',
  '',
  'LIS AUSSI dans le depot, ne deduis rien :',
  '- docs/requirements.md : catalogue complet, contraintes C-001 a C-008, hypotheses H-001 a H-011, et la section',
  '  "Limites relevees le 2026-09-10 par la revue contradictoire de C-008".',
  '- docs/entity-model.md, internal/models/models.go, internal/models/matrix.go : ce que le banc sait produire.',
  '- internal/service/criteria.go et criteria_c008.go : comment un critere est execute a la lettre.',
  '- internal/harness/templates/probe.go.tmpl et cell.go.tmpl : les corps mesures.',
  '',
  'MESURES ETABLIES PAR LA CONTRE-EPREUVE, a tenir pour acquises sauf si tu les refais toi-meme :',
  'Machine : go1.27.0 windows/amd64, L1d 49152 o, dernier niveau 37748736 o, 24 processeurs logiques.',
  'Sondes POINTER_CHASE au repos, -cpu=1 -benchtime=250ms, mediane de 21 : residente 24576 o vaut 0,80 ns ;',
  'intermediaire 4194304 o vaut 12,29 ns ; non residente 150994944 o vaut 132,4 ns ; rapport 165,6.',
  'Sous charge memoire graduee, la non residente monte jusqu a 274,4 ns pendant que la residente ne bouge que de',
  '10 pour cent ; le rapport atteint 317. L ordre des trois sondes est conserve sous toute charge.',
  'Cellules NAMED_FIELDS en profil LOCAL, trois replicats, matrice M-8f03757ac206 :',
  '  24 octets, sans champ pointeur : deltas +0,3796 +0,4035 +0,3680, significatifs, plancher 0,0436.',
  '  24 octets, avec champ pointeur : deltas +0,3585 +0,2599 +0,4111.',
  '  16 octets : medianes ciHigh -0,0177 et +0,0180, planchers 0,2310 et 0,2147.',
  '  8 octets, avec champ pointeur : deltas -0,2706 -0,1930 -0,2170, ciHigh medians -0,1946, plancher 0,0824.',
  '  8 octets, sans champ pointeur : medianes valeur 0,7913 0,8156 0,8248.',
  '  Temoins de sensibilite : 128 octets, effet 0,97 ns ; 1024 octets, effet 6,47 ns. Les deux qualifient.',
  '',
  'CE QUI DOIT ETRE FERME. La contre-epreuve a etabli, mesures a l appui :',
  'H-013, faille BLOQUANTE. Aucun plafond absolu de quietude ne peut fonctionner. La branche "rapport superieur',
  '  a 200" bascule des que la latence non residente depasse 200 fois la mediane residente, soit environ 161 ns',
  '  ici ; un plafond a 200 ns laisse une fenetre de 39 ns ou une charge legere infirme a tort. Poser le plafond',
  '  la ou il protege revient a le poser la ou l infirmation commence, donc a tuer la branche.',
  'H-013, faille MAJEURE. Au repos, aucune des deux branches d infirmation n est atteignable sur cette machine :',
  '  132,4 ns est au-dessus de 100, et le rapport 165,6 est sous 200. La campagne propre ne peut que confirmer.',
  '  C est exactement le reproche que le catalogue fait a H-006, confirmee par un corpus incapable de la refuter.',
  'H-013, faille MAJEURE. Le temoin d ordonnancement est aveugle : une charge gonfle les trois sondes en',
  '  conservant leur ordre, donc il passe dans toutes les campagnes contaminees.',
  'H-012, faille MAJEURE. La garde de resolution est inerte : son seuil, la moitie de l effet du temoin de',
  '  sensibilite, vaut 0,49 ou 3,24 ns alors que les planchers reels valent 0,04 a 0,23 ns.',
  'H-012, faille MAJEURE. "Le temoin de sensibilite de sa serie" ne designe pas un objet unique : deux tailles',
  '  qualifient, et les deux lectures donnent des seuils qui different d un facteur 6,6.',
  'H-012, faille MAJEURE. Le plancher est l etendue de trois tirages : coefficient de variation d environ 52 pour',
  '  cent, non borne en haut, et l erreur pousse toujours vers CONFIRMED. Un replicat dissident fixe le plancher.',
  'H-012, faille MAJEURE. La regle "au moins deux des trois petites tailles" est inatteignable sur amd64 : a 24',
  '  octets le pointeur est significativement plus LENT parce que trois mots nommes tiennent dans les registres',
  '  d argument, a 16 octets l effet est nul, seule la taille 8 avec champ pointeur bascule. Au plus une taille',
  '  peut donc basculer, et la branche d infirmation est inatteignable en pratique.',
  'Plus plusieurs failles mineures, toutes decrites dans le dossier.',
  '',
  'TA TACHE : produire une version de H-012 et de H-013 qui ferme ces failles, ou qui declare honnetement ce',
  'qu elle renonce a eprouver. Les deux sont acceptables ; ce qui ne l est pas, c est un critere qui ne peut que',
  'confirmer sans le dire. Le catalogue prefere un verdict NON CONCLUANT honnete a un CONFIRMED faux : c est la',
  'lecon qu il a tiree de H-006, et elle est ecrite dans docs/requirements.md.',
  '',
  'Il est legitime de conclure qu une des deux hypotheses ne doit PAS etre ecrite en l etat, et de dire quelle',
  'capacite ou quelle machine la rendrait eprouvable. Il est legitime de retirer une branche du critere si elle',
  'est indissociable d un artefact, a condition de le dire dans la source et dans l enonce.',
  '',
  'CONTRAINTES DE FORME, imperatives :',
  '- Chaque hypothese est UNE ligne de tableau Markdown a cinq cellules : ID, Source, Enonce, Critere, UC lies.',
  '- Le parseur coupe sur la barre verticale : AUCUNE barre verticale dans le texte d une cellule.',
  '- Francais canadien. Termes techniques, commandes et identifiants en anglais. Citations de BEPG en anglais.',
  '- Un critere nomme explicitement ses conditions de non-conclusion.',
  '',
  'Reponds uniquement par l objet structure demande.',
].join('\n')

const STRATEGIES = [
  {
    cle: 'renoncer',
    prompt: 'STRATEGIE : renoncer proprement a ce qui n est pas eprouvable, et le dire. Pour H-013, prends au serieux '
      + 'l issue que la contre-epreuve juge honnete : retirer la branche du rapport, indissociable d un artefact de '
      + 'contention, et ne garder que ce que la page 254 permet d eprouver sur ce banc. Pour H-012, examine si la '
      + 'regle des deux tailles sur trois doit tomber a une seule assortie d une exigence de marge. Le prix a payer '
      + 'doit apparaitre dans la cellule Source et dans l enonce, pas seulement dans une justification.',
  },
  {
    cle: 'estimateur',
    prompt: 'STRATEGIE : reparer par la metrologie plutot que par le renoncement. Pour H-012, remplace l etendue de '
      + 'trois tirages par un estimateur dont la dispersion propre est maitrisee, et fixe le nombre de replicats dans '
      + 'le critere pour que le plancher ne grandisse pas avec l effort de mesure. Ancre la garde de resolution sur '
      + 'l echelle ou le verdict se decide, pas sur celle du temoin. Pour H-013, cherche une garde de quietude qui '
      + 'soit une grandeur MESUREE et non un seuil choisi, en te servant de la sonde intermediaire ou d un rapport '
      + 'entre sondes ; verifie contre les chiffres du dossier qu elle separe reellement le repos de la charge, et si '
      + 'elle ne separe pas, dis-le au lieu de la proposer.',
  },
  {
    cle: 'machine',
    prompt: 'STRATEGIE : rendre explicite le domaine de validite. Une hypothese peut etre eprouvable sur une classe de '
      + 'machines et pas sur une autre. Ecris dans le critere la condition de machine sous laquelle chaque branche '
      + 'd infirmation est atteignable, exprimee sur des grandeurs que la Provenance ou les mesures portent deja, et '
      + 'declare la non concluante hors de ce domaine. Verifie ensuite, chiffres du dossier en main, ce que cela donne '
      + 'sur la machine du catalogue : si H-013 y est non concluante par construction, dis-le franchement plutot que '
      + 'de maquiller un CONFIRMED. Pense aussi a C-006, qui prevoit deja arm64.',
  },
]

const LENTILLES = [
  {
    cle: 'atteignabilite',
    prompt: 'LENTILLE : atteignabilite chiffree. Pour chaque branche d infirmation, calcule avec les chiffres du '
      + 'dossier ce qu il faudrait mesurer pour la faire basculer, et dis si c est atteignable au repos sur cette '
      + 'machine. Une branche qui ne bascule que sous contamination ne compte pas. Une branche qui exige une machine '
      + 'que le catalogue ne possede pas doit etre declaree comme telle dans le critere. Refais les mesures si tu en '
      + 'as besoin, mais dis-le.',
  },
  {
    cle: 'ambiguite',
    prompt: 'LENTILLE : ambiguite d execution. Un evaluateur Go devra appliquer ce texte a la lettre. Traque chaque '
      + 'expression qui ne designe pas un objet unique, chaque seuil dont l unite ou la reference est implicite, '
      + 'chaque ordre d evaluation non dit entre une garde et une branche. Le precedent est au dossier : evaluateH007 '
      + 'tranche une ambiguite par un break sur le premier element du fichier. Verifie aussi le format des cinq '
      + 'cellules et l absence de barre verticale.',
  },
  {
    cle: 'direction-erreur',
    prompt: 'LENTILLE : direction de l erreur. Pour chaque estimateur et chaque garde, demande-toi de quel cote une '
      + 'erreur pousse le verdict. Le defaut central de H-007 etait que son plancher sous-estimait le bruit ; la '
      + 'premiere version de H-012 le surestime de facon incontrolee, et les deux erreurs poussent du meme cote, '
      + 'celui du livre. Verifie que la nouvelle version ne reproduit pas ce biais, et qu une erreur de mesure peut '
      + 'pousser le verdict dans les deux sens.',
  },
]

phase('Reparer')
log('3 strategies de reparation, chacune eprouvee par 3 lentilles')

const eprouvees = await pipeline(
  STRATEGIES,
  (s) => agent(CONTEXTE + '\n\n' + s.prompt, {
    label: 'repare:' + s.cle, phase: 'Reparer', schema: SCHEMA_REPARATION, effort: 'high',
  }),
  (rep, strat) => {
    if (!rep) return null
    const rendu = JSON.stringify(rep, null, 2)
    return parallel(LENTILLES.map((l) => () => agent(
      CONTEXTE + '\n\n' + l.prompt
      + '\n\nREPARATION A EPROUVER (strategie ' + strat.cle + ') :\n' + rendu
      + '\n\nTon but est de la faire tomber. Prouve chaque affirmation par une lecture de fichier avec sa ligne, '
      + 'par un chiffre du dossier, ou par une mesure que tu executes. Si elle tient, dis-le et rapporte les '
      + 'failles mineures qui restent.',
      { label: 'eprouve:' + strat.cle + ':' + l.cle, phase: 'Eprouver', schema: SCHEMA_EPREUVE, effort: 'high' },
    ))).then((v) => ({ strategie: strat.cle, reparation: rep, epreuves: v.filter(Boolean) }))
  },
)

const dossier2 = eprouvees.filter(Boolean).map((r) => 'REPARATION ' + r.strategie + '\n'
  + JSON.stringify(r.reparation, null, 2) + '\n\nEPREUVES SUBIES :\n'
  + JSON.stringify(r.epreuves, null, 2)).join('\n\n' + '='.repeat(80) + '\n\n')

const bloquants = eprouvees.filter(Boolean).flatMap((r) => r.epreuves.flatMap((e) => e.failles.filter((f) => f.gravite === 'bloquant')))
log(eprouvees.filter(Boolean).length + ' reparations eprouvees, ' + bloquants.length + ' failles bloquantes restantes')


const finale = await agent(
  CONTEXTE
  + '\n\nTrois reparations ont ete proposees et chacune eprouvee par trois lentilles adversariales.'
  + '\nVoici le dossier complet de la seconde passe.\n\n' + dossier2
  + '\n\nArrete la version FINALE de H-012 et de H-013, et de C-009 si elle reste necessaire.'
  + '\nPrends la meilleure formulation de chaque reparation et greffe ce que les autres ont de bon.'
  + '\nToute faille bloquante ou majeure est soit fermee dans le texte, soit assumee avec sa raison ecrite.'
  + '\nSi une hypothese ne doit pas etre ecrite en l etat, mets son champ ecrire a faux et dis ce qui la rendrait'
  + '\neprouvable. Une hypothese qui ne peut que confirmer sur toute machine disponible ne doit pas etre gelee.'
  + '\nDonne aussi la specification de matrice exacte qu une campagne portant ces hypotheses devrait employer.'
  + '\nRelis une derniere fois : cinq cellules, aucune barre verticale, conditions de non-conclusion explicites.',
  { label: 'version-finale', phase: 'Arreter', schema: SCHEMA_FINAL, effort: 'high' },
)

```

## 6. `audit-escapebench` : audit du code

- **Exécution :** `wf_e30c209a-4c3`, session `2f7ab848`, lancée le 2026-09-11 à 16 h 18 UTC, reprise quatre fois le 2026-09-12 entre 11 h 34 et 11 h 58 UTC, terminée à 12 h 19 UTC. La dernière reprise a duré 21 min.
- **Agents : 581** dans l'état final = 18 découvreurs (un par paquet ou lentille transversale) + 1 agent de fusion + 561 vérificateurs (187 constats × 3 rôles; 373 ont répondu, 188 ont échoué) + 1 critique de complétude (échoué). On compte **1 378 fichiers de transcription** de sous-agents (1 377 horodatés), parce que chaque reprise relance des agents : 921 sans aucune réponse, 303 sous `claude-fable-5-1`, 154 sous `claude-opus-5`. Le fichier d'exécution déclare pourtant `claude-opus-5` pour les 581 agents.
- **Rôles :** le *réfutateur* cherche la garde en amont ou l'erreur de lecture. Le *reproducteur* travaille dans un `git worktree` isolé et y exécute un test jetable. Le *juge de spécification* cherche si `docs/` ou une décision prescrit le comportement dénoncé.
- **Règle d'accord :** un constat est confirmé si au moins 2 des votes valides le maintiennent. La sévérité retenue est la plus fréquente parmi ces votes. Un vérificateur sans réponse est affiché comme réfutation, mais n'entre pas dans le décompte.
- **Interruption :** les 189 échecs viennent des limites d'usage du compte (168 sur la limite de dépense mensuelle, 21 sur la limite de session). La seconde ronde, prévue après la critique de complétude, n'a pas eu lieu.
- **Résultat (journal) :** 290 constats bruts, 187 après dédoublonnage, 119 confirmés. `Doc/AUDIT.md` publie un autre décompte : 190 items, 102 confirmés, 4 rejetés, 84 non vérifiés. L'écart est expliqué depuis le 2026-09-22 par D-65 ([`Doc/DECISION.md`](../Doc/DECISION.md)) et l'[erratum de `Doc/AUDIT.md`](../Doc/AUDIT.md#erratum-du-2026-09-22--décomptes-de-la-méthode) : constats inscrits deux fois, confirmés publiés comme non vérifiés, non vérifiés que le fichier réfute par défaut.
- **Paramètres d'exécution.** `ROOT` vaut `<DEPOT>/EscapeBench`. `BASELINE` est injecté dans le prompt commun; voici son texte, abrégé : « go1.27.0 windows/amd64 ; go vet OK ; gofmt vide ; go test -race -shuffle=on OK ; couverture par paquet de 83,8 % à 98,1 % ; selftest des hooks OK ; staticcheck inutilisable avec go1.27 ; results/.campaign-lock absent ; git status propre ; aucun fichier de test ne semble porter le tag integration_test (à confirmer) ».
- **Jetons :** 21,49 M en entrée nouvelle, 261,4 M en lecture de cache, 2,44 M en sortie.

```js
const ROOT = args.root
const BASELINE = args.baseline

const SEV = ['bloquant', 'majeur', 'mineur', 'suggestion']

const COMMON = `Tu audites le dépôt Go EscapeBench. Racine du module : ${ROOT} (module github.com/agbruneau/escapebench, Go 1.25+, bibliothèque standard seulement, architecture hexagonale : cmd/escapebench, internal/{models,ports,service,adapters/*,harness}).
Règles absolues :
- Tu ne modifies AUCUN fichier du dépôt. Lecture, grep, go doc, go vet, go test, go run en lecture seule uniquement. Aucune écriture sous results/, matrices/, docs/.
- Lis ENTIÈREMENT chaque fichier de ton périmètre, du début à la fin, puis trace les appelants et appelés hors périmètre avec Grep quand un constat en dépend. Ne rapporte que ce que tu as lu.
- Un constat = un défaut précis, ancré sur fichier:ligne, avec un extrait de code exact et un scénario d'échec concret (entrées ou état -> sortie fausse, plantage, ressource fuitée, règle nommée violée). Pas d'impression, pas de généralité.
- Sévérités : bloquant (résultat de mesure ou verdict faux, perte ou corruption de données, violation d'une règle BR/C/NFR sans garde, plantage sur un chemin normal) ; majeur (comportement erroné sur un cas limite plausible, fuite de ressource, portabilité Windows/Linux cassée, test qui ne peut pas échouer sur une règle) ; mineur (défaut réel mais contenu) ; suggestion (simplification, lisibilité, test manquant sans risque immédiat).
- Zéro constat est une réponse valable : ne baisse pas la barre pour remplir. Mais ne tais rien non plus : les constats mineurs et les suggestions comptent, l'utilisateur veut tout.
- Pour chaque correctif, dis s'il touche internal/harness/templates/*.tmpl (cela change Campaign.harnessDigest et invalide toute matrice et campagne antérieures : C-005, BR-003-1) ou s'il exigerait de modifier le texte d'un critère gelé H-### dans docs/requirements.md (interdit par le processus : on crée une nouvelle H-###).
- Contexte déjà vérifié par l'orchestrateur, ne le refais pas : ${BASELINE}
- Le projet est déclaré clos le 2026-09-10 (README) ; l'audit vise à préparer une planification de correctifs, il ne juge pas la clôture.
Rends tes constats en français canadien, identifiants et noms Go en anglais.`

const LENSES = [
  { key: 'models', title: 'Paquet internal/models', files: ['internal/models/models.go', 'internal/models/matrix.go', 'internal/models/models_test.go', 'internal/models/matrix_test.go', 'internal/models/replicat_test.go', 'internal/models/revue_test.go', 'internal/models/layout_test.go'],
    focus: 'Exactitude des invariants (Validate, Canonical, Expand, ProfileSegment, identifiants), stabilité des identifiants de matrice (BR-001-1 : une même demande donne le même identifiant), Expand sur toutes les combinaisons (dispositions, répétitions, charges, réplicats), cas limites numériques (tailles, k, sizeBytes/8), champs optionnels de Provenance et Measurement, absence de dépendance hors stdlib et de tags (C-004).' },
  { key: 'service-generation', title: 'Services UC-001, UC-002, UC-003', files: ['internal/service/matrix.go', 'internal/service/escape.go', 'internal/service/campaign.go', 'internal/service/matrix_test.go', 'internal/service/escape_test.go', 'internal/service/campaign_test.go', 'internal/service/fakes_test.go', 'internal/ports/ports.go'],
    focus: 'Flux principaux et alternatifs (A1..An) des UC-001, UC-002, UC-003 tels que décrits dans docs/use-cases ; reprise de campagne (--resume), verrou de campagne (AcquireLock/ReleaseLock sur toutes les sorties, y compris annulation de contexte et erreur), gel des hypothèses (freezeCriteria, hypothesesDigest), refus H-007 sur matrice à réplicats (C-009), ordre d exécution des sujets, propagation du contexte, absence d I/O directe dans service (CLAUDE.md), erreurs enveloppées avec %w, aucun panic.' },
  { key: 'service-analysis', title: 'Services UC-004, UC-005 et évaluateurs de critères', files: ['internal/service/compare.go', 'internal/service/verdict.go', 'internal/service/criteria.go', 'internal/service/criteria_c008.go', 'internal/service/criteria_c009.go', 'internal/service/criteria_c010.go', 'internal/service/compare_test.go', 'internal/service/verdict_test.go', 'internal/service/criteria_c008_test.go', 'internal/service/criteria_c009_test.go', 'internal/service/criteria_c010_test.go', 'internal/service/criteria_h011_test.go'],
    focus: 'Fidélité de chaque évaluateur H-001..H-013 au texte gelé de docs/requirements.md (lis la ligne de chaque hypothèse) : clauses de non-conclusion évaluées dans l ordre prescrit, bornes inclusives/exclusives, médianes, rapports, bandes de résidence (H-008/H-013), plancher et barrière de H-012, quiétude de H-013 ; appariement valeur/pointeur (ValuePointerPairs), points de bascule (TippingPoints, tri sort.Slice non stable), empreinte des critères (Digest) et refus si le texte change ; VerdictReports multi-campagnes pour le tableau de bord.' },
  { key: 'store', title: 'Adapter store (persistance, DTO, verrou)', files: ['internal/adapters/store/store.go', 'internal/adapters/store/dto.go', 'internal/adapters/store/store_test.go', 'internal/adapters/store/layout_test.go'],
    focus: 'Immutabilité des résultats (NFR-004 : jamais de réécriture d un dossier de campagne ; WriteMeasurement additif), atomicité des écritures (fichier partiel en cas de plantage, rename atomique), NextCampaignID (collisions, tri lexicographique de suffixes numériques 1..10+), verrou results/.campaign-lock (acquisition, libération, verrou orphelin), conversion entité<->DTO (omitempty et renormalisation replicate/repeat/payload, champs optionnels de Provenance C-008 et quiétude C-010, valeurs par défaut), chemins construits à partir d identifiants (traversée, caractères interdits sous Windows), LatestComparisonSet et LatestVerdictReport (ordre), erreurs %w et errors.Is aux bords, fermeture des fichiers.' },
  { key: 'gotool-system', title: 'Adapters gotool et system (processus, plateforme)', files: ['internal/adapters/gotool/gotool.go', 'internal/adapters/gotool/treecpu_windows.go', 'internal/adapters/gotool/treecpu_other.go', 'internal/adapters/gotool/gotool_test.go', 'internal/adapters/gotool/quietude_test.go', 'internal/adapters/system/system.go', 'internal/adapters/system/quietude.go', 'internal/adapters/system/quietude_linux.go', 'internal/adapters/system/quietude_windows.go', 'internal/adapters/system/quietude_other.go', 'internal/adapters/system/topology.go', 'internal/adapters/system/topology_linux.go', 'internal/adapters/system/topology_windows.go', 'internal/adapters/system/topology_other.go', 'internal/adapters/system/system_test.go', 'internal/adapters/system/quietude_test.go', 'internal/adapters/system/topology_test.go'],
    focus: 'Lancement de go build / go test (exec.CommandContext, propagation de l annulation, arbre de processus sous Windows via Job Object, fuite de handles, sortie tronquée), analyse de la sortie de go test -bench (formats ns/op, B/op, allocs/op, unités, lignes multiples, -cpu), drapeaux C-003 (count, benchtime, cpu), attestation de quiétude C-010 (GetSystemTimes, /proc/stat, /proc/vmstat, soustraction du temps propre, division par zéro, fenêtre trop courte, Measured=false), topologie de cache (CPUID, /sys, valeurs absentes), Clock, ProvenanceProbe (NFR-001 champs obligatoires), tags de build corrects et complets pour les trois plateformes (windows, linux, autres), syscall seulement (C-002).' },
  { key: 'escape-harness', title: 'Classificateur d échappement et harnais', files: ['internal/adapters/escape/escape.go', 'internal/adapters/escape/escape_test.go', 'internal/adapters/escape/revue_test.go', 'internal/harness/harness.go', 'internal/harness/harness_test.go', 'internal/harness/layout_test.go', 'internal/harness/templates/cell.go.tmpl', 'internal/harness/templates/probe.go.tmpl', 'internal/harness/templates/subject_test.go.tmpl', 'internal/harness/templates/go.mod.tmpl'],
    focus: 'Analyse des lignes -gcflags=-m (formats du compilateur Go 1.25-1.27 : "moved to heap", "escapes to heap", "leaking param", positions fichier:ligne:col, chemins Windows avec lettre de lecteur), go/ast pour retrouver la variable et sa cause (retour, closure, canal, conteneur ; alias non traversés, D-20), verdict OTHER, NFR-002 reproductibilité ; gabarits : exactitude des types générés (unsafe.Sizeof, NAMED_FIELDS, champ pointeur), b.ReportAllocs, setup hors boucle, élimination par le compilateur (sink, résultat non consommé), sondes (SEQUENTIAL_SCAN, SCATTERED_SCAN, APPEND_*, POINTER_CHASE : permutation, anneau, taille d un nœud), go.mod généré (version), Digest (SHA-256 sur les gabarits embarqués, ordre des fichiers déterministe). Signale explicitement tout correctif qui changerait le digest.' },
  { key: 'cli-specs-dashboard-cmd', title: 'Adapters cli, specs, dashboard et composition root', files: ['internal/adapters/cli/params.go', 'internal/adapters/cli/render.go', 'internal/adapters/cli/cli_test.go', 'internal/adapters/cli/layout_test.go', 'internal/adapters/specs/specs.go', 'internal/adapters/specs/index.go', 'internal/adapters/specs/specs_test.go', 'internal/adapters/dashboard/dashboard.go', 'internal/adapters/dashboard/dashboard_test.go', 'cmd/escapebench/main.go', 'cmd/escapebench/main_test.go'],
    focus: 'ParseParameters (clés inconnues, doublons, valeurs vides, espaces, ordre, valeurs par défaut D-19), ParseHypotheses, FindRoot (remontée, symlinks, racine absente), rendu (format, arrondis, nil), lecture de docs/requirements.md (analyse du tableau Markdown : cellules contenant des barres verticales, colonnes, lignes de continuation, identifiants H-### dans le texte), lecture des statuts de UC, Index de code (build tag integration_test référencé mais aucun fichier ne le porte ?), écriture de docs/dashboard.md (rendu déterministe, ordre), câblage de main.go (chaque dépendance injectée au bon port, signal.NotifyContext, codes de sortie, sortie stdout/stderr, RenderMatrix imprimé même si err != nil), usage vs drapeaux réellement déclarés.' },
  { key: 'build-ci-hooks', title: 'Makefile, CI, hooks, skills, agents', files: ['Makefile', '.github/workflows/ci.yml', '.claude/settings.json', '.claude/hooks/guard-paths.sh', '.claude/hooks/go-check.sh', '.claude/hooks/go-test.sh', '.claude/hooks/spec-lint.sh', '.claude/hooks/selftest.sh', '.claude/skills/bench/SKILL.md', '.claude/skills/go-test/SKILL.md', '.claude/skills/implement/SKILL.md', '.claude/skills/refute/SKILL.md', '.claude/skills/spec-coverage/SKILL.md', '.claude/skills/spec-review/SKILL.md', '.claude/agents/code-reviewer.md', '.claude/agents/spec-reviewer.md', '.gitignore', '.gitattributes', 'go.mod'],
    focus: 'Cible integration_test du Makefile sans fichier taggé (vérifie par grep -r "go:build" ) ; cohérence Makefile/CI/CLAUDE.md (commandes, drapeaux, versions Go : go.mod dit 1.25, poste go1.27) ; hooks : robustesse à jq absent, chemins Windows, set -e et codes de sortie, guard-paths contournable (chemin relatif, ../, casse sous Windows, tool_input.file_path absent pour MultiEdit/NotebookEdit), go-test au Stop sur gros dépôt, spec-lint (regex, faux positifs sur "should" dans les citations anglaises du livre) ; skills et agents qui citent des commandes, chemins ou drapeaux qui n existent pas dans le code ; .gitattributes et LF ; CI sans cache ni timeout ; CI n exécute pas les tests d intégration.' },
  { key: 'claude-md-conformity', title: 'Lentille transversale : conformité à CLAUDE.md et C-004', files: ['cmd/escapebench/main.go', 'internal/models/models.go', 'internal/models/matrix.go', 'internal/service/campaign.go', 'internal/service/matrix.go', 'internal/service/escape.go', 'internal/service/compare.go', 'internal/service/verdict.go', 'internal/service/criteria.go', 'internal/service/criteria_c008.go', 'internal/service/criteria_c009.go', 'internal/service/criteria_c010.go', 'internal/ports/ports.go', 'internal/adapters/store/store.go', 'internal/adapters/gotool/gotool.go', 'internal/adapters/escape/escape.go', 'internal/adapters/specs/specs.go', 'internal/adapters/dashboard/dashboard.go', 'internal/harness/harness.go'],
    focus: 'Vérifie mécaniquement chaque règle de construction du CLAUDE.md du projet : internal/models n importe que la stdlib et ne porte aucun tag ; internal/service ne fait aucune I/O directe (os, io, net, exec, fmt.Print) ; adapters n importent jamais service ; toute fonction d I/O prend context.Context en premier ; erreurs enveloppées %w et non fmt.Errorf("%v") ni errors.New perdant la cause ; aucun panic sur un chemin de requête (grep panic, log.Fatal, os.Exit hors main) ; commentaire d en-tête citant le UC sur chaque fonction de service ; tests nommés TestUC###_... et table-driven ; usage de testing/synctest pour le temporel et absence de time.Sleep dans les tests ; benchmarks avec b.ReportAllocs. Utilise go list -deps, grep, et lis les fichiers.' },
  { key: 'concurrency-resources', title: 'Lentille transversale : concurrence, contexte, ressources', files: ['internal/service/campaign.go', 'internal/service/escape.go', 'internal/service/matrix.go', 'internal/adapters/gotool/gotool.go', 'internal/adapters/gotool/treecpu_windows.go', 'internal/adapters/gotool/treecpu_other.go', 'internal/adapters/system/quietude.go', 'internal/adapters/system/quietude_windows.go', 'internal/adapters/system/quietude_linux.go', 'internal/adapters/store/store.go', 'cmd/escapebench/main.go'],
    focus: 'Goroutines et canaux (fuite, blocage, absence de WaitGroup), annulation du contexte (SIGINT pendant une campagne : verrou libéré ? statut ABORTED écrit ? processus enfant go test tué ? fichiers partiels ?), exec.CommandContext et WaitDelay, lecture de stdout/stderr (deadlock sur pipe pleine, CombinedOutput), handles Windows (Job Object, CloseHandle), fichiers ouverts non fermés, defer dans les boucles, accès concurrents à des maps ou à des variables de paquet, horloge injectée vs time.Now direct, déterminisme des itérations de map (ordre des sujets, des fichiers écrits, des rapports).' },
  { key: 'security-robustness', title: 'Lentille transversale : robustesse aux entrées et sécurité', files: ['internal/adapters/cli/params.go', 'internal/adapters/store/store.go', 'internal/adapters/store/dto.go', 'internal/adapters/specs/specs.go', 'internal/adapters/gotool/gotool.go', 'internal/adapters/escape/escape.go', 'internal/harness/harness.go', 'internal/models/matrix.go', 'cmd/escapebench/main.go'],
    focus: 'Frontières de confiance : identifiants de matrice et de campagne passés en drapeau puis utilisés dans des chemins (filepath.Join avec ../, identifiants vides, caractères réservés Windows) ; paramètres de matrice injectés dans des gabarits Go (text/template sans échappement : un nom de profil ou une taille peut-il produire du code non voulu ?) ; arguments passés à go test/go build (drapeau -benchtime non validé -> injection de drapeau ?) ; JSON lu du disque (champs manquants, valeurs négatives, NaN, très grands nombres) ; requirements.md analysé (lignes malformées) ; ReadFile sans limite ; permissions des fichiers créés ; chemins absolus dans les résultats (fuite du chemin utilisateur dans results/ versionné ?).' },
  { key: 'portability', title: 'Lentille transversale : portabilité Windows/Linux/macOS', files: ['internal/adapters/store/store.go', 'internal/adapters/gotool/gotool.go', 'internal/adapters/gotool/treecpu_windows.go', 'internal/adapters/gotool/treecpu_other.go', 'internal/adapters/system/system.go', 'internal/adapters/system/quietude_windows.go', 'internal/adapters/system/quietude_linux.go', 'internal/adapters/system/quietude_other.go', 'internal/adapters/system/topology_windows.go', 'internal/adapters/system/topology_linux.go', 'internal/adapters/system/topology_other.go', 'internal/adapters/escape/escape.go', 'internal/adapters/specs/index.go', 'internal/adapters/cli/params.go', 'internal/harness/harness.go', '.claude/hooks/guard-paths.sh', '.claude/hooks/selftest.sh'],
    focus: 'Le poste de référence est Windows, la CI est ubuntu. Cherche : path vs filepath, séparateurs codés en dur, comparaison de chemins sensibles à la casse, fins de ligne CRLF dans la sortie du compilateur ou de go test (strings.Split sur \\n avec \\r résiduel), lettre de lecteur dans les positions fichier:ligne:col du compilateur (C:\\... contient un deux-points), tags de build manquants (darwin, freebsd tombent dans _other ?), syscall Windows (chargement de kernel32, signatures), /proc absent, CPU count, GOOS/GOARCH dans la provenance, exécutable go introuvable, temps système en unités différentes selon plateforme, tests qui ne s exécutent que sur une plateforme et masquent une régression sur l autre.' },
  { key: 'test-quality', title: 'Lentille transversale : qualité des tests', files: ['internal/service/campaign_test.go', 'internal/service/verdict_test.go', 'internal/service/compare_test.go', 'internal/service/criteria_c008_test.go', 'internal/service/criteria_c009_test.go', 'internal/service/criteria_c010_test.go', 'internal/service/criteria_h011_test.go', 'internal/service/matrix_test.go', 'internal/service/escape_test.go', 'internal/service/fakes_test.go', 'internal/adapters/store/store_test.go', 'internal/adapters/gotool/gotool_test.go', 'internal/adapters/escape/escape_test.go', 'internal/harness/harness_test.go', 'internal/models/models_test.go', 'internal/models/matrix_test.go', 'cmd/escapebench/main_test.go'],
    focus: 'Tests qui ne peuvent pas échouer (assertion absente, tautologie, erreur ignorée, t.Skip large), tests qui vérifient l implémentation plutôt que la postcondition, fakes qui masquent le comportement réel (fake store qui ne persiste rien), flux alternatifs et règles BR sans test (compare avec docs/use-cases/UC-00x.md : liste les A# et BR-###-# sans TestUC###_ correspondant), dépendance à l ordre (rompue par -shuffle ?), dépendance au système de fichiers réel ou au binaire go dans des tests unitaires, temps réel (time.Now, time.Sleep) au lieu de synctest, tests lents, tests de layout (layout_test.go) : que vérifient-ils vraiment ? Propose pour chaque test faible la mutation qui devrait le faire échouer.' },
  { key: 'statistics', title: 'Lentille transversale : statistiques et arithmétique des verdicts', files: ['internal/service/compare.go', 'internal/service/criteria.go', 'internal/service/criteria_c008.go', 'internal/service/criteria_c009.go', 'internal/service/criteria_c010.go', 'internal/service/verdict.go', 'internal/models/models.go', 'internal/adapters/gotool/gotool.go'],
    focus: 'Médiane sur nombre pair, bootstrap percentile (2000 rééchantillonnages, graine dérivée de la paire : math/rand vs math/rand/v2, déterminisme entre versions de Go et entre plateformes, quantiles 2,5/97,5 : interpolation ou index), signification = intervalle exclut zéro (BR-004-2, strict ou large ?), division par zéro dans les rapports (médiane valeur 0), comparaison de flottants en égalité (H-010 : "exactement le double" avec des float64), NaN et Inf, conversion int/float, ordre de tri et stabilité (sort.Slice), point de bascule (D-15 : plus long suffixe favorable), rapport (k+1)/k de H-010, plancher et barrière de H-012 (étendue des trois deltas de rang médian sur cinq), fraction d occupation C-010 (bornes 0..1, cœurs), unités (ns, octets), arrondis dans les rendus qui pourraient masquer une différence.' },
  { key: 'simplification', title: 'Lentille transversale : sur-ingénierie et simplification (ponytail)', files: ['internal/models/models.go', 'internal/models/matrix.go', 'internal/service/campaign.go', 'internal/service/verdict.go', 'internal/service/criteria.go', 'internal/service/criteria_c008.go', 'internal/service/criteria_c009.go', 'internal/adapters/store/store.go', 'internal/adapters/store/dto.go', 'internal/adapters/cli/render.go', 'internal/adapters/specs/specs.go', 'internal/harness/harness.go', 'internal/ports/ports.go'],
    focus: 'Code réinventant la stdlib (slices, maps, sort.SliceStable, strings.Cut, errors.Join, cmp), abstractions à une seule implémentation non exigées par un test (interfaces jamais mockées), duplication entre fichiers criteria_*.go (helpers de médiane, de filtrage, de bandes copiés), fonctions mortes (go vet ne le voit pas : grep chaque symbole exporté et non exporté), paramètres jamais variés, code de compatibilité pour des fichiers antérieurs qui n existent pas dans results/, commentaires qui contredisent le code. Sévérité suggestion sauf si la duplication cache déjà une divergence réelle (alors majeur). Cite pour chaque item ce qui le remplace et le nombre de lignes retirées.' },
  { key: 'spec-drift-uc1-2', title: 'Dérive spécification/code : UC-001 et UC-002', files: ['docs/use-cases/UC-001-generer-matrice.md', 'docs/use-cases/UC-002-classer-echappement.md', 'docs/entity-model.md', 'internal/service/matrix.go', 'internal/service/escape.go', 'internal/models/matrix.go', 'internal/models/models.go', 'internal/adapters/escape/escape.go', 'internal/harness/harness.go'],
    focus: 'Lis chaque UC section par section (préconditions, scénario principal étape par étape, flux alternatifs, postconditions succès/échec, règles BR-001-#, BR-002-#) et vérifie que le code fait exactement cela, ni plus ni moins : étape observable absente, précondition non vérifiée (verrou de campagne, matrice existante), flux alternatif non branché, règle sans garde, comportement ajouté hors UC. Vérifie aussi que docs/entity-model.md nomme les attributs que le code porte (Cell.Replicate, Provenance.l1DataCacheBytes, quiétude) et que le statut déclaré (Deployed) correspond. Un écart où la spec est lacunaire se rapporte en catégorie conformite-spec avec la mention "spec à mettre à jour d abord".' },
  { key: 'spec-drift-uc3', title: 'Dérive spécification/code : UC-003', files: ['docs/use-cases/UC-003-executer-campagne.md', 'docs/entity-model.md', 'internal/service/campaign.go', 'internal/adapters/store/store.go', 'internal/adapters/gotool/gotool.go', 'internal/models/models.go'],
    focus: 'Lis UC-003 section par section (préconditions, scénario principal, A1 harnais modifié, A2.., A4 reprise, postconditions, BR-003-1 à BR-003-5, C-003, C-005, C-009, C-010, NFR-001, NFR-003, NFR-004) et vérifie que le code fait exactement cela : verrou et harnais figé, empreintes (harnessDigest, hypothesesDigest), count >= 20, provenance complète, immutabilité, reprise sans remesurer, statut final, ordre des sujets (réplicats séparés par une passe), refus H-007 + réplicats, attestation de quiétude attachée à chaque Measurement. Signale tout comportement du code absent du UC et toute étape du UC absente du code.' },
  { key: 'spec-drift-uc4-5', title: 'Dérive spécification/code : UC-004 et UC-005', files: ['docs/use-cases/UC-004-comparer-valeur-pointeur.md', 'docs/use-cases/UC-005-produire-verdicts.md', 'docs/dashboard.md', 'internal/service/compare.go', 'internal/service/verdict.go', 'internal/service/criteria.go', 'internal/adapters/dashboard/dashboard.go', 'internal/adapters/specs/specs.go', 'internal/adapters/specs/index.go'],
    focus: 'Lis UC-004 et UC-005 section par section et vérifie le code : appariement, IC bootstrap, point de bascule (étape 5, D-15, artefact C-009 : UC-004 ne doit pas publier la bascule d une série répliquée), BR-004-#, BR-005-# (dashboard régénéré jamais édité, empreinte gelée refusée si texte changé, hypothèse sans évaluateur -> INCONCLUSIVE), étape 7 dashboard (colonnes Code/Unit/Integration/Regression/Integrity : d où viennent-elles, sont-elles calculées ou constantes ?), verdicts multi-campagnes. Compare docs/dashboard.md actuel au rendu attendu par dashboard.go (colonnes, ordre, valeurs) et signale toute divergence ou valeur codée en dur.' },
]

function findingsToText(list) {
  return list.map(f => `- [${f.id}] ${f.severity} | ${f.category} | ${f.file}:${f.line} | ${f.title}`).join('\n')
}

function finderPrompt(l, seen) {
  const seenBlock = seen.length
    ? `\nConstats DÉJÀ rapportés par d'autres découvreurs (ne les répète pas ; rapporte seulement ce qui est nouveau ou ce qui les contredit) :\n${findingsToText(seen)}\n`
    : ''
  return `${COMMON}

Ton périmètre : ${l.title}.
Fichiers à lire intégralement (relatifs à la racine) :
${l.files.map(f => '- ' + f).join('\n')}
Ce que tu cherches : ${l.focus}
${seenBlock}
Méthode : lis tout, puis pour chaque soupçon trace le chemin d'exécution réel (appelants, tests existants) avant de le rapporter. Vérifie qu'un test existant ne couvre pas déjà le cas. Si tu peux confirmer par une commande en lecture seule (go test -run, go vet, go doc, go list), fais-le et cite la sortie dans evidence ou failure_scenario. Rapporte ensuite chaque constat, du plus grave au moins grave, dans le schéma demandé. Dans notes, dis ce que tu as lu, ce que tu n'as pas pu vérifier et pourquoi.`
}

const VERIFIER_LENSES = [
  { key: 'refute', effort: 'high', isolation: undefined, prompt: f => `${COMMON}

Ton rôle : RÉFUTER un constat d'audit produit par un autre agent. Par défaut, si tu n'arrives pas à établir concrètement que le défaut est réel, refuted=true.
Constat :
${JSON.stringify(f, null, 2)}

Procède ainsi : lis entièrement ${f.file}, puis chaque appelant et appelé pertinent (Grep). Cherche la garde en amont qui rend le scénario impossible, le test existant qui le couvre, la spécification (docs/use-cases, docs/requirements.md, CLAUDE.md, ../Doc/DECISION.md) qui prescrit ce comportement, ou l'erreur de lecture du découvreur (mauvaise ligne, extrait inexact, version de Go). Si le scénario d'échec tient réellement, dis précisément pourquoi (fichier:ligne, valeurs) et retiens une sévérité. Si le constat est vrai mais la sévérité exagérée, refuted=false avec la sévérité corrigée. Si le correctif proposé est faux ou incomplet, corrige-le dans corrected_fix.` },
  { key: 'reproduce', effort: 'high', isolation: 'worktree', prompt: f => `${COMMON}

Ton rôle : REPRODUIRE un constat d'audit produit par un autre agent, par l'exécution quand c'est possible.
Constat :
${JSON.stringify(f, null, 2)}

Tu travailles dans une copie isolée du dépôt (git worktree) : tu PEUX y ajouter un fichier de test jetable nommé zz_audit_repro_test.go dans le paquet concerné (package identique au paquet, ou paquet _test), l'exécuter avec go test -run <Nom> -count=1 ./chemin/du/paquet, consigner la sortie exacte, puis SUPPRIMER ce fichier. N'écris jamais sous results/, matrices/ ni docs/. Ne modifie aucun fichier existant.
Si la reproduction par test est impossible (dépend du système, de go test réel, d'un signal), trace à la main un chemin d'exécution complet avec des valeurs concrètes, ligne par ligne, et dis explicitement que c'est une trace et non une exécution.
Pour un constat de catégorie tests, simplification, documentation ou conformite-*, la reproduction consiste à vérifier le fait affirmé (grep, go list -deps, lecture) et, pour un test faible, à appliquer la mutation proposée et constater que le test passe encore (puis restaurer).
refuted=true si le comportement observé n'est pas celui que le constat annonce ou si le fait affirmé est faux. Mets la commande et la sortie dans reproduction.` },
  { key: 'spec', effort: 'medium', isolation: undefined, prompt: f => `${COMMON}

Ton rôle : JUGER un constat d'audit au regard de la spécification et du processus du projet.
Constat :
${JSON.stringify(f, null, 2)}

Lis les documents cités ou pertinents : docs/use-cases/UC-###.md concerné, la ligne de docs/requirements.md pour chaque FR/NFR/C/H cité, docs/entity-model.md, CLAUDE.md du projet, ../Doc/DECISION.md (décisions D-01 à D-38 qui assument certains écarts). Puis lis ${f.file} autour de la ligne ${f.line}.
Tranche : (1) le comportement dénoncé est-il prescrit ou assumé par la spécification ou une décision D-## ? Alors refuted=true, en citant le passage. (2) Le défaut est-il réel mais la spec lacunaire ? Alors refuted=false, et dis dans corrected_fix que la spec doit changer d'abord (quel document, quelle section), conformément au processus. (3) Le correctif proposé violerait-il une règle du processus (texte d'un critère gelé, écriture dans results/, gabarit du harnais qui invalide les campagnes sans nouvelle H-###) ? Alors garde refuted=false si le défaut est réel, mais réécris le correctif dans corrected_fix pour qu'il respecte le processus. Retiens la sévérité selon l'échelle du projet.` },
]

function tally(f, votes) {
  const valid = votes.filter(Boolean)
  const kept = valid.filter(v => !v.refuted)
  const confirmed = kept.length >= 2
  const counts = {}
  for (const v of kept) counts[v.severity] = (counts[v.severity] || 0) + 1
  let severity = f.severity
  let best = 0
  for (const s of SEV) if ((counts[s] || 0) > best) { best = counts[s]; severity = s }
  return {
    ...f,
    confirmed,
    final_severity: confirmed ? severity : f.severity,
    votes: VERIFIER_LENSES.map((l, i) => ({ lens: l.key, ...(votes[i] || { refuted: true, confidence: 'low', reasoning: 'agent sans réponse', severity: f.severity }) })),
  }
}

async function verifyAll(list, phaseName) {
  return pipeline(list, f => parallel(VERIFIER_LENSES.map(l => () =>
    agent(l.prompt(f), { label: `${l.key}:${f.id}`, phase: phaseName, schema: VERDICT_SCHEMA, effort: l.effort, isolation: l.isolation })
  )).then(votes => tally(f, votes)))
}

async function mergeFindings(fresh, seen, phaseName) {
  if (!fresh.length) return []
  const seenBlock = seen.length ? `\nConstats déjà retenus d'une ronde précédente (un constat neuf qui les répète est un doublon : keep = identifiant du constat ancien) :\n${findingsToText(seen)}\n` : ''
  const res = await agent(`Tu dédoublonnes des constats d'audit sur le dépôt Go EscapeBench (racine ${ROOT}). Deux constats sont des doublons s'ils désignent le même défaut au même endroit ou le même défaut dupliqué à plusieurs endroits par copier-coller (alors un seul cluster, keep = le plus précis, et note les autres emplacements dans note). Deux constats sur le même fichier mais des défauts différents ne sont PAS des doublons. Ne fusionne pas deux constats de catégories différentes portant sur la même ligne si les défauts sont distincts. Ne rapporte que les clusters contenant au moins un doublon ; un constat unique n'a pas besoin de cluster. Lis les fichiers concernés si le titre ne suffit pas à trancher.
${seenBlock}
Constats neufs :
${JSON.stringify(fresh.map(f => ({ id: f.id, file: f.file, line: f.line, severity: f.severity, category: f.category, title: f.title, description: f.description, fix: f.fix })), null, 1)}`,
    { label: 'fusion', phase: phaseName, schema: MERGE_SCHEMA, effort: 'medium' })

// ---------------- Ronde 1 ----------------
phase('Découverte')
const r1raw = await parallel(LENSES.map(l => () => agent(finderPrompt(l, []), { label: `find:${l.key}`, phase: 'Découverte', schema: FINDINGS_SCHEMA, effort: 'high' })))
const finderNotes = {}
let r1 = []
r1raw.forEach((r, i) => {
  const lens = LENSES[i].key
  if (!r) { finderNotes[lens] = 'AGENT SANS RÉPONSE'; log(`Découvreur ${lens} sans réponse`); return }
  finderNotes[lens] = r.notes
  r1.push(...r.findings.map(f => ({ ...f, lens })))
})
r1 = stamp(r1, 'A')
log(`Ronde 1 : ${r1.length} constats bruts de ${LENSES.length} découvreurs`)

phase('Fusion')
const merged1 = await mergeFindings(r1, [], 'Fusion')

phase('Vérification')
const verified1 = (await verifyAll(merged1, 'Vérification')).filter(Boolean)
const conf1 = verified1.filter(v => v.confirmed)
log(`Ronde 1 : ${conf1.length} confirmés / ${verified1.length} vérifiés`)

// ---------------- Complétude ----------------
phase('Complétude')
const gaps = await agent(`${COMMON}

Tu es le critique de complétude d'un audit. Voici les lentilles déjà exécutées (périmètre et focus) :
${LENSES.map(l => `- ${l.key} : ${l.title} — ${l.focus}`).join('\n')}

Constats confirmés jusqu'ici :
${findingsToText(conf1)}

Constats réfutés (ne pas relancer) :
${findingsToText(verified1.filter(v => !v.confirmed))}

Question : qu'est-ce qui manque ? Un fichier ou un chemin d'exécution qu'aucune lentille n'a lu (compare avec git ls-files '*.go' '*.tmpl' '*.sh' '*.md' sous ${ROOT}), une classe de défauts non cherchée (p. ex. cohérence entre les rapports de campagne dans ../Campagnes et ce que le code produit réellement ; results/ versionnés : les fichiers JSON présents sont-ils lisibles par les DTO actuels, go run ./cmd/escapebench compare --campaign C-2026-09-10-1 rejoue-t-il sans erreur ? ; docs/dashboard.md à jour ; LANCEMENT.md et README exacts ; go.mod 1.25 vs fonctionnalités 1.26/1.27 utilisées ; comportement sous go vet de Go 1.27 ; interactions entre constats confirmés). Explore le dépôt pour fonder chaque angle sur un fait, puis propose au plus 6 angles nouveaux, chacun avec des fichiers précis et un focus qu'un découvreur peut exécuter. N'inclus pas d'angle déjà couvert.`,
  { label: 'critique', phase: 'Complétude', schema: GAPS_SCHEMA, effort: 'high' })

let r2 = [], merged2 = [], verified2 = []
const angles = gaps ? gaps.angles.slice(0, 6) : []
if (angles.length) {
  log(`Complétude : ${angles.length} angles nouveaux — ${angles.map(a => a.key).join(', ')}`)
  const seen = verified1
  const r2raw = await parallel(angles.map(a => () => agent(finderPrompt(a, seen), { label: `find2:${a.key}`, phase: 'Complétude', schema: FINDINGS_SCHEMA, effort: 'high' })))
  r2raw.forEach((r, i) => {
    const lens = 'r2:' + angles[i].key
    if (!r) { finderNotes[lens] = 'AGENT SANS RÉPONSE'; return }
    finderNotes[lens] = r.notes
    r2.push(...r.findings.map(f => ({ ...f, lens })))
  })
  r2 = stamp(r2, 'B')
  log(`Ronde 2 : ${r2.length} constats bruts`)
  merged2 = await mergeFindings(r2, verified1, 'Complétude')
  verified2 = (await verifyAll(merged2, 'Complétude')).filter(Boolean)
  log(`Ronde 2 : ${verified2.filter(v => v.confirmed).length} confirmés / ${verified2.length} vérifiés`)
}

```
