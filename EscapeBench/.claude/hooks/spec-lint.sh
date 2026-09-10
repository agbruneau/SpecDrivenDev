#!/usr/bin/env bash
# PostToolUse (Edit|Write) — contrôle de forme d'un cas d'utilisation
# (gabarit SDD, ch. 3-4) après édition d'un fichier docs/use-cases/UC-###-*.md.
# Code de sortie 2 = avertissement montré à Claude (documentation hooks, 2026-09-10).
set -uo pipefail
ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
INPUT="$(cat)"
FILE="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)"
FILE="${FILE//\\//}"; ROOT="${ROOT//\\//}"   # chemins Windows C:\... -> C:/...
case "$FILE" in */docs/use-cases/UC-[0-9][0-9][0-9]-*.md) ;; *) exit 0 ;; esac
[ -f "$FILE" ] || exit 0

missing=()
for section in '## Overview' '**Use Case ID:**' '**Primary Actor:**' '**Goal:**' '**Status:**' \
               '**Linked Requirements:**' '## Preconditions' '## Main Success Scenario' \
               '## Alternative Flows' '## Postconditions' '**Success:**' '**Failure:**' '## Business Rules'; do
  grep -qF -- "$section" "$FILE" || missing+=("$section")
done

# Identifiant du fichier = identifiant déclaré
BASENAME="$(basename "$FILE")"
DECLARED="$(grep -oE '\*\*Use Case ID:\*\* UC-[0-9]{3}' "$FILE" | grep -oE 'UC-[0-9]{3}' || true)"
[ "${BASENAME:0:6}" = "$DECLARED" ] || missing+=("identifiant du fichier ($BASENAME) ≠ Use Case ID ($DECLARED)")

# Exigences liées existantes
REQ="$ROOT/docs/requirements.md"
if [ -f "$REQ" ]; then
  for id in $(grep -oE '\b(FR|NFR|C|H)-[0-9]{3}\b' "$FILE" | sort -u); do
    grep -qE "^\| $id \|" "$REQ" || missing+=("$id absent de docs/requirements.md")
  done
fi

# Mots vagues (SDD p. 51) dans les flux
VAGUE="$(grep -niE '\b(normalement|rapidement|si possible|au besoin|devrait|normally|quickly|as needed|should)\b' "$FILE" | grep -vE 'Notes de revue|^[0-9]+:>' || true)"

if [ ${#missing[@]} -gt 0 ] || [ -n "$VAGUE" ]; then
  {
    echo "spec-lint — $BASENAME"
    for m in "${missing[@]}"; do echo " - manquant ou incohérent : $m"; done
    [ -n "$VAGUE" ] && { echo " - formulations vagues à remplacer par une règle explicite (SDD p. 51) :"; echo "$VAGUE" | sed 's/^/   /'; }
  } >&2
  exit 2
fi
exit 0
