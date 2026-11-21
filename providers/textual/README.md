# Textual

Every source patch-it reads for Textual, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Textual changelog (`textual/textual`)

### What it watches

Removals and breaking changes in each Textual release, read from that release's section of its changelog: symbols removed (`App.dark`, …) or renamed. One `package_release` event per changelog bullet the plugin can read with certainty. The releases are listed from PyPI (final releases only) and each is read at its tag (`v<version>`); 0.86.0 is the release the golden test pins. A bullet under an "Added" or "Features" heading that plainly names what the release adds is an `added` event.

Kind: `changelog` (`kinds/changelog`).

### Why this source

- **Source:** [CHANGELOG.md at v0.86.0](https://github.com/Textualize/textual/blob/v0.86.0/CHANGELOG.md), read raw at the tag (`https://raw.githubusercontent.com/Textualize/textual/v0.86.0/CHANGELOG.md`). Checked on 2026-11-14.
- **Why this one:** it's the maintainers' own changelog in the [Keep a Changelog](https://keepachangelog.com) format, with `Removed` and `Changed` sections and explicit "Breaking change:" bullets. At a tag it never changes, so the URL is its own citation.
- **Alternatives considered:** GitHub release notes (the same text, but editable after the fact and not in one file); the docs site's changelog page (rendered from this file); diffing the package's code (finds removals but not the maintainers' wording or intent).
- **Quote:** "Breaking change: Removed `App.dark` reactive attribute" (0.86.0, Changed).

### How the provider announces changes

- Textual is pre-1.0 and releases often; breaking changes can land in any minor release and are marked in the changelog.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the release's section only; a release missing from the file is an error.
- An event needs a bullet whose first sentence is one plain template over code spans: `Removed X`, `Removed the X method from Y`, `Renamed X to Y`, `X property removed`, `Deprecated X in favor of Y`, or a bare name under `Removed`. Any bullet starting "Breaking change:" counts as breaking.
- Drops (and logs) everything else: behavior changes, signature changes, removed options and parameters (of which call?), dropped Python versions. 2 events from 0.86.0.

### Known gaps

- A release is read at its tag; one whose tag or changelog section is missing fails that pair, it isn't guessed.
- Breaking behavior changes written as prose aren't events.

## Textual's public API (`textual/python-surface`)

### What it watches

The public API of the `textual` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/textual/>), each release's wheel read as published; the code is at <https://github.com/Textualize/textual>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install textual==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** CHANGELOG.md (read by `textual/textual`, kind `changelog`).
- **Quote:** `def walk_selectable_widgets( root: DOMNode, bounds: Shape, bounded: set[DOMNode] ) -> Iterable[Widget]` (`textual/walk.py` line 176) at 8.2.5; the third parameter is `containers` at 8.2.6.

### How the provider announces changes

- Textual releases often and changes public helpers in patch releases; its changelog (the `textual/textual` source) says so in prose.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `textual/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
