# Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of a provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `datadog-2.60.0.yaml`, `datadog-2.61.0.yaml` | https://raw.githubusercontent.com/DataDog/datadog-api-client-python/2.60.0/.generator/schemas/v2/openapi.yaml (and `2.61.0`) | 2026-11-13 | Apache-2.0 (datadog-api-client-python) | `queried_tags_window_seconds:` (in 2.60.0, gone in 2.61.0) |
| `github-2022-11-28.json`, `github-2026-03-10.json` | https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.2022-11-28.json (and `.2026-03-10.json`) | 2026-11-14 | MIT (github/rest-api-description) | "Remove deprecated rate property from rate limit endpoint" (GitHub's breaking-changes page for 2026-03-10) |
| `gitlab-v18.11.0-ee.yaml`, `gitlab-v19.0.0-ee.yaml` | https://gitlab.com/gitlab-org/gitlab/-/raw/v18.11.0-ee/doc/api/openapi/openapi_v3.yaml (and `v19.0.0-ee`) | 2026-11-15 | CC BY-SA 4.0 (GitLab's `doc/` directory) | `bitbucket_app_password:` (in v18.11.0-ee, gone in v19.0.0-ee) |
| `vercel-v1.13.4.json`, `vercel-v1.13.5.json` | https://raw.githubusercontent.com/vercel/sdk/v1.13.4/vercel-spec.json (and `v1.13.5`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | "Check the price for a domain (deprecated)" (summary of `GET /v4/domains/price`, new in v1.13.5) |
| `vercel-v1.18.6.json`, `vercel-v1.18.7.json` | https://raw.githubusercontent.com/vercel/sdk/v1.18.6/vercel-spec.json (and `v1.18.7`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v4/domains/price": {` (in v1.18.6, gone in v1.18.7) |
| `vercel-v1.6.2.json`, `vercel-v1.6.3.json` | https://raw.githubusercontent.com/vercel/sdk/v1.6.2/vercel-spec.json (and `v1.6.3`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v1/log-drains": {` (in v1.6.2, gone in v1.6.3 without being deprecated, back in v1.15.0): the drop test |
| `vercel-v1.28.64.json`, `vercel-v1.28.65.json` | https://raw.githubusercontent.com/vercel/sdk/v1.28.64/vercel-spec.json (and `v1.28.65`) | 2026-11-18 | Apache-2.0 (vercel/sdk) | `"/v2/sandboxes/drives/{nameOrId}": {` (`DELETE` moved here from `.../{name}` in v1.28.65) |

Stripe's spec isn't saved here: its tests use small specs written inline.
