# urllib3 (`urllib3/python-surface`)

## What it watches

The public API of the `urllib3` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`plugins/apisurface`), with the Python reader (`plugins/apisurface/pysurface`).

## Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/urllib3/>), each release's wheel read as published; the code is at <https://github.com/urllib3/urllib3>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install urllib3==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGES.rst (read by `urllib3/urllib3`, kind `changelog`).
- **Quote:** `def getheader(self, name: str, default: str | None = None) -> str | None` (`BaseHTTPResponse`, `urllib3/response.py` line 552) at 2.5.0, gone at 2.6.0, deprecated since 2.0.

## How the provider announces changes

- Urllib3 deprecates for a major and removes in a later minor.
- **Forced on upgrade**, for a repo whose pin reaches the release.

## What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `urllib3/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
- Ignores private names: a leading underscore, a module under a private one or under one named `internal` or `vendor`, a name left out of `__all__`, and tests. Also classes nested in classes, a property's parameters (and its setter), and `self`/`cls` (callers never pass them).
- Drops (and logs): an overloaded function's parameters; a removed parameter when the function now takes `**kwargs`; a new parameter without a default (the change-event contract has no subject for it yet).
- A removal is a rename only when the same scope loses one and gains one of the same kind with identical parameters, at least one of them besides `self`/`cls`; anything less sure is a removal.
- Drops (and logs) a removal that may not be one: a name its module still binds in the newer release (an alias such as `DeadlineExceededError = GatewayTimeoutError`, an import from a private module, a star import from outside the package, a module-level `__getattr__` that hands out old names); a symbol whose file the newer release has too big to read (`MaxFileBytes`); a method whose class is still there but whose bases changed, which one of its bases now defines, or which the class now defines as an attribute.

## Known gaps

- A positional parameter renamed reads as the old one removed: a call that passes it by keyword breaks, one that passes it by position doesn't.
- A class with no `__init__` of its own (generated pydantic or dataclass-like types) has no parameters to compare, so its rename reads as a removal of the old name, and its fields aren't compared.
- A module that star-imports from outside the package, or defines `__getattr__`, hides every removal from it: those are notes, not events.
- Type changes (an annotation narrowed, a return type changed) aren't events.
- The wheel read is the first one by file name (a `-none-any` wheel when there is one); a release whose platform wheels differ in their Python files is read from that one.
