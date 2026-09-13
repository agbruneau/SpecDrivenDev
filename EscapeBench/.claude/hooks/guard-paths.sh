#!/usr/bin/env bash
# PreToolUse (Edit|Write|NotebookEdit) — garde des chemins protégés.
# Règles appliquées : BR-003-1 / C-005 (harnais figé pendant une campagne),
# BR-003-3 / NFR-004 (results/ en écriture seule par le runner), BR-005-3
# (dashboard régénéré, jamais édité), UC-001 BR-001-2 (matrices immuables).
# Contrat hook (documentation Claude Code, consultée le 2026-09-10) : JSON sur
# stdin ; code de sortie 2 = bloquer l'action, stderr montré à Claude.
#
# Révision du 2026-09-12 (A-102, A-103, A-105, A-112) : la garde échoue fermée.
# Sans jq, ou sur une entrée illisible, elle refuse l'écriture au lieu de la laisser
# passer en silence ; un chemin non normalisé (`..`, `//`) est refusé ; la casse est
# ignorée sur les systèmes de fichiers insensibles ; notebook_path est gardé aussi.
set -euo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"

deny() { printf 'EscapeBench guard: %s\nChemin : %s\nLa spécification prime : modifier docs/ d abord, puis synchroniser.\n' "$1" "${REL:-<inconnu>}" >&2; exit 2; }

command -v jq >/dev/null 2>&1 \
  || { echo 'EscapeBench guard: jq introuvable, écriture refusée par précaution (installer jq).' >&2; exit 2; }
FILE="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // .tool_input.notebook_path // empty')" \
  || { echo 'EscapeBench guard: entrée JSON illisible, écriture refusée par précaution.' >&2; exit 2; }
# Un outil sans chemin de fichier n'écrit rien que cette garde protège.
[ -z "$FILE" ] && exit 0
# Sous Windows natif, Claude Code transmet des chemins C:\... : normaliser en C:/...
FILE="${FILE//\\//}"; ROOT="${ROOT//\\//}"

# Un chemin non normalisé contourne la suppression de préfixe et le `case` : refuser
# d'emblée plutôt que de tenter une résolution (realpath change la forme du chemin).
case "$FILE" in
  *..*|*//*) REL="$FILE"; deny "chemin non normalisé (segment .. ou //) ; fournir le chemin canonique." ;;
esac

# Chemin relatif au projet
REL="${FILE#"$ROOT"/}"
REL="${REL#./}"

# NTFS et APFS sont insensibles à la casse : Results/x.json désigne results/x.json.
# Les motifs ci-dessous sont en minuscules ; comparer sur une copie repliée.
MATCH="$REL"
case "${OSTYPE:-}" in msys*|cygwin*|win*|darwin*) MATCH="${REL,,}" ;; esac

case "$MATCH" in
  results/*)
    deny "results/ est en écriture seule pour le runner de campagne (BR-003-3, NFR-004)." ;;
  matrices/*)
    deny "une Matrix est immuable après génération (BR-001-2) ; générer une nouvelle matrice via le binaire." ;;
  docs/dashboard.md)
    deny "docs/dashboard.md est régénéré par /spec-coverage et /refute (BR-005-3)." ;;
  internal/harness/*)
    if [ -f "$ROOT/results/.campaign-lock" ]; then
      deny "une campagne est en cours (results/.campaign-lock) ; le harnais est figé (BR-003-1, C-005)."
    fi ;;
esac
exit 0
