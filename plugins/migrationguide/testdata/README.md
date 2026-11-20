# Golden events

`golden/anthropic-1.0.0.json` is what the `anthropic` plugin stores from anthropic-sdk-python's MIGRATION.md at v1.0.0. The saved guide itself is shared with the `sdkguide` package: see [`sdkguide/testdata/README.md`](../../../sdkguide/testdata/README.md) for its URL, date and license.

Rewrite it after an intended change with `go test ./plugins/migrationguide -update`, and say in the PR why the new output is right.

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `pypi-anthropic-excerpt.json` | https://pypi.org/pypi/anthropic/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |
