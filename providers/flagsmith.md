# Flagsmith (`flagsmith`)

## What it watches

Flagsmith's REST API (the Admin/Management API and the SDK-facing endpoints together) between releases of Flagsmith's own OpenAPI 3.1 spec: whole operations removed, and request parameters callers send that are removed, renamed or newly required.

Kind: `openapi` (`plugins/openapi`), with `adapter.Flagsmith` and a request-side filter (`openapi.Plugin.Keep`, same `requestSide` GitLab uses).

## Why this source

- **Source:** `openapi.yaml` at the root of [Flagsmith/flagsmith](https://github.com/Flagsmith/flagsmith), fetched at a release tag (`https://raw.githubusercontent.com/Flagsmith/flagsmith/<tag>/openapi.yaml`); the newest tag from `releases/latest`. Checked on 2026-11-19.
- **Why this one:** it's generated straight from the Django backend with drf-spectacular (`make generate-full-openapi`) and regenerated on every pull request that touches `api/**` by a bot that commits the result back to the PR branch (`.github/workflows/api-fix-docs-artefacts.yml`), so a release tag always carries the spec for the code it actually ships. Flagsmith tags a release on almost every merge to `main` (release-please), so two consecutive tags are usually one pull request apart.
- **Alternatives considered:**
  - **`sdk/openapi.yaml`**, the same spec filtered down to the SDK-only endpoints (`openapi-filter-flagsmith-sdk.yml`). Reading the full spec instead covers Admin/Management API removals too (the one real breaking change found, below, is an Admin-API-adjacent endpoint `sdk/openapi.yaml` would have caught as well, but there's no reason to read the narrower copy).
  - **A deprecations or sunset page.** Searched docs.flagsmith.com on 2026-11-19 and found none: the [release process page](https://docs.flagsmith.com/platform/release-process) says breaking changes "are announced in advance" and ship with migration guides, but points to GitHub Releases for the actual list, not a dedicated page.
  - **The main repo's GitHub Releases / `CHANGELOG.md`.** Release-please categorizes them (Features, Bug Fixes, ⚠ BREAKING CHANGES), but entries are prose feature summaries ("Remove the experimental flag update endpoints"), never a request parameter or symbol name in backticks: `changelog`'s classifier (built for a package's `CHANGELOG.md`) found zero matching bullets across the file's history.
  - **The Python SDK** ([flagsmith-python-client](https://github.com/Flagsmith/flagsmith-python-client), PyPI `flagsmith`) **and the JS/TS SDK** ([flagsmith-js-client](https://github.com/Flagsmith/flagsmith-js-client), npm `@flagsmith/flagsmith`, formerly `flagsmith`). Both keep a release-please `CHANGELOG.md` with "⚠ BREAKING CHANGES" sections, checked 2026-11-19: the Python client's say things like "drop Python 3.9 support" and "Restore v3 `OfflineHandler` interface"; the JS client's 3.0.4 entry ("The Edge API will no longer allow for `flagsmith.incrementTrait`, this PR removes it from the SDK") is the one bullet in either file that names a removed symbol in backticks, but its sentence shape ("will no longer allow for X, this PR removes it") doesn't match `changelog`'s plain "Removed `X`" templates either. Neither SDK has a MIGRATION.md or UPGRADING.md. A client-library plugin for either is a separate, future addition (see Known gaps).
  - **An RSS or Atom feed.** None found: Flagsmith's own blog and changelog have no feed; `docs.flagsmith.com/releases` is a static page, not a feed.
- **Quote:** `"**EXPERIMENTAL ENDPOINT** - Subject to change without notice."` (every operation removed in the replay fixture, below, carried this in its own `description`).

## How the provider announces changes

- **No API version in the path or a header.** `/api/v1/...` is the only prefix, and `v1` hasn't changed; `info.version` in the spec is a fixed `"v1"` that doesn't move between releases (checked 2026-11-19), so the event's `version` is the release tag, not `info.version`.
- **A release on almost every merge to `main`**, via release-please; the spec is regenerated and committed in the same pull request, so it's never stale relative to a tag.
- **No stated deprecation window.** The one real removal found (below) was tagged "EXPERIMENTAL ENDPOINT - Subject to change without notice" from the day it shipped and removed three releases later with no deprecation period; nothing in the spec or the release notes marks an operation `deprecated: true` before removing it.

## What the plugin reads and ignores

- **Reads** whole operations removed between two tags, and the request-side changes `core/diff/` already tracks for every `openapi`-format provider (removed or newly-required parameters and request fields).
- **Drops response-side changes**, same reason as GitLab (`requestSide`): a request body's schema is a named, shared component (`UpdateFlagV2`, `FeatureValue`), not inlined like GitLab's hash-named bodies, so the differ reports its removal as its own "schema removed" change, side `response` by default, whether or not the operations that used it are actually gone. Of the 11 changes between `v2.286.0` and `v2.287.0`, 8 are these schema-level echoes of the 3 real operation removals; the filter drops them and keeps the 3 operations.
- The replay fixture: `v2.286.0` to `v2.287.0` (2026-10-15 to 2026-10-20), where PR [#8736](https://github.com/Flagsmith/flagsmith/pull/8736) ("Remove the experimental flag update endpoints") removed `POST /api/experiments/environments/{environment_key}/delete-segment-override/`, `.../update-flag-v1/` and `.../update-flag-v2/` and their eight request-body schemas, listed under "Features" in the [v2.287.0 release](https://github.com/Flagsmith/flagsmith/releases/tag/v2.287.0). All three were marked experimental from the day they shipped; `/api/__future__/...` is where Flagsmith puts their eventual, stable replacements.

## Known gaps

- **The experimental caveat.** The one real removal this plugin has seen is of endpoints Flagsmith itself called "subject to change without notice", not a stable, long-shipped one. It's still a genuine removal a caller of those three endpoints would have hit, and the mechanism (an operation vanishing between two tagged specs) is the same one that would catch a stable endpoint's removal.
- **The Python and JS/TS SDKs aren't watched.** Their `CHANGELOG.md` files don't name removed symbols plainly enough for the `changelog` format plugin (see Alternatives considered); a client-library plugin for either would need its own reader, or a looser classifier.
- **No deprecation signal.** Nothing here reads `deprecated: true` on an operation (none exists yet in Flagsmith's spec to learn the shape from) or any advance notice; only an operation already gone becomes an event.
