# `make check` is what CI runs on every pull request: the CI_go steps (each one
# is a target here, and CI_go calls these same targets) plus the internal-names
# check of CI_leak_guard over every tracked file. Run it before you open a PR.
#
#   make check          everything below, in CI's order
#   make plugins-json   regenerate docs/plugins.json from the registry and docs/plugins.md
#
# The `::error` lines are GitHub annotations; locally they're plain messages.

GO     ?= go
PYTHON ?= python3

# Packages allowed outside the standard library: this module and the approved
# dependencies. Adding a dependency means adding it here, in review.
MODULE  := github.com/patch-it-dev/patch-it-plugins
ALLOWED := ^($(MODULE)(/|$$)|gopkg\.in/yaml\.v3$$)

.PHONY: check tidy build fmt vet imports nethttp test leak-guard plugins-json

check: tidy build fmt vet imports nethttp test leak-guard
	@echo "make check: all passed"

# go.mod and go.sum are tidy.
tidy:
	$(GO) mod tidy
	@git diff --exit-code -- go.mod go.sum || { echo "::error file=go.mod::go.mod/go.sum are not tidy. Run 'go mod tidy' and commit the result."; exit 1; }

build:
	$(GO) build ./...

fmt:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		for f in $$unformatted; do echo "::error file=$$f::not gofmt-formatted. Run 'gofmt -w $$f'."; done; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

# An allowlist, so a plugin can't pull in a database driver, a cloud SDK or
# anything private.
imports:
	@others=$$($(GO) list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./... | sed 's/ \[.*//' | sort -u | grep -vE '$(ALLOWED)' || true); \
	if [ -n "$$others" ]; then \
		echo "::error::imports outside the allowlist (Makefile, ALLOWED):"; echo "$$others"; exit 1; \
	fi

# Plugins fetch through env.Get / kit, never on their own.
nethttp:
	@bad=$$($(GO) list -f '{{.ImportPath}} {{join .Imports " "}}' ./... \
		| grep -E '(^| )net/http( |$$)' | cut -d' ' -f1 \
		| grep -vE '^$(MODULE)/(fetch|kit|adapter)(/|$$)' || true); \
	if [ -n "$$bad" ]; then \
		echo "::error::these packages import net/http; fetch through env.Get or kit instead:"; echo "$$bad"; exit 1; \
	fi

# -race: the fetcher and the tests' HTTP servers run on other goroutines.
# Includes the check that docs/plugins.json is up to date (internal/pluginlist).
test:
	$(GO) test -race -count=1 ./...

# CI_leak_guard checks only what a PR adds (plus gitleaks); this checks every
# tracked file.
leak-guard:
	$(PYTHON) scripts/leak_guard.py --all

plugins-json:
	$(GO) run ./cmd/list-plugins
