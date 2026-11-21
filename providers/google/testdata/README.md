# Google: saved copies and goldens

Real copies of what Google publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/google -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved feed

A real copy, saved once with `curl`; the golden test reads it through the whole `notices` plugin.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `vertex-ai-generative-ai.atom.xml` | https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml (Atom; `https://cloud.google.com/feeds/...` 301s here; `Last-Modified` sent) | 2026-11-14 08:27 UTC | CC BY 4.0 (Google Cloud documentation), kept whole | "Vertex AI Extensions is deprecated and will be shut down after November 26, 2026." |

## Saved model deprecations pages

The deprecations and model-versions pages, read by the golden tests through the whole `notices` plugin.

**License:** © Google, CC BY 4.0 (Google's developer documentation).

| File | URL | Fetched | Quote |
|---|---|---|---|
| `gemini-deprecations.md` | https://aistudio.google.com/docs/deprecations (where https://ai.google.dev/gemini-api/docs/deprecations redirects; served as Markdown). Kept whole, byte for byte | 2026-11-21 | "\| `gemini-2.0-flash` \| February 5, 2025 \| June 1, 2026 \| `gemini-3.6-flash` \|", as Markdoc `* cell` lines |
| `vertex-model-versions.html` | https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions. Excerpt: every `<h2>`/`<h3>` line and every line of every `<table>` (982 of 10,074 lines), unchanged; scripts, navigation and prose left out. `retire.Parse` finds the same 39 retirements in the excerpt as in the full page | 2026-11-21 | "&lt;td&gt;June 30, 2026&lt;/td&gt;" in the "Retired models" table |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/google-genai/`: `google_genai-0.8.0-py3-none-any.trimmed.whl`, `google_genai-1.0.0-py3-none-any.trimmed.whl`, `google_genai-3.3.0-py3-none-any.trimmed.whl`, `google_genai-3.4.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/74/6d/c5b1757ffe28fdfb38df3fda79c614ec840ebac0b72a3fcbe2ed969e254d/google_genai-0.8.0-py3-none-any.whl; https://files.pythonhosted.org/packages/24/9d/63dbd2b6c630f44cbbf09c4e04b4c9012da01f6e585d34ae53d07931bb67/google_genai-1.0.0-py3-none-any.whl; https://files.pythonhosted.org/packages/ae/e4/619de4d0f985cfafc7a420339796279716aba5c6cac362ab4f97b856279f/google_genai-3.3.0-py3-none-any.whl; https://files.pythonhosted.org/packages/2b/7a/daed011805af7b3ef07c18d4ea8e9534891c7dfeeac438b0d22361f65e0a/google_genai-3.4.0-py3-none-any.whl | 2026-11-21 | Apache-2.0 (the wheel's own license files are kept in it). Trimmed: only `google/genai/__init__.py`, `google/genai/interactions.py`, `google/genai/types.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/google-genai/`: `pypi-google-genai-excerpt.json` and `pypi-google-genai-0.8.0.json`, `pypi-google-genai-1.0.0.json`, `pypi-google-genai-3.3.0.json`, `pypi-google-genai-3.4.0.json` | https://pypi.org/pypi/google-genai/json, https://pypi.org/pypi/google-genai/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/google-genai/`: `genai-0.15.0.trimmed.tgz`, `genai-1.0.0.trimmed.tgz`, `genai-1.1.0.trimmed.tgz` | https://registry.npmjs.org/@google/genai/-/genai-0.15.0.tgz; https://registry.npmjs.org/@google/genai/-/genai-1.0.0.tgz; https://registry.npmjs.org/@google/genai/-/genai-1.1.0.tgz | 2026-11-21 | Apache-2.0 (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `dist/genai.d.ts`, `dist/node/node.d.ts`, `dist/web/web.d.ts`, each copied byte for byte | see the source's page under `providers/` |
| `npm/google-genai/`: `npm-google-genai-excerpt.json` and `npm-google-genai-0.15.0.json`, `npm-google-genai-1.0.0.json`, `npm-google-genai-1.1.0.json` | https://registry.npmjs.org/@google%2Fgenai, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |
