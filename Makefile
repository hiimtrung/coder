BINARY  := coder
CMD     := ./cmd/coder
DIST    := dist
PKG     := github.com/trungtran/coder/internal/version
VERSION_FILE := VERSION
CHANGELOG_FILE := CHANGELOG.md
REF ?= origin/main

# Version from git tag or "dev"
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -ldflags "-X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).BuildDate=$(BUILD_DATE) -s -w"

.PHONY: build build-all install clean test release-prepare release-note release-check release-main release-tag tag

## build: Build for the current platform
build:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(BINARY) $(CMD)
	@echo "Built: $(DIST)/$(BINARY)"

## build-all: Cross-compile for all supported platforms
build-all: clean
	@mkdir -p $(DIST)
	@echo "Building $(BINARY) $(VERSION) for all platforms...\n"

	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64   $(CMD)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe $(CMD)

	@echo "\nBuilt binaries:"
	@ls -lh $(DIST)/

## install: Build and install to /usr/local/bin (requires write permission)
install: build
	cp $(DIST)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed: /usr/local/bin/$(BINARY)"

## install-user: Build and install to ~/bin (no sudo required)
install-user: build
	@mkdir -p $(HOME)/bin
	cp $(DIST)/$(BINARY) $(HOME)/bin/$(BINARY)
	@echo "Installed: $(HOME)/bin/$(BINARY)"
	@echo "Make sure $(HOME)/bin is in your PATH"

## clean: Remove build artifacts
clean:
	rm -rf $(DIST)

## test: Quick smoke test of the built binary
test: build
	@echo "--- version ---"
	$(DIST)/$(BINARY) version
	@echo "\n--- list ---"
	$(DIST)/$(BINARY) list
	@echo "\n--- list be ---"
	$(DIST)/$(BINARY) list be
	@echo "\n--- install be --dry-run ---"
	$(DIST)/$(BINARY) install be --dry-run --target /tmp/tas-test

help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## release-prepare: Update VERSION and scaffold CHANGELOG on your working branch (usage: make release-prepare VERSION=v0.5.1)
release-prepare:
	@test -n "$(VERSION)" || (echo "Usage: make release-prepare VERSION=v0.5.1" && exit 1)
	@printf '%s\n' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$$' || \
		(echo "Error: VERSION must look like v1.2.3 or v1.2.3-rc.1" && exit 1)
	@test -f "$(VERSION_FILE)" || (echo "Error: $(VERSION_FILE) is missing" && exit 1)
	@test -f "$(CHANGELOG_FILE)" || (echo "Error: $(CHANGELOG_FILE) is missing" && exit 1)
	@printf '%s\n' "$(VERSION)" > "$(VERSION_FILE)"
	@if grep -Eq '^## \[$(VERSION)\] ' "$(CHANGELOG_FILE)"; then \
		echo "$(CHANGELOG_FILE) already has a '## [$(VERSION)]' section."; \
	else \
		$(MAKE) release-note VERSION="$(VERSION)"; \
	fi
	@echo "Release branch metadata prepared:"
	@echo "  version : $(VERSION)"
	@echo "  files   : $(VERSION_FILE), $(CHANGELOG_FILE)"
	@echo "Next:"
	@echo "  1. Fill in $(CHANGELOG_FILE)"
	@echo "  2. Commit and merge your branch into main"
	@echo "  3. Run: make release-main VERSION=$(VERSION)"

