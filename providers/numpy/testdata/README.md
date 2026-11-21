# NumPy: saved copies and goldens

Real copies of what NumPy publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/numpy -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved removed-members guide

Read through the `migrationguide` kind's removed-members reader; the golden file is what it stores from the guide.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `numpy_2_0_migration_guide-v2.0.0.rst` | https://github.com/numpy/numpy/blob/v2.0.0/doc/source/numpy_2_0_migration_guide.rst | 2026-11-04 | BSD-3-Clause (NumPy), kept whole | "The table below provides replacements for the removed members:" |
| `pypi-numpy-excerpt.json` | https://pypi.org/pypi/numpy/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/numpy/`: `numpy-2.4.6-cp311-cp311-macosx_10_9_x86_64.trimmed.whl`, `numpy-2.5.0-cp312-cp312-macosx_10_13_x86_64.trimmed.whl`, `numpy-2.5.1-cp312-cp312-macosx_10_13_x86_64.trimmed.whl` | https://files.pythonhosted.org/packages/b3/49/ec46835a70be8fa6446c495126ac84fdb28cb2558e1620ffb87a10c8b64c/numpy-2.4.6-cp311-cp311-macosx_10_9_x86_64.whl; https://files.pythonhosted.org/packages/fa/0a/11486d02add7b1384dff7374d124b1cfbb0ee864dcc9f6a2c0380638cf84/numpy-2.5.0-cp312-cp312-macosx_10_13_x86_64.whl; https://files.pythonhosted.org/packages/62/7b/14687aa674250e5e546f616f486b0d56d3631cd5b2415739141ce40bdcea/numpy-2.5.1-cp312-cp312-macosx_10_13_x86_64.whl | 2026-11-21 | BSD-3-Clause (and the licenses its wheel lists) (the wheel's own license files are kept in it). Trimmed: only `numpy/__init__.py`, `numpy/__init__.pyi`, `numpy/lib/_shape_base_impl.py`, `numpy/lib/_shape_base_impl.pyi` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/numpy/`: `pypi-numpy-excerpt.json` and `pypi-numpy-2.4.6.json`, `pypi-numpy-2.5.0.json`, `pypi-numpy-2.5.1.json` | https://pypi.org/pypi/numpy/json, https://pypi.org/pypi/numpy/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
