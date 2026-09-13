# Décisions de construction — LeakLab (P3)

**Date :** 2026-09-13 · **Portée :** spécification, corpus, banc et première campagne de `LeakLab/`.

Ce journal consigne les décisions qui s'écartent de la lettre du processus, figent un choix que la spécification laissait ouvert, ou tirent une leçon d'EscapeBench. Il ne répète pas ce que le code et les cas d'utilisation disent déjà.

## 1. Processus

**D-01 — Les trois cas d'utilisation sont passés à `Approved` sans revue humaine, puis à `Deployed` à la clôture.** Le mandat était de développer le projet au complet, tests, validation et documentation compris, sans intervention. Comme pour EscapeBench (D-01 de `DECISION.md`), l'agent a assumé la transition que l'AIUP réserve au chercheur. La revue humaine prévue par Martinelli (2026) n'a pas eu lieu ; elle reste la première chose à faire avant d'étendre le banc.

**D-02 — Le gel des critères est daté par un commit, avant le code.** Le commit `6ad40aa` porte la vision, le catalogue d'exigences, les treize critères et le corpus de référence ; le code arrive au commit suivant, et aucun cas du corpus n'avait été exécuté entre les deux. L'empreinte de chaque critère est en outre recalculée à chaque production de verdicts (C-008). Deux réserves, écrites avant les mesures dans `requirements.md` : l'auteur connaissait les changements de Go qui touchent H-011 et H-013, et C-006 a été révisée après le gel, en construisant le pilote de LEAKPROFILE, avant toute campagne (D-07). Une contrainte n'est pas un critère : sa révision ne change aucune empreinte. Après `R-2026-09-13-1`, trois autres modifications de `requirements.md` ont été datées et committées (`6c74693`) avant tout code et avant la campagne qui les juge : la précision de C-004, le bras `AFTERFUNC_WITNESS` de C-007 et l'hypothèse successeur H-014 (D-13, D-14). Les treize critères d'origine n'ont pas bougé : les empreintes consignées par les deux campagnes sont identiques.

**D-03 — L'oracle a été exécuté pendant la construction ; les détecteurs, non.** Valider la vérité terrain du corpus exige d'exécuter ses scénarios : c'est ce que fait l'oracle, par les piles de goroutines. Aucun détecteur dynamique (BARE, RACE, SYNCTEST, NUMGOROUTINE, LEAKPROFILE, PROGRAM) ni `go vet` n'a tourné sur le corpus avant la campagne `R-2026-09-13-1`. Une exception : le test `TestUC003_CommandeCtxvet` passe `ctxvet` sur le paquet du corpus pour vérifier qu'il y trouve l'appel à `net.Dial`. Ce détecteur n'est jugé par aucun critère, et son résultat découle de la table BR-003-1. Les autres tests éprouvent la chaîne sur une fixture (`internal/campaign/testdata/fixture`) et sur des campagnes synthétiques.

## 2. Architecture

**D-04 — Deux modules Go (C-003).** Le corpus est fautif par construction : `go vet ./...` et `go test -race ./...` du module principal échoueraient sur lui. Le module imbriqué `lab/` le rend invisible des motifs `./...` ; le module principal l'importe par une directive `replace` pour lire son catalogue, sans jamais exécuter ses scénarios.

**D-05 — Layout plat, sans ports.** EscapeBench suivait le layout hexagonal du chapitre 14 du livre. LeakLab n'a aucun adaptateur à remplacer : chaque interface y aurait une seule implémentation, ce que le livre lui-même appelle de la cérémonie (p. 517). La tension entre ces deux règles du livre est l'objet de P4 (HexaGuard) ; LeakLab ne la tranche pas, il évite de la créer. Les entités persistées portent directement leurs tags JSON : aucune contrainte ne l'interdit ici, contrairement à EscapeBench.

**D-06 — Un processus par observation, répétitions en boucle externe.** NFR-003 exige l'isolation : une fuite, une minuterie ou une erreur fatale ne passe pas d'une observation à la suivante. Les répétitions forment la boucle externe (toutes les cellules à la répétition 1, puis à la 2…) pour étaler sur la matrice entière la dérive lente de la machine, au lieu de la concentrer sur une cellule.

## 3. Corpus et détecteurs

