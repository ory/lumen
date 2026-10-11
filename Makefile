GO       := go
GOTAGS   := fts5
GOFLAGS  := -tags=$(GOTAGS)

XGORELEASER_IMAGE := oryd/xgoreleaser:1.26.0-2.14.1

.PHONY: build build-local build-bench-swe test eval e2e e2e-bm25 e2e-semantic e2e-lang lint vet tidy clean format plugin-dev

build:
	docker run --platform linux/amd64 --mount type=bind,source="$$(pwd)",target=/project \
		$(XGORELEASER_IMAGE) --snapshot --clean

build-local:
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) \
		-o bin/lumen .

build-bench-swe:
	cd bench-swe && $(GO) build -o ../bin/bench-swe .

test:
	CGO_ENABLED=1 $(GO) test $(GOFLAGS) ./...

# Offline retrieval catalog report (BM25, no embedding service).
eval:
	CGO_ENABLED=1 LUMEN_EVAL_REPORT=$${LUMEN_EVAL_REPORT:-$(CURDIR)/bin/retrieval-bm25.md} \
		$(GO) test $(GOFLAGS) -count=1 -run 'TestRetrievalCatalogBM25' ./internal/index

# All E2E suites. Semantic suites skip without Ollama + all-minilm.
e2e:
	CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=15m -v -count=1 -run 'TestE2E_' ./...

# BM25 executable suites; must pass with no embedding service.
e2e-bm25:
	CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=15m -v -count=1 \
		-run 'TestE2E_BM25|TestE2E_Catalog_BM25|TestLang_BM25SnapshotDiagnostics' .

# Semantic executable suites; a missing embedding service is a failure.
e2e-semantic:
	LUMEN_E2E_SEMANTIC=require CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=20m -v -count=1 -run 'TestE2E_' ./...

e2e-lang:
	LUMEN_E2E_SEMANTIC=require CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=30m -v -count=1 -run 'TestLang_' ./...

lint:
	golangci-lint run

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/ dist/

format:
	goimports -w .
	npx --yes prettier --write "**/*.{json,md,mdx,yaml,yml}"
	npx --yes doctoc --github README.md

plugin-dev: build-local
	@echo "Run: claude --plugin-dir ."

vhs:
	export CLAUDE_PLUGIN_ROOT=$(pwd) && cd testdata/fixtures/go && vhs ../../../docs/demo/demo.tape
