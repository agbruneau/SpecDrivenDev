#!/usr/bin/env bash
# PreToolUse (Edit|Write|MultiEdit) — garde des chemins protégés.
# Règles appliquées : BR-003-1 / C-005 (harnais figé pendant une campagne),
# BR-003-3 / NFR-004 (results/ en écriture seule par le runner), BR-005-3
# (dashboard régénéré, jamais édité), UC-001 BR-001-2 (matrices immuables).
# Contrat hook : JSON sur stdin ; code de sortie 2 = bloquer l'action, stderr
# remonté à Claude (à vérifier dans la documentation hooks de Claude Code).
set -euo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"
FILE="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)"
[ -z "$FILE" ] && exit 0

# Chemin relatif au projet
REL="${FILE#"$ROOT"/}"
REL="${REL#./}"

deny() { printf 'EscapeBench guard: %s\nChemin : %s\nLa spécification prime : modifier docs/ d abord, puis synchroniser.\n' "$1" "$REL" >&2; exit 2; }

case "$REL" in
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
