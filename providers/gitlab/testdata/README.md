# GitLab: saved copies and goldens

Real copies of what GitLab publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/gitlab -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `gitlab-v18.11.0-ee.yaml`, `gitlab-v19.0.0-ee.yaml` | https://gitlab.com/gitlab-org/gitlab/-/raw/v18.11.0-ee/doc/api/openapi/openapi_v3.yaml (and `v19.0.0-ee`) | 2026-11-15 | CC BY-SA 4.0 (GitLab's `doc/` directory) | `bitbucket_app_password:` (in v18.11.0-ee, gone in v19.0.0-ee) |

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/gitlab-tags-1.json` | `https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/repository/tags?order_by=version&sort=desc&per_page=100&page=1` | all 100 tags; `name` and `commit.created_at` |
| `versions/gitlab-tags-2.json` | the same, `page=2` | 14 tags (18.10.0, 18.10.1, 18.9.x, 18.1.4); same fields |
