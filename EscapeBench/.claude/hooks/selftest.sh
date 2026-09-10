#!/usr/bin/env bash
# Contrôle des hooks : chemins POSIX et Windows, cas bloquants et cas passants.
# Lancé par la CI et à la main : bash .claude/hooks/selftest.sh
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
fails=0

# Émet un JSON de hook valide pour un chemin donné. Le chemin passe par stdin :
# sous Git Bash, jq est un binaire Windows et MSYS réécrit les chemins passés en
# argument, ce qui fausserait le contrôle.
input() { printf '%s' "$1" | jq -Rc '{tool_input:{file_path:.}}'; }

check() { # check <attendu> <script> <projectDir> <chemin> <libellé>
  local want="$1" script="$2" dir="$3" path="$4" label="$5" got
  input "$path" > "$TMP/in.json"
  CLAUDE_PROJECT_DIR="$dir" bash "$ROOT/.claude/hooks/$script" < "$TMP/in.json" >/dev/null 2>&1
  got=$?
  if [ "$got" != "$want" ]; then
    echo "ÉCHEC  $label : attendu $want, obtenu $got" >&2; fails=$((fails+1))
  fi
}

for form in posix windows; do
  if [ "$form" = windows ]; then
    command -v cygpath >/dev/null 2>&1 || continue
    DIR="$(cygpath -w "$ROOT")"; SEP='\\'
  else
    DIR="$ROOT"; SEP='/'
  fi
  j() { printf '%s%s%s' "$DIR" "${SEP:0:1}" "$(printf '%s' "$1" | tr '/' "$SEP")"; }

  # guard-paths : chemins protégés (BR-003-3, BR-001-2, BR-005-3) et chemin libre.
  check 2 guard-paths.sh "$DIR" "$(j results/x.json)"        "$form guard results/"
  check 2 guard-paths.sh "$DIR" "$(j matrices/m/x.go)"       "$form guard matrices/"
  check 2 guard-paths.sh "$DIR" "$(j docs/dashboard.md)"     "$form guard dashboard"
  check 0 guard-paths.sh "$DIR" "$(j internal/models/x.go)"  "$form guard code libre"
  # internal/harness/ : bloqué seulement si une campagne est en cours (C-005).
  check 0 guard-paths.sh "$DIR" "$(j internal/harness/x.go)" "$form harness sans verrou"
  : > "$ROOT/results/.campaign-lock"
  check 2 guard-paths.sh "$DIR" "$(j internal/harness/x.go)" "$form harness sous verrou"
  rm -f "$ROOT/results/.campaign-lock"

  # spec-lint : les UC du dépôt passent ; un UC amputé est signalé.
  for uc in "$ROOT"/docs/use-cases/UC-*.md; do
    check 0 spec-lint.sh "$DIR" "$(j "docs/use-cases/$(basename "$uc")")" "$form spec-lint $(basename "$uc")"
  done
done

# spec-lint sur un UC volontairement fautif (hors dépôt : sections absentes, mot vague).
mkdir -p "$TMP/docs/use-cases"
printf '# Use Case: Faux\n\n**Use Case ID:** UC-999\n\nLe système devrait faire quelque chose.\n' \
  > "$TMP/docs/use-cases/UC-999-faux.md"
check 2 spec-lint.sh "$TMP" "$TMP/docs/use-cases/UC-999-faux.md" "spec-lint détecte un UC incomplet"

# go-test (Stop) : ne relance rien quand un hook Stop est déjà actif.
printf '{"stop_hook_active":true}' > "$TMP/stop.json"
CLAUDE_PROJECT_DIR="$ROOT" bash "$ROOT/.claude/hooks/go-test.sh" < "$TMP/stop.json" >/dev/null 2>&1 \
  || { echo "ÉCHEC  go-test stop_hook_active" >&2; fails=$((fails+1)); }

[ "$fails" -eq 0 ] && echo "hooks : tous les contrôles passent" || exit 1
