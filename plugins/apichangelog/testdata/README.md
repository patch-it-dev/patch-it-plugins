# Saved changelogs

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `mongodb-atlas-changelog-c0f99ca9.json` | https://raw.githubusercontent.com/mongodb/openapi/c0f99ca9145e5aa74d08ae4451020857e93b5f28/changelog/changelog.json (the "Release API Changelog" commit of 2026-11-18 20:15 UTC; the same file as `main` on 2026-11-19) | 2026-11-19 | Apache-2.0 (mongodb/openapi) | "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" |
| `mongodb-atlas-changelog-2026-11-18-day.json` | The same file's newest day (`2026-11-18`), kept whole and unchanged: the drop test | 2026-11-19 | Apache-2.0 (mongodb/openapi) | `"changeCode": "request-property-one-of-removed"`, `"backwardCompatible": false` |
| `commits-c0f99ca9.json`, `commits-day.json` | Stand-ins for `https://api.github.com/repos/mongodb/openapi/commits?path=changelog/changelog.json&per_page=1`, keeping only the field the plugin reads (`sha`) | 2026-11-19 | n/a (test fixtures) | |

**Trimmed, positions kept:** the real file is 10 MB. `mongodb-atlas-changelog-c0f99ca9.json` keeps every day, endpoint, API version and change in place. Every change the plugin doesn't read is replaced by `{}`: all but the three life-cycle codes and `endpoint-reactivated`. So is every endpoint or version left with none. So every JSON pointer in `golden/mongodb-atlas.json` names the same change in the real file. The 105 kept changes are copied unchanged. The plugin reads exactly the same 81 findings from this copy as from the real file (checked when the copy was made). The copy is re-indented.
