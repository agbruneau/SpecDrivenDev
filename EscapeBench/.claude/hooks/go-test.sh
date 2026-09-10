#!/usr/bin/env bash
# Stop — exécute la suite unitaire (-race, -shuffle=on) avant que Claude ne conclue
# sa réponse, uniquement si des fichiers Go existent. Code 2 = Claude reçoit l'échec
# et poursuit ; le champ stop_hook_active évite la boucle infinie (à vérifier dans la
# documentation hooks de Claude Code).
set -uo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"
ACTIVE="$(printf '%s' "$INPUT" | jq -r '.stop_hook_active // false' 2>/dev/null || echo false)"
[ "$ACTIVE" = "true" ] && exit 0
cd "$ROOT" || exit 0
command -v go >/dev/null 2>&1 || exit 0
find . -name '*.go' -not -path './matrices/*' | grep -q . || exit 0
if ! OUT="$(go test -race -shuffle=on -count=1 ./... 2>&1)"; then
  printf 'go test a échoué (hook Stop) :\n%s\n' "$(printf '%s' "$OUT" | tail -n 60)" >&2
  exit 2
fi
exit 0
