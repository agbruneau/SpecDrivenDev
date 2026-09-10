# Vision — EscapeBench

**EscapeBench** est un banc de mesure reproductible qui éprouve les affirmations du chapitre 8 (et des chapitres 5 et 6) de *Building Enterprise Projects with Go* (Shahsavan, Apress 2026) sur l'analyse d'échappement et la frontière entre passage par valeur et passage par pointeur en Go.

Le banc n'est pas un outil d'optimisation : il produit des **verdicts** (confirmée, infirmée, non concluante) sur des hypothèses numérotées, à partir de mesures dont la provenance (version de Go, `GOOS/GOARCH`, CPU, date) est consignée avec chaque résultat.

Le projet suit le *Spec-Driven Development* (Martinelli, Apress 2026) : le dossier `/docs` fait autorité, chaque hypothèse est écrite et gelée avant la première mesure, et le code est dérivé des cas d'utilisation.

## Objectifs

- Cartographier, pour des structures de 8 à 4 096 octets et plusieurs profils de durée de vie, le point où le passage par pointeur devient moins coûteux que la copie par valeur.
- Vérifier la règle « valeur si 1–3 mots machine » et les rapports de coût annoncés par le livre (cache, allocations, préallocation).
- Constituer un catalogue observé des motifs d'échappement, comparé à la liste du livre.
- Laisser un banc rejouable sur une autre version de Go ou une autre architecture, sans modification du harnais.

## Hors périmètre

- Mesures sur une base de code applicative réelle (voir l'alternative documentée dans `Projets-candidats_Building-Enterprise-Projects-with-Go.md`, projet P1).
- Optimisation du compilateur Go ou contribution en amont.
- Toute conclusion transposée automatiquement à une version de Go non mesurée.
