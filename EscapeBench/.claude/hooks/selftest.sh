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

# A-104 : le contrôle « harnais sous verrou » posait et supprimait le verrou RÉEL du dépôt.
# Lancé pendant une campagne il la déprotégeait (BR-003-1, BR-003-3, C-005). Il travaille
# désormais sur un faux projet sous $TMP, que le trap EXIT nettoie.
mkdir -p "$TMP/proj/results" "$TMP/proj/internal/harness"

for form in posix windows; do
  if [ "$form" = windows ]; then
    command -v cygpath >/dev/null 2>&1 || continue
    DIR="$(cygpath -w "$ROOT")"; PDIR="$(cygpath -w "$TMP/proj")"; SEP='\\'
  else
    DIR="$ROOT"; PDIR="$TMP/proj"; SEP='/'
  fi
  j() { printf '%s%s%s' "$DIR" "${SEP:0:1}" "$(printf '%s' "$1" | tr '/' "$SEP")"; }
  jp() { printf '%s%s%s' "$PDIR" "${SEP:0:1}" "$(printf '%s' "$1" | tr '/' "$SEP")"; }

  # guard-paths : chemins protégés (BR-003-3, BR-001-2, BR-005-3) et chemin libre.
  check 2 guard-paths.sh "$DIR" "$(j results/x.json)"        "$form guard results/"
  check 2 guard-paths.sh "$DIR" "$(j matrices/m/x.go)"       "$form guard matrices/"
  check 2 guard-paths.sh "$DIR" "$(j docs/dashboard.md)"     "$form guard dashboard"
  check 0 guard-paths.sh "$DIR" "$(j internal/models/x.go)"  "$form guard code libre"
  # A-103 : un chemin non normalisé contournait la garde.
  check 2 guard-paths.sh "$DIR" "$(j internal/../results/x.json)" "$form guard segment .."
  # internal/harness/ : bloqué seulement si une campagne est en cours (C-005).
  check 0 guard-paths.sh "$PDIR" "$(jp internal/harness/x.go)" "$form harnais sans verrou"
  : > "$TMP/proj/results/.campaign-lock"
  check 2 guard-paths.sh "$PDIR" "$(jp internal/harness/x.go)" "$form harnais sous verrou"
  rm -f "$TMP/proj/results/.campaign-lock"

  # spec-lint : les UC du dépôt passent ; un UC amputé est signalé.
  for uc in "$ROOT"/docs/use-cases/UC-*.md; do
    check 0 spec-lint.sh "$DIR" "$(j "docs/use-cases/$(basename "$uc")")" "$form spec-lint $(basename "$uc")"
  done
done

# A-103 : `//` est l'autre forme de chemin non normalisé.
printf '%s//results/x.json' "$ROOT" > "$TMP/dslash.txt"
check 2 guard-paths.sh "$ROOT" "$(cat "$TMP/dslash.txt")" "guard double barre oblique"

# A-102 : sans jq la garde doit refuser (fail closed), pas laisser passer.
input "$ROOT/results/x.json" > "$TMP/nojq.json"
mkdir -p "$TMP/emptybin"
if PATH="$TMP/emptybin" CLAUDE_PROJECT_DIR="$ROOT" \
     bash "$ROOT/.claude/hooks/guard-paths.sh" < "$TMP/nojq.json" >/dev/null 2>&1; then
  echo "ÉCHEC  guard-paths sans jq : attendu un refus, obtenu 0" >&2; fails=$((fails+1))
fi

# A-112 : la garde Bash refuse une écriture shell vers un chemin protégé et laisse passer
# l'invocation du binaire, qui est le seul producteur légitime de ces fichiers.
bashcmd() { printf '%s' "$1" | jq -Rc '{tool_input:{command:.}}'; }
bguard() { # bguard <attendu> <commande> <libellé>
  local want="$1" got
  bashcmd "$2" > "$TMP/bash.json"
  CLAUDE_PROJECT_DIR="$ROOT" bash "$ROOT/.claude/hooks/guard-bash.sh" < "$TMP/bash.json" >/dev/null 2>&1
  got=$?
  [ "$got" = "$want" ] || { echo "ÉCHEC  $3 : attendu $want, obtenu $got" >&2; fails=$((fails+1)); }
}
bguard 2 'echo x > docs/dashboard.md'            "guard-bash dashboard"
bguard 2 'rm -rf results/campaigns/C-1'          "guard-bash results/"
bguard 2 'cp /tmp/x.json matrices/m/matrix.json' "guard-bash matrices/"
bguard 0 'go run ./cmd/escapebench dashboard'    "guard-bash binaire admis"
bguard 0 'cat results/verdicts/x.json'           "guard-bash lecture admise"
bguard 0 'go test -race -shuffle=on ./...'       "guard-bash commande neutre"

# A-108 : une citation verbatim du livre n'est pas une formulation vague du spécificateur.
mkdir -p "$TMP/cite/docs/use-cases"
cp "$ROOT/docs/use-cases/UC-001-generer-matrice.md" "$TMP/cite/docs/use-cases/UC-001-generer-matrice.md"
cp "$ROOT/docs/requirements.md" "$TMP/cite/docs/requirements.md"
printf '\n> Citation : « a small struct should be passed by value ».\n' \
  >> "$TMP/cite/docs/use-cases/UC-001-generer-matrice.md"
check 0 spec-lint.sh "$TMP/cite" "$TMP/cite/docs/use-cases/UC-001-generer-matrice.md" \
  "spec-lint tolère une citation verbatim"

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
