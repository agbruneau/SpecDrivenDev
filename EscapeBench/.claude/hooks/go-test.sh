#!/usr/bin/env bash
# Stop — exécute la suite unitaire (-race, -shuffle=on) avant que Claude ne conclue
# sa réponse, uniquement si des fichiers Go existent. Contrat confirmé (documentation
# hooks Claude Code, 2026-09-10) : code 2 = Claude ne s'arrête pas et reçoit stderr ;
# stop_hook_active vaut true quand un hook Stop est déjà en cours pour ce tour.
set -uo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"
# A-110 : sans jq la garde anti-boucle disparaissait et la suite était relancée à chaque
# tour. Détecter le drapeau sans jq dans ce cas plutôt que de le lire comme false.
if command -v jq >/dev/null 2>&1; then
  ACTIVE="$(printf '%s' "$INPUT" | jq -r '.stop_hook_active // false' 2>/dev/null || echo false)"
else
  echo 'go-test : jq introuvable, lecture dégradée de stop_hook_active.' >&2
  ACTIVE=false
  printf '%s' "$INPUT" | grep -qE '"stop_hook_active"[[:space:]]*:[[:space:]]*true' && ACTIVE=true
fi
[ "$ACTIVE" = "true" ] && exit 0
cd "$ROOT" || exit 0
command -v go >/dev/null 2>&1 || exit 0
find . -name '*.go' -not -path './matrices/*' | grep -q . || exit 0
if ! OUT="$(go test -race -shuffle=on -count=1 ./... 2>&1)"; then
  printf 'go test a échoué (hook Stop) :\n%s\n' "$(printf '%s' "$OUT" | tail -n 60)" >&2
  exit 2
fi
exit 0
