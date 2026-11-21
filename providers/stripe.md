# Stripe (`stripe/api`)

## What it watches

Stripe's REST API, API version by API version (`2026-10-28.endive` → `2026-11-18.endive`): endpoints, parameters, fields, schemas and enum values that are removed, renamed, retyped or added between two versions of Stripe's own OpenAPI spec. Breaking events carry Stripe's own explanation from its API changelog as the migration hint when the changelog has one.

Kind: `openapi` (`plugins/openapi`), with `adapter.Stripe` and the changelog hint source in `hint/`.

## Why this source

- **Source:** [github.com/stripe/openapi](https://github.com/stripe/openapi), the file `latest/openapi.spec3.json` pinned to a release tag (`https://raw.githubusercontent.com/stripe/openapi/<tag>/latest/openapi.spec3.json`); the API versions from the releases' notes, `GET https://api.github.com/repos/stripe/openapi/releases` (checked 2026-11-20). Migration hints from [docs.stripe.com/changelog](https://docs.stripe.com/changelog), read as markdown (each page's URL plus `.md`). Checked on 2026-10-13.
- **Why this one:** it's Stripe's own machine-readable description of the API, maintained by Stripe and published on every release. Two specs can be diffed exactly, with no guessing: a removed field is a removed field. Stripe recommends `latest/openapi.spec3.json` (v1 and v2 endpoints).
- **Alternatives considered:** the API changelog alone (human-written, one entry per change; precise about *why* but not exhaustive at the field level, so it supplies hints, not events); `openapi/spec3.json` (legacy, v1 only); the `*.sdk.*` variants (include internal annotations and pre-release features); `preview/` (public-preview endpoints, not stable).
- **Quote:** "Removes the `charges` attribute from the `PaymentIntent` object" ([changelog, 2022-11-15](https://docs.stripe.com/changelog/2022-11-15), listed as Breaking).

## How the provider announces changes

- Versions are named `YYYY-MM-DD.<release>` (`2026-09-30.endive`). Major releases include backward-incompatible changes; monthly releases only backward-compatible ones.
- **Breaking changes are opt-in.** An account has a default API version, and a request can override it with the `Stripe-Version` header; old versions keep working. A breaking change reaches code when someone upgrades the account version or the SDK. Modern SDKs pin the API version they were released with (stripe-node 12+, stripe-python 6+, stripe-ruby 9+, stripe-php 11+; Java, Go and .NET always did).
- The spec repo tags releases (`vNNNN`) several times a day, many per API version. Since v2191 (2026-03-11) each release's notes name the API version its SDK spec carries, in a table: `| openapi/spec3.sdk.{json,yaml} | 2026-11-18.endive |`. That is the version Stripe's SDKs pin ("This release changes the pinned API version to `2026-11-18.endive`").
- **Versions are API versions.** The listed versions are those named in the notes, oldest first, each read at the newest release that names it and dated by the first one. The SDK changelog sources' `MapsTo` pins (`providers/stripe-sdks.md`) name the same ids, so a feed query on an SDK's versions finds this source's changes between the two API versions. A pair is one API version to the next. The event's `version` and `previous_version` are those ids, which are also the spec's `info.version`.

## What the plugin reads and ignores

- Reads: the whole spec at each API version's newest release, diffed against the version before's (`core/diff/`, `core/normalize/`). `$ref`s are never followed: a change inside a shared schema is reported once, on that schema, not once per endpoint that uses it.
- Hints: for each breaking event, the changelog entry for that version whose "REST API" table names the same parameter and operation; the entry's "What's new" text and link become `migration_hint`. An entry describing an addition never explains a removal, and the reverse.
- `latest/openapi.spec3.json` can lag a version bump by several releases (at v2602 the notes name `2026-11-18.endive` while the file still declares `2026-10-28.endive`). A spec that doesn't declare the listed version is an error, never read as that version. When a later release's file catches up, it also names the version in its notes, so the version is read there on a later run.

## Known gaps

- Releases before v2191 (2026-03-11) don't name their API version, so versions older than `2026-02-25.clover` aren't listed: naming them would mean reading every old spec or guessing.
- While a version is the newest, the release it's read at moves as Stripe publishes more of it. The pair into it is compared once, at the release current then.
- If Stripe ever moved its SDK spec on to the next version before `latest/openapi.spec3.json` had declared the current one, that version's release would stop moving and it would fail to read on every run, loudly. It hasn't happened: at each listed version's release (v2203 … v2600) the file declares that version (checked 2026-11-21); v2602 is the lag described above.
- Changes the spec doesn't express (behavior changes, webhook payload semantics) are only in the changelog, which isn't read for events.
- Enum value removals and some tax-related changes have no parameter row in the changelog, so those events keep the normalizer's one-line hint.
- Webhooks use the version set when the endpoint was created; the events don't say which webhook endpoints are affected.
