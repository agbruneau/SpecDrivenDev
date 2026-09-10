# EscapeBench — contrat exécutable (BEPG ch. 11, p. 324-325) : le Makefile dit QUOI, le CI dit OÙ.
COUNT ?= 20
MATRIX ?=
CAMPAIGN ?=

.PHONY: fmt vet test integration_test bench matrix escape campaign compare verdict dashboard tidy build clean

fmt:
	gofmt -l -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test -race -shuffle=on -count=1 ./...

## Tests d'intégration (adapters réels : compilateur, système de fichiers) — build tag BEPG ch. 17.
integration_test:
	go test -race -count=1 -tags=integration_test -timeout 300s ./...

build:
	go build -o bin/escapebench ./cmd/escapebench

## Sous-commandes du banc — chacune correspond à un cas d'utilisation (docs/use-cases).
matrix: build            ## UC-001 : matrice de référence si MATRIX est vide
	./bin/escapebench matrix $(if $(MATRIX),--params $(MATRIX),--reference)

escape: build            ## UC-002
	./bin/escapebench escape --matrix $(MATRIX)

campaign: build          ## UC-003 (C-003 : count ≥ 20)
	./bin/escapebench campaign --matrix $(MATRIX) --count $(COUNT)

compare: build           ## UC-004
	./bin/escapebench compare --campaign $(CAMPAIGN)

verdict: build           ## UC-005
	./bin/escapebench verdict --campaign $(CAMPAIGN)

dashboard: build         ## UC-005 étape 7 / FR-007
	./bin/escapebench dashboard

bench: campaign compare  ## Chaîne complète de mesure pour une matrice

tidy:
	go mod tidy

clean:
	rm -rf bin
