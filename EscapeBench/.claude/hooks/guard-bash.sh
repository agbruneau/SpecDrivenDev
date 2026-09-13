#!/usr/bin/env bash
# PreToolUse (Bash) — étend la garde des chemins protégés aux commandes shell (A-112).
# CLAUDE.md affirme que results/, matrices/ et docs/dashboard.md ne sont jamais écrits
# par un agent ; guard-paths.sh ne couvre que les outils d'édition, alors qu'un simple
# `echo > docs/dashboard.md` ou `rm -rf results/` les contourne.
#
# Ces fichiers se produisent en exécutant le binaire du banc, jamais en les éditant :
# les invocations du banc sont donc admises telles quelles.
set -euo pipefail
INPUT="$(cat)"

command -v jq >/dev/null 2>&1 \
  || { echo 'EscapeBench guard: jq introuvable, commande refusée par précaution (installer jq).' >&2; exit 2; }
CMD="$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')" \
  || { echo 'EscapeBench guard: entrée JSON illisible, commande refusée par précaution.' >&2; exit 2; }
[ -z "$CMD" ] && exit 0

# Le banc écrit sous results/ et matrices/ : c'est sa fonction (BR-003-3, BR-001-2).
case "$CMD" in
  *escapebench*|*'make matrix'*|*'make escape'*|*'make campaign'*|*'make compare'*|*'make verdict'*|*'make dashboard'*)
    exit 0 ;;
esac

# Une commande qui ne fait que lire est sans effet sur l'immutabilité.
printf '%s' "$CMD" | grep -qE '(>|>>|\brm\b|\bmv\b|\bcp\b|\btee\b|\btruncate\b|\btouch\b|\bmkdir\b|\bdd\b|\bsed\b +-[a-z]*i|\bchmod\b|\bln\b)' || exit 0

if printf '%s' "$CMD" | grep -qiE '(^|[^[:alnum:]_./-])(\./)?(results|matrices)/|docs/dashboard\.md'; then
  printf 'EscapeBench guard: results/, matrices/ et docs/dashboard.md ne sont jamais écrits par un agent (BR-003-3, BR-001-2, BR-005-3, NFR-004).\nCommande : %s\nCes fichiers se produisent en exécutant le binaire du banc.\n' "$CMD" >&2
  exit 2
fi
exit 0
