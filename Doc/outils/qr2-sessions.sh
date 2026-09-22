#!/usr/bin/env bash
# qr2-sessions.sh — agrégats de sessions Claude Code, sans contenu (QR2, lot 4, D-58).
#
# Appel :
#   bash Doc/outils/qr2-sessions.sh [DOSSIER_TRANSCRIPTIONS] [DOSSIER_SORTIE]
#   bash Doc/outils/qr2-sessions.sh ~/.claude/projects/C--Users-agbru-OneDrive-Documents-GitHub-Prospection /tmp/qr2
#
# Variables : QR2_IDLE_MIN (défaut 15) = écart au-delà duquel deux enregistrements
# successifs ne comptent pas comme temps actif; QR2_PHASES = fichier TSV
# « phase<TAB>début ISO UTC inclus<TAB>fin ISO UTC exclue » (défaut : phases de Prospection ci-dessous).
# Requiert bash, jq ≥ 1.6, find, sort. Ne lit que des horodatages, des noms d'outils,
# des compteurs de jetons et des noms de modèle : aucun texte de message n'est écrit.
#
# Sorties (TSV) dans DOSSIER_SORTIE :
#   sessions.tsv   une ligne par transcription principale
#   workflows.tsv  une ligne par exécution de workflow (dossier subagents/workflows/wf_*)
#   phases.tsv     agrégats par phase, chaque événement classé par son horodatage
#   days.tsv       sessions actives par jour UTC
set -euo pipefail

DIR=${1:-"$HOME/.claude/projects/C--Users-agbru-OneDrive-Documents-GitHub-Prospection"}
OUT=${2:-./qr2-out}
IDLE=${QR2_IDLE_MIN:-15}
command -v jq >/dev/null || { echo "jq introuvable" >&2; exit 1; }
[ -d "$DIR" ] || { echo "dossier introuvable : $DIR" >&2; exit 1; }
mkdir -p "$OUT"
EV="$OUT/events.jsonl"; : > "$EV"

PHASES_TSV=$(if [ -n "${QR2_PHASES:-}" ]; then cat "$QR2_PHASES"; else cat <<'EOF'
hors-projet	0000	2026-09-08T00:00:00Z
cadrage	2026-09-08T00:00:00Z	2026-09-10T13:54:00Z
escapebench	2026-09-10T13:54:00Z	2026-09-11T16:00:00Z
audit	2026-09-11T16:00:00Z	2026-09-13T09:00:00Z
leaklab	2026-09-13T09:00:00Z	2026-09-14T00:00:00Z
depot-final	2026-09-14T00:00:00Z	2026-09-15T10:00:00Z
apres-evaluation	2026-09-15T10:00:00Z	9999
EOF
fi)

# Normalisation d'une transcription en événements sans contenu.
# rec = horodatage brut, turn = message humain, tool = appel d'outil, msg = réponse (jetons), agent = transcription de sous-agent.
NORM='
def txt: if type=="string" then . else ([.[]?|select(.type?=="text")|.text]|join(" ")) end;
def hasres: type=="array" and any(.[]; .type?=="tool_result");
. as $a
| [$a[]|select(.timestamp)|.timestamp] as $ts
| ( if $src=="main" then
      ($ts[]|{k:"rec",t:.}),
      ($a[]|select(.type=="user" and (.isMeta|not) and (.isSidechain|not) and (.message.content|hasres|not))
           |select(.message.content|txt|length>0 and ((startswith("<")|not) or startswith("<command-name>")))|{k:"turn",t:.timestamp})
    else {k:"agent",t:($ts|min),t2:($ts|max)} end ),
  ([$a[]|select(.type=="assistant")|.timestamp as $t|.message.content[]?|select(.type?=="tool_use")|{id,name,t:$t}]
     |unique_by(.id)[]|{k:"tool",t,name}),
  ([$a[]|select(.type=="assistant" and .message.usage and .message.model!="<synthetic>")]
     |group_by(.message.id)[]|last
     |{k:"msg",t:.timestamp,model:.message.model,
       inn:((.message.usage.input_tokens//0)+(.message.usage.cache_creation_input_tokens//0)),
       cr:(.message.usage.cache_read_input_tokens//0),out:(.message.usage.output_tokens//0)})
| select(.t!=null) + {s:$s,src:$src,wf:$wf}'

for f in "$DIR"/*.jsonl; do
  s=$(basename "$f" .jsonl)
  jq -cs --arg s "$s" --arg src main --arg wf "" "$NORM" "$f" >> "$EV"
  [ -d "$DIR/$s/subagents" ] || continue
  while IFS= read -r g; do
    wf=$(echo "$g" | sed -n 's#.*/workflows/\(wf_[^/]*\)/.*#\1#p')
    jq -cs --arg s "$s" --arg src sub --arg wf "$wf" "$NORM" "$g" >> "$EV"
  done < <(find "$DIR/$s/subagents" -name '*.jsonl')
done

# Métadonnées des workflows : nom et agentCount final, quand le fichier d'exécution existe.
WFMETA=$(for j in "$DIR"/*/workflows/wf_*.json; do [ -f "$j" ] && jq -c '{wf:.runId,name:.workflowName,final:.agentCount}' "$j"; done | jq -cs '.')

jq -rs --arg idle "$IDLE" --arg phases "$PHASES_TSV" --argjson meta "$WFMETA" --arg out "$OUT" '
def ep: sub("\\.[0-9]+Z$";"Z")|fromdate;
def ph: . as $t | ([$phases|split("\n")[]|select(length>0)|split("\t")|select($t>=.[1] and $t<.[2])|.[0]][0] // "?");
def active: [.[]|select(.k=="rec")|.t|ep]|sort
   | [range(1;length) as $i|(.[$i]-.[$i-1])|select(. <= ($idle|tonumber)*60)]|((add // 0)/60|floor);
def tok: {inn:([.[]|select(.k=="msg")|.inn]|add//0), cr:([.[]|select(.k=="msg")|.cr]|add//0), out:([.[]|select(.k=="msg")|.out]|add//0)};
. as $e
| ($e|map(select(.src=="main"))|group_by(.s)|map(
    . as $m | ($e|map(select(.s==$m[0].s and .src=="sub"))) as $sub
    | [$m[0].s,
       ([$m[]|select(.k=="rec")|.t]|min), ([$m[]|select(.k=="rec")|.t]|max),
       ((([$m[]|select(.k=="rec")|.t|ep]|max)-([$m[]|select(.k=="rec")|.t|ep]|min))/3600*10|floor/10),
       ($m|active),
       ([$m[]|select(.k=="turn")]|length),
       ([$m[]|select(.k=="tool")]|length),
       ([$m[]|select(.k=="tool")|.name]|group_by(.)|map("\(.[0])=\(length)")|join(",")),
       ([$m[]|select(.k=="tool" and (.name=="Agent" or .name=="Task"))]|length),
       ([$m[]|select(.k=="tool" and .name=="Workflow")]|length),
       ([$sub[]|select(.k=="agent")]|length),
       ($m|tok|.inn,.cr,.out),
       ($sub|tok|.inn,.cr,.out),
       ([$m[],$sub[]|select(.k=="msg")|.model]|unique|join(","))]
  )) as $sessions
| ($e|map(select(.src=="sub" and .wf!=""))|group_by(.wf)|map(
    . as $w | ($meta|map(select(.wf==$w[0].wf))[0]) as $mm
    | [$w[0].s, $w[0].wf, ($mm.name//"?"),
       ([$w[]|select(.k=="agent")]|length), ($mm.final//""),
       ([$w[]|select(.k=="agent")|.t]|min), ([$w[]|select(.k=="agent")|.t2]|max),
       ((([$w[]|select(.k=="agent")|.t2|ep]|max)-([$w[]|select(.k=="agent")|.t|ep]|min))/60|floor),
       ($w|tok|.inn,.cr,.out),
       ([$w[]|select(.k=="msg")|.model]|unique|join(","))]
  )) as $wfs
| ($e|map(. + {p:(.t|ph)})|group_by(.p)|map(
    . as $p
    | [$p[0].p,
       ([$p[]|select(.src=="main")|.s]|unique|length),
       ([$p[]|select(.k=="turn")]|length),
       ([$p[]|select(.k=="tool" and .src=="main")]|length),
       ($p|map(select(.src=="main"))|group_by(.s)|map(active)|add),
       ([$p[]|select(.k=="agent")]|length),
       ($p|map(select(.src=="main"))|tok|.inn,.cr,.out),
       ($p|map(select(.src=="sub"))|tok|.inn,.cr,.out)]
  )) as $phases_out
| ($e|map(select(.src=="main" and .k=="rec")|{d:.t[0:10],s})|unique|group_by(.d)|map([.[0].d,length])) as $days
| "#sessions", (["session","premier","dernier","duree_h","actif_min","tours","outils","outils_par_type","agent_calls","workflow_calls","sous_agents","main_in","main_cache_read","main_out","sub_in","sub_cache_read","sub_out","modeles"]|@tsv), ($sessions[]|@tsv),
  "#workflows", (["session","wf","nom","transcriptions","agentCount_final","debut","fin","duree_min","in","cache_read","out","modeles"]|@tsv), ($wfs[]|@tsv),
  "#phases", (["phase","sessions","tours","outils","actif_min","sous_agents","main_in","main_cache_read","main_out","sub_in","sub_cache_read","sub_out"]|@tsv), ($phases_out[]|@tsv),
  "#days", (["jour","sessions"]|@tsv), ($days[]|@tsv)
' "$EV" | awk -v out="$OUT" '/^#/{f=out"/"substr($0,2)".tsv"; printf "" > f; next} {print >> f}'

rm -f "$EV"
for t in sessions workflows phases days; do echo "$OUT/$t.tsv : $(($(wc -l < "$OUT/$t.tsv")-1)) lignes"; done
