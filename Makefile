# GoPress development Makefile.

.PHONY: build test lint generate deps check oracle-up fixture golden diff e2e porting-status

GO ?= go
export PATH := $(HOME)/go/bin:$(PATH)
export GOTOOLCHAIN ?= local

build:
	$(GO) build -o bin/gopress ./cmd/gopress
	$(GO) build -o bin/gopress-tools ./cmd/gopress-tools
	$(GO) build -o bin/gopress-lint ./cmd/gopress-lint

test:
	$(GO) test ./...

lint: build
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	$(GO) vet ./...
	staticcheck ./...
	./bin/gopress-lint ./...

generate:
	@echo "generate: hookgen/apigen/gen-assets land in P1; nothing to do yet"

deps:
	@echo "deps-check lands in P0.8; verifying modules readonly"
	$(GO) mod verify

check: lint deps test

oracle-up:
	@bash scripts/oracle-up.sh

fixture:
	@bash scripts/fixture.sh

golden:
	@bash scripts/golden.sh

diff: build
	./bin/gopress-tools diff --ref http://127.0.0.1:8081 --go http://127.0.0.1:8081

e2e:
	@echo "e2e lands with Playwright" >&2; exit 1

porting-status:
	$(GO) run ./cmd/gopress-tools porting-status
