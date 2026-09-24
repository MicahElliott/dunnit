.PHONY: run build package clean vet release tag-release dun

# Go toolchain downloads need checksum verification; don't inherit GOSUMDB=off.
export GOSUMDB := sum.golang.org

TARGET_OS ?= $(shell go env GOOS)

build: dun

# dunnit is deliberately unconditional (.PHONY, no file-based
# prerequisites) rather than depending on `$(shell find . -name
# '*.go')` -- that pattern is vulnerable to a real gotcha: Make's
# staleness check is strictly mtime(prerequisite) > mtime(target), so
# if a .go edit and the resulting `go build` both land within the same
# filesystem-mtime tick, the next `make build`/`make run` can see "no
# .go file newer than dunnit" and silently skip rebuilding even though
# the source changed -- exactly what happened during the 2026-09-02
# session (compounded there by also running a stale packaged
# Dunnit.app instead of this binary at all). `go build` itself is
# already near-instant when nothing changed (its own content-hash
# based cache), so always invoking it here costs essentially nothing
# and removes the whole class of tie/staleness bugs.
dun:
	go build -trimpath -ldflags "-s -w" -o dunnit .

run: build
	./dunnit

# Requires the `fyne` CLI (go install fyne.io/tools/cmd/fyne@latest).
# Produces a native desktop package with the custom icon.
package:
	@if [ "$(TARGET_OS)" != "darwin" ] && [ "$(TARGET_OS)" != "linux" ] && [ "$(TARGET_OS)" != "windows" ]; then \
		echo "Unsupported desktop target: $(TARGET_OS) (supported: darwin, linux, windows)"; \
		exit 1; \
	fi
	fyne package -os $(TARGET_OS) -name Dunnit -icon Icon.png

vet:
	go vet ./...

# Validates the current commit and pushes an annotated version tag. The tag
# triggers GitHub Actions, which builds and publishes all native artifacts.
#
# Usage: make tag-release VERSION=v0.1.0
tag-release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make tag-release VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	@if ! printf '%s\n' "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$$'; then \
		echo "VERSION must look like v1.2.3 or v1.2.3-rc.1"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree is not clean -- commit or stash changes first."; \
		git status --short; \
		exit 1; \
	fi
	@if git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null; then \
		echo "Tag already exists: $(VERSION)"; \
		exit 1; \
	fi
	$(MAKE) build
	$(MAKE) vet
	git push origin HEAD
	git tag -a "$(VERSION)" -m "Dunnit $(VERSION)"
	git push origin "$(VERSION)"

clean:
	rm -f dunnit
	rm -rf Dunnit.app
	rm -f Dunnit-*-macos.zip Dunnit-*-linux.tar.xz Dunnit-*-windows.zip
	rm -f Dunnit.tar.xz Dunnit.exe

# Cuts a local native release: packages for the host OS, tags
# the current commit, pushes the tag, and creates a GitHub Release
# with the native artifact attached (via `gh`, using auto-generated notes from
# commits since the last tag). No CI/GoReleaser involved -- this is
# purely a local, manual-trigger convenience wrapper.
#
# Usage: make release VERSION=v0.1.0
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	@if ! git diff --quiet || ! git diff --cached --quiet; then \
		echo "Working tree has uncommitted changes -- commit or stash first."; \
		exit 1; \
	fi
	$(MAKE) package
	@if [ "$(TARGET_OS)" = "darwin" ]; then \
		rm -f Dunnit-$(VERSION)-macos.zip; \
		zip -r -X Dunnit-$(VERSION)-macos.zip Dunnit.app; \
	else \
		mv Dunnit.tar.xz Dunnit-$(VERSION)-linux.tar.xz; \
	fi
	@# `fyne package` bumps FyneApp.toml's internal Build counter as a
	@# side effect -- discard that so tagging happens on a clean tree
	@# matching what was already committed.
	git checkout -- FyneApp.toml
	git tag $(VERSION)
	env -u GH_TOKEN git push origin $(VERSION)
	@if [ "$(TARGET_OS)" = "darwin" ]; then \
		artifact=Dunnit-$(VERSION)-macos.zip; \
	else \
		artifact=Dunnit-$(VERSION)-linux.tar.xz; \
	fi; \
	env -u GH_TOKEN gh release create $(VERSION) "$$artifact" --title "$(VERSION)" --generate-notes
