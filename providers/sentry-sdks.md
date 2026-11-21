# Sentry SDKs (`sentry/python-changelog`, `sentry/javascript-changelog`, `sentry/go`, `sentry/dotnet`)

## What it watches

Names removed, renamed or deprecated in each release of Sentry's Python, JavaScript, Go and .NET SDKs, from that release's section of the SDK's `CHANGELOG.md` at its tag. One `package_release` event per bullet that names what it changes plainly. Sentry's API is a separate source (`sentry/api`).

Kind: `changelog` (`plugins/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Tag |
|---|---|---|---|
| `sentry/python-changelog` | `pypi:sentry-sdk` | PyPI | `X.Y.Z` |
| `sentry/javascript-changelog` | `npm:@sentry/node`, `npm:@sentry/browser` (released together) | npm (`@sentry/node`) | `X.Y.Z` |
| `sentry/go` | `go:github.com/getsentry/sentry-go` | the repo's `vX.Y.Z` tags | `vX.Y.Z` |
| `sentry/dotnet` | `nuget:Sentry` | the repo's `X.Y.Z` tags | `X.Y.Z` |

## Why this source

- **Source:** https://github.com/getsentry/sentry-python/blob/master/CHANGELOG.md, https://github.com/getsentry/sentry-javascript/blob/develop/CHANGELOG.md, https://github.com/getsentry/sentry-go/blob/master/CHANGELOG.md, https://github.com/getsentry/sentry-dotnet/blob/main/CHANGELOG.md, each read raw at the release's tag; checked on 2026-11-20.
- **Why this one:** Sentry writes every major's removals into the changelog itself, under "Removed" / "Breaking Changes" / "Deprecations" headings or as conventional-commit bullets (`feat(core)!: Remove \`BaseClient\``). The Python 2.0 changelog section carries the same "Removed" and "Deprecated" lists as its `MIGRATION_GUIDE.md`, so the guide isn't a second source (it would repeat the same events under another id).
- **Alternatives considered:** the migration guides (`MIGRATION_GUIDE.md`, `MIGRATION.md`: prose sections per major, mostly behavior and options; the JavaScript one's headings aren't plain enough to read without false positives); GitHub release notes (the same text).
- **Quote:** "Removed `sentry_sdk.tracing.Span.new_span`. Use `sentry_sdk.tracing.Span.start_child` instead." (sentry-python 2.0.0).

## How the provider announces changes

A major a year or so per SDK, announced with a migration guide and a changelog section; deprecations land in minors before. sentry-python has a 3.0.0rc1 tagged: its 3.0.0 section will be read when 3.0.0 is on PyPI.

## What the plugin reads and ignores

- Reads the release's section only; a bullet counts under a breaking, removal or deprecation heading, or marked `!` (conventional commits), when its first sentence names what it removes plainly.
- Drops (and logs): removed options and integrations' settings, dropped runtime and framework versions, behavior changes, and bullets that list names in prose.

## Known gaps

- Options passed to `init()` (`with_locals`, `enableLogs`) aren't symbols: their removal isn't an event.
- A release whose section is missing at its tag is skipped as nothing to read.
