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
| `REVUE-PRELANCEMENT_2026-09-10.md` | Revue du dépôt avant développement : constats, corrections appliquées, points laissés au chercheur | Avant `/spec-review UC-001` |

## Ordre de lecture

1. `Projets-candidats…` §1–5 (conclusion, hypothèses, points de vigilance, cartographie, grille).
2. `Guide-implementation…` §1–7 (méthode et cycle de travail).
3. `EscapeBench/docs/` dans l'ordre AIUP : vision → requirements → entity-model → use_cases.puml → use-cases ; puis `EscapeBench/LANCEMENT.md`.
4. Fiches P2–P8 de `Projets-candidats…` au moment de cadrer le projet suivant.

## État

| Projet | Statut | Prochaine étape |
|---|---|---|
| P1 EscapeBench | Spécification complète (UC-001 à UC-005 au statut Reviewed), outillage Claude Code revu et vérifié le 2026-09-10 (`bash .claude/hooks/selftest.sh`), squelette compilé avec go1.25.0 et go1.27.0 ; aucun code de service | Créer le dépôt dédié, `/spec-review UC-001` → `Approved` → `/implement UC-001` (voir `EscapeBench/LANCEMENT.md`) |
| P2–P8 | Fiches de cadrage seulement | Rédiger `docs/vision.md` et le catalogue `H-###` avant tout code |

Conventions : français pour la prose, anglais pour les identifiants et le code ; pages citées = folios imprimés des PDF ; marqueurs *Confirmé / Probable / Hypothèse / À vérifier / Adaptation*.
