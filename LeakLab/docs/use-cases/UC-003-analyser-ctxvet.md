# Use Case: Analyser un paquet avec ctxvet

## Overview

**Use Case ID:** UC-003
**Use Case Name:** Analyser un paquet avec ctxvet
**Primary Actor:** Développeur (Pipeline CI en acteur secondaire)
**Goal:** Obtenir la liste des appels d'I/O qui ignorent le contexte dans un répertoire Go, avec l'API qui le respecte
**Status:** Deployed

**Linked Requirements:** FR-003, FR-007, C-002
**Linked Hypotheses:** aucune (l'analyseur alimente la matrice, pas un critère)
**Entities:** Observation

## Preconditions

- Le répertoire désigné contient des fichiers `.go`.

## Main Success Scenario

1. Le développeur lance `ctxvet` sur un répertoire.
2. Le système analyse syntaxiquement chaque fichier `.go` du répertoire, tests exclus.
3. Pour chaque appel à une fonction de la liste de BR-003-1, désignée par le chemin d'import réel de son paquet, le système émet un diagnostic `fichier:ligne:colonne: ctxvet: <appel> ignore le contexte ; utiliser <remplaçant>`.
4. Le système sort avec le code 1 s'il a émis au moins un diagnostic, 0 sinon.

## Alternative Flows

### A1: Fichier illisible ou syntaxe invalide
**Trigger:** À l'étape 2, un fichier ne s'analyse pas.
**Flow:**
1. Le système affiche l'erreur et sort avec le code 2, sans diagnostic partiel.

## Postconditions

**Success:** Les diagnostics sont écrits sur la sortie standard, triés par fichier puis par position.

## Business Rules

### BR-003-1: Appels visés
| Paquet | Fonction | Remplaçant |
|---|---|---|
| `net` | `Dial`, `DialTimeout` | `(*net.Dialer).DialContext` |
| `net` | `Listen` | `(*net.ListenConfig).Listen` |
| `net/http` | `Get`, `Head`, `Post`, `PostForm` | `http.NewRequestWithContext` et `(*http.Client).Do` |
| `net/http` | `NewRequest` | `http.NewRequestWithContext` |
| `os/exec` | `Command` | `exec.CommandContext` |

### BR-003-2: Résolution par import
Un appel `x.F` n'est visé que si `x` désigne, dans le fichier, l'import du paquet de la table (alias compris). Un identifiant local homonyme n'est pas visé.

### BR-003-3: Portée assumée
L'analyse est syntaxique : les méthodes (`db.Query`, `conn.Read`) ne sont pas visées, faute de types. C'est une limite déclarée, mesurée par la matrice.

## Notes de revue

- Clôture du 2026-09-13 : statut porté à `Deployed`. Ce qui l'établit : `ctxvet` a diagnostiqué dans la campagne de référence le seul cas `CTX_IO` du corpus et aucun autre ; ses règles BR-003-1 à BR-003-3 sont couvertes par `TestUC003_*`.
