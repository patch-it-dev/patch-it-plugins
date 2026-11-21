# MongoDB

Every source patch-it reads for MongoDB, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## MongoDB (`mongodb/atlas-admin-spec`)

### What it watches

The MongoDB Atlas Administration API v2 (`cloud.mongodb.com/api/atlas/v2`): operations MongoDB marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec MongoDB publishes for it.

Kind: `openapi` (`kinds/openapi`), with its adapter, `MongoDBAtlas` ([`atlas_admin_spec.go`](atlas_admin_spec.go)). The API's own changelog is a separate source, `mongodb/atlas-admin-api` (kind `changelog`).

### Why this source

- **Source:** `openapi/v2.json` in [mongodb/openapi](https://github.com/mongodb/openapi), pinned to a commit (`https://raw.githubusercontent.com/mongodb/openapi/<sha>/openapi/v2.json`). The versions come from the GitHub commits API. Checked on 2026-11-21.
- **Why this one:**
  - It's MongoDB's own spec of the whole API, committed by its release workflow ("ci(prod): Release OpenAPI Spec") a few times a week: 768 commits from 2024-07-25 to 2026-11-19.
  - It marks deprecated operations with the `deprecated` flag.
  - Every commit can be cited and replayed.
- **Alternatives considered:**
  - **The repo's `v0.0.N` tags:** they mark the repo's tooling releases, not the spec.
  - **The per-resource-version files in `openapi/v2/`:** one file per API resource version; `v2.json` holds every operation with all its versions, which is what a caller of any version sees removed.
  - **`changelog/changelog.json`:** read by `mongodb/atlas-admin-api`; it reports changes per resource version, in prose, not removals of whole operations.
- **Quote:** `"deprecated": true` on `GET /api/atlas/v2/users/byName/{userName}` ("Returns the details for one MongoDB Cloud user account with the specified username."), new on 2025-02-20.

### How the provider announces changes

- **One version per day,** the day's last commit, its date the event's `version`: `info.version` is a fixed `2.0`, and the dated resource versions (`application/vnd.atlas.2025-03-12+json`) are per operation, several in one spec.
- **Deprecations use the flag** (53 operations on 2026-11-19). Fourteen descriptions mention "deprecated" without it, each about the deprecated invitation flow, not the operation.
- **Deprecated operations do leave:** the tenant backup operations were flagged, then removed between 2025-10 and 2026-11.
- **Operations also leave unmarked:** GET `/rest/unauth/version` on 2024-08-07.

### What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag, quoting the whole operation.
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Ignores everything else:** removals of operations never marked deprecated, schema and property changes (the changelog source reports those per resource version), new operations.
- **The golden test** replays 2025-02-18 → 2025-02-20, when 14 user-management operations were deprecated at once; three are saved, three events. **The drop test** replays 2024-07-25 → 2024-08-07: GET `/rest/unauth/version` gone unmarked, no event.

### Known gaps

- **Deprecated resource versions** (an operation's `2023-01-01` content type sunset while newer ones stay) aren't events: only whole operations are. The changelog source covers them.
- **Removals MongoDB didn't flag first aren't events.**
- **History starts on 2024-07-25,** when `openapi/v2.json` was first committed.
- **License:** the spec is CC BY-NC-SA 3.0 US; only small excerpts are saved for tests.

## MongoDB Atlas (`mongodb/atlas-admin-api`)

### What it watches

The MongoDB Atlas Administration API (`/api/atlas/v2`): an endpoint's API version deprecated (unless MongoDB reactivates it), and an endpoint's API version removed at its sunset date, as MongoDB lists them in its own machine-readable API changelog.

Kind: `changelog` (`kinds/changelog`), with its JSON parser (`kinds/changelog/json.go`) and the `AtlasAdminAPI` log.

### Why this source

- **Source:** `changelog/changelog.json` in [mongodb/openapi](https://github.com/mongodb/openapi), read at the commit that last changed it (`https://raw.githubusercontent.com/mongodb/openapi/<sha>/changelog/changelog.json`). The commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:** it's MongoDB's own record of what changed, released by MongoDB's CI ("Release API Changelog") a few times a month. It's also what the [API changelog docs page](https://www.mongodb.com/docs/atlas/reference/api-resources-spec/changelog/) shows.
  - It covers 188 days from 2023-02-01 to 2026-11-18. Each day lists every endpoint that changed, per API version, with a change code from oasdiff, a sentence, and whether the change is backward compatible.
  - The life-cycle entries say what happens and when: "Resource version 2023-01-01 deprecated and marked for removal on 2025-06-01", "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'".
- **Alternatives considered:**
  - **The per-version OpenAPI specs in the same repo** (`openapi/v2/openapi-<version>.json`). The changelog is MongoDB's own diff of them, already classified, so diffing them again would only re-derive it.
  - **The docs changelog page.** It's built from the same data, as HTML.
  - **The drivers' changelogs** (PyMongo, the Node.js driver). They're a different surface, the database driver, not this API, and a separate plugin if added.
- **Quote:** "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" (the shared-tier backup endpoints, 2026-01-22)

### How the provider announces changes

- **Versioning by date, per resource.** A caller picks the version per request in the `Accept` header (`application/vnd.atlas.2023-01-01+json`). Each API version of an endpoint is deprecated when a newer one ships, and removed at a sunset date, usually a year or more later.
- **The changelog gets a release a few times a month.** It holds 22,273 changes, 481 of them marked not backward compatible.

### What the plugin reads and ignores

- **Reads** the change codes about a whole endpoint at one API version, replaying the days oldest first:
  - `endpoint-deprecated` and `resource-version-deprecated` become **`deprecation`** events, unless a later `endpoint-reactivated` withdraws them. MongoDB has reactivated 13 times. For example, `GET /api/atlas/v2/groups/{groupId}/clusters` at `2024-05-30` was deprecated on 2024-06-25 and reactivated the next day.
  - `endpoint-removed` becomes a **`breaking`** event, unless the same endpoint and version shows up again on a later day (none has): then the removal goes, and so does any deprecation before it. A removal announced twice counts from its first announcement. All 38 in the file say the version "has reached its sunset date".
  - **`version`** is the API version the change is about (`2023-01-01`): the one a caller's `Accept` header names.
  - A deprecation counts from its first announcement since the last reactivation: the changelog repeats some.
- **Cites** the file at its commit, with a JSON pointer to the change and the change itself as the excerpt: every field MongoDB wrote, in its order, whitespace aside.
- **Ignores** every other change code. Most are additions. The other "not backward compatible" ones are property-level, and the flag is broad: a response enum gaining a value counts (68 times). Removed or newly required request properties are probably real breaks, but they aren't read until measured.
- **Not read:** `api-path-removed-without-deprecation`. Its one entry (`GET …/streams/activeVpcPeeringConnections` at `2024-11-13`, 2025-01-10) comes with `endpoint-added` the same day, and the endpoint changes again later, so it wasn't removed. Also not read: `sunset-deleted`, whose one entry keeps the endpoint deprecated.
- On the changelog released 2026-11-18 this gives 81 events: 43 deprecations still standing and 38 removals.

### Known gaps

- **Property-level breaking changes aren't events.** Examples are a request property removed (4), made required (11) or added as required (3), or an enum value no longer accepted (36), even where MongoDB marks them not backward compatible.
- **The changelog doesn't always announce a deprecation before a removal.** None of the 38 removals had an `endpoint-deprecated` or `resource-version-deprecated` entry for the same endpoint and version in the changelog, so some removals arrive without an earlier deprecation event. The removal entries do say the sunset date was reached.
- **An event stored before it's withdrawn stays stored:** a deprecation before its reactivation, or a removal before its version comes back. The plugin doesn't emit it again, but nothing withdraws an event already in the feed.
- **The drivers aren't covered,** and they're what most application code uses.
- **The patch-it client can't place these events yet.** Their `version` is a dated API version (`2023-01-01`), chosen per request in an `Accept` header, a scheme the client's version check doesn't know.

## MongoDB Node.js driver (`mongodb/node-surface`)

### What it watches

The public API of the `mongodb` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/mongodb>), each version's `dist.tarball`, read as published; the code is at <https://github.com/mongodb/node-mongodb-native>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install mongodb@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** HISTORY.md; the upgrade guides (majors only).
- **Quote:** `interface AsyncDisposable_2` (exported as `AsyncDisposable`, `mongodb.d.ts` line 613) at 6.22.0, gone at 7.0.0.

### How the provider announces changes

- The driver ships one declaration file (`mongodb.d.ts`, an API Extractor rollup); majors remove what the previous one deprecated.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `mongodb.MongoClient.connect`.
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

## PyMongo (`mongodb/pymongo-surface`)

### What it watches

The public API of the `pymongo` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/pymongo/>), each release's wheel read as published; the code is at <https://github.com/mongodb/mongo-python-driver>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install pymongo==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the changelog in the docs (`doc/changelog.rst`); the migration guides (majors only).
- **Quote:** `explicit_session: bool = False,` in `AsyncCommandCursor.__init__` (`pymongo/asynchronous/command_cursor.py`) at 4.15.5, gone at 4.16.0.

### How the provider announces changes

- The driver releases a minor every few months; constructors' parameters change in minors.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `pymongo/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
- Ignores private names: a leading underscore, a module under a private one or under one named `internal` or `vendor`, a name left out of `__all__`, and tests. Also classes nested in classes, a property's parameters (and its setter), and `self`/`cls` (callers never pass them).
- Drops (and logs): an overloaded function's parameters; a removed parameter when the function now takes `**kwargs`; a new parameter without a default (the change-event contract has no subject for it yet).
- A removal is a rename only when the same scope loses one and gains one of the same kind with identical parameters, at least one of them besides `self`/`cls`; anything less sure is a removal.
- Drops (and logs) a removal that may not be one: a name its module still binds in the newer release (an alias such as `DeadlineExceededError = GatewayTimeoutError`, an import from a private module, a star import from outside the package, a module-level `__getattr__` that hands out old names); a symbol whose file the newer release has too big to read (`MaxFileBytes`); a method whose class is still there but whose bases changed, which one of its bases now defines, or which the class now defines as an attribute.

### Known gaps

- A positional parameter renamed reads as the old one removed: a call that passes it by keyword breaks, one that passes it by position doesn't.
- A class with no `__init__` of its own (generated pydantic or dataclass-like types) has no parameters to compare, so its rename reads as a removal of the old name, and its fields aren't compared.
- A module that star-imports from outside the package, or defines `__getattr__`, hides every removal from it: those are notes, not events.
- Type changes (an annotation narrowed, a return type changed) aren't events.
- The wheel read is the first one by file name (a `-none-any` wheel when there is one); a release whose platform wheels differ in their Python files is read from that one.
- Wheels are per platform; the first by file name is read (the Python files are the same in each).
