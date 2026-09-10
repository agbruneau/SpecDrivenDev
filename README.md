# Prospection

Dépôt de prospection : identification, évaluation et cadrage de projets d'exploration à développer avec Claude Code. Le dépôt contient les sources analysées, les documents de cadrage et, pour le premier projet retenu, un noyau de spécification prêt à être déplacé dans son propre dépôt.

## Sources

| Fichier | Référence |
|---|---|
| `Building_Enterprise_Projects_with_Go.pdf` | Shahsavan, S. *Building Enterprise Projects with Go: Clarity at Scale in Production-Grade Go Systems*. Apress, 2026. DOI 10.1007/979-8-8688-2370-1 |
| `Spec-Driven Development.pdf` | Martinelli, S. *Spec-Driven Development: From Specs to Code with AI Agents*. Apress Pocket Guides, 2026. DOI 10.1007/979-8-8688-2851-5 |

## Documents

| Fichier | Contenu | Lire quand |
|---|---|---|
| `Projets-candidats_Building-Enterprise-Projects-with-Go.md` | Cartographie des affirmations réfutables du livre de Shahsavan, grille d'évaluation, huit fiches de projets (P1–P8), séquence recommandée, méthode d'implémentation commune | Pour choisir un projet |
| `Guide-implementation_AIUP-Claude-Code.md` | Méthode d'implémentation : *AI Unified Process* (Martinelli) adapté aux bancs de réfutation Go, outillage Claude Code (`aiup-core`, skills de projet, `CLAUDE.md`, hooks, sous-agents), cycle par cas d'utilisation, traçabilité, pièges | Avant d'ouvrir le dépôt d'un projet |
| `EscapeBench/` | Noyau de spécification du projet P1 : `CLAUDE.md`, `docs/vision.md`, `docs/requirements.md` (FR, NFR, C, H), `docs/entity-model.md`, `docs/use_cases.puml`, `docs/use-cases/UC-002`, `UC-003`, `docs/dashboard.md` | Comme gabarit pour tout nouveau projet |

## Ordre de lecture

1. `Projets-candidats…` §1–5 (conclusion, hypothèses, points de vigilance, cartographie, grille).
2. `Guide-implementation…` §1–7 (méthode et cycle de travail).
3. `EscapeBench/docs/` dans l'ordre AIUP : vision → requirements → entity-model → use_cases.puml → use-cases.
4. Fiches P2–P8 de `Projets-candidats…` au moment de cadrer le projet suivant.

## État

| Projet | Statut | Prochaine étape |
|---|---|---|
| P1 EscapeBench | Noyau de spécification rédigé ; UC-002 et UC-003 en revue ; aucun code | Revue à froid des deux UC, puis `/implement UC-002` dans un dépôt dédié |
| P2–P8 | Fiches de cadrage seulement | Rédiger `docs/vision.md` et le catalogue `H-###` avant tout code |

Conventions : français pour la prose, anglais pour les identifiants et le code ; pages citées = folios imprimés des PDF ; marqueurs *Confirmé / Probable / Hypothèse / À vérifier / Adaptation*.
