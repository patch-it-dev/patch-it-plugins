# Stripe

Every source patch-it reads for Stripe, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Stripe (`stripe/api`)

### What it watches

Stripe's REST API, API version by API version (`2026-10-28.endive` → `2026-11-18.endive`): endpoints, parameters, fields, schemas and enum values that are removed, renamed, retyped or added between two versions of Stripe's own OpenAPI spec, and what Stripe adds inside a version after naming it (`added` events with `previous_version` equal to `version`). Breaking events carry Stripe's own explanation from its API changelog as the migration hint when the changelog has one.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Stripe` ([`api.go`](api.go)) and the changelog hint source in `core/hint`.

### Why this source

- **Source:** [github.com/stripe/openapi](https://github.com/stripe/openapi), the file `latest/openapi.spec3.json` pinned to a release tag (`https://raw.githubusercontent.com/stripe/openapi/<tag>/latest/openapi.spec3.json`); the API versions from the releases' notes, `GET https://api.github.com/repos/stripe/openapi/releases` (checked 2026-11-20). Migration hints from [docs.stripe.com/changelog](https://docs.stripe.com/changelog), read as markdown (each page's URL plus `.md`). Checked on 2026-10-13.
- **Why this one:** it's Stripe's own machine-readable description of the API, maintained by Stripe and published on every release. Two specs can be diffed exactly, with no guessing: a removed field is a removed field. Stripe recommends `latest/openapi.spec3.json` (v1 and v2 endpoints).
- **Alternatives considered:** the API changelog alone (human-written, one entry per change; precise about *why* but not exhaustive at the field level, so it supplies hints, not events); `openapi/spec3.json` (legacy, v1 only); the `*.sdk.*` variants (include internal annotations and pre-release features); `preview/` (public-preview endpoints, not stable).
- **Quote:** "Removes the `charges` attribute from the `PaymentIntent` object" ([changelog, 2022-11-15](https://docs.stripe.com/changelog/2022-11-15), listed as Breaking).

### How the provider announces changes

- Versions are named `YYYY-MM-DD.<release>` (`2026-09-30.endive`). Major releases include backward-incompatible changes; monthly releases only backward-compatible ones.
- **Breaking changes are opt-in.** An account has a default API version, and a request can override it with the `Stripe-Version` header; old versions keep working. A breaking change reaches code when someone upgrades the account version or the SDK. Modern SDKs pin the API version they were released with (stripe-node 12+, stripe-python 6+, stripe-ruby 9+, stripe-php 11+; Java, Go and .NET always did).
- The spec repo tags releases (`vNNNN`) several times a day, many per API version. Since v2191 (2026-03-11) each release's notes name the API version its SDK spec carries, in a table: `| openapi/spec3.sdk.{json,yaml} | 2026-11-18.endive |`. That is the version Stripe's SDKs pin ("This release changes the pinned API version to `2026-11-18.endive`").
- **Versions are API versions.** The listed versions are those named in the notes, oldest first, each read at the newest release that names it and dated by the first one. The SDK changelog sources' `MapsTo` pins (below) name the same ids, so a feed query on an SDK's versions finds this source's changes between the two API versions. A pair is one API version to the next. The event's `version` and `previous_version` are those ids, which are also the spec's `info.version`.

### What the plugin reads and ignores

- Reads: the whole spec at each API version's newest release and at the first release whose `latest/openapi.spec3.json` declares it (`Releases`, `adapter.Rereleased`). The change into a version is the version before's newest release against this version's first: what Stripe added later inside the version isn't filed with it. The version's first release against its newest is the additions inside the version, filed under the version itself (`previous_version` = `version`), additions only: a named version only gains things, so a removal there is a correction, not a release. Measured on 2026-09-30.endive (v2525 → v2592, checked 2026-11-21): two enum values added to `subscriptions_resource_paused_status_details_subscription.type`, nothing removed (`TestStripeAdditionsInsideAVersionGolden`). (`core/diff/`, `core/normalize/`) `$ref`s are never followed: a change inside a shared schema is reported once, on that schema, not once per endpoint that uses it.
- Hints: for each breaking event, the changelog entry for that version whose "REST API" table names the same parameter and operation; the entry's "What's new" text and link become `migration_hint`. An entry describing an addition never explains a removal, and the reverse.
- `latest/openapi.spec3.json` can lag a version bump by several releases (at v2602 the notes name `2026-11-18.endive` while the file still declares `2026-10-28.endive`). A spec that doesn't declare the listed version is an error, never read as that version. When a later release's file catches up, it also names the version in its notes, so the version is read there on a later run.

### Known gaps

- Releases before v2191 (2026-03-11) don't name their API version, so versions older than `2026-02-25.clover` aren't listed: naming them would mean reading every old spec or guessing.
- While a version is the newest, the release it's read at moves as Stripe publishes more of it. The pair into it is compared once, at the release current then; what the version gains after that is compared by the hosted ingestion service as a pair of two releases of the version (additions only) when it sees the newest release move.
- If Stripe ever moved its SDK spec on to the next version before `latest/openapi.spec3.json` had declared the current one, that version's release would stop moving and it would fail to read on every run, loudly. It hasn't happened: at each listed version's release (v2203 … v2600) the file declares that version (checked 2026-11-21); v2602 is the lag described above.
- Changes the spec doesn't express (behavior changes, webhook payload semantics) are only in the changelog, which isn't read for events.
- Enum value removals and some tax-related changes have no parameter row in the changelog, so those events keep the normalizer's one-line hint.
- Webhooks use the version set when the endpoint was created; the events don't say which webhook endpoints are affected.

## Stripe Node.js SDK (`stripe/node-surface`)

### What it watches

The public API of the `stripe` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/stripe>), each version's `dist.tarball`, read as published; the code is at <https://github.com/stripe/stripe-node>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install stripe@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md ("⚠️" on breaking bullets); the API spec (`stripe/api`).
- **Quote:** `namespace ErrorType` (`esm/stripe.esm.node.d.ts` line 482) at 22.6.2, gone at 23.0.0.

### How the provider announces changes

- Each major pins a new Stripe API version and removes what that version removed.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `stripe.Stripe.constructEventWithoutVerification`.
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
- The `Stripe` namespace's thousands of resource types are types inside a namespace: not read (the API spec covers the fields).

## Stripe Python SDK (`stripe/python-surface`)

### What it watches

The public API of the `stripe` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/stripe/>), each release's wheel read as published; the code is at <https://github.com/stripe/stripe-python>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install stripe==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (prose per release, "⚠️" on breaking bullets); the API spec (`stripe/api`: the HTTP API, not Python names).
- **Quote:** `def request( self, method: Literal["get", "post", "delete"], url: str, params: Optional[Dict[str, Any]] = None, *, base_address: BaseAddress = "api", ) -> "StripeObject"` (`StripeObject`, `stripe/_stripe_object.py` line 452) at 15.6.2, gone at 16.0.0.

### How the provider announces changes

- Each major pins a new Stripe API version and removes what that version removed; minors add. `stripe/__init__.py` hands out most names lazily (`__getattr__`), so removals from the top-level namespace are notes, not events.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `stripe/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- Most top-level names are lazy (`__getattr__`): a class removed from `stripe` itself is only a note.

## Stripe SDKs (`stripe/python`, `stripe/node`, `stripe/go`, `stripe/java`, `stripe/ruby`, `stripe/dotnet`, `stripe/php`)

### What it watches

Classes, methods, properties, constants and resources removed or renamed in each release of Stripe's seven official server SDKs, read from that release's section of the SDK's own `CHANGELOG.md`. One `package_release` event per bullet that names what it removes plainly. Stripe's API itself is a separate source (`stripe/api`, kind `openapi`). A bullet under an "Added" or "Features" heading that plainly names what the release adds is an `added` event.

Kind: `changelog` (`kinds/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Tag |
|---|---|---|---|
| `stripe/python` | `pypi:stripe` | PyPI | `vX.Y.Z` |
| `stripe/node` | `npm:stripe` | npm | `vX.Y.Z` |
| `stripe/go` | `go:github.com/stripe/stripe-go/v87` (events name the module at each release's major: `…/v73`) | the repo's `vX.Y.Z` tags | `vX.Y.Z` |
| `stripe/java` | `maven:com.stripe:stripe-java` | the repo's tags | `vX.Y.Z` |
| `stripe/ruby` | `rubygems:stripe` | the repo's tags | `vX.Y.Z` |
| `stripe/dotnet` | `nuget:Stripe.net` | the repo's tags | `vX.Y.Z` |
| `stripe/php` | `packagist:stripe/stripe-php` | the repo's tags | `vX.Y.Z` |

### Why this source

- **Source:** each SDK's `CHANGELOG.md`, read raw at the release's tag, e.g. https://github.com/stripe/stripe-python/blob/master/CHANGELOG.md (checked on 2026-11-20; `https://raw.githubusercontent.com/stripe/stripe-python/v4.0.0/CHANGELOG.md` for the golden).
- **Why this one:** it's Stripe's own record of each release, and every breaking bullet is marked `⚠️`, in all seven SDKs. At a tag it never changes, so the URL is its own citation.
- **Alternatives considered:** GitHub release notes (the same text, release by release, editable afterwards); the SDKs' wiki migration guides (one page per major, prose); Stripe's API changelog (about the API, not the SDK's names); diffing the published packages (the `apisurface` kind, a separate source when it's built).
- **Quote:** "⚠️ Removed the deprecated `Charge.refund` method. Prefer [Refund.create](https://stripe.com/docs/api/refunds/create)" (stripe-python 4.0.0).

### How the provider announces changes

Stripe ships a new major of every SDK when it pins a new API version (twice a year or so), and marks each breaking bullet `⚠️`. Older changelogs head releases `## X.Y.Z - date`, newer ones `## <a id="X-Y-Z"></a>X.Y.Z - date`; both are read.

A release that changes the API version the SDK pins says so in its section, in the same words in all seven SDKs (checked 2026-11-20): "This release changes the pinned API version to `2026-11-18.endive`." A few say they keep it: "This release **doesn't** change the pinned API version; it still uses `2026-05-27.dahlia`." (Go, Java, .NET), "…**does not** change the pinned API version. It's still `2026-06-24.dahlia`." (PHP), "This release uses the same pinned API version to `2026-03-25.dahlia` as the last major release." (Node).

### What the plugin reads and ignores

- Reads the release's section only. A bullet counts when it's marked `⚠️` or sits under a breaking or deprecation heading, and its first sentence is a plain template over code spans: `Remove X`, `Remove X from Y`, `Rename X to Y`, `X has been removed`, `Deprecated X in favor of Y`. The pull-request link it starts with is read past.
- Drops (and logs) everything else: type changes ("Change type of …"), nullable fields, pinned API versions, dropped language versions, bullets that list names inside prose or parentheses, and "Generated changes from …" roll-ups.
- **What a release pins** (`MapsTo`, `pins.go`): while listing the releases, it reads the changelog at the newest release's tag (or the one before, when that tag isn't pushed yet) and sets each release's `MapsTo` to `stripe/api` at the API version its own section states (the same id `stripe/api` lists its versions by, above), in one of the sentences above, exactly as written (`2026-11-18.endive`). A release whose section doesn't say pins nothing here, even though it keeps the release before's; two different versions in one section are none. Older wordings without "pinned" ("Major version release for API version 2022-08-01", "Move to API version …") aren't read.

### Known gaps

- Stripe's big generated roll-ups ("⚠️ Generated changes from #…") list their removals in nested bullets the reader doesn't open: those releases give no event.
- A release whose tag or section is missing is skipped as nothing to read, never guessed.
- Only releases that state their pin have one: most minors and patches don't say, so a release between two pin changes has none.
