# Anthropic

Every source patch-it reads for Anthropic, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Anthropic Python SDK (`anthropic/python-sdk`)

### What it watches

Request parameters removed from `messages.create()` and its siblings in anthropic-sdk-python 1.0.0 (`temperature`, `top_p`, `top_k`): passing one is a `TypeError`. One breaking event per parameter. Pinned to that release.

Kind: `migrationguide` (`kinds/migrationguide`), reading with `kinds/migrationguide/sdkguide/`.

### Why this source

- **Source:** MIGRATION.md at the first release of each major ([at v1.0.0](https://github.com/anthropics/anthropic-sdk-python/blob/v1.0.0/MIGRATION.md)), the majors listed from PyPI; read raw at the tag (`https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/v1.0.0/MIGRATION.md`). Checked on 2026-10-17.
- **Why this one:** it's the SDK maintainers' own guide to the major release, with a table of the methods, the removed parameter and what to use instead. A tag never changes, so the URL is its own permanent citation.
- **Alternatives considered:** the GitHub release notes and `CHANGELOG.md` (list commits, not "what to use instead"); diffing the SDK's code between versions (finds signature changes but not the maintainers' advice, and is a lot more machinery); Anthropic's model deprecations page (mentions the change, but it's the models page).
- **Quote:** "Removed: deprecated request parameters" (section heading), with the row "`temperature`, `top_p`, `top_k` | Remove them."

### How the provider announces changes

- Package versions (PEP 440). 1.0.0 shipped on 2026-08-20.
- **Forced on a fresh install** for a repo whose `anthropic` pin is open above 1.0, otherwise when the team upgrades.
- The fix works on both sides of the upgrade: removing the parameter is right for current models, and `extra_body` passes it through for an older one.

### What the plugin reads and ignores

- The guide must be the "Migrating to v1" guide. Every table with a "Removed parameter" and a "Use instead" column; a row counts only when its method cell names the plain `messages.create()`.
- Ignores the beta-only `output_format` row: `output_format=Model` is still valid on the `parse()` / `stream()` helpers, so it isn't a removal everywhere.
- `migration_hint` is the guide's "Use instead" cell, as written.

### Known gaps

- Only 1.0.0. A later major's guide is a new entry.
- Removed type aliases, exports and helper arguments in the same guide aren't events yet.

## Anthropic model retirements (`anthropic/models`)

### What it watches

Claude models that are retired or have a retirement date on the Claude API, with the replacement Anthropic recommends. One breaking event per model name.

Kind: `notices` (`kinds/notices`), reading with the deprecations-page reader, `kinds/notices/retire/`.

### Why this source

- **Source:** [platform.claude.com/docs/en/about-claude/model-deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations), read as markdown at the same URL plus `.md`. Checked on 2026-10-13.
- **Why this one:** it's Anthropic's own deprecation history, one table per announcement (retirement date, deprecated model, recommended replacement), served as markdown. `robots.txt` disallows only `/api/`.
- **Alternatives considered:** the models API (`GET /v1/models`: what exists now, no dates); release notes and emails (not a complete list); the old docs.anthropic.com URL (redirects here).
- **Quote:** "| November 30, 2026 | `claude-sonnet-4-5-20250929` | `claude-sonnet-5-5` |" (Deprecation history, 2026-09-30: Claude Sonnet 4.5 model).

### How the provider announces changes

- A few announcements a year, at least 60 days before retirement (per the page).
- **Forced, by date**, on the Claude API: the event's `version` is the retirement date. Amazon Bedrock and Google Cloud set their own dates, so code calling Claude through them may be affected on a different day.

### What the plugin reads and ignores

- Same reader as [`openai/models`](../openai/README.md#openai-model-retirements-openaimodels): every dated table with a model and a replacement column, newest row wins, retired replacements followed, every event read from and citing a Wayback snapshot.
- Keeps every `claude-` name. Ignores the "Model status" table (active models with tentative dates) and any model without a date yet.

### Known gaps

- Retirements on Amazon Bedrock or Google Cloud aren't on this page.
- The page also documents SDK changes (the Python SDK 1.0 removing sampling parameters); those are the [`anthropic/python-sdk`](#anthropic-python-sdk-anthropicpython-sdk) source, not this one.

## Anthropic Python SDK's public API (`anthropic/python-surface`)

### What it watches

The public API of the `anthropic` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/anthropic/>), each release's wheel read as published; the code is at <https://github.com/anthropics/anthropic-sdk-python>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install anthropic==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the repo's CHANGELOG.md (commit titles, read by the `changelog` kind where registered); the migration guide (`anthropic/python-sdk`, kind `migrationguide`: majors only); there is no public OpenAPI spec.
- **Quote:** `class BetaBillingFrozenError(PermissionDeniedError)` (`anthropic/types/beta/beta_billing_frozen_error.py` line 9) at 2.6.0, gone at 2.7.0.

### How the provider announces changes

- The SDK is generated from Anthropic's API definition and released every week or two; beta types come and go with the betas.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `anthropic/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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

## Anthropic TypeScript SDK (`anthropic/typescript-surface`)

### What it watches

The public API of the `@anthropic-ai/sdk` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@anthropic-ai/sdk>), each version's `dist.tarball`, read as published; the code is at <https://github.com/anthropics/anthropic-sdk-typescript>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @anthropic-ai/sdk@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (commit titles); the migration guide (majors only).
- **Quote:** `class BetaBillingFrozenError extends PermissionDeniedError` (`resources/beta/organization/organization.d.mts` line 45) at 1.6.0, gone at 1.7.0.

### How the provider announces changes

- Generated from Anthropic's API definition and released every week or two; beta types come and go with the betas.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@anthropic-ai/sdk/resources/messages.Messages.create`.
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
- The client re-exports every resource through its namespaces, so one removed class has many paths (16 for `BetaBillingFrozenError`), all in one event.
