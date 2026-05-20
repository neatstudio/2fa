APP := 2fa
MAIN := ./cmd/2fa
BIN_DIR := bin
DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: test build install package clean release-notes

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) $(MAIN)

install: build
	mkdir -p $(HOME)/.local/bin
	ln -sf $(CURDIR)/$(BIN_DIR)/$(APP) $(HOME)/.local/bin/$(APP)

package:
	VERSION=$(VERSION) ./scripts/package.sh

release-notes:
	VERSION=$(VERSION) ./scripts/release-notes.sh

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
