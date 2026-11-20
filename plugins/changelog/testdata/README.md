# Saved changelogs

Real copies, saved once with `curl` from the release tag and never edited. The
golden files under `golden/` are what the plugin stores from them
(`go test ./plugins/changelog -update`).

| File | URL | Tag commit | Saved | License | Quote |
|---|---|---|---|---|---|
| `textual-v0.86.0-CHANGELOG.md` | https://raw.githubusercontent.com/Textualize/textual/v0.86.0/CHANGELOG.md | `5a738fa` | 2026-11-14 08:36 UTC | MIT (Textual), kept whole | "Breaking change: Removed `App.dark` reactive attribute" |
| `urllib3-2.0.0-CHANGES.rst` | https://raw.githubusercontent.com/urllib3/urllib3/2.0.0/CHANGES.rst | `303ba81` | 2026-11-14 08:35 UTC | MIT (urllib3), kept whole | "Removed support for Python 2.7, 3.5, and 3.6" |

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `pypi-urllib3-excerpt.json` | https://pypi.org/pypi/urllib3/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |
