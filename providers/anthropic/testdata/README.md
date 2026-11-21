# Anthropic: saved copies and goldens

Real copies of what Anthropic publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/anthropic -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved model deprecations pages

The deprecations page, as markdown (the page's URL plus `.md`), read by the golden test through the whole `notices` plugin.

**License:** excerpts for testing; © Anthropic. Not covered by this repo's license.

| File | URL | Fetched | Quote |
|---|---|---|---|
| `anthropic-deprecations.md` | https://platform.claude.com/docs/en/about-claude/model-deprecations.md | 2026-10-13 | "\| November 30, 2026 \| `claude-sonnet-4-5-20250929` \| `claude-sonnet-5-5` \|" |

## Saved SDK migration guide

Read through the `migrationguide` kind's SDK-guide reader; `golden/anthropic-1.0.0.json` is what it stores from MIGRATION.md at v1.0.0.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `anthropic-MIGRATION-v1.0.0.md` | https://github.com/anthropics/anthropic-sdk-python/blob/v1.0.0/MIGRATION.md | 2026-10-17 | MIT (anthropic-sdk-python), kept whole | "Removed: deprecated request parameters" |

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `pypi-anthropic-excerpt.json` | https://pypi.org/pypi/anthropic/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/anthropic/`: `anthropic-2.4.0-py3-none-any.trimmed.whl`, `anthropic-2.5.0-py3-none-any.trimmed.whl`, `anthropic-2.6.0-py3-none-any.trimmed.whl`, `anthropic-2.7.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/8b/5d/e399d9028411ce404d7ddbe6b8060d878b2549ed8cd671a7bb496ae20ffc/anthropic-2.4.0-py3-none-any.whl; https://files.pythonhosted.org/packages/f4/10/c4e6f015111c702c2d5a386ad57745b83c79c08eae68e541dbb16b5e3965/anthropic-2.5.0-py3-none-any.whl; https://files.pythonhosted.org/packages/38/0b/705a05ab68558914e725cf1143d7e409fa3bb596c35bca1eceab42402abe/anthropic-2.6.0-py3-none-any.whl; https://files.pythonhosted.org/packages/8a/da/ed144ab553eb2bc7b5febbfb37858f28b33ec332c37f0fdf260012ae3fe2/anthropic-2.7.0-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `anthropic/__init__.py`, `anthropic/_exceptions.py`, `anthropic/types/__init__.py`, `anthropic/types/beta/__init__.py`, `anthropic/types/beta/beta_billing_frozen_error.py`, `anthropic/types/beta/beta_billing_frozen_error_details.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/anthropic/`: `pypi-anthropic-excerpt.json` and `pypi-anthropic-2.4.0.json`, `pypi-anthropic-2.5.0.json`, `pypi-anthropic-2.6.0.json`, `pypi-anthropic-2.7.0.json` | https://pypi.org/pypi/anthropic/json, https://pypi.org/pypi/anthropic/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/anthropic-ai-sdk/`: `sdk-1.6.0.trimmed.tgz`, `sdk-1.7.0.trimmed.tgz`, `sdk-1.8.0.trimmed.tgz` | https://registry.npmjs.org/@anthropic-ai/sdk/-/sdk-1.6.0.tgz; https://registry.npmjs.org/@anthropic-ai/sdk/-/sdk-1.7.0.tgz; https://registry.npmjs.org/@anthropic-ai/sdk/-/sdk-1.8.0.tgz | 2026-11-21 | MIT (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `client.d.mts`, `index.d.mts`, `resources/beta.d.mts`, `resources/beta/beta.d.mts`, `resources/beta/index.d.mts`, `resources/beta/organization.d.mts`, `resources/beta/organization/index.d.mts`, `resources/beta/organization/organization.d.mts`, `resources/index.d.mts`, each copied byte for byte | see the source's page under `providers/` |
| `npm/anthropic-ai-sdk/`: `npm-anthropic-ai-sdk-excerpt.json` and `npm-anthropic-ai-sdk-1.6.0.json`, `npm-anthropic-ai-sdk-1.7.0.json`, `npm-anthropic-ai-sdk-1.8.0.json` | https://registry.npmjs.org/@anthropic-ai%2Fsdk, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |
