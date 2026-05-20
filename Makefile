APP := 2fa
MAIN := ./cmd/2fa
BIN_DIR := bin
DIST_DIR := dist
VERSION ?= $(shell tr -d '[:space:]' < VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: test build install package clean release-notes tag

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

tag:
	git diff --quiet
	git diff --cached --quiet
	git tag $(VERSION)
	git push origin main $(VERSION)

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
