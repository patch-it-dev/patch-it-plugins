# Neon

Every source patch-it reads for Neon, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Neon (`neon/api`)

### What it watches

The Neon API (`console.neon.tech/api/v2`), the management API for Neon projects, branches, computes, databases and roles: operations Neon marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Neon` ([`api.go`](api.go)).

### Why this source

- **Source:** `packages/sdk/spec/neon-openapi.json` in [neondatabase/neon-pkgs](https://github.com/neondatabase/neon-pkgs), pinned to the commit that last changed it (`https://raw.githubusercontent.com/neondatabase/neon-pkgs/<sha>/packages/sdk/spec/neon-openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:**
  - It's Neon's own spec, a copy of the one Neon publishes for its API, kept in Neon's own repo. A daily job in the repo ("SDK Spec Drift") pulls `https://neon.com/api_spec/release/v2.json` and fails when it differs from the committed copy; a maintainer then commits the new one.
  - It marks deprecations with the `deprecated` flag, and the description says why or until when.
  - There are 23 versions from 2026-06-24 to 2026-11-17, and a commit can be cited and replayed.
- **Alternatives considered:**
  - **`https://neon.com/api_spec/release/v2.json`.** The same spec served live, but it has no versions to pin, cite or replay.
  - **The npm package `@neondatabase/api-client`,** which ships the spec as YAML in each release. `@neon/sdk` succeeded it (neon-pkgs's README calls it the "successor to `@neondatabase/api-client`"), and there's been no release since 2.7.3 on 2026-07-01.
  - **The [changelog](https://neon.com/docs/changelog)** (RSS at `https://neon.com/docs/changelog/rss.xml`). It's a weekly digest of everything Neon shipped, with Postgres versions, Console features and AI models in the same post as an API endpoint's deprecation. It does name some API changes ("The deprecated `GET /consumption_history/account` endpoint has been removed.", 2026-06-19), but as prose in a digest, not something to read reliably.
  - **The SDKs** (`@neon/sdk`, `@neondatabase/serverless`, the `neon-api` Python package). Their changelogs don't list removed API operations in a form to read.
- **Quote:** "Deprecated: to test a branch's already-saved configuration, use `sendNeonAuthEmailProviderTest`" (`POST /projects/{project_id}/branches/{branch_id}/auth/send_test_email`, deprecated in the spec on 2026-08-23)

### How the provider announces changes

- **Operations are named by the path a caller sends,** `POST /api/v2/projects/…`: the spec's server is `https://console.neon.tech/api/v2` and its paths leave out `/api/v2`, so the adapter moves it into the paths (as Sentry's does with `/api/0`). Quotes cite the spec as published. If Neon ever moves `/api/v2` between the server and the paths, the operations stay the same, rather than every deprecated one reading as removed.
- **No API version beyond `/api/v2`.** `info.version` is `v2` in all 23 commits, so the event's `version` is the date of the spec commit, the closest thing to the API version a change arrived in.
- **The spec changes about once a week**, mostly new operations (144 operations on 2026-06-24, 194 on 2026-11-17).
- **Deprecations use the flag.** In all 23 commits no operation is described as deprecated without it. Two unflagged operations deprecate a parameter in words (`include_v1_metrics` on `GET /consumption_history/projects`, and `name` on `POST …/snapshots/{snapshot_id}/restore`); those are parameter deprecations, not operation ones, so they make no event. Each flagged operation also has an `x-sunset` date, and most descriptions say "Removal scheduled for <date>" or "use X instead".
- **Sunset dates aren't removal dates.** On 2026-11-17, 14 operations say "Removal scheduled for March 1, 2026" (`x-sunset` 2026-03-01) and one "Removal scheduled for July 1, 2026", and all 15 are still in the spec, the live one included (checked 2026-11-19).
- **The changelog announces API deprecations and removals in prose:** `GET /consumption_history/account` was "deprecated, with a planned sunset of June 1, 2026" (2026-04-03) and "has been removed" (2026-06-19).

### What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so Neon's reason and its `x-sunset` date are in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal. None was renamed in the history.
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** One operation left the spec in its history, unmarked: `POST /projects/{project_id}/branches/{branch_id}/recover`, on 2026-08-09. neon-pkgs's own PR for it (#408) found "The endpoint was not removed from the API. It was removed from the published spec": it still routes.
  - **`x-sunset` dates and "Removal scheduled" words:** quoted in the deprecation's excerpt, but not events of their own, since the operations outlive them.
  - **Request and response property changes, and new operations.**
- **History:** over the 23 commits this keeps 1 event: the deprecation of `POST /projects/{project_id}/branches/{branch_id}/auth/send_test_email` (beta) on 2026-08-23. The other 15 deprecated operations were already flagged in the first commit.
- **Reads** a historical pair (a version pair, `5c57d009` → `93b93dc9`) and recreates that deprecation (the golden test). The drop test replays `4c6b3be4` to `4497de89`, the `recover` removal.
- **A kept removal:** none happened in neon-pkgs's history, so a test replays the same spec from `@neondatabase/api-client` 2.7.2 to 2.7.3 (a different source, used only for this test): `GET /consumption_history/account`, deprecated and then removed, the removal Neon's changelog announced. That's one breaking event.

### Known gaps

- **Removals Neon didn't mark first aren't events,** even if the endpoint really went away.
- **A sunset date that passes isn't an event.** The 15 operations past their "Removal scheduled" date are reported once, when they were deprecated, and again only if they leave the spec.
- **Lag.** neon-pkgs commits the spec when someone acts on the drift check, days after the live spec changes. On 2026-11-19 the live spec had one operation more than the newest commit.
- **History before 2026-06-24** isn't in neon-pkgs, so deprecations and removals before then (`GET /consumption_history/account`, removed by 2026-06-19) aren't replayable from this source.
- **Only the management API is covered,** not the serverless driver (`@neondatabase/serverless`) or the Data API.
- **Two spec commits on one day are one version,** read at the day's last commit: the event's `version` is a day.
- **The patch-it client needs `neon` in its list of date-versioned APIs.** The events' `version` is a date (`2026-08-23`), because the API has no version; without the entry, the client reads it as a package version and can't place these events. That list is in the client used by patch-it's hosted ingestion service, not in this repo.

## Neon serverless driver (`neon/serverless-surface`)

### What it watches

The public API of the `@neondatabase/serverless` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@neondatabase/serverless>), each version's `dist.tarball`, read as published; the code is at <https://github.com/neondatabase/serverless>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @neondatabase/serverless@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md.
- **Quote:** `export { native } from "pg"` (`index.d.mts` line 8) at 0.10.4, gone at 1.0.0.

