# raycanvas — build system

MODULE   := github.com/ha1tch/raycanvas
BIN      := bin
GOFLAGS  := -v
EXAMPLES := basic text paths curves grid zui shadows icons

.PHONY: help build build-examples lint clean fmt version release patch package $(EXAMPLES)

# ── Help (default target) ─────────────────────────────────────────────────────
help:
	@echo "raycanvas $(shell cat VERSION)"
	@echo ""
	@echo "Library"
	@echo "  make build                    Type-check the library (no main)"
	@echo ""
	@echo "Examples"
	@echo "  make build-examples           Compile all examples to bin/"
	@echo "  make basic                    Build and run the basic example"
	@echo "  make text                     Build and run the text example"
	@echo "  make paths                    Build and run the paths example"
	@echo "  make curves                   Build and run the curves example"
	@echo "  make grid                     Build and run the grid (joxel) example"
	@echo "  make zui                      Build and run the zui (quag) example"
	@echo "  make shadows                  Build and run the shadows & blur example"
	@echo ""
	@echo "Quality"
	@echo "  make lint                     go vet on library and all examples"
	@echo "  make fmt                      gofmt -w on all .go files"
	@echo ""
	@echo "Versioning"
	@echo "  make version                  Print current version"
	@echo "  make release V=0.2.1          Cut a full versioned release"
	@echo "  make patch   V=0.2.1-patched01  Patched checkpoint (no CHANGELOG check)"
	@echo "  make package                  Quick zip of current state"
	@echo "  make package S=wip            Quick zip with suffix (e.g. -wip)"
	@echo ""
	@echo "Housekeeping"
	@echo "  make clean                    Remove bin/ and raycanvas-*.zip"

# ── Library build ─────────────────────────────────────────────────────────────
build:
	go build $(GOFLAGS) ./...

# ── Build all example binaries ────────────────────────────────────────────────
# Each example has its own go.mod with a replace directive pointing at ../../
build-examples: $(patsubst %, $(BIN)/%, $(EXAMPLES))

$(BIN)/%: examples/%/main.go
	@mkdir -p $(BIN)
	cd examples/$* && go build $(GOFLAGS) -o ../../$(BIN)/$* .

# ── Run individual examples (build if needed) ─────────────────────────────────
basic: $(BIN)/basic
	./$(BIN)/basic

text: $(BIN)/text
	./$(BIN)/text

paths: $(BIN)/paths
	./$(BIN)/paths

curves: $(BIN)/curves
	./$(BIN)/curves

grid: $(BIN)/grid
	./$(BIN)/grid

zui: $(BIN)/zui
	./$(BIN)/zui

shadows: $(BIN)/shadows
	./$(BIN)/shadows

icons: $(BIN)/icons
	./$(BIN)/icons

# ── Quality ───────────────────────────────────────────────────────────────────
lint:
	go vet ./...
	@for ex in $(EXAMPLES); do \
		echo "vet examples/$$ex"; \
		(cd examples/$$ex && go vet ./...); \
	done

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

# ── Versioning ────────────────────────────────────────────────────────────────
version:
	@cat VERSION

# Cut a full versioned release: make release V=0.2.1
release:
ifndef V
	$(error V is required: make release V=0.2.1)
endif
	./release.sh $(V)

# Cut a patched checkpoint: make patch V=0.2.1-patched01
patch:
ifndef V
	$(error V is required: make patch V=0.2.1-patched01)
endif
	./release.sh $(V) --short

# Quick zip of current state: make package [S=suffix]
package:
ifdef S
	./package.sh $(S)
else
	./package.sh
endif

# ── Housekeeping ──────────────────────────────────────────────────────────────
clean:
	rm -rf $(BIN) raycanvas-*.zip
