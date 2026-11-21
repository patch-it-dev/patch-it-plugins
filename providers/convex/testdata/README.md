# Convex: saved copies and goldens

Real copies of what Convex publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/convex -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `convex-f35879bb.json`, `convex-5e5159cb.json` | https://raw.githubusercontent.com/convex/convex-backend/f35879bb396491942bf8ecbc8ef5e4280499f89f/npm-packages/@convex-dev/platform/deployment-openapi.json (and `5e5159cbea5188e9da5400febda1b031a233eea0`) | 2026-11-19 | FSL-1.1-Apache-2.0 (convex/convex-backend `LICENSE.md`; the spec's own `info.license` is `LicenseRef-FSL-1.1-Apache-2.0`): excerpt for testing, © Convex, Inc.; maintainers, please confirm | "Deprecated: use `GET /integrations` instead." (`GET /list_log_streams`, and five more log stream operations, new in `5e5159cb`) |
| `convex-3f6edf7a.json`, `convex-3ead9091.json` | https://raw.githubusercontent.com/convex/convex-backend/3f6edf7ab8f3f98e027f317633246a19348eabcc/npm-packages/@convex-dev/platform/deployment-openapi.json (and `3ead909103b7f5897f529c031e80991ad174b07e`) | 2026-11-19 | FSL-1.1-Apache-2.0 (convex/convex-backend `LICENSE.md`; the spec's own `info.license` is `LicenseRef-FSL-1.1-Apache-2.0`): excerpt for testing, © Convex, Inc.; maintainers, please confirm | `"name": {` in `UsageLimitConfigRequest` and `UsageLimitConfigResponse` (in `3f6edf7a`, gone in `3ead9091` without being deprecated): the drop test |

The Convex copies keep `openapi`, `info`, `servers` and the path items involved, whole and in their original order, every line copied unchanged (bar the trailing comma after a path item or schema that's now last); the usage-limit pair also keeps the component schemas those operations reach, the log stream pair none. The source is FSL-1.1-Apache-2.0 (each version becomes Apache-2.0 two years after it's published), which allows copying for a permitted purpose; these are short excerpts kept for testing, but a maintainer should confirm that before release. Each pair gives the same changes as the full files, with and without the filter. The management API spec (`management-openapi.json`, `info.license` `LicenseRef-Convex`, not defined anywhere in the repo) is neither read nor saved.

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `npm/convex/`: `convex-1.35.0.trimmed.tgz`, `convex-1.35.1.trimmed.tgz`, `convex-1.42.2.trimmed.tgz`, `convex-1.42.3.trimmed.tgz` | https://registry.npmjs.org/convex/-/convex-1.35.0.tgz; https://registry.npmjs.org/convex/-/convex-1.35.1.tgz; https://registry.npmjs.org/convex/-/convex-1.42.2.tgz; https://registry.npmjs.org/convex/-/convex-1.42.3.tgz | 2026-11-21 | Apache-2.0 (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `dist/esm-types/browser/index.d.ts`, `dist/esm-types/browser/query_options.d.ts`, `dist/esm-types/react/client.d.ts`, `dist/esm-types/react/index.d.ts`, each copied byte for byte | see the source's page under `providers/` |
| `npm/convex/`: `npm-convex-excerpt.json` and `npm-convex-1.35.0.json`, `npm-convex-1.35.1.json`, `npm-convex-1.42.2.json`, `npm-convex-1.42.3.json` | https://registry.npmjs.org/convex, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |
