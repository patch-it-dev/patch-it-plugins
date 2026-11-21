# OpenAI: saved copies and goldens

Real copies of what OpenAI publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/openai -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved model deprecations pages

The deprecations page, as markdown (the page's URL plus `.md`), read by the golden test through the whole `notices` plugin.

**License:** excerpts for testing; © OpenAI. Not covered by this repo's license.

**Trimmed.** Each file keeps the page title and, for every retirement table that names at least one model the plugin keeps (text models code sends), the section heading above it and the whole table, copied unchanged. Prose, notice-period policy, endpoint tables and tables of only image, audio or embedding models were left out. `retire.Parse` returns exactly the same retirements from the trimmed and the full pages.

`openai-deprecations.md` is the page today, read by `golden/openai-models.json`. `openai-deprecations-2026-10-13.md` is the page's layout before 2026-11 (one table per announcement, no Status or Notice date column), kept as a case while it still reads: `layouts/openai-models-2026-10-13.json` is its events.

| File | URL | Fetched | Quote |
|---|---|---|---|
| `openai-deprecations.md` | https://developers.openai.com/api/docs/deprecations.md | 2026-11-21 | "\| **Dec 11, 2026** \| `gpt-5`, `gpt-5-2025-08-07` \| Deprecated \| `gpt-6.1-sol` \| Jun 11, 2026 \|" |
| `openai-deprecations-2026-10-13.md` | https://developers.openai.com/api/docs/deprecations.md | 2026-10-13 | "\| October 23, 2026 \| `gpt-4-0613` \\\| `gpt-4`, … \| `gpt-5.6-sol` \|" |

`openai-deprecations-2026-11-21.md` is the page read by `golden/openai-deprecations.json` (the endpoint rows), saved with `curl` the day the source was built.

| File | URL | Fetched | Quote |
|---|---|---|---|
| `openai-deprecations-2026-11-21.md` | https://developers.openai.com/api/docs/deprecations.md | 2026-11-21 | "\| **Nov 30, 2026** \| `/v1/prompts` and reusable prompt objects \| Deprecated \|" |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the Python package's own pair, 1.1.1 → 1.1.2, from the files at the top level of `testdata/`).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `openai-1.1.1-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/f7/86/ea99e3f966a639a3256e2f79d78cd63d45d5eb651a5457ccbdcfd9eed24a/openai-1.1.1-py3-none-any.whl | 2026-11-20 | Apache-2.0 (openai-python; the wheel's `licenses/LICENSE` is kept in it). Trimmed: only `openai/__init__.py`, `openai/types/__init__.py`, every file under `openai/types/beta/`, and the dist-info `METADATA`, `WHEEL` and `LICENSE`, each copied byte for byte (46 of the wheel's files); the plugin reads the same symbols under `openai.types.beta` from it as from the whole wheel. | `class AsssitantDeleted(BaseModel):` (`openai/types/beta/asssitant_deleted.py`, line 10) |
| `openai-1.1.2-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/dd/82/b92f73453ea318c0d46f31aeb56e9d94a42606c010fb72a513f4a3cd4bac/openai-1.1.2-py3-none-any.whl | 2026-11-20 | The same, trimmed the same way. | `from .assistant_deleted import AssistantDeleted as AssistantDeleted` (`openai/types/beta/__init__.py`) |
| `pypi-openai-excerpt.json` | https://pypi.org/pypi/openai/json | 2026-11-20 | PyPI metadata. Trimmed: `releases` to 1.0.0rc1 (a pre-release, left out) and 1.1.0 to 1.1.2, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked` | |
| `pypi-openai-1.1.1.json`, `pypi-openai-1.1.2.json` | https://pypi.org/pypi/openai/1.1.1/json, …/1.1.2/json | 2026-11-20 | PyPI metadata. Trimmed: `info.name`, `info.version`, and `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/openai/`: `openai-7.59.0.trimmed.tgz`, `openai-7.60.0.trimmed.tgz`, `openai-7.61.0.trimmed.tgz` | https://registry.npmjs.org/openai/-/openai-7.59.0.tgz; https://registry.npmjs.org/openai/-/openai-7.60.0.tgz; https://registry.npmjs.org/openai/-/openai-7.61.0.tgz | 2026-11-21 | Apache-2.0 (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `resources/beta.d.mts`, `resources/beta/agents.d.mts`, `resources/beta/agents/agents.d.mts`, `resources/beta/agents/index.d.mts`, `resources/beta/index.d.mts`, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `npm/openai/`: `npm-openai-excerpt.json` and `npm-openai-7.59.0.json`, `npm-openai-7.60.0.json`, `npm-openai-7.61.0.json` | https://registry.npmjs.org/openai, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |
