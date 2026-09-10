---
name: spec-review
description: >
  Lance la revue d'exécutabilité d'un cas d'utilisation (UC-###) par le sous-agent spec-reviewer
  et rapporte ses constatations sans modifier la spécification. Invoqué par slash command :
  /spec-review UC-004. Étape obligatoire avant le passage d'un UC au statut Approved.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Agent
---
# /spec-review $ARGUMENTS

1. Vérifie que `docs/use-cases/$ARGUMENTS-*.md` existe ; sinon arrête-toi.
2. Délègue au sous-agent **spec-reviewer** avec ce prompt, sans ajouter d'historique de conversation :
   « Revois le cas d'utilisation $ARGUMENTS (fichier docs/use-cases/$ARGUMENTS-*.md) selon ta procédure. Contexte : projet EscapeBench, banc de réfutation ; les hypothèses H-### sont dans docs/requirements.md. Réponds dans ton format obligatoire. »
3. Reproduis intégralement la réponse du sous-agent, puis ajoute une seule ligne :
   `Prochaine étape : <corriger les constatations bloquantes dans la spécification | passer Status à Approved après relecture humaine à froid>`.
4. N'édite aucun fichier. Les corrections sont faites par le chercheur ou sur demande explicite, dans `docs/`, avant tout code.
