# NumPy (`numpy`)

## What it watches

Members removed from or renamed in NumPy's main namespace in 2.0.0 (`np.float_`, `np.NaN`, `np.product` and about fifty more): code using one raises `AttributeError`. One event per member. Pinned to that release.

Format plugin: `removedmembers` (`plugins/removedmembers`).

## Why this source

- **Source:** [`doc/source/numpy_2_0_migration_guide.rst` at v2.0.0](https://github.com/numpy/numpy/blob/v2.0.0/doc/source/numpy_2_0_migration_guide.rst), read raw at the tag. Checked on 2026-11-04.
- **Why this one:** it's NumPy's own migration guide, with a table of every removed member and its migration guideline. A tag never changes, so the URL is its own citation.
- **Alternatives considered:** the 2.0.0 release notes (list the removals in prose, spread over several sections); the rendered guide on numpy.org (the same text as HTML); NEP 52 (the plan, not what shipped).
- **Quote:** "The table below provides replacements for the removed members:" (Main namespace), with rows such as "NaN — Use ``np.nan`` instead."

## How the provider announces changes

- Package versions (PEP 440). 2.0.0 shipped on 2024-06-16.
- **Forced on a fresh install** for an unpinned or open pin (`>=1.24`), otherwise when the team upgrades.
- Every replacement the guide names exists in NumPy 1.x too, so a rename works on either side of the upgrade.

## What the plugin reads and ignores

- Reads the "Main namespace" section's table: removed member, migration guideline (a guideline may continue over several lines).
- `renamed_symbol` with the replacement when the guideline is exactly "Use ``np.<member>`` instead." (`float_` → `float64`, `NaN` → `nan`, `product` → `prod`…); `removed_symbol` otherwise (a builtin, an expression such as `-np.inf`, or advice). 56 events.

## Known gaps

- The guide's other tables (`np.lib` members, changed behavior) aren't read.
- Later releases' removals aren't watched.
