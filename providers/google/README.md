# Google

Every source patch-it reads for Google, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Google Gemini API models (`google/gemini-models`)

### What it watches

Gemini API models Google shuts down: every model on Google's own Gemini API deprecations page with a shutdown date, one `model` event per model name, with the replacement Google recommends. Google gives each date as the earliest a model may be shut down; the event says so.

Kind: `notices` (`kinds/notices`), with the deprecations-table reader (`kinds/notices/retire`, `GeminiModels` in [`models.go`](models.go)), reading Markdoc tables.

### Why this source

- **Source:** [Gemini deprecations](https://ai.google.dev/gemini-api/docs/deprecations), which redirects to `https://aistudio.google.com/docs/deprecations`; that URL serves the page as Markdown (Markdoc `{% table %}` blocks), and the plugin reads it there. Checked on 2026-11-21.
- **Why this one:** it's Google's own table of every Gemini API model with its release date, its shutdown date and the recommended replacement, one table per model family, the shut-down ones kept under "Deprecated models". It's where Google says deprecation dates are tracked: "Deprecation announcements are made on the Release notes page, and the announced earliest shutdown dates are tracked on this page."
- **Alternatives considered:**
  - **The Gemini API release notes** (`https://aistudio.google.com/docs/changelog`): the same deprecations, announced in prose, one entry per day. The table is the same facts, already structured.
  - **Vertex AI's model versions page** (`google/vertex-model-retirements`): the same models on Google Cloud, with Google Cloud's own retirement dates, which differ from the Gemini API's (gemini-2.5-pro: 2026-10-20 on Vertex AI, 2026-12-01 here). A separate source: a repo reaches one or the other.
  - **The Gemini API's models list** (`GET /v1beta/models`): what's served now, with no dates.
- **Quote:** "| `gemini-2.0-flash` | February 5, 2025 | June 1, 2026 | `gemini-3.6-flash` |" (Gemini 2.0 models, Deprecated models), and above the tables: "The shutdown dates listed in the table indicate the earliest possible dates on which a model might be retired".
- **Anchors** (checked 2026-11-21, declared in `provider.go`): `gemini-2.0-flash` and `embedding-001`, two shut-down models the page keeps. A read without them fails as a change of layout, never as nothing new.

### How the provider announces changes

- Models are named by version (`gemini-2.5-flash`, `gemini-2.5-flash-preview-09-2025`); a preview gets a shutdown date months after its stable model ships, a stable model about a year after its successor.
- A model is announced in the release notes, then its date appears here. Some shut-down models' requests are routed to a successor ("effective December 1, 2026, API requests to `gemini-2.5-pro` will automatically route to `gemini-3.8-flash`"); the replacement cell says so, and the hint quotes it.

### What the plugin reads and ignores

- Reads every Markdoc table with a "Shutdown date", a "Model" and a "Recommended replacement" column: one `breaking` `model` event per model name with a date, `version` the date. `replacement` is set when the cell names exactly one plain model; otherwise the hint quotes the cell.
- Ignores a row whose date isn't a date ("No shutdown date announced"), the managed agents' tables (an "Agent" column, not a model), and a cell that isn't a plain model id (a footnote mark). The pinned drop test: the Gemini 4 models and the agents, next to the two shut-down Gemini 3 previews, give only those two events.
- A read with far fewer models than the last one fails (the table changed shape), as do a page with no table and a block page.

### Known gaps

- Google's dates are the earliest possible; a later exact date only shows up when Google edits the row, which is then a new event.
- Models served only on Vertex AI are the other source's.

## Google Gen AI JavaScript SDK (`google/genai-js-surface`)

### What it watches

The public API of the `@google/genai` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@google/genai>), each version's `dist.tarball`, read as published; the code is at <https://github.com/googleapis/js-genai>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @google/genai@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (release-please, "⚠ BREAKING CHANGES").
- **Quote:** `interface LiveMusicSetClientContentParameters` (`dist/genai.d.ts` line 3757) at 0.15.0, gone at 1.0.0.

### How the provider announces changes

- Ships every week or two as one bundled declaration file per platform (`dist/genai.d.ts`, `dist/node/node.d.ts`, `dist/web/web.d.ts`).
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@google/genai.GoogleGenAI`.
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
- The node and web entry points are separate bundles: a removal there is one event per bundle.

## Google Gen AI Python SDK (`google/genai-python-surface`)

### What it watches

The public API of the `google-genai` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/google-genai/>), each release's wheel read as published; the code is at <https://github.com/googleapis/python-genai>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install google-genai==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (release-please, "⚠ BREAKING CHANGES"; for the `changelog` kind); the Gemini API Discovery document (the HTTP API).
- **Quote:** `class TuningJobOrOperation(_common.BaseModel)` (`google/genai/types.py` line 5924) at 0.8.0, gone at 1.0.0.

### How the provider announces changes

- The SDK ships every week or two; `google.genai.interactions` re-exports a generated client from a private package, so its names are read as still bound when they move.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `google/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- Symbols are named from the `google` namespace (`genai.types.Tool`), the import path code writes after `from google import genai`.

