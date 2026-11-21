# PostHog Go SDK (`posthog/go`)

## What it watches

Names removed or renamed in each release of posthog-go, from that release's section of its `CHANGELOG.md` at its tag. Package `go:github.com/posthog/posthog-go` (events name the module at each release's major, `…/v2` from 2.0.0); releases listed from the repo's `vX.Y.Z` tags. The Python SDK's 8.0 migration guide is a separate source (`posthog/python-sdk`).

Kind: `changelog` (`plugins/changelog`, `sdks.go`).

## Why this source

- **Source:** https://github.com/PostHog/posthog-go/blob/main/CHANGELOG.md, read raw at the release's tag; checked on 2026-11-20.
- **Why this one:** changesets writes each major's changes under "### Major Changes", one bullet per change starting with its commit (`fc05013: Remove …`).
- **Alternatives considered:** GitHub release notes (the same text); PostHog's SDK docs (one page, not versioned).
- **Quote:** "fc05013: Remove `Capture.Library` and `Exception.Library`." (2.0.0).

## How the provider announces changes

Majors are rare; v2 moved the module path to `github.com/posthog/posthog-go/v2`.

## What the plugin reads and ignores

- Reads "Major Changes" bullets (and any breaking heading); the commit prefix is read past.
- Drops (and logs) behavior changes and bullets that don't name what they remove plainly.

## Known gaps

- A release whose section is missing at its tag is skipped as nothing to read.
