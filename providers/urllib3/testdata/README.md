# urllib3: saved copies and goldens

Real copies of what urllib3 publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/urllib3 -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/urllib3/`: `urllib3-2.2.1-py3-none-any.trimmed.whl`, `urllib3-2.2.2-py3-none-any.trimmed.whl`, `urllib3-2.5.0-py3-none-any.trimmed.whl`, `urllib3-2.6.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/a2/73/a68704750a7679d0b6d3ad7aa8d4da8e14e151ae82e6fee774e6e0d05ec8/urllib3-2.2.1-py3-none-any.whl; https://files.pythonhosted.org/packages/ca/1c/89ffc63a9605b583d5df2be791a27bc1a42b7c32bab68d3c8f2f73a98cd4/urllib3-2.2.2-py3-none-any.whl; https://files.pythonhosted.org/packages/a7/c2/fe1e52489ae3122415c51f387e221dd0773709bad6c6cdaa599e8a2c5185/urllib3-2.5.0-py3-none-any.whl; https://files.pythonhosted.org/packages/56/1a/9ffe814d317c5224166b23e7c47f606d6e473712a2fad0f704ea9b99f246/urllib3-2.6.0-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `urllib3/__init__.py`, `urllib3/response.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/urllib3/`: `pypi-urllib3-excerpt.json` and `pypi-urllib3-2.2.1.json`, `pypi-urllib3-2.2.2.json`, `pypi-urllib3-2.5.0.json`, `pypi-urllib3-2.6.0.json` | https://pypi.org/pypi/urllib3/json, https://pypi.org/pypi/urllib3/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |

## Saved changelogs

Real copies, saved once with `curl` from the release tag and never edited unless said.

| File | URL | Tag commit | Saved | License | Quote |
|---|---|---|---|---|---|
| `urllib3-2.0.0-CHANGES.rst` | https://raw.githubusercontent.com/urllib3/urllib3/2.0.0/CHANGES.rst | `303ba81` | 2026-11-14 08:35 UTC | MIT (urllib3), kept whole | "Removed support for Python 2.7, 3.5, and 3.6" |

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `pypi-urllib3-excerpt.json` | https://pypi.org/pypi/urllib3/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |
