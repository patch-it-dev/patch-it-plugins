# Flagsmith: saved copies and goldens

Real copies of what Flagsmith publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/flagsmith -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `flagsmith-v2.286.0.yaml`, `flagsmith-v2.287.0.yaml` | https://raw.githubusercontent.com/Flagsmith/flagsmith/v2.286.0/openapi.yaml (and `v2.287.0`) | 2026-11-19 | BSD-3-Clause (Flagsmith/flagsmith; also the spec's own `info.license`) | "**EXPERIMENTAL ENDPOINT** - Subject to change without notice." (`POST .../delete-segment-override/` and two more, removed in `v2.287.0`) |
| `flagsmith-desc-c3ef832.yaml`, `flagsmith-desc-f504f16.yaml` | https://raw.githubusercontent.com/Flagsmith/flagsmith/c3ef83244bfb4b1d65ab6fc5ff45412659dd61f3/openapi.yaml (and `f504f160dcf40b16baace0506a04dfda8fd6332c`, commits not tags) | 2026-11-19 | BSD-3-Clause (Flagsmith/flagsmith; also the spec's own `info.license`) | "Short-lived access token for the Admin API." (in `c3ef832`), "...for the Management API." (in `f504f16`, the same commit as "feat: rename admin api to management" #8464): the drop test |

The Flagsmith copies are trimmed by lines: `openapi`, `info` and `servers`, then the path items involved (`paths: {}` where none remain, the one line in `v2.287.0.yaml` that isn't in the original) and, under `components: schemas:`, every schema those paths' request bodies reach, transitively, whole and in original order, every line copied unchanged; `tags` and `security` left out. `FeatureValue`, `FeatureValueTypeEnum` and `MultivariateValue` are kept unchanged in both files because the real spec still has them at `v2.287.0` (other, unremoved operations use them too); every other schema the removed paths reached (`DeleteSegmentOverride`, `EnvironmentDefault`, `FeatureIdentifier`, `FeatureUpdateSegmentData`, `SegmentIdentifier`, `SegmentOverride`, `UpdateFlag`, `UpdateFlagV2`) really is gone in `v2.287.0`, confirmed against the full file. The drop-test pair is two commits, not tags, kept whole but for the same trim (`TokenExchangeResponse` and the `Master API Key` security scheme, the only two places that commit touched `openapi.yaml`).
