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

## JSON changelogs

Read with `Plugin{Log: ...}` (json.go); `golden/mongodb-atlas.json` is what it stores from the first file.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `mongodb-atlas-changelog-c0f99ca9.json` | https://raw.githubusercontent.com/mongodb/openapi/c0f99ca9145e5aa74d08ae4451020857e93b5f28/changelog/changelog.json (the "Release API Changelog" commit of 2026-11-18 20:15 UTC; the same file as `main` on 2026-11-19) | 2026-11-19 | Apache-2.0 (mongodb/openapi) | "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" |
| `mongodb-atlas-changelog-2026-11-18-day.json` | The same file's newest day (`2026-11-18`), kept whole and unchanged: the drop test | 2026-11-19 | Apache-2.0 (mongodb/openapi) | `"changeCode": "request-property-one-of-removed"`, `"backwardCompatible": false` |
| `commits-c0f99ca9.json`, `commits-day.json` | Stand-ins for `https://api.github.com/repos/mongodb/openapi/commits?path=changelog/changelog.json&per_page=1`, keeping only the field the plugin reads (`sha`) | 2026-11-19 | n/a (test fixtures) | |

**Trimmed, positions kept:** the real file is 10 MB. `mongodb-atlas-changelog-c0f99ca9.json` keeps every day, endpoint, API version and change in place. Every change the plugin doesn't read is replaced by `{}`: all but the three life-cycle codes and `endpoint-reactivated`. So is every endpoint or version left with none. So every JSON pointer in `golden/mongodb-atlas.json` names the same change in the real file. The 105 kept changes are copied unchanged. The plugin reads exactly the same 81 findings from this copy as from the real file (checked when the copy was made). The copy is re-indented.

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `log-history/commits.json`, `log-history/changelog-{1,2,3}.json` | Stand-ins for the GitHub commits list of `changelog/changelog.json` and the file at three commits, in the real file's shape | written for the test, 2026-11-20 | n/a (test fixtures) | for the pair-by-pair test only (three versions, two pairs); no golden reads them |
