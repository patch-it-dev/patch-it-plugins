# MongoDB: saved copies and goldens

Real copies of what MongoDB publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/mongodb -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/pymongo/`: `pymongo-4.15.4-cp310-cp310-macosx_10_9_x86_64.trimmed.whl`, `pymongo-4.15.5-cp310-cp310-macosx_10_9_x86_64.trimmed.whl`, `pymongo-4.16.0-cp310-cp310-macosx_10_9_x86_64.trimmed.whl` | https://files.pythonhosted.org/packages/5c/9e/5d6bd240a8d32e088078b37dcfa1579028c51d91168c0e992827ec1e87e6/pymongo-4.15.4-cp310-cp310-macosx_10_9_x86_64.whl; https://files.pythonhosted.org/packages/33/e4/d80061be4e53125597dd2916171c87986043b190e50c1834fff455e71d42/pymongo-4.15.5-cp310-cp310-macosx_10_9_x86_64.whl; https://files.pythonhosted.org/packages/4d/93/c36c0998dd91ad8b5031d2e77a903d5cd705b5ba05ca92bcc8731a2c3a8d/pymongo-4.16.0-cp310-cp310-macosx_10_9_x86_64.whl | 2026-11-21 | Apache-2.0 (the wheel's own license files are kept in it). Trimmed: only `pymongo/__init__.py`, `pymongo/asynchronous/__init__.py`, `pymongo/asynchronous/command_cursor.py`, `pymongo/command_cursor.py`, `pymongo/synchronous/__init__.py`, `pymongo/synchronous/command_cursor.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/pymongo/`: `pypi-pymongo-excerpt.json` and `pypi-pymongo-4.15.4.json`, `pypi-pymongo-4.15.5.json`, `pypi-pymongo-4.16.0.json` | https://pypi.org/pypi/pymongo/json, https://pypi.org/pypi/pymongo/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/mongodb/`: `mongodb-6.22.0.trimmed.tgz`, `mongodb-7.0.0.trimmed.tgz`, `mongodb-7.1.0.trimmed.tgz` | https://registry.npmjs.org/mongodb/-/mongodb-6.22.0.tgz; https://registry.npmjs.org/mongodb/-/mongodb-7.0.0.tgz; https://registry.npmjs.org/mongodb/-/mongodb-7.1.0.tgz | 2026-11-21 | Apache-2.0 (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `mongodb.d.ts`, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `npm/mongodb/`: `npm-mongodb-excerpt.json` and `npm-mongodb-6.22.0.json`, `npm-mongodb-7.0.0.json`, `npm-mongodb-7.1.0.json` | https://registry.npmjs.org/mongodb, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |

## JSON changelogs

Read with `Plugin{Log: ...}` (`kinds/changelog/json.go`); `golden/mongodb-atlas.json` is what it stores from the first file.

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `mongodb-atlas-changelog-c0f99ca9.json` | https://raw.githubusercontent.com/mongodb/openapi/c0f99ca9145e5aa74d08ae4451020857e93b5f28/changelog/changelog.json (the "Release API Changelog" commit of 2026-11-18 20:15 UTC; the same file as `main` on 2026-11-19) | 2026-11-19 | Apache-2.0 (mongodb/openapi) | "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" |
| `mongodb-atlas-changelog-2026-11-18-day.json` | The same file's newest day (`2026-11-18`), kept whole and unchanged: the drop test | 2026-11-19 | Apache-2.0 (mongodb/openapi) | `"changeCode": "request-property-one-of-removed"`, `"backwardCompatible": false` |
| `log-history/commits.json`, `log-history/changelog-{1,2,3}.json` | Stand-ins for the GitHub commits list of `changelog/changelog.json` and the file at three commits, in the real file's shape | written for the test, 2026-11-20 | n/a (test fixtures) | for the pair-by-pair test only (three versions, two pairs); no golden reads them |

**Trimmed, positions kept:** the real file is 10 MB. `mongodb-atlas-changelog-c0f99ca9.json` keeps every day, endpoint, API version and change in place. Every change the plugin doesn't read is replaced by `{}`: all but the three life-cycle codes and `endpoint-reactivated`. So is every endpoint or version left with none. So every JSON pointer in `golden/mongodb-atlas.json` names the same change in the real file. The 105 kept changes are copied unchanged. The plugin reads exactly the same 81 findings from this copy as from the real file (checked when the copy was made). The copy is re-indented.

## Saved lifecycle page

Read by `golden/mongodb-server.json` through the whole `notices` plugin, kept whole as `curl` saved it.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `mongodb-lifecycles.html` | https://www.mongodb.com/legal/support-policy/lifecycles | 2026-11-21 | © MongoDB, Inc.; quoted as a citation of MongoDB's published policy | "MongoDB 6.0 \| July 2022 \| July 31, 2025" |
