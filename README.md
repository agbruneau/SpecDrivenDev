# Prospection : Cadrage et Développement IA avec Claude Code

Dépôt de prospection : identification, évaluation et cadrage de projets d'exploration à développer avec Claude Code. Le dépôt contient les ouvrages de référence analysés, les documents de cadrage et le premier projet mené à terme, EscapeBench (P1), clos le 2026-09-10 avec un verdict pour chacune de ses treize hypothèses.

## Ouvrages de référence

Les trois PDF sont à la racine du dépôt. Les pages citées ailleurs dans le dépôt sont les folios imprimés.

1. **Shahsavan, Saeed** (2026). *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*. Apress. ISBN 979-8-8688-2369-5 (imprimé), 979-8-8688-2370-1 (numérique). DOI [10.1007/979-8-8688-2370-1](https://doi.org/10.1007/979-8-8688-2370-1).
   Fichier : [`Building_Enterprise_Projects_with_Go.pdf`](Building_Enterprise_Projects_with_Go.pdf). Rôle : source des affirmations réfutables cartographiées dans `Projets-candidats…` et éprouvées par EscapeBench (H-001 à H-013).
2. **Martinelli, Simon** (2026). *Spec-Driven Development: From Specs to Code with AI Agents*. Apress, coll. « Apress Pocket Guides ». ISBN 979-8-8688-2850-8 (imprimé), 979-8-8688-2851-5 (numérique). DOI [10.1007/979-8-8688-2851-5](https://doi.org/10.1007/979-8-8688-2851-5).
   Fichier : [`Spec-Driven Development.pdf`](Spec-Driven%20Development.pdf). Rôle : *AI Unified Process* (AIUP) — catalogue d'exigences, modèle d'entités, cas d'utilisation, traçabilité — adapté dans `Guide-implementation…` et appliqué au noyau `EscapeBench/docs/`.
3. **Marco, Eden** (2026). *Agentic Coding with Claude Code: The everyday developer's guide to agentic coding with Claude Code*. Birmingham (R.-U.) : Packt Publishing. Première publication mars 2026 (© 2025). ISBN 978-1-80602-259-5.
   Fichier : [`Agentic_Coding_with_Claude_Code.pdf`](Agentic_Coding_with_Claude_Code.pdf). Rôle : réglages Claude Code retenus — mémoire, skills, sous-agents, hooks, MCP, plan mode, parallélisme — repris dans `Guide-implementation…` et dans `EscapeBench/.claude/`.

## Documents

| Fichier | Contenu | Lire quand |
|---|---|---|
| `RAPPORT-FINAL_EscapeBench.md` | **Verdicts consolidés des treize hypothèses** : sept infirmations, six confirmations, ce que chaque confirmation vaut, les quatre défauts de construction trouvés et ce qui reste ouvert | **À lire en premier pour connaître les résultats**, et pour citer un verdict |
| `Projets-candidats_Building-Enterprise-Projects-with-Go.md` | Cartographie des affirmations réfutables du livre de Shahsavan, grille d'évaluation, huit fiches de projets (P1–P8), séquence recommandée, méthode d'implémentation commune | Pour choisir un projet |
| `Guide-implementation_AIUP-Claude-Code.md` | Méthode d'implémentation : AIUP (Martinelli) adapté aux bancs de réfutation Go ; réglages Claude Code retenus de Marco ; cycle par cas d'utilisation, traçabilité, pièges | Avant d'ouvrir le dépôt d'un projet |
| `EscapeBench/` | Projet P1, clos : noyau de spécification (`docs/` — vision, FR/NFR/C/H, modèle d'entités, diagramme, UC-001 à UC-005 à `Deployed`, tableau de bord), `CLAUDE.md`, six skills, deux sous-agents, quatre hooks avec leur contrôle (`selftest.sh`), implémentation Go (`cmd/`, `internal/`), CI, résultats archivés (`results/` — matrices, campagnes, verdicts), `LANCEMENT.md` | Pour rejouer une campagne ou reprendre le banc ; gabarit pour P2–P8 |
| `DECISION.md` | Décisions D-01 à D-38 : écarts de processus assumés, conception du harnais, méthode statistique, décisions issues des revues, clôture, ce qui est délibérément absent | Avant de modifier le banc ou de lancer une campagne |
| `REVUE-PRELANCEMENT_2026-09-10.md` | Revue du dépôt avant développement : constats, corrections appliquées, points laissés au chercheur | Pour l'historique du cadrage |
| `RAPPORT-CAMPAGNE_C-2026-09-10-1.md` | Campagne de référence : six premiers verdicts (H-001 à H-006), audit contradictoire, trois défauts de construction avec leurs contre-épreuves, hypothèses successeurs proposées | Pour retracer la première génération d'hypothèses |
| `REVUE-C008_2026-09-10.md` | Revue contradictoire des capacités ajoutées par C-008 : sept constats survivants sur trente-cinq, cinq correctifs, deux limites renvoyées à des hypothèses successeurs | Pour comprendre les capacités C-008 et leurs limites |
| `RAPPORT-CAMPAGNE_C-2026-09-10-3.md` | Première épreuve de H-007 à H-013 : sept verdicts, infirmation de H-010 et de H-011, comparaison d'une campagne chargée et d'une campagne propre | Pour retracer la seconde génération d'hypothèses |
| `RAPPORT-CAMPAGNE_C-2026-09-10-4.md` | Première épreuve de H-012 sur une matrice à réplicats : confirmation, et ce que la cellule écartée du jugement mesure | Pour retracer H-012 |
| `RAPPORT-CAMPAGNE_C-2026-09-10-5.md` | Première épreuve de H-013 : confirmation sur une machine attestée au repos, et la même campagne rendue non concluante sous charge délibérée | Pour comprendre l'attestation de quiétude (C-010) et la réserve sur H-008 |

Les campagnes finales `C-2026-09-10-11` et `C-2026-09-10-12`, dont viennent les treize verdicts, n'ont pas de rapport propre : elles sont consolidées dans `RAPPORT-FINAL_EscapeBench.md`.

## Ordre de lecture

1. `RAPPORT-FINAL_EscapeBench.md` (résultats de P1).
2. `Projets-candidats…` §1–5 (conclusion, hypothèses, points de vigilance, cartographie, grille).
3. `Guide-implementation…` §1–7 (méthode et cycle de travail).
4. `EscapeBench/docs/` dans l'ordre AIUP : vision → requirements → entity-model → use_cases.puml → use-cases ; puis `EscapeBench/LANCEMENT.md`.
5. Fiches P2–P8 de `Projets-candidats…` au moment de cadrer le projet suivant.

## État

| Projet | Statut | Prochaine étape |
|---|---|---|
| P1 EscapeBench | **Clos le 2026-09-10.** Treize hypothèses, treize verdicts : sept infirmées, six confirmées. UC-001 à UC-005 à `Deployed` ; contraintes C-008 à C-010 satisfaites ; verdicts issus de `C-2026-09-10-11` (douze hypothèses, 541 sujets) et `C-2026-09-10-12` (H-012, 82 sujets) | Aucune sur cette machine. Restent ouverts : rejouer H-008 et H-013 sur un boîtier plus rapide (`arm64`), désassembler les deux bras d'une paire, borner la contention mémoire par compteurs de performance |
| P2–P8 | Fiches de cadrage seulement | Rédiger `docs/vision.md` et le catalogue `H-###` avant tout code, en partant d'`EscapeBench/` comme gabarit |

Conventions : français pour la prose, anglais pour les identifiants et le code ; pages citées = folios imprimés des PDF ; marqueurs *Confirmé / Probable / Hypothèse / À vérifier / Adaptation*.
