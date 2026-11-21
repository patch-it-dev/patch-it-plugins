# PlanetScale: saved copies and goldens

Real copies of what PlanetScale publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/planetscale -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `planetscale-e631095a.yaml`, `planetscale-fc13e09d.yaml` | https://raw.githubusercontent.com/planetscale/terraform-provider-planetscale/e631095aa649a6797578a1d51d861b91e0568aa1/schemas/openapi.yaml (and `fc13e09d0c18b893aabd3e02d175d13957bba677`) | 2026-11-19 | none stated for the file (its `info` says "&copy; 2026 PlanetScale, Inc."); the repo, planetscale/terraform-provider-planetscale, is Apache-2.0. Excerpt for testing; © PlanetScale | "Deprecated. Use GET /organizations/:organization/databases/:database/branches/:branch/change-requests instead." (`GET .../neki-changes`, new in `fc13e09d`) |
| `planetscale-b49c3540.yaml`, `planetscale-f6d6ad4d.yaml` | https://raw.githubusercontent.com/planetscale/terraform-provider-planetscale/b49c354080667638b79781801f0517ae07bc55b8/schemas/openapi.yaml (and `f6d6ad4d3929f6c1ef021eb13c1541ba31f72107`) | 2026-11-19 | none stated for the file (its `info` says "&copy; 2026 PlanetScale, Inc."); the repo, planetscale/terraform-provider-planetscale, is Apache-2.0. Excerpt for testing; © PlanetScale | `/organizations/{organization}/databases/{database}/branches/{branch}/read-only-replicas:` (in `b49c3540`, never deprecated, gone in `f6d6ad4d`, where `.../dedicated-read-replicas:` is): the drop test |

The PlanetScale copies are trimmed by lines: everything before `paths:` (`openapi`, `info`), then the path items involved, whole, and the two-line `servers` block (`https://api.planetscale.com/v1`, the same in every version), every line copied unchanged; the other paths, `tags`, `security`, `x-readme` and `components` left out. The file states no license, so only what the tests need is kept. The golden pair gives the full files' 5 deprecations (their only ones), the drop-test pair their 7 removals (their only ones). The tests serve them as GitHub would, with each commit's date.
