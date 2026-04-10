BINARY    := bipolar
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)
BUILD_DIR := dist

GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build build-all clean tidy lint release

# ── Local build ───────────────────────────────────────────────────────────────

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/bipolar

# ── Cross-compile for all platforms ───────────────────────────────────────────

build-all: clean
	@mkdir -p $(BUILD_DIR)

	@echo "→ darwin/amd64"
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64      ./cmd/bipolar

	@echo "→ darwin/arm64  (Apple Silicon)"
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64      ./cmd/bipolar

	@echo "→ linux/amd64"
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64       ./cmd/bipolar

	@echo "→ linux/arm64"
	GOOS=linux   GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64       ./cmd/bipolar

	@echo "→ windows/amd64"
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/bipolar

	@echo ""
	@echo "Binaries written to $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

# ── Release archives (tar.gz / zip) ───────────────────────────────────────────

release: build-all
	@echo "Packaging release archives..."

	cd $(BUILD_DIR) && tar czf $(BINARY)-$(VERSION)-darwin-amd64.tar.gz   $(BINARY)-darwin-amd64
	cd $(BUILD_DIR) && tar czf $(BINARY)-$(VERSION)-darwin-arm64.tar.gz   $(BINARY)-darwin-arm64
	cd $(BUILD_DIR) && tar czf $(BINARY)-$(VERSION)-linux-amd64.tar.gz    $(BINARY)-linux-amd64
	cd $(BUILD_DIR) && tar czf $(BINARY)-$(VERSION)-linux-arm64.tar.gz    $(BINARY)-linux-arm64
	cd $(BUILD_DIR) && zip     $(BINARY)-$(VERSION)-windows-amd64.zip     $(BINARY)-windows-amd64.exe

	@echo ""
	@echo "Release archives ready:"
	@ls -lh $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip

# ── Utilities ─────────────────────────────────────────────────────────────────

tidy:
	go mod tidy

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
