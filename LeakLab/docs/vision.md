# Vision — LeakLab

**LeakLab** est un banc reproductible qui éprouve les affirmations de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur les anti-patrons de concurrence — fuite de goroutine, interblocage de canal, contexte non annulé, I/O qui ignore le contexte, course de données — et sur les outils qui prétendent les révéler (chapitres 7, 9 et 20).

La question de recherche : pour chaque anti-patron, quel mécanisme de détection le révèle — test ordinaire, détecteur de courses, bulle `testing/synctest`, surveillance de `runtime.NumGoroutine()`, profil `goroutineleak`, analyse statique —, avec quels faux négatifs et quels faux positifs ?

Comme EscapeBench, le banc ne corrige rien : il produit des **verdicts** (confirmée, infirmée, non concluante) sur des hypothèses numérotées dont le critère de réfutation est gelé avant la première mesure, et une **matrice de détectabilité** cas × détecteur dont chaque cellule cite ses observations.

Le projet suit le *Spec-Driven Development* (Martinelli, Apress 2026) : `docs/` fait autorité, le code est dérivé des cas d'utilisation.

## Objectifs

- Constituer un corpus versionné de cas minimaux, chacun avec sa vérité terrain établie indépendamment des détecteurs, et sa correction.
- Mesurer la détectabilité de chaque cas par chaque détecteur, en processus isolés et répétés.
- Éprouver les chiffres et les généralisations du livre sur `synctest`, `-race`, `NumGoroutine`, `time.After` et l'annulation des contextes.
- Laisser une recommandation de chaîne d'intégration continue fondée sur la matrice, et un analyseur `ctxvet` utilisable seul.

## Hors périmètre

- Fuites enfouies dans une base de code réelle : le corpus synthétique surestime la détectabilité, en contrepartie d'une vérité terrain connue.
- Outils hors bibliothèque standard (`goleak`, `golang.org/x/tools/go/analysis`) : écartés par C-002, et signalés comme tels.
- Toute conclusion transposée à une version de Go non mesurée.
