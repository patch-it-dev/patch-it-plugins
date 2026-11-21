# Stripe (`stripe/api`)

## What it watches

Stripe's REST API, release by release: endpoints, parameters, fields, schemas and enum values that are removed, renamed, retyped or added between two versions of Stripe's own OpenAPI spec. Breaking events carry Stripe's own explanation from its API changelog as the migration hint when the changelog has one.

Kind: `openapi` (`plugins/openapi`), with `adapter.Stripe` and the changelog hint source in `hint/`.

## Why this source

- **Source:** [github.com/stripe/openapi](https://github.com/stripe/openapi), the file `latest/openapi.spec3.json` pinned to a release tag (`https://raw.githubusercontent.com/stripe/openapi/<tag>/latest/openapi.spec3.json`); the newest tag from `GET https://api.github.com/repos/stripe/openapi/releases/latest`. Migration hints from [docs.stripe.com/changelog](https://docs.stripe.com/changelog), read as markdown (each page's URL plus `.md`). Checked on 2026-10-13.
- **Why this one:** it's Stripe's own machine-readable description of the API, maintained by Stripe and published on every release. Two specs can be diffed exactly, with no guessing: a removed field is a removed field. Stripe recommends `latest/openapi.spec3.json` (v1 and v2 endpoints).
- **Alternatives considered:** the API changelog alone (human-written, one entry per change; precise about *why* but not exhaustive at the field level, so it supplies hints, not events); `openapi/spec3.json` (legacy, v1 only; used only for tags older than the `latest/` directory, such as `v207` in 2022); the `*.sdk.*` variants (include internal annotations and pre-release features); `preview/` (public-preview endpoints, not stable).
- **Quote:** "Removes the `charges` attribute from the `PaymentIntent` object" ([changelog, 2022-11-15](https://docs.stripe.com/changelog/2022-11-15), listed as Breaking).

## How the provider announces changes

- Versions are named `YYYY-MM-DD.<release>` (`2026-09-30.endive`). Major releases include backward-incompatible changes; monthly releases only backward-compatible ones.
- **Breaking changes are opt-in.** An account has a default API version, and a request can override it with the `Stripe-Version` header; old versions keep working. A breaking change reaches code when someone upgrades the account version or the SDK. Modern SDKs pin the API version they were released with (stripe-node 12+, stripe-python 6+, stripe-ruby 9+, stripe-php 11+; Java, Go and .NET always did).
- The spec repo tags releases (`vNNNN`) several times a day, but most tags don't change the spec: the plugin polls by tag and compares by content hash, so most runs find nothing.
- The event's `version` is the spec's `info.version` (`2026-09-30.endive`).

## What the plugin reads and ignores

- Reads: the whole spec at the newest tag, diffed against the last one it saw (`core/diff/`, `core/normalize/`). `$ref`s are never followed: a change inside a shared schema is reported once, on that schema, not once per endpoint that uses it.
- Hints: for each breaking event, the changelog entry for that version whose "REST API" table names the same parameter and operation; the entry's "What's new" text and link become `migration_hint`. An entry describing an addition never explains a removal, and the reverse.
- Reads a historical pair (a version pair, `v207` → `v208`) and recreates the 2022-11-15 change (38 events, 36 breaking).
- `latest/openapi.spec3.json` can lag a version bump by several tags; the plugin follows the file, not the release notes.

## Known gaps

- Changes the spec doesn't express (behavior changes, webhook payload semantics) are only in the changelog, which isn't read for events.
- Enum value removals and some tax-related changes have no parameter row in the changelog, so those events keep the normalizer's one-line hint.
- Webhooks use the version set when the endpoint was created; the events don't say which webhook endpoints are affected.
