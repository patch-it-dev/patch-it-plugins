# Datadog

Every source patch-it reads for Datadog, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Datadog (`datadog/api`)

### What it watches

Datadog's API v2, release by release: endpoints, request and response properties and schemas removed or changed between two versions of the OpenAPI spec Datadog generates its official clients from.

Kind: `openapi` (`kinds/openapi`), with its adapter, `Datadog` ([`api.go`](api.go)).

### Why this source

- **Source:** `.generator/schemas/v2/openapi.yaml` in [DataDog/datadog-api-client-python](https://github.com/DataDog/datadog-api-client-python), pinned to a release tag (`https://raw.githubusercontent.com/DataDog/datadog-api-client-python/<tag>/.generator/schemas/v2/openapi.yaml`); the newest tag from the GitHub Releases API. Checked on 2026-11-13.
- **Why this one:** Datadog publishes no standalone spec repo. Its generator writes the same spec, byte for byte, into every official client repo on every release, so this file *is* the API as Datadog's own clients see it, versioned by tag.
- **Alternatives considered:** the same file in the Go, Java, Ruby or TypeScript client repos (identical content; reading more than one would report every change twice); the repo's `CHANGELOG.md` (one section per release with `Removed` / `Deprecated` bullets linking each PR; prose, not wired as a hint source yet); the API reference on docs.datadoghq.com (rendered by JavaScript, not readable with a plain fetch).
- **Quote:** "Remove deprecated queried_tags_window_seconds and related_asset_tags fields" ([PR #4045](https://github.com/DataDog/datadog-api-client-python/pull/4045), released in 2.61.0).

### How the provider announces changes

- REST paths are versioned by prefix only (`/api/v1/…`, `/api/v2/…`), one spec file each. `info.version` is a fixed `"1.0"` that never moves.
- **Breaking changes are forced**: a removed property is gone from the next release, with no account-level version to stay on. Some go through a `Deprecated` changelog entry first; not all do.
- The client's release tag (`2.61.0`) is the only version signal, and it's what users pin (`datadog-api-client==2.61.0`). So the event's `version` is the release tag. Releases come roughly weekly.
- The spec is YAML only (~10 MB). The adapter converts it to JSON before diffing; excerpts are re-serialized from the node they quote either way.

### What the plugin reads and ignores

- Reads: the v2 spec at the newest tag, diffed against the last one.
- Ignores: changes to OpenAPI `tags` (they only rename generated SDK methods; the REST API is unchanged). The v1 spec isn't read yet.
- Reads a historical pair (a version pair, `2.60.0` → `2.61.0`) and recreates the removal of two properties from `TagIndexingRuleDynamicTags` (the golden test uses trimmed excerpts of both specs).

### Known gaps

- API v1 (`v1/openapi.yaml`) isn't watched.
- Deprecations announced only in the changelog (with no spec change yet) aren't events.

## Datadog ddtrace (Python) (`datadog/ddtrace-python-surface`)

### What it watches

The public API of the `ddtrace` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/ddtrace/>), each release's wheel read as published; the code is at <https://github.com/DataDog/dd-trace-py>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install ddtrace==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** GitHub release notes; the docs' upgrade notes.
- **Quote:** `def resolve_strands_hook_provider() -> type` (`ddtrace/appsec/ai_guard/integrations/__init__.py` line 67) at 4.12.3, gone at 4.13.0.

### How the provider announces changes

- The tracer releases often; most of the package is `ddtrace.internal` and `ddtrace.contrib.internal`, which this source treats as private. Moved APIs keep their old module forwarding names (`__getattr__`) until the next major.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `ddtrace/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- Wheels are per platform; the first by file name is read.
