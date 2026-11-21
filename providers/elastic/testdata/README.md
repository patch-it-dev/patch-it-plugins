# Elastic: saved copies and goldens

Real copies of what Elastic publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/elastic -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `elasticsearch-8.19-fd5d063c.json`, `elasticsearch-9.0-57ecfe5f.json` | https://raw.githubusercontent.com/elastic/elasticsearch-specification/fd5d063c4824de5d33618b7fca4df0e6c1b8489f/output/openapi/elasticsearch-openapi.json (the head of `8.19`; and `57ecfe5fda6976b94fc8b8830ecb51f4969dccd6`, the head of `9.0`) | 2026-11-19 | Apache-2.0 (elastic/elasticsearch-specification) | `"/{index}/_unfreeze": {` (in 8.19, deprecated; gone in 9.0) |
| `elasticsearch-9.1-12639d0e.json`, `elasticsearch-9.2-87a82640.json` | https://raw.githubusercontent.com/elastic/elasticsearch-specification/12639d0e5ec07f064c909f15160ff68d608dbc2c/output/openapi/elasticsearch-openapi.json (the head of `9.1`; and `87a82640d8aa1139163dc7071bf00f971ce5e453`, the head of `9.2`) | 2026-11-19 | Apache-2.0 (elastic/elasticsearch-specification) | "designed for indirect use by Elasticsearch Service, Elastic Cloud Enterprise, and Elastic Cloud on Kubernetes. Direct use is not supported." (`/_autoscaling/capacity`, in 9.1, gone in 9.2): the drop test |

The Elasticsearch copies keep `openapi`, `info` and the paths involved, every line unchanged; the 9.2 copy's paths are all gone in 9.2, so its `"paths": {}` is the one line that isn't in the original.

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/es-releases.json` | `https://api.github.com/repos/elastic/elasticsearch/releases?per_page=100` | all 100; `tag_name`, `prerelease`, `draft` |
| `versions/es-branches.json` | `https://api.github.com/repos/elastic/elasticsearch-specification/git/matching-refs/heads/` | the minor branches, `main` and `gh-pages` (33 of 250) |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/elasticsearch/`: `elasticsearch-9.1.1-py3-none-any.trimmed.whl`, `elasticsearch-9.1.2-py3-none-any.trimmed.whl`, `elasticsearch-9.1.3-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/cf/4c/c0c95d3d881732a5d1b28e12c9be4dea5953ade71810f94565bd5bd2101a/elasticsearch-9.1.1-py3-none-any.whl; https://files.pythonhosted.org/packages/89/95/ee3bcf867cdbaba15a0118d6ca0b9b4477892ff3b92e7d6135cc42b262af/elasticsearch-9.1.2-py3-none-any.whl; https://files.pythonhosted.org/packages/bf/10/cab92a57625aeb2d787e3f5e3a698948bc5b34c151422afacff1f0f1c691/elasticsearch-9.1.3-py3-none-any.whl | 2026-11-21 | Apache-2.0 (the wheel's own license files are kept in it). Trimmed: only `elasticsearch/__init__.py`, `elasticsearch/_sync/client/__init__.py`, `elasticsearch/client.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/elasticsearch/`: `pypi-elasticsearch-excerpt.json` and `pypi-elasticsearch-9.1.1.json`, `pypi-elasticsearch-9.1.2.json`, `pypi-elasticsearch-9.1.3.json` | https://pypi.org/pypi/elasticsearch/json, https://pypi.org/pypi/elasticsearch/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
