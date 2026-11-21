# Algolia: saved copies and goldens

Real copies of what Algolia publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/algolia -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `algolia-search-2025-03-25-5c28957ec.yml`, `algolia-search-2025-04-01-c19ee6afb.yml` | https://raw.githubusercontent.com/algolia/api-clients-automation/released-2025-03-25-5c28957ec/specs/bundled/search.yml (and `released-2025-04-01-c19ee6afb`) | 2026-11-19 | MIT (algolia/api-clients-automation) | `deprecated: true` under `/1/clusters:` `get:` (new in `2025-04-01`) |
| `algolia-ingestion-2025-01-07-d0417f325.yml`, `algolia-ingestion-2025-01-22-232affcac.yml` | https://raw.githubusercontent.com/algolia/api-clients-automation/released-2025-01-07-d0417f325/specs/bundled/ingestion.yml (and `released-2025-01-22-232affcac`) | 2026-11-19 | MIT (algolia/api-clients-automation) | "Retrieves a list of tasks using the v1 endpoint, please use `getTasks` instead." (`GET /1/tasks`) |
| `algolia-recommend-2024-03-19-2eac6f410.yml`, `algolia-recommend-2024-03-25-30207fcd0.yml` | https://raw.githubusercontent.com/algolia/api-clients-automation/released-2024-03-19-2eac6f410/specs/bundled/recommend.yml (and `released-2024-03-25-30207fcd0`) | 2026-11-19 | MIT (algolia/api-clients-automation) | `/1{path}:` (in `2024-03-19`), `/{path}:` (in `2024-03-25`): `algolia-recommend`'s drop test |
| `algolia-ingestion-2024-09-18-cc190fae3.yml`, `algolia-ingestion-2024-09-19-a62c27d3b.yml` | https://raw.githubusercontent.com/algolia/api-clients-automation/released-2024-09-18-cc190fae3/specs/bundled/ingestion.yml (and `released-2024-09-19-a62c27d3b`) | 2026-11-19 | MIT (algolia/api-clients-automation) | `/1/transformations/models:` (in `2024-09-18`, never deprecated, gone in `2024-09-19`): the drop test |

The Algolia copies are trimmed by lines: `openapi`, `info` and the paths involved, every line copied unchanged; `components` left out.

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/algolia-refs.json` | `https://api.github.com/repos/algolia/api-clients-automation/git/matching-refs/tags/released-` | 9 of 186 tags, five of them on two days with several releases |
| `versions/algolia-commit-*.json` | `https://api.github.com/repos/algolia/api-clients-automation/commits/<tag>` | `sha` and the committer date |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/algoliasearch/`: `algoliasearch-4.44.1-py3-none-any.trimmed.whl`, `algoliasearch-4.44.3-py3-none-any.trimmed.whl`, `algoliasearch-4.44.4-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/a8/a7/74795993b739e721583653812155f0719159b79268b6eb4452594dc103fa/algoliasearch-4.44.1-py3-none-any.whl; https://files.pythonhosted.org/packages/32/05/84c46d0844055735c1192ba1a0364d871251dbf03e26a465eedf148e63f1/algoliasearch-4.44.3-py3-none-any.whl; https://files.pythonhosted.org/packages/39/53/d20cdf0238aa64a31aaea58e359aa761fe76515a7ff9872c345b5b06eaab/algoliasearch-4.44.4-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `algoliasearch/__init__.py`, `algoliasearch/recommend/__init__.py`, `algoliasearch/recommend/models/__init__.py`, `algoliasearch/recommend/models/facet_filters.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/algoliasearch/`: `pypi-algoliasearch-excerpt.json` and `pypi-algoliasearch-4.44.1.json`, `pypi-algoliasearch-4.44.3.json`, `pypi-algoliasearch-4.44.4.json` | https://pypi.org/pypi/algoliasearch/json, https://pypi.org/pypi/algoliasearch/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
