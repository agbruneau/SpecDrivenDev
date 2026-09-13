#!/usr/bin/env bash
# PreToolUse (Edit|Write|NotebookEdit) — garde des chemins protégés de LeakLab.
# NFR-004 / BR-001-5 : results/ n'est écrit que par le binaire. Repris d'EscapeBench : la garde
# échoue fermée (sans jq ou sur entrée illisible, elle refuse), refuse les chemins non normalisés
# et ignore la casse sur les systèmes de fichiers insensibles.
# Contrat hook : JSON sur stdin ; code de sortie 2 = bloquer l'action, stderr montré à Claude.
set -euo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"

deny() { printf 'LeakLab guard: %s\nChemin : %s\n' "$1" "${REL:-<inconnu>}" >&2; exit 2; }

command -v jq >/dev/null 2>&1 \
  || { echo 'LeakLab guard: jq introuvable, écriture refusée par précaution (installer jq).' >&2; exit 2; }
FILE="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // .tool_input.notebook_path // empty')" \
  || { echo 'LeakLab guard: entrée JSON illisible, écriture refusée par précaution.' >&2; exit 2; }
[ -z "$FILE" ] && exit 0
FILE="${FILE//\\//}"; ROOT="${ROOT//\\//}"

case "$FILE" in
  *..*|*//*) REL="$FILE"; deny "chemin non normalisé (segment .. ou //) ; fournir le chemin canonique." ;;
esac

REL="${FILE#"$ROOT"/}"
REL="${REL#./}"
MATCH="$REL"
case "${OSTYPE:-}" in msys*|cygwin*|win*|darwin*) MATCH="${REL,,}" ;; esac

case "$MATCH" in
  results/*)
    deny "results/ est en écriture seule pour le binaire (NFR-004, BR-001-5) ; lancer go run ./cmd/leaklab." ;;
esac
exit 0
