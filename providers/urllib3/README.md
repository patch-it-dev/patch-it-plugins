# urllib3

Every source patch-it reads for urllib3, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## urllib3 changelog (`urllib3/urllib3`)

### What it watches

Removals and deprecations in each urllib3 release, read from that release's section of its changelog: classes, methods and modules removed or renamed. One `package_release` event per bullet the plugin can read with certainty. The releases are listed from PyPI (final releases only) and each is read at its git tag (`<version>`); 2.0.0 is the release the golden test pins.

Kind: `changelog` (`kinds/changelog`).

### Why this source

- **Source:** [CHANGES.rst at 2.0.0](https://github.com/urllib3/urllib3/blob/2.0.0/CHANGES.rst), read raw at the tag (`https://raw.githubusercontent.com/urllib3/urllib3/2.0.0/CHANGES.rst`). Checked on 2026-11-14.
- **Why this one:** it's the maintainers' own changelog, with a "Removed" and a "Deprecated" section per release, each bullet linking its issue. At a tag it never changes.
- **Alternatives considered:** the [v2 migration guide](https://urllib3.readthedocs.io/en/stable/v2-migration-guide.html) (prose and examples, harder to read deterministically); GitHub release notes (same content as the changelog); diffing the package's code.
- **Quote:** "Removed ``urllib3.HTTPResponse.from_httplib``" (2.0.0, Removed).

### How the provider announces changes

- Semantic versions; 2.0.0 (2023-04-26) was the major release that removed long-deprecated APIs.
- **Forced on a fresh install** for an unpinned `urllib3`, which many packages pull in indirectly.

### What the plugin reads and ignores

- reStructuredText titles: release and section levels are told apart by their underline character.
- Same bullet templates as [`textual/textual`](../textual/README.md#textual-changelog-textualtextual). A name the same release's "Added" section adds back isn't reported as removed: 2.0.0 removed the `urllib3.request` module and added a `urllib3.request()` function.
- Drops (and logs) dropped Python and OpenSSL versions and behavior changes. 12 events from 2.0.0.

### Known gaps

- A release is read at its tag; one whose tag or changelog section is missing fails that pair, it isn't guessed.
- Changed defaults (TLS versions, retries) are behavior changes and aren't events.

## urllib3's public API (`urllib3/python-surface`)

### What it watches

The public API of the `urllib3` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/urllib3/>), each release's wheel read as published; the code is at <https://github.com/urllib3/urllib3>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install urllib3==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGES.rst (read by `urllib3/urllib3`, kind `changelog`).
- **Quote:** `def getheader(self, name: str, default: str | None = None) -> str | None` (`BaseHTTPResponse`, `urllib3/response.py` line 552) at 2.5.0, gone at 2.6.0, deprecated since 2.0.

### How the provider announces changes

- Urllib3 deprecates for a major and removes in a later minor.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `urllib3/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