**D-07 — Pause de 100 ms avant le profil `goroutineleak`.** Le profil ne reconnaît qu'une goroutine déjà garée : une goroutine lancée par le scénario, encore exécutable à l'instant du profil, ne peut pas être dite fuitée. Sans pause, LEAKPROFILE aurait été jugé sur la course entre le retour du scénario et l'ordonnanceur. La révision de C-006 le consigne ; elle précède toute campagne.

**D-08 — L'oracle lit les piles de goroutines, pas les détecteurs.** C'est la leçon de H-006 dans EscapeBench : un verdict faux y est venu d'un classificateur du banc, non d'une observation. Ici, la vérité terrain d'un cas `leak` est une goroutine bloquée dans la fonction `Worker` du cas, sur l'état que `runtime.Stack` imprime pour la primitive déclarée (`chan send`, `sync.Mutex.Lock`, `IO wait`…) ; celle d'un autre cas est l'absence de cette fonction dans toutes les piles après au plus une seconde. Les noms de `Worker` sont tirés des valeurs de fonction, et un test garantit qu'ils sont distincts. Ce que l'oracle ne vérifie pas : les courses (il tourne sans `-race`), l'accessibilité d'une primitive depuis une racine du ramasse-miettes, les défauts statiques et les deux cas d'interblocage, qu'il ne ferait qu'attendre. Ces attributs tiennent par construction et sont relus dans le code de chaque cas.

**D-09 — TCP brut au lieu de `http.Get` (Adaptation).** L'exemple de la p. 568 appelle `http.Get`. Le transport HTTP garde des goroutines de lecture et d'écriture par connexion, et des connexions oisives pendant 90 s : un cas corrigé aurait laissé des goroutines vivantes sans fuite, et NUMGOROUTINE aurait été jugé sur le transport. `net.Dial` puis `Read` gardent le motif — une I/O bloquante sans contexte — sans ce bruit. `ctxvet` vise `http.Get` comme `net.Dial`.

**D-10 — Le délai par défaut de `go test` est conservé (C-004).** Les binaires de test ne reçoivent pas `-test.timeout` ; la campagne tue elle-même un processus au bout de 5 s. Le banc observe ainsi ce qu'obtient un développeur qui lance `go test` sans option, ce qui est l'objet de H-008.

**D-11 — Chaque hypothèse a son témoin.** Autre leçon d'EscapeBench : une confirmation ne vaut que si l'infirmation était possible, et un verdict ne distingue pas un banc muet d'une affirmation vraie sans témoin. `witness-fail` garde H-001 et H-002 (un échec doit être vu, son message capturé) ; le bras `REAL` garde H-006 ; le parent `OPAQUE`, dont `context` surveille chaque enfant par une goroutine, garde H-010 ; `RETAINED_WITNESS` garde H-011. H-005 et H-013 sont non concluantes si le détecteur ne diagnostique aucun cas.

**D-12 — `ctxvet` est syntaxique.** C-002 exclut `golang.org/x/tools/go/analysis`. L'analyseur résout les noms d'import du fichier et écarte un identifiant local homonyme, mais ne vise pas les méthodes (`db.Query`, `conn.Read`), faute de types. C'est une limite déclarée (BR-003-3) que la matrice mesure.

## 4. Décisions prises après la campagne `R-2026-09-13-1`

La première campagne a tourné sur le banc du commit `4ff35fd`. Avant d'en croire les verdicts, chaque résultat contraire à l'attente de l'auteur a été relu jusqu'à sa cause. Trois venaient du banc, un du détecteur jugé.

**D-13 — Le banc n'appliquait pas C-004 ; la campagne est refaite, la première archivée.** Le paquet `testing` fixe `-test.timeout` à 0 (`testing.go`, ligne 476) : c'est `go test` qui transmet 10 minutes au binaire. Lancés directement, les binaires de la campagne n'avaient donc aucune minuterie d'alarme. Or `checkdead` renonce à déclarer l'interblocage dès qu'une minuterie est en attente (`proc.go`, lignes 6511 à 6519). Sans alarme, BARE a rendu l'erreur fatale « all goroutines are asleep » sur les deux cas d'interblocage (H-008 confirmée), et SYNCTEST l'a rendue sur `mutex-leak` et `global-channel-leak`, dont la goroutine bloquée ne l'est pas durablement : ce n'est pas `synctest` qui les diagnostiquait, c'est le runtime, faute d'alarme. La spécification était juste ; le code ne la suivait pas. Le binaire passe désormais `-test.timeout=10m0s` (test `TestUC001_C004_DelaiParDefaut`), C-004 le dit explicitement, et `R-2026-09-13-2` rejoue la campagne entière. `R-2026-09-13-1` reste dans `results/`, avec ses verdicts, comme trace du défaut.

**D-14 — Le résidu de H-009 mesurait l'expiration des minuteries ; H-014 lui succède.** H-009 compare des contextes expirés à des contextes annulés. Une contre-épreuve hors campagne a montré que 20 000 `time.AfterFunc` expirés sans aucun contexte laissent 182 à 234 octets par opération dans le tas, et 20 000 contextes expirés 147 à 156 : le rappel de chaque minuterie expirée tourne dans une goroutine, et le runtime garde ensuite ses structures. Le résidu de H-009 dépasse donc son seuil sans qu'aucun contexte ne soit retenu. Le critère de H-009 est gelé ; il n'est pas retouché. H-014 reprend l'affirmation en retranchant le bras témoin `AFTERFUNC_WITNESS`, avec la réserve écrite dans `requirements.md` : son critère connaît l'ordre de grandeur du témoin.

**D-15 — La mesure nulle de H-006 est une résolution d'horloge, pas un défaut.** Les cinq répétitions du bras `SYNCTEST` valent exactement 0 ns. Une contre-épreuve l'explique : sur ce poste Windows, 997 boucles de calcul sur 1 000 sont mesurées à 0 par `time.Since`, et 200 bulles consécutives prennent 1,06 ms, soit 5,3 µs chacune. Une valeur nulle signifie « sous la résolution de l'horloge monotone », bien en deçà du seuil de 50 ms. Le verdict tient ; la sonde ne mesure pas la durée d'une bulle, elle en borne l'ordre de grandeur.

**D-16 — L'échec du profil `goroutineleak` sur `mutex-leak` est réel.** Le runtime prévoit les mutex (`mgc.go`, ligne 1189) ; le profil a pourtant manqué ce cas cinq fois sur cinq. Contre-épreuve : un `sync.Mutex` alloué seul (8 octets) n'est jamais signalé, le même mutex au début d'une structure de 72 octets l'est chaque fois (trois essais chacun). La taille de l'objet qui porte le mutex décide. Le mécanisme le plus probable, non vérifié dans le code, est l'allocateur *tiny* : un objet de moins de 16 octets sans pointeur partage son bloc avec d'autres, et le bloc reste marqué tant que l'un d'eux vit. Ce n'est pas un artefact du banc : `new(sync.Mutex)` est une écriture ordinaire. L'infirmation de H-013 est retenue.

**D-17 — H-002 ne dépend pas du sous-test.** Le livre parle d'un sous-test qui oublie une goroutine ; le corpus fuit dans un test de premier niveau. Contre-épreuve : un sous-test séquentiel et un sous-test parallèle qui laissent chacun une goroutine bloquée produisent, sous `-v`, les seules lignes `=== RUN`, `=== PAUSE`, `=== CONT` et `--- PASS`.

## 5. Vérification

| Contrôle | Résultat |
|---|---|
| `go vet ./...`, `gofmt -l` (module principal) | propre |
| `go test -race -shuffle=on -count=1 ./...` | tous les paquets au vert, windows/amd64, go1.27.0 |
| Oracle `cd lab && go test -count=1 ./corpus` | 29 cas exécutés, vérité terrain confirmée |
| Conformité catalogue ↔ `requirements.md` | identiques (test `TestUC001_BR1_CorpusConformeSpec`) |
| Hook `guard-paths.sh` | refuse `results/` (casse ignorée) et les chemins avec `..` ; laisse `docs/` |
| Campagne de référence `R-2026-09-13-2` | 960 observations dynamiques, 64 statiques, 75 mesures de sonde, 9 min 01 ; aucune cellule instable |
| Verdicts | six confirmées, huit infirmées ; rapport `RAPPORT-FINAL_LeakLab.md` |
| Linux | non exécuté : toolchain de WSL trop ancienne, aucun téléchargement lancé |
