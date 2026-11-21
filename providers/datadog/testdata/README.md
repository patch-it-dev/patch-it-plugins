# Datadog: saved copies and goldens

Real copies of what Datadog publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/datadog -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `datadog-2.60.0.yaml`, `datadog-2.61.0.yaml` | https://raw.githubusercontent.com/DataDog/datadog-api-client-python/2.60.0/.generator/schemas/v2/openapi.yaml (and `2.61.0`) | 2026-11-13 | Apache-2.0 (datadog-api-client-python) | `queried_tags_window_seconds:` (in 2.60.0, gone in 2.61.0) |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/ddtrace/`: `ddtrace-4.12.3-cp310-cp310-macosx_14_0_arm64.trimmed.whl`, `ddtrace-4.13.0-cp310-cp310-macosx_14_0_arm64.trimmed.whl`, `ddtrace-4.16.0-cp310-cp310-macosx_14_0_arm64.trimmed.whl`, `ddtrace-4.16.1-cp310-cp310-macosx_14_0_arm64.trimmed.whl` | https://files.pythonhosted.org/packages/d7/18/90b574aa3c046bd9f4862b0353d8f8a42aeba504c601c3235bd8d91f057f/ddtrace-4.12.3-cp310-cp310-macosx_14_0_arm64.whl; https://files.pythonhosted.org/packages/52/97/9d4ec43fa15991f195e62795e1d46fb158cab3df29c370d272889b8f8640/ddtrace-4.13.0-cp310-cp310-macosx_14_0_arm64.whl; https://files.pythonhosted.org/packages/6d/af/859a2d709008db2c95842aaab04f0d242a9b338cf8279f655b0f5946c9df/ddtrace-4.16.0-cp310-cp310-macosx_14_0_arm64.whl; https://files.pythonhosted.org/packages/ac/0a/ccaa73a0d201d570b3b4a3e9ef8ff1617056d552dbd3d4f81292ee7af717/ddtrace-4.16.1-cp310-cp310-macosx_14_0_arm64.whl | 2026-11-21 | Apache-2.0 or BSD-3-Clause (the wheel's own license files are kept in it). Trimmed: only `ddtrace/appsec/ai_guard/__init__.py`, `ddtrace/appsec/ai_guard/_api_client.py`, `ddtrace/appsec/ai_guard/integrations/__init__.py`, `ddtrace/appsec/ai_guard/integrations/litellm.py`, `ddtrace/appsec/ai_guard/integrations/strands.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/ddtrace/`: `pypi-ddtrace-excerpt.json` and `pypi-ddtrace-4.12.3.json`, `pypi-ddtrace-4.13.0.json`, `pypi-ddtrace-4.16.0.json`, `pypi-ddtrace-4.16.1.json` | https://pypi.org/pypi/ddtrace/json, https://pypi.org/pypi/ddtrace/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