## Google Vertex AI model retirements (`google/vertex-model-retirements`)

### What it watches

Generative AI models Google Cloud retires on Vertex AI (now called Gemini Enterprise Agent Platform): every model on Google Cloud's model versions page with a plain retirement date, one `model` event per model id, with the replacement Google names.

Kind: `notices` (`kinds/notices`), with the deprecations-table reader (`kinds/notices/retire`, `VertexModelRetirements` in [`models.go`](models.go)), reading HTML tables.

### Why this source

- **Source:** [Model versions and lifecycle](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions) (the old `cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versions` redirects there). Checked on 2026-11-21.
- **Why this one:** it's Google Cloud's own table of every model id with its release date, its retirement date and its replacement, including the "Retired models" table. The release-notes feed (`google/vertex-ai-models`) only announces some deprecations, in prose; this page is where the dates are kept.
- **Alternatives considered:**
  - **The Vertex AI release notes feed**: already a source (`google/vertex-ai-models`), for announcements; not every retirement is announced there with a table of ids.
  - **The deprecations page** (`docs.cloud.google.com/vertex-ai/generative-ai/docs/deprecations`): SDK modules and features, not model dates.
  - **The Gemini API deprecations page** (`google/gemini-models`): the Gemini API's own dates, which differ.
- **Quote:** "| `veo-3.0-generate-001` | July 29, 2025 | June 30, 2026 | `veo-3.1-generate-001` |" (Retired models).
- **Anchors** (checked 2026-11-21, declared in `provider.go`): `gemini-1.5-pro-002` and `text-bison`, two retired models the page keeps. A read without them fails as a change of layout, never as nothing new.

### How the provider announces changes

- A GA model is kept for at least a year, written as a minimum ("October 13, 2027 or later", "No sooner than May 20, 2028"); when Google decides to retire it, the cell becomes a plain date, and the model later moves to "Retired models".

### What the plugin reads and ignores

- Reads every table with a "Retirement date", a "Model ID" and a "Replacement model ID" (or "Recommended upgrade") column: one `breaking` `model` event per model id with a plain date, `version` the date. `replacement` is set when the cell names exactly one model.
- Ignores a minimum lifetime ("… or later", "No sooner than …", "No retirement date announced"): it isn't a retirement notice. The pinned drop test: the Gemini image models' table, four rows with such dates and one dated retirement, gives one event.
- Ignores a cell that isn't a plain model id ("textembedding-gecko@003 *", with a footnote mark).

### Known gaps

- The page is about 430 KB of HTML; only its headings and tables are read, so a date given only in prose beside a table is missed.
- Partner models (Claude, Llama…) on Vertex AI are on other pages.

## Vertex AI generative models (`google/vertex-ai-models`)

### What it watches

Generative AI model endpoints on Google Cloud's Vertex AI that Google deprecates: one `model` event per model id in a "Discontinued endpoints" table, with the replacement when the row names exactly one.

Kind: `notices` (`kinds/notices`), with the feed reader (`kinds/notices/feed`) and its `VertexAIModels` feed ([`vertex_ai_models.go`](vertex_ai_models.go)).

### Why this source

- **Source:** the [Generative AI on Vertex AI release notes](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/release-notes) Atom feed, `https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml` (the `cloud.google.com/feeds/…` URL redirects here). Checked on 2026-11-14.
- **Why this one:** it's Google's own release notes as a feed, one entry per day, each change under a heading naming its kind (`Deprecated`), and deprecations come with a table of discontinued endpoints and replacements.
- **Alternatives considered:** the model versions and lifecycle docs page (HTML, rewritten in place, no history); the Google Cloud deprecations page (all of Cloud, prose); the models list API (what exists now, not when it stops).
- **Quote:** "We recommend updating your model endpoints before June 30, 2026, to avoid service disruption." (Deprecated: Imagen generation GA endpoints deprecation).
- **Anchors** (checked 2026-11-21, declared in `provider.go`): the feed's own title, "Generative AI on Vertex AI - Release notes" (its entries roll off). A read without them fails as a change of layout, never as nothing new.

### How the provider announces changes

- Deprecations are announced in the release notes with a date to move by; shutdown dates aren't always given.
- Content is licensed CC BY 4.0.

### What the plugin reads and ignores

- Only `Deprecated` sections with a "Discontinued endpoints" table; one event per model id, `provider: google`. `replacement` only when the row names one other plain id.
- The date the section says to move by is the event's `version`; a section saying it "is deprecated" with only a "before …" date is a deprecation, not breaking.
- Drops (and logs) product names in prose (third-party models mentioned in a sentence) and non-model deprecations (Vertex AI Extensions). 35 events from the saved feed.
- Every event cites a Wayback Machine snapshot of the feed and is read from it.

### Known gaps

- Whether an "update before <date>" deprecation should wait for a stated shutdown date is open; today it's a deprecation with that date.
- Partner models (Claude, Llama on Vertex) are only named in prose, so they're dropped.
