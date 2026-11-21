# Textual: saved copies and goldens

Real copies of what Textual publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/textual -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/textual/`: `textual-8.2.5-py3-none-any.trimmed.whl`, `textual-8.2.6-py3-none-any.trimmed.whl`, `textual-8.2.7-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/cd/01/c4555f9c8a692ff83d84930150540f743ce94c89234f9e9a15ff4baba3a8/textual-8.2.5-py3-none-any.whl; https://files.pythonhosted.org/packages/b8/b4/c2b876f445e52522824cb900f2c7db3a7c24f89d20449ef278b4195d0ecb/textual-8.2.6-py3-none-any.whl; https://files.pythonhosted.org/packages/a8/f5/c1e18bc0707300a0e90204343abbf7d7acd6fb7ebe03a6d4893b99a234b8/textual-8.2.7-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `textual/walk.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/textual/`: `pypi-textual-excerpt.json` and `pypi-textual-8.2.5.json`, `pypi-textual-8.2.6.json`, `pypi-textual-8.2.7.json` | https://pypi.org/pypi/textual/json, https://pypi.org/pypi/textual/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |

## Saved changelogs

Real copies, saved once with `curl` from the release tag and never edited unless said.

| File | URL | Tag commit | Saved | License | Quote |
|---|---|---|---|---|---|
| `textual-v0.86.0-CHANGELOG.md` | https://raw.githubusercontent.com/Textualize/textual/v0.86.0/CHANGELOG.md | `5a738fa` | 2026-11-14 08:36 UTC | MIT (Textual), kept whole | "Breaking change: Removed `App.dark` reactive attribute" |