## release-note: Scaffold a changelog section (usage: make release-note VERSION=v0.5.1)
release-note:
	@test -n "$(VERSION)" || (echo "Usage: make release-note VERSION=v0.5.1" && exit 1)
	@printf '%s\n' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$$' || \
		(echo "Error: VERSION must look like v1.2.3 or v1.2.3-rc.1" && exit 1)
	@test -f "$(CHANGELOG_FILE)" || (echo "Error: $(CHANGELOG_FILE) is missing" && exit 1)
	@grep -Eq '^## \[$(VERSION)\] ' "$(CHANGELOG_FILE)" && \
		(echo "Error: $(CHANGELOG_FILE) already has a '## [$(VERSION)]' section." && exit 1) || true
	@tmp_file="$$(mktemp)"; \
	awk -v version="$(VERSION)" -v today="$$(date +%Y-%m-%d)" '\
		BEGIN { inserted = 0 } \
		{ print } \
		!inserted && $$0 == "---" { \
			print ""; \
			print "## [" version "] — " today; \
			print ""; \
			print "### Added"; \
			print "- "; \
			print ""; \
			print "### Changed"; \
			print "- "; \
			print ""; \
			print "### Fixed"; \
			print "- "; \
			print ""; \
			inserted = 1; \
		} \
	' "$(CHANGELOG_FILE)" > "$$tmp_file" && mv "$$tmp_file" "$(CHANGELOG_FILE)"
	@echo "Scaffolded changelog section for $(VERSION) in $(CHANGELOG_FILE)"
	@echo "Edit the bullets, commit the changelog update, then run:"
	@echo "  make release-main VERSION=$(VERSION)"

## release-check: Validate release metadata and target ref (usage: make release-check VERSION=v0.5.1 [REF=origin/main])
release-check:
	@test -n "$(VERSION)" || (echo "Usage: make release-check VERSION=v0.5.1 [REF=origin/main]" && exit 1)
	@printf '%s\n' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$$' || \
		(echo "Error: VERSION must look like v1.2.3 or v1.2.3-rc.1" && exit 1)
	@test -f "$(VERSION_FILE)" || (echo "Error: $(VERSION_FILE) is missing" && exit 1)
	@test -f "$(CHANGELOG_FILE)" || (echo "Error: $(CHANGELOG_FILE) is missing" && exit 1)
	@git rev-parse --verify "$(REF)" >/dev/null 2>&1 || \
		(echo "Error: REF '$(REF)' does not exist locally. Try: git fetch origin --tags" && exit 1)
	@git diff --quiet || (echo "Error: working tree has unstaged changes. Commit or stash them first." && exit 1)
	@git diff --cached --quiet || (echo "Error: index has staged changes. Commit or stash them first." && exit 1)
	@test "$$(tr -d '[:space:]' < $(VERSION_FILE))" = "$(VERSION)" || \
		(echo "Error: $(VERSION_FILE) must contain $(VERSION) before releasing." && exit 1)
	@grep -Eq '^## \[$(VERSION)\] ' "$(CHANGELOG_FILE)" || \
		(echo "Error: $(CHANGELOG_FILE) is missing a '## [$(VERSION)]' section." && exit 1)
	@git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null 2>&1 && \
		(echo "Error: local tag $(VERSION) already exists." && exit 1) || true
	@git ls-remote --exit-code --tags origin "refs/tags/$(VERSION)" >/dev/null 2>&1 && \
		(echo "Error: remote tag $(VERSION) already exists on origin." && exit 1) || true
	@echo "Release checks passed:"
	@echo "  version : $(VERSION)"
	@echo "  ref     : $(REF)"
	@echo "  commit  : $$(git rev-parse --short $(REF))"
	@echo "  subject : $$(git log -1 --format=%s $(REF))"

## release-main: Validate origin/main and push the release tag (usage: make release-main VERSION=v0.5.1 [REF=origin/main])
release-main:
	@$(MAKE) release-tag VERSION="$(VERSION)" REF="$(REF)"

## release-tag: Create and push an annotated tag from a merged ref (usage: make release-tag VERSION=v0.5.1 [REF=origin/main])
release-tag: release-check
	@echo "Creating annotated tag $(VERSION) from $(REF)..."
	@git tag -a "$(VERSION)" "$(REF)" -m "Release $(VERSION)"
	@echo "Pushing tag $(VERSION) to origin..."
	@git push origin "refs/tags/$(VERSION)"
	@echo "Release tag pushed:"
	@echo "  version : $(VERSION)"
	@echo "  ref     : $(REF)"
	@echo "GitHub Actions will build and publish the release from this tag."

## tag: Backward-compatible alias for release-tag (usage: make tag VERSION=v0.5.1 [REF=origin/main])
tag:
	@echo "make tag is deprecated; using release-tag instead."
	@$(MAKE) release-tag VERSION="$(VERSION)" REF="$(REF)"
