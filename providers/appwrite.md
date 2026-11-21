# Appwrite (`appwrite/api`)

## What it watches

The Appwrite REST API as a server SDK calls it (the operations an Appwrite server SDK, in a repo's backend code, can call): operations Appwrite marks deprecated between two commits of its OpenAPI spec. SDK method renames that don't deprecate the route are left out, and so are removals.

Kind: `openapi` (`plugins/openapi`), with `adapter.Appwrite`.

## Why this source

- **Source:** [appwrite/specs](https://github.com/appwrite/specs) (BSD-3-Clause), the newest minor's spec at the newest commit that changed `specs/`: `https://raw.githubusercontent.com/appwrite/specs/<sha>/specs/<X.Y>.x/open-api3-<X.Y>.x.json`, or `open-api3-<X.Y>.x-server.json` in a directory without the single document. The newest minor is the highest `<X.Y>.x` directory by number (`specs/latest` is a copy and is skipped). Checked on 2026-11-19.
- **Why this one:**
  - It's Appwrite's own description of the API, one spec per server minor, published by Appwrite. Its API reference reads it: "The Console's Cloud API reference reads the newest numbered spec" (commit `06e6fafa`).
  - It marks deprecations twice over: the `deprecated` flag, and `x-appwrite.deprecated` with what to use instead (`{"since": "1.8.0", "replaceWith": "tablesDB.createTransaction"}`).
  - Each commit can be cited and replayed: 139 commits on `main` from 2026-02-17 to 2026-11-17.
- **Alternatives considered:**
  - **`app/config/specs` in [appwrite/appwrite](https://github.com/appwrite/appwrite).** The same specs, but they stopped there: commit `8d82bc89` (2026-02-18) removed them from the repo ("remove specs from git tracking"), and the last release tag with them is `1.8.1`.
  - **The SDK changelogs** (`CHANGELOG.md` in `appwrite/sdk-for-node` and the others). They list real removals ("* Breaking: removed `account.listLogs`, `users.listLogs` ..."), but per SDK and in SDK versions, not API operations: a possible later `changelog` plugin, not this one.
  - **No deprecations page:** `https://appwrite.io/docs/deprecations` redirects to the quick starts.
  - **The [changelog](https://appwrite.io/changelog)** (RSS at `https://appwrite.io/changelog/rss.xml`): its 50 posts from 2026-09-30 to 2026-11-19 never say "deprecated" or "breaking". Product news.
- **Quote:** `"deprecated": {"since": "1.8.0", "replaceWith": "tablesDB.createTransaction"}` (`POST /databases/transactions`, flagged in the 1.9.x server spec on 2026-06-24)

## How the provider announces changes

- **One spec per server minor**, `specs/<X.Y>.x/`, updated on `main` many times per minor. No tags or releases in the repo, so a commit SHA pins a spec. Commits don't line up with Appwrite's releases: on 2026-11-19 the newest spec says `2.7.2` while appwrite/appwrite has released `2.7.4`.
- **The layout changed once.** Up to 1.9.x a directory has one document per platform (`-client`, `-console`, `-server`); from 2.0.x there's one document, each operation listing its platforms in `x-appwrite.platforms` (2.3.x and 2.4.x also have the per-platform files). The adapter reads the single document when there is one and keeps the operations on `server`. At `0ccd20ab` (1.9.x and 2.0.x) and `98cc1a25` (2.4.x's two) that gives the operations the server document has. Where they part, the single document is the one kept up: `06e6fafa` changed 2.4.x's single document only.
- **The `deprecated` flag marks two things.** A route Appwrite wants callers off (the legacy Databases API, replaced by TablesDB in 1.8.0). Or an SDK method name renamed while the route stays: `POST /account/mfa/challenges` is flagged with `"replaceWith": "account.createMFAChallenge"`, which is the same operation's other name in `x-appwrite.methods` (`createMfaChallenge` became `createMFAChallenge`). 23 of the 99 operations flagged in 2.7.x's server operations are renames. All 23 have been flagged since 1.8.0, and 1.8.x's server spec at `ac705cd3` has the same 23.
- **`since` isn't a date and isn't when the flag came.** The 14 flags added on 2026-06-24 all say `"since": "1.8.0"`, in a spec whose `info.version` was 1.9.5. It's quoted, not used.
- **The event's `version` is the date of the spec commit** (UTC, `2026-06-24`), not the spec's `info.version` (the server version, `1.9.5`). A repo pins an SDK, and the SDKs are versioned on their own: node-appwrite 32.0.0 and the Python SDK 28.0.0 both send `X-Appwrite-Response-Format: 2.7.2`. A server version can't be placed against that pin, and spec commits don't line up with server releases anyway. The server version is in the event's source, the spec at that commit.
- **Every server URL ends in `/v1`** (`https://cloud.appwrite.io/v1` and the regional `https://{region}.cloud.appwrite.io/v1`, `<REGION>` up to 1.8.x), at every commit read; the paths don't repeat it. The adapter moves `/v1` into the paths (`adapter.ServerBasePath`), so an operation is named by the path a caller sends, `POST /v1/databases/transactions`; each event's `source_pointer` names the path as published.

## What the plugin reads and ignores

- **Reads:** the newest minor's server operations at the newest commit, diffed against the last ones seen. One kind of event comes out: **deprecations**, an operation that gains the `deprecated` flag. The event quotes `x-appwrite.deprecated` (since and replaceWith), which is where the spec says what to use instead; descriptions don't say. Quoting the whole operation could lose it to the 2,000-character cut, because `x-appwrite` sorts last.
- **Matches operations by path template**, as the other specs do.
- **Ignores:**
  - **SDK method deprecations.** A flag whose `replaceWith` names a method of the same operation's `x-appwrite.methods`, or on an operation with a method in `x-appwrite.methods` that isn't itself deprecated (a live SDK method remains), isn't read as a deprecation (`diff.Options.FlagNotDeprecation`). An operation with no methods listed (`"methods": []`, as the 14 below) is read by its flag. In 2.7.x the two rules pick the same 23 operations. If such an operation is later deprecated for another operation, that's an event then. From 1.7.x to 1.8.x (both at `ac705cd3`) this leaves out 21 of the 71 new flags. The other 50 are the Databases API's deprecations in favor of TablesDB.
  - **Removals, all of them,** even of deprecated operations. Read as the plugin reads it (139 commits on `main`), 78 operations left the server operations (from the spec, or from the `server` platform). Only one had been flagged first, and it was a path move: `PATCH .../attributes/{key}/relationship` became `.../attributes/relationship/{key}` on 2026-03-20. 13 are back now: on 2026-10-12, `06e6fafa` took `server` off 13 operations of 2.4.x's single document ("listed server and client methods no released SDK ships yet"), and 2.5.x had them on `server` again the next day. Nothing in the spec says when an operation stops answering.
  - **Request and response property changes, and new operations.**
- Over the 139 commits this keeps 14 events, all on 2026-06-24 (`7d74bf9f`): the six `/v1/databases/transactions` operations and the `longtext`, `mediumtext`, `text` and `varchar` attribute operations, each replaced by a `tablesDB` method. None is a rename or has a live method, so the filter drops none of them.
- **Reads** a pair of commits (a version pair, as the golden test does from `8c00ba35` to `7d74bf9f`). A tag can name an older minor's directory, `1.7.x@<sha>`, as the rename test does. The pinned drop test is the 2.4.x pair (13 removals, no event). A single document where no operation is on `server` (the platforms gone or renamed) fails the fetch, rather than read as nothing deprecated.

## Known gaps

- **Removals aren't events.** The spec doesn't mark them first, so real ones are missed. The SDK changelogs say them ("* Breaking: removed `X`"); a `changelog` plugin per SDK could read them.
- **Deprecations from before the plugin's first run aren't events,** nor operations that first appear already flagged. When the newest minor changes, the plugin diffs the new directory against the previous minor's last spec, so an operation that was unflagged there and is flagged in the new one is an event. The Databases to TablesDB deprecations of 1.8.0 can be replayed (`1.7.x@<sha>` to `<sha>`), and a run never sees them.
- **The layout changed once (2.0.x) and may again.** The adapter fails, rather than guess, when a directory has neither file.
- **Spec commits aren't releases.** A deprecation's `version` is the day the spec changed, which can trail the release. Two commits on one day are one version, read at the day's last commit.
- **Client and console operations aren't watched.** Only what a server SDK can call.
- **The patch-it client needs `appwrite` in its list of date-versioned APIs,** as for `neon`, `convex` and `planetscale`. Without it, it reads the date as a package version and drops the event.
