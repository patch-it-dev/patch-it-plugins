# `make check` is what CI runs on every pull request: the CI_go steps (each one
# is a target here, and CI_go calls these same targets), the internal-names
# check of CI_leak_guard over every tracked file, and CI_contract's fixture
# comparison when you point it at a patch-it checkout. Run it before you open a PR.
#
#   make check                       everything below, in CI's order
#   make check PATCH_IT=../patch-it  also compare contract/fixtures/ with patch-it's
#   make plugins                     print the providers and sources from the registry (JSON)
#   make new-plugin ARGS='--provider acme --source api --kind openapi'
#                                    scaffold a source (cmd/new-plugin; asks for what's missing)
#
# The `::error` lines are GitHub annotations; locally they're plain messages.

GO     ?= go
PYTHON ?= python3

# Packages allowed outside the standard library: this module and the approved
# dependencies. Adding a dependency means adding it here, in review.
MODULE  := github.com/patch-it-dev/patch-it-plugins
ALLOWED := ^($(MODULE)(/|$$)|gopkg\.in/yaml\.v3$$)

.PHONY: check tidy build fmt vet imports nethttp test todos leak-guard contract plugins new-plugin

check: tidy build fmt vet imports nethttp test todos leak-guard contract
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
		| grep -vE '^$(MODULE)/(core/fetch|core/kit|plugins/openapi/adapter)(/|$$)' || true); \
	if [ -n "$$bad" ]; then \
		echo "::error::these packages import net/http; fetch through env.Get or kit instead:"; echo "$$bad"; exit 1; \
	fi

# -race: the fetcher and the tests' HTTP servers run on other goroutines.
test:
	$(GO) test -race -count=1 ./...

# A provider started with new-plugin is done when nothing it wrote still says
# TODO(new-plugin).
TODO_DIRS := plugins providers
todos:
	@left=$$(grep -rl 'TODO(new-plugin)' $(TODO_DIRS) || true); \
	if [ -n "$$left" ]; then \
		for f in $$left; do echo "::error file=$$f::TODO(new-plugin) left: fill it in (grep -n 'TODO(new-plugin)' $$f)."; done; \
		exit 1; \
	fi

# CI_leak_guard checks only what a PR adds (plus gitleaks); this checks every
# tracked file.
leak-guard:
	$(PYTHON) scripts/leak_guard.py --all

# CI_contract: contract/fixtures/ hold the same events as patch-it's fixtures/.
# Needs a patch-it checkout; skipped without one, as CI_contract skips while
# patch-it can't be read anonymously.
contract:
	@if [ -n "$(PATCH_IT)" ]; then \
		$(PYTHON) scripts/check_contract.py "$(PATCH_IT)/fixtures"; \
	else \
		echo "contract: skipped (make check PATCH_IT=<a patch-it checkout> compares contract/fixtures/)"; \
	fi

# The list of what patch-it watches, from the registry. Never committed: the
# Release workflow attaches it to each release as plugins.json.
plugins:
	@$(GO) run ./cmd/list-plugins

# Scaffold a provider's source on a kind: cmd/new-plugin with ARGS. In a
# terminal it asks for whatever ARGS leaves out.
new-plugin:
	$(GO) run ./cmd/new-plugin $(ARGS)
