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
	# -p 1: packages share the fixture MySQL DB; parallel boots race on
	# session_tokens / usermeta read-modify-write and flake auth tests.
	$(GO) test -p 1 ./...

lint: build
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	$(GO) vet ./...
	staticcheck ./...
	./bin/gopress-lint ./...

generate:
	@echo "generate: hookgen/apigen/gen-assets land in P1; nothing to do yet"

deps: build
	$(GO) mod verify
	./bin/gopress-tools deps-check

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
	@if ! command -v node >/dev/null 2>&1; then \
		echo "e2e: node not available; skipping Playwright smoke" >&2; \
		exit 0; \
	fi; \
	if ! command -v npm >/dev/null 2>&1; then \
		echo "e2e: npm not available; skipping Playwright smoke" >&2; \
		exit 0; \
	fi; \
	cd tests/e2e && \
		if [ ! -d node_modules ]; then npm install; fi && \
		(npx playwright install --with-deps chromium >/dev/null 2>&1 || npx playwright install chromium) && \
		GOPRESS_URL="$${GOPRESS_URL:-http://127.0.0.1:8082}" npm test

porting-status:
	$(GO) run ./cmd/gopress-tools porting-status