### How the provider announces changes

- The driver is one declaration file per module format; its `Pool` and `Client` extend `pg`'s, whose members aren't read.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@neondatabase/serverless.neon`.
- Parameters compare by position, as TypeScript passes them: a parameter gone off the end (with no rest parameter now) is removed; one optional before and required now is a `changed_default`. A renamed parameter is no change.
- Ignores names marked `@internal`, `private` and `protected` members, `#private` fields, setters, index signatures, interface and object-type members, interfaces and type aliases inside a namespace (an entry point's own are read), entry points under a private path (`_shims`, `internal`, `src`), the `./index` duplicate of the root, and what the default export holds (it's the same declaration as a named one).
- One declaration reachable at several import paths is one event, every path in its `symbols`.
- Drops (and logs) a removal that may not be one: a member of a name now re-exported from another package, now a value, an interface or a type (whose members aren't read); a method gone from a class that extends a class of another package (it may define it); a name an entry point may still get from a package it re-exports wholesale (`export * from "other"`).

### Known gaps

- What a property's type holds isn't followed: `client.chat.completions.create` is read as the `Completions` class's `create`, reached through namespaces and entry points, not through the `chat` property's type.
- Type changes (a parameter's type narrowed, a return type changed, an interface's fields) aren't events.
- A package that re-exports another wholesale (`export * from "@scope/core"`) hides that package's API: it's a source of its own, or not read.
- Classes are indexed by name across files to follow a method to its base class: two classes of the same name (in two namespaces) are read as one, which can only turn a removal into a note, never add one.
- A parameter added without being optional is a note (the change-event contract has no subject for it yet).
- Members inherited from `pg` aren't read: a method gone from `Pool` is a note (pg may define it).
