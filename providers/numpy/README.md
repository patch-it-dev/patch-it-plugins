# NumPy

Every source patch-it reads for NumPy, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## NumPy migration guides (`numpy/numpy`)

### What it watches

Members a new major removes from or renames in NumPy's main namespace, from that major's migration guide: in 2.0.0, `np.float_`, `np.NaN`, `np.product` and about fifty more (code using one raises `AttributeError`). One event per member, with the major's first release as `version` and the release before it on PyPI as `previous_version`. The releases are listed from PyPI; the guide is read at each major's first tag (`v<X>.0.0`), so nothing in the code names a release.

Kind: `migrationguide` (`kinds/migrationguide`), with the removed-members reader (`kinds/migrationguide/members`).

### Why this source

- **Source:** `doc/source/numpy_<X>_0_migration_guide.rst` at tag `v<X>.0.0`, read raw at the tag; the one so far is [2.0's](https://github.com/numpy/numpy/blob/v2.0.0/doc/source/numpy_2_0_migration_guide.rst). A major with no guide at its tag has nothing to read (the pair is skipped). Releases from [PyPI](https://pypi.org/project/numpy/). Checked on 2026-11-04 (2.0) and 2026-11-20 (no other guide yet).
- **Why this one:** it's NumPy's own migration guide, with a table of every removed member and its migration guideline. A tag never changes, so the URL is its own citation.
- **Alternatives considered:** the 2.0.0 release notes (list the removals in prose, spread over several sections); the rendered guide on numpy.org (the same text as HTML); NEP 52 (the plan, not what shipped).
- **Quote:** "The table below provides replacements for the removed members:" (Main namespace), with rows such as "NaN — Use ``np.nan`` instead."

### How the provider announces changes

- Package versions (PEP 440). 2.0.0 shipped on 2024-06-16.
- **Forced on a fresh install** for an unpinned or open pin (`>=1.24`), otherwise when the team upgrades.
- Every replacement the guide names exists in NumPy 1.x too, so a rename works on either side of the upgrade.

### What the plugin reads and ignores

- Reads the "Main namespace" section's table: removed member, migration guideline (a guideline may continue over several lines).
- `renamed_symbol` with the replacement when the guideline is exactly "Use ``np.<member>`` instead." (`float_` → `float64`, `NaN` → `nan`, `product` → `prod`…); `removed_symbol` otherwise (a builtin, an expression such as `-np.inf`, or advice). 56 events.

### Known gaps

- The guide's other tables (`np.lib` members, changed behavior) aren't read.
- Removals in a minor release aren't watched: only a major gets a guide.
- A later major's guide is read with 2.0's rule: a guideline that is exactly "Use ``np.<member>`` instead." is a rename. NumPy 2.0's guide says each such replacement works on 1.x; a later guide might name one that's new, which would make the rename fail on the release before.

## NumPy's public API (`numpy/python-surface`)

### What it watches

The public API of the `numpy` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/numpy/>), each release's wheel read as published; the code is at <https://github.com/numpy/numpy>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install numpy==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** the release notes (one file per release; the `numpy/numpy` migration guide source covers 2.0); the deprecation warnings in the code.
- **Quote:** `def row_stack( tup: Sequence[ArrayLike], *, dtype: DTypeLike | None = None, casting: _CastingKind = "same_kind", ) -> NDArray[Any]` (`numpy/lib/_shape_base_impl.pyi` line 129, re-exported as `numpy.row_stack`) at 2.4.6, gone at 2.5.0.

### How the provider announces changes

- NumPy deprecates for two releases and then removes; the `.pyi` stubs, read in place of the `.py` files, carry every public signature.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `numpy/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
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
- Functions implemented in C appear only through the `.pyi` stubs; one missing from the stubs isn't seen.
