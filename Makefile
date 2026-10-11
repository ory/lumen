GO       := go
GOTAGS   := fts5
GOFLAGS  := -tags=$(GOTAGS)

XGORELEASER_IMAGE := oryd/xgoreleaser:1.26.0-2.14.1

.PHONY: build build-local build-bench-swe test e2e e2e-lang e2e-worktrees test-worktree-mutations lint vet tidy clean format plugin-dev

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

e2e:
	CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=10m -v -count=1 -run 'TestE2E_' ./...

e2e-lang:
	CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),e2e -timeout=30m -v -count=1 -run 'TestLang_' ./...

# Offline: real Lumen binaries, Git worktrees, and MCP stdio against a
# deterministic embedding server. Requires git; no Ollama or LM Studio.
e2e-worktrees:
	CGO_ENABLED=1 $(GO) test -tags=$(GOTAGS),worktree_e2e -timeout=20m -v -count=1 ./e2e/worktrees/

# Curated production mutants must each fail their designated assertion.
test-worktree-mutations:
	CGO_ENABLED=1 $(GO) run ./e2e/worktrees/mutate $(if $(MUTATIONS_OUT),-out $(MUTATIONS_OUT))

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
