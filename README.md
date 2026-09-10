# Prospection

Dépôt de prospection : identification, évaluation et cadrage de projets d'exploration à développer avec Claude Code. Le dépôt contient les sources analysées, les documents de cadrage et, pour le premier projet retenu, un noyau de spécification prêt à être déplacé dans son propre dépôt.

## Sources

| Fichier | Référence |
|---|---|
| `Building_Enterprise_Projects_with_Go.pdf` | Shahsavan, S. *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*. Apress, 2026. DOI 10.1007/979-8-8688-2370-1 |
| `Spec-Driven Development.pdf` | Martinelli, S. *Spec-Driven Development: From Specs to Code with AI Agents*. Apress Pocket Guides, 2026. DOI 10.1007/979-8-8688-2851-5 |
| `Agentic_Coding_with_Claude_Code.pdf` | Marco, E. *Agentic Coding with Claude Code*. Packt, mars 2026. ISBN 978-1-80602-259-5 |

## Documents

| Fichier | Contenu | Lire quand |
|---|---|---|
| `Projets-candidats_Building-Enterprise-Projects-with-Go.md` | Cartographie des affirmations réfutables du livre de Shahsavan, grille d'évaluation, huit fiches de projets (P1–P8), séquence recommandée, méthode d'implémentation commune | Pour choisir un projet |
| `Guide-implementation_AIUP-Claude-Code.md` | Méthode d'implémentation : *AI Unified Process* (Martinelli) adapté aux bancs de réfutation Go ; réglages Claude Code retenus d'*Agentic Coding with Claude Code* (Marco) — mémoire, skills, sous-agents, hooks, MCP, plan mode, parallélisme ; cycle par cas d'utilisation, traçabilité, pièges | Avant d'ouvrir le dépôt d'un projet |
| `EscapeBench/` | Projet P1 prêt au lancement : noyau de spécification complet (`docs/` — vision, FR/NFR/C/H, modèle d'entités, diagramme, UC-001 à UC-005, tableau de bord), `CLAUDE.md`, six skills, deux sous-agents, quatre hooks avec leur contrôle (`selftest.sh`), squelette Go compilable (`go.mod`, `Makefile`, `cmd/`, `internal/`), CI, `LANCEMENT.md` | À copier dans un dépôt dédié ; gabarit pour P2–P8 |
| `REVUE-PRELANCEMENT_2026-09-10.md` | Revue du dépôt avant développement : constats, corrections appliquées, points laissés au chercheur | Avant d'ouvrir le code |
| `DECISION.md` | Décisions prises pendant la construction d'EscapeBench : écarts de processus assumés, conception du harnais, méthode statistique, limites | Avant de modifier le banc ou de lancer une campagne |
| `RAPPORT-CAMPAGNE_C-2026-09-10-1.md` | Première campagne de référence : six verdicts, audit contradictoire, trois défauts de construction avec leurs contre-épreuves, hypothèses successeurs proposées | Avant de citer un verdict, et avant la campagne suivante |
| `REVUE-C008_2026-09-10.md` | Revue contradictoire des capacités ajoutées par C-008 : sept constats survivants sur trente-cinq, cinq correctifs, deux limites renvoyées à des hypothèses successeurs | Avant de lancer une campagne sur H-007 à H-011 |
| `RAPPORT-CAMPAGNE_C-2026-09-10-3.md` | Première épreuve de H-007 à H-013 : sept verdicts, infirmation de H-010 et de H-011, comparaison d'une campagne chargée et d'une campagne propre | Avant de citer un verdict de la seconde génération d'hypothèses |

## Ordre de lecture

1. `Projets-candidats…` §1–5 (conclusion, hypothèses, points de vigilance, cartographie, grille).
2. `Guide-implementation…` §1–7 (méthode et cycle de travail).
3. `EscapeBench/docs/` dans l'ordre AIUP : vision → requirements → entity-model → use_cases.puml → use-cases ; puis `EscapeBench/LANCEMENT.md`.
4. Fiches P2–P8 de `Projets-candidats…` au moment de cadrer le projet suivant.

## État

| Projet | Statut | Prochaine étape |
|---|---|---|
| P1 EscapeBench | **Première campagne exécutée, seconde génération d'hypothèses écrite.** UC-001 à UC-005 implémentés et testés (couverture 93,7 %) ; campagne `C-2026-09-10-1`, 230 sujets, 29 min, six verdicts ; audit contradictoire mené, trois défauts de construction démontrés ; H-007 à H-011 au catalogue, contrainte `C-008` pour les capacités qu'elles exigent | Satisfaire `C-008` (dispositions de type, sonde à chaîne dépendante, deux profils de conteneur, tailles de cache en provenance), puis campagne portant H-007 à H-011 ; rejouer sur `arm64` (C-006) |
| P2–P8 | Fiches de cadrage seulement | Rédiger `docs/vision.md` et le catalogue `H-###` avant tout code |

Conventions : français pour la prose, anglais pour les identifiants et le code ; pages citées = folios imprimés des PDF ; marqueurs *Confirmé / Probable / Hypothèse / À vérifier / Adaptation*.
