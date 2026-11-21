# Convex

Every source patch-it reads for Convex, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Convex (`convex/deployment-api`)

### What it watches

The Convex Deployment API (`{deployment-url}/api/v1`, the platform API for managing one deployment: environment variables, log streams and integrations, usage limits, data sync): operations Convex marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Convex` ([`deployment_api.go`](deployment_api.go)).

### Why this source

- **Source:** `npm-packages/@convex-dev/platform/deployment-openapi.json` in [convex/convex-backend](https://github.com/convex/convex-backend) (the org was `get-convex` until 2026; GitHub redirects the old URLs, checked 2026-11-20), pinned to the commit that last changed it (`https://raw.githubusercontent.com/convex/convex-backend/<sha>/npm-packages/@convex-dev/platform/deployment-openapi.json`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why a commit, not a release tag:** the repo tags `precompiled-YYYY-MM-DD-<sha>` builds (1,213 of them), but those are daily builds of the self-hosted backend, not releases of the API. There are several on some days, ordered only by commit date, and the spec is the same in most of them. The spec at a tag is the spec at the last commit before it that changed the file, so the commits see every version a tag would, and finding the newest takes one call instead of paging through the tags. A commit never changes, so it can be cited and replayed.
- **Why this one:**
  - It's Convex's own description of the API: its docs build renders the API reference from it.
  - It marks deprecations with the `deprecated` flag and says what to use instead ("Deprecated: use `GET /integrations` instead."). Convex's docs build drops flagged operations, "since they stay served only for existing callers" (`npm-packages/docs/scripts/prepare-openapi-specs.js`).
  - There are 68 versions from 2025-08-19, when the spec was added, to 2026-11-16.
- **Alternatives considered:**
  - **The Management API spec** (`management-openapi.json`, next to it: teams, projects, deployments). Its `info.license` is `LicenseRef-Convex`, which nothing in the repo defines, so the plugin neither reads nor saves it.
  - **The SDK changelogs** (`convex` on npm, `convex/convex-js` `CHANGELOG.md`; `convex-py`). One flat list of bullets per version (95 versions in `convex-js`), with no removed or deprecated sections for the `changelog` plugin to read.
  - **[Convex News](https://news.convex.dev/)** (RSS at `https://news.convex.dev/rss/`). Product news and announcements, newest post 2026-09-21; it doesn't name API operations.
  - **The `@convex-dev/platform` npm package** (0.1.11). Its tarball has only `package.json` and the generated client, not the specs.
- **Quote:** "Deprecated: use `GET /integrations` instead." (`GET /api/v1/list_log_streams`, deprecated in the spec on 2026-11-05)

### How the provider announces changes

- **No API version beyond `/api/v1`.** `info.version` is `1.0.0` in every commit, so the event's `version` is the spec commit's committer date (UTC), the closest thing to the API version a change arrived in. The patch-it client keeps a date-versioned API's events only if it's in its list of date-versioned APIs, so `convex` needs to be in it.
- **The spec changes a few times a month,** mostly with new operations and properties as Convex builds out the platform API.
- **Deprecations are rare:** one so far. On 2026-10-30 (`f35879bb`, "Rename log_streaming to integrations") Convex added six `/integrations` operations; on 2026-11-05 (`5e5159cb`) it marked the six log stream operations deprecated, each naming its replacement. The commit says "The routes still work, so existing callers and older dashboards keep working."

### What the plugin reads and ignores

- **Reads:** the spec at the newest commit, diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so the replacement Convex names is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated. None yet.
- **Operations are named by the path a caller sends,** `GET /api/v1/list_log_streams`: the server URL is `{deployment-url}/api/v1` (since the first commit) and the spec's paths leave out `/api/v1`, so the adapter moves it into the paths (as Sentry's does with `/api/0`). Quotes cite the spec as published. If Convex ever moves `/api/v1` between the server and the paths, the operations stay the same, rather than every deprecated one reading as removed.
- **Matches operations by path template,** as for every spec here. No path parameter has been renamed in this spec yet.
- **Ignores everything else:**
  - **Request and response property changes,** and new operations. Over the history the differ finds 18 property removals, 16 type changes, 10 schema removals and 3 properties no longer required, most of them in the data sync, usage-limit and S3 export APIs. The drop test is one of them: on 2026-07-14 (`3ead9091`, "Remove name from usage limits") the optional `name` left the usage-limit request and response; the commit says it "was never editable in the UI (always `None` on create, only echoed back)", and the request type at that commit doesn't reject unknown fields, so a caller still sending it isn't refused.
  - **Removals of operations that were never deprecated.** None in the history: no operation has left the spec.
- Over the 68 commits this keeps 6 events, all deprecations on 2026-11-05: `GET /api/v1/list_log_streams`, `GET /api/v1/get_log_stream/{id}`, `POST /api/v1/create_log_stream`, `POST /api/v1/update_log_stream/{id}`, `POST /api/v1/delete_log_stream/{id}`, `POST /api/v1/rotate_webhook_secret/{id}`.
- **Reads** a historical pair (a version pair, `f35879bb` → `5e5159cb`) and recreates those six. The drop test replays `3f6edf7a` to `3ead9091`.

### Known gaps

- **Only the Deployment API is covered.** Not the Management API (see above), the client SDKs (`convex` on npm, `convex-py`), or the functions runtime. SDK removals are written as plain bullets in their changelogs, so nothing reads them yet.
- **Removals Convex doesn't mark first aren't events,** even if the endpoint really went away. None has happened yet.
- **Two spec commits on one day are one version** (13 days in the history), read at the day's last commit: a change made and undone the same day isn't seen.
- **The patch-it client can't place these events until `convex` is in its list of date-versioned APIs.**

## Convex JavaScript client (`convex/js-surface`)

### What it watches

The public API of the `convex` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/convex>), each version's `dist.tarball`, read as published; the code is at <https://github.com/convex/convex-js>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install convex@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md.
- **Quote:** `type QueryOptions<Query extends FunctionReference<"query">>` (`dist/esm-types/browser/query_options.d.ts` line 9) at 1.35.0, gone at 1.35.1.

### How the provider announces changes

- Has many entry points (`convex/browser`, `convex/react`, `convex/server`, …), released every week or two.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `convex/react.useQuery`.
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
