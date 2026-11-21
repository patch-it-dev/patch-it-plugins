# OpenAI

Every source patch-it reads for OpenAI, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## OpenAI (`openai/api`)

### What it watches

The OpenAI REST API (`api.openai.com/v1`): operations OpenAI marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec its SDKs are generated from.

Kind: `openapi` (`kinds/openapi`), with its adapter, `OpenAI` ([`api.go`](api.go)).

### Why this source

- **Source:** `openapi.yaml` in [openai/openai-openapi](https://github.com/openai/openai-openapi) (MIT), pinned to a commit (`https://raw.githubusercontent.com/openai/openai-openapi/<sha>/openapi.yaml`). The versions come from the GitHub commits API. Checked on 2026-11-21.
- **Why this one:**
  - It's OpenAI's own description of the API, the one `openai-python` and `openai-node` are generated from, and it's kept current: 589 commits changed it from 2022-01-19 to 2026-11-21, several on most days.
  - It marks deprecated operations with the `deprecated` flag, and the operation's description says until when.
  - Every commit can be cited and replayed.
- **Alternatives considered:**
  - **The repo's tags** (`1.3.0`, `2.0.0`): too sparse to follow the API.
  - **`openapi.json` beside it:** the same spec, but only committed from 2026-07-16 on; the YAML has the whole history.
  - **The [deprecations page](https://developers.openai.com/api/docs/deprecations):** read for model shutdowns by `openai/models` (kind `notices`). It announces endpoint shutdowns in prose and tables, not per operation.
  - **The changelog:** product news, not keyed by operation.
- **Quote:** "The endpoint and model are deprecated and scheduled to shut down on February 26, 2027." (`POST /audio/translations`, flagged deprecated on 2026-10-28)

### How the provider announces changes

- **One version per day.** The spec is committed several times a day, and `info.version` (`2.3.0`) isn't bumped with each change, so a version is a day: the day's last commit, its date the event's `version`.
- **Operations are named by the path a caller sends** (`POST /v1/audio/translations`): the server is `https://api.openai.com/v1` and the paths leave out `/v1`, so the adapter moves it into the paths.
- **Deprecations use the flag.** On 2026-11-21, 17 operations carry it, and none is called deprecated only in words (one description mentions deprecated models, not the operation). Flagged descriptions give the shutdown date.
- **Much of the daily churn is the spec's own shape**: enum lists rewritten as `const`, schemas regenerated, work in progress added and pulled again.

### What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so the shutdown date is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal.
- **Ignores everything else:** removals of operations that were never deprecated, schema and property changes, new operations. The drop test replays 2026-11-17 → 2026-11-18, when the session spending-limit schemas added the day before were pulled again: three schema removals, no event.
- **The golden test** replays 2026-10-27 → 2026-10-28: `POST /audio/translations` deprecated, one event; the two new `/organization/external_keys` operations of the same day are dropped.

### Known gaps

- **Deprecated parameters, fields and enum values** (`user` in favor of `safety_identifier`, a model in a `model` enum) aren't events: only whole operations are.
- **Removals OpenAI didn't flag first aren't events,** even if the endpoint really went away.
- **History:** the GitHub commits API lists at most 1,000 commits. The spec has 589 (back to 2022-01-19), so all are listed today; at its pace, the oldest will start dropping off in 2027.
- **The patch-it client needs `openai` in its list of date-versioned APIs** to place the events' date `version`; that list is outside this repo.

## OpenAI model retirements (`openai/models`)

### What it watches

OpenAI models that are being shut down: every text model name code sends to the Chat Completions or Responses API, with its shutdown date and the replacement OpenAI recommends. One breaking event per model name.

Kind: `notices` (`kinds/notices`), reading with the deprecations-page reader, `kinds/notices/retire/`.

### Why this source

- **Source:** [developers.openai.com/api/docs/deprecations](https://developers.openai.com/api/docs/deprecations), read as markdown at `https://developers.openai.com/api/docs/deprecations.md`. Checked on 2026-11-21.
- **Why this one:** it's OpenAI's own list of every deprecation: a table of upcoming shutdowns and one of past ones, each row a shutdown date, the models, a status, the recommended replacement and the notice date. The markdown version is machine-readable without scraping HTML. `robots.txt` allows it.
- **Alternatives considered:** the models list API (`GET /v1/models`: says what exists now, not when it stops); OpenAI's blog and email notices (announce only some changes, no table); the old `platform.openai.com/docs/deprecations` URL (redirects here).
- **Quote:** "| **Dec 11, 2026** | `gpt-5`, `gpt-5-2025-08-07` | Deprecated | `gpt-6.1-sol` | Jun 11, 2026 |" ("Upcoming deprecations", 2026-11-21).

### How the provider announces changes

- An announcement every few weeks, shutdowns a few months later (at least 6 months for generally available models, 3 for specialized variants, possibly 2 weeks for previews, per the page).
- **Forced, by date**: a request naming a retired model fails on every SDK version. So the event's `version` is the shutdown date.
- The page keeps past shutdowns in a "Past deprecations" table. Until 2026-11 it was one table per announcement, newest first, with plain dates; since then it is two tables (upcoming, past) with bold dates and a Status and a Notice date column. The reader reads both layouts.

### What the plugin reads and ignores

- Reads every table with a shutdown or retirement date column, a model column and a replacement column. Each backticked name in the model cell is one model (aliases after an escaped pipe too). The feature and endpoint table has no model column, so it isn't read.
- A model listed twice keeps the row with the latest notice date (the first row on a page without that column); a newer "To be announced", or a status that isn't a retirement (only exactly `Deprecated`, `Shutdown`, `Shut down` or `Retired` are; "Deprecation postponed" isn't), withdraws an older date.
- A replacement that is itself retired is followed to one that still works (`gpt-3.5-turbo-0613` → `gpt-3.5-turbo` → `gpt-5.6-terra`), and the hint says so.
- Keeps every `gpt-<number>` model (a new generation needs no change) and the `o1`, `o3`, `o4`, `chatgpt-`, `computer-use` and `codex-` names. Ignores image, audio, realtime, transcription, embedding and moderation models, fine-tuned bases (`ft-gpt-4`; code sends `ft:gpt-4-0613:org::id`) and the legacy `-completions` aliases: the events are for the operations that take text models.
- Every event cites a Wayback Machine snapshot of the page, never the live page, and is read from that snapshot. No snapshot, or one that differs from the live page, stores nothing that run. A page that suddenly lists far fewer retirements fails the run (a renamed column probably hid a table).

### Known gaps

- Replacements that need a setting (`gpt-5.6-sol` with `reasoning.mode: pro`) or name no model are passed on as written; a fix can't be generated from the second kind.
- Image, audio and embedding model retirements aren't events.

## OpenAI Node.js SDK (`openai/node-surface`)

### What it watches

The public API of the `openai` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/openai>), each version's `dist.tarball`, read as published; the code is at <https://github.com/openai/openai-node>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install openai@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the repo's CHANGELOG.md (commit titles); the API spec (`openai/api`: the HTTP API).
- **Quote:** `interface AgentSessionBudgetReachedEvent` (`resources/beta/agents/agents.d.mts` line 1188) at 7.59.0, gone at 7.60.0.

### How the provider announces changes

- Generated from the same spec as the Python SDK and released several times a week; types follow the API, so a pulled preview feature removes its types in a minor.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `openai/resources/chat/completions.Completions.create`.
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

## OpenAI Python SDK (`openai/python-surface`)

### What it watches

The public API of the `openai` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/openai/>), each release's wheel read as published; the code is at <https://github.com/openai/openai-python>. Checked on 2026-11-20.
- **Why this one:** the wheel is exactly what `pip install openai==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the repo's CHANGELOG.md (generated from commit titles; it says "chore(types): rename" without the old and new names a fix needs); GitHub release notes (the same text); the API's OpenAPI spec (`openai/api`: the HTTP API, not the Python names code uses).
- **Quote:** `class AsssitantDeleted(BaseModel)`, `openai/types/beta/asssitant_deleted.py` line 10 at 1.1.1, gone at 1.1.2 (renamed `AssistantDeleted`).

### How the provider announces changes

- The SDK is generated from OpenAI's API spec and released several times a week; type names follow the spec, so a renamed schema renames a class without a deprecation step.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `openai/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
