#!/usr/bin/env bash
# PostToolUse (Edit|Write) — vérification rapide après édition d'un fichier Go.
# Volontairement léger (formatage + vet) ; les tests tournent au hook Stop et via make.
set -uo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"
command -v jq >/dev/null 2>&1 || { echo 'go-check : jq introuvable, contrôle ignoré.' >&2; exit 0; }
FILE="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)"
FILE="${FILE//\\//}"; ROOT="${ROOT//\\//}"   # chemins Windows C:\... -> C:/...
case "$FILE" in *.go) ;; *) exit 0 ;; esac
cd "$ROOT" || exit 0
command -v go >/dev/null 2>&1 || { echo "go introuvable dans PATH (C-001)" >&2; exit 2; }

UNFORMATTED="$(gofmt -l "$FILE" 2>/dev/null || true)"
if [ -n "$UNFORMATTED" ]; then
  gofmt -w "$FILE" && echo "gofmt appliqué : $FILE" >&2
fi
if ! OUT="$(go vet ./... 2>&1)"; then
  printf 'go vet a échoué :\n%s\n' "$OUT" >&2
  exit 2
fi
exit 0
