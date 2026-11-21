# Vercel: saved copies and goldens

Real copies of what Vercel publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/vercel -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `vercel-v1.13.4.json`, `vercel-v1.13.5.json` | https://raw.githubusercontent.com/vercel/sdk/v1.13.4/vercel-spec.json (and `v1.13.5`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | "Check the price for a domain (deprecated)" (summary of `GET /v4/domains/price`, new in v1.13.5) |
| `vercel-v1.18.6.json`, `vercel-v1.18.7.json` | https://raw.githubusercontent.com/vercel/sdk/v1.18.6/vercel-spec.json (and `v1.18.7`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v4/domains/price": {` (in v1.18.6, gone in v1.18.7) |
| `vercel-v1.6.2.json`, `vercel-v1.6.3.json` | https://raw.githubusercontent.com/vercel/sdk/v1.6.2/vercel-spec.json (and `v1.6.3`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v1/log-drains": {` (in v1.6.2, gone in v1.6.3 without being deprecated, back in v1.15.0): the drop test |
| `vercel-v1.28.64.json`, `vercel-v1.28.65.json` | https://raw.githubusercontent.com/vercel/sdk/v1.28.64/vercel-spec.json (and `v1.28.65`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v2/sandboxes/drives/{nameOrId}": {` (`DELETE` moved here from `.../{name}` in v1.28.65) |
