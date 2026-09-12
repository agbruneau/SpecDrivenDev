# EscapeBench — contrat exécutable (BEPG ch. 11, p. 324-325) : le Makefile dit QUOI, le CI dit OÙ.
COUNT ?= 20
PARAMS ?=
MATRIX ?=
CAMPAIGN ?=
HYPOTHESES ?=

.PHONY: fmt vet test integration_test matrix escape campaign compare verdict dashboard tidy build clean

fmt:
	gofmt -l -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test -race -shuffle=on -count=1 ./...

## Tests d'intégration (adapters réels : compilateur, système de fichiers) — build tag BEPG ch. 17.
## Ils rejouent les verdicts archivés sous results/ (non-régression de UC-005) et classent la
## sortie véritable de `go build -gcflags=-m` (UC-002). `-shuffle=on` comme la commande de
## référence de CLAUDE.md : l'ordre des tests ne doit pas porter de résultat.
integration_test:
	go test -race -shuffle=on -count=1 -tags=integration_test -timeout 300s ./...

build:
	go build -o bin/escapebench ./cmd/escapebench

## Sous-commandes du banc — chacune correspond à un cas d'utilisation (docs/use-cases).
matrix: build            ## UC-001 : matrice de référence si PARAMS est vide
	./bin/escapebench matrix $(if $(PARAMS),--params "$(PARAMS)",--reference)

escape: build            ## UC-002
	./bin/escapebench escape --matrix $(MATRIX)

campaign: build          ## UC-003 (C-003 : count ≥ 20) ; HYPOTHESES=H-001,H-002 restreint le gel (exigé pour H-012, C-009)
	./bin/escapebench campaign --matrix $(MATRIX) --count $(COUNT) $(if $(HYPOTHESES),--hypotheses $(HYPOTHESES))

compare: build           ## UC-004
	./bin/escapebench compare --campaign $(CAMPAIGN)

verdict: build           ## UC-005
	./bin/escapebench verdict --campaign $(CAMPAIGN)

dashboard: build         ## UC-005 étape 7 / FR-007
	./bin/escapebench dashboard

tidy:
	go mod tidy

clean:
	rm -rf bin
