# OpenAI Python SDK (`openai/python-sdk`)

## What it watches

The public API of the `openai` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`plugins/apisurface`), with the Python reader (`plugins/apisurface/pysurface`).

## Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/openai/>): each release's wheel, `openai-<version>-py3-none-any.whl`, read as published. Checked on 2026-11-20.
- **Why this one:** the wheel is exactly what `pip install openai==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the repo's CHANGELOG.md (generated from commit titles; it says "chore(types): rename" without the old and new names a fix needs); GitHub release notes (the same text); the API's OpenAPI spec (the HTTP API, not the Python names and parameters code uses; the `openai/models` source covers the models).
- **Quote:** `class AsssitantDeleted(BaseModel)`, `openai/types/beta/asssitant_deleted.py` line 10 at 1.1.1, gone at 1.1.2 (renamed `AssistantDeleted`).

## How the provider announces changes

- The SDK is generated from OpenAI's API spec and released often (several times a week at times). Type names follow the spec, so a renamed schema renames a class without a deprecation step.
- **Forced on upgrade**, for a repo whose pin reaches the release.

## What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `openai/` as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, and an `__init__`'s re-exports.
- Ignores private names (a leading underscore, a module under a private one, a name left out of `__all__`), classes nested in classes, and tests.
- Drops (and logs): an overloaded function's parameters; a removed parameter when the function now takes `**kwargs`; a new parameter without a default (the change-event contract has no subject for it yet).
- A removal is a rename only when the same scope loses one and gains one of the same kind with identical parameters, at least one of them besides `self`/`cls`; anything less sure is a removal.
- Drops (and logs) a removal that may not be one: a symbol whose file the newer release has too big to read (`MaxFileBytes`); a method whose class is still there but whose bases changed, which one of its bases (followed through the package's own classes, private ones included) now defines, or which the class now defines as an attribute (`create = _create`, a property made a plain attribute).

## Known gaps

- A class with no `__init__` of its own (the generated pydantic types) has no parameters to compare, so its rename reads as a removal of the old name, and its fields aren't compared.
- A method moved to a base class from outside the package (a dependency's class) is still read as removed when the bases themselves didn't change: the dependency's code isn't read.
- Type changes (an annotation narrowed) aren't events.
