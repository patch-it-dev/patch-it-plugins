# Neon: saved copies and goldens

Real copies of what Neon publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/neon -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `neon-5c57d009.json`, `neon-93b93dc9.json` | https://raw.githubusercontent.com/neondatabase/neon-pkgs/5c57d0095a9a4594a3c8b09d499b132902e7c42e/packages/sdk/spec/neon-openapi.json (and `93b93dc9e4d1b4ab48dac6c8ccba76b35541b476`) | 2026-11-19 | Excerpt for testing; © Neon. neondatabase/neon-pkgs is Apache-2.0, but the spec's own `info.license` says "Proprietary" | "Deprecated: to test a branch's already-saved configuration, use `sendNeonAuthEmailProviderTest`" (`POST /projects/{project_id}/branches/{branch_id}/auth/send_test_email`, new in `93b93dc9`) |
| `neon-4c6b3be4.json`, `neon-4497de89.json` | https://raw.githubusercontent.com/neondatabase/neon-pkgs/4c6b3be46b1b333e7f910db51218ad144de27179/packages/sdk/spec/neon-openapi.json (and `4497de89a5e84b38956fde686d271f99317b8fdb`) | 2026-11-19 | Excerpt for testing; © Neon. neondatabase/neon-pkgs is Apache-2.0, but the spec's own `info.license` says "Proprietary" | `"/projects/{project_id}/branches/{branch_id}/recover": {` (in `4c6b3be4`, gone in `4497de89` without being deprecated): the drop test |
| `neon-api-client-2.7.2.yaml`, `neon-api-client-2.7.3.yaml` | `public-v2-for-users.gen.yaml` in https://registry.npmjs.org/@neondatabase/api-client/-/api-client-2.7.2.tgz (and `api-client-2.7.3.tgz`) | 2026-11-19 | Excerpt for testing; © Neon. The package is MIT, but the spec's own `info.license` says "Proprietary" | "**Deprecated.** Use `GET /consumption_history/projects` (`getConsumptionHistoryPerProject`)" (`GET /consumption_history/account`, in 2.7.2, gone in 2.7.3) |

The Neon copies are excerpts, not whole files: the spec says its license is "Proprietary" whatever the license of the repo or package that ships it. The JSON copies keep `openapi`, `servers`, `info` (without its `contact`) and the path items involved, whole and in their original order, every line copied unchanged (bar the trailing comma after a member that's now last); `components` left out. The YAML copies keep, by lines, `openapi`, `servers`, `info`'s `title`, `version` and `license`, and the path items involved; they're a different source from the plugin's (the npm package @neondatabase/api-client, succeeded by @neon/sdk and not released since 2.7.3), used only to check that the removal of an operation marked deprecated is kept.

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `npm/neondatabase-serverless/`: `serverless-0.10.4.trimmed.tgz`, `serverless-1.0.0.trimmed.tgz`, `serverless-1.0.1.trimmed.tgz` | https://registry.npmjs.org/@neondatabase/serverless/-/serverless-0.10.4.tgz; https://registry.npmjs.org/@neondatabase/serverless/-/serverless-1.0.0.tgz; https://registry.npmjs.org/@neondatabase/serverless/-/serverless-1.0.1.tgz | 2026-11-21 | MIT (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `index.d.mts`, `index.d.ts`, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `npm/neondatabase-serverless/`: `npm-neondatabase-serverless-excerpt.json` and `npm-neondatabase-serverless-0.10.4.json`, `npm-neondatabase-serverless-1.0.0.json`, `npm-neondatabase-serverless-1.0.1.json` | https://registry.npmjs.org/@neondatabase%2Fserverless, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |
