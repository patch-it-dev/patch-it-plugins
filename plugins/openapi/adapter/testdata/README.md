# Recorded API responses

The version-listing tests (`versions_test.go`) read these instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/github-versions.json` | `https://api.github.com/versions` | whole |
| `versions/gitlab-tags-1.json` | `https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/repository/tags?order_by=version&sort=desc&per_page=100&page=1` | all 100 tags; `name` and `commit.created_at` |
| `versions/gitlab-tags-2.json` | the same, `page=2` | 14 tags (18.10.0, 18.10.1, 18.9.x, 18.1.4); same fields |
| `versions/es-releases.json` | `https://api.github.com/repos/elastic/elasticsearch/releases?per_page=100` | all 100; `tag_name`, `prerelease`, `draft` |
| `versions/es-branches.json` | `https://api.github.com/repos/elastic/elasticsearch-specification/git/matching-refs/heads/` | the minor branches, `main` and `gh-pages` (33 of 250) |
| `versions/sentry-api-releases.json` | `https://api.github.com/repos/getsentry/sentry/releases?per_page=100&page=1` | the 19 `api@` releases; `tag_name`, `draft`, `prerelease`, `published_at`, asset names |
| `versions/sentry-schema-releases.json` | `https://api.github.com/repos/getsentry/sentry-api-schema/releases` (pages 1-4) | 2.3.2 and 2.4.0; same fields |
| `versions/algolia-refs.json` | `https://api.github.com/repos/algolia/api-clients-automation/git/matching-refs/tags/released-` | 9 of 186 tags, five of them on two days with several releases |
| `versions/algolia-commit-*.json` | `https://api.github.com/repos/algolia/api-clients-automation/commits/<tag>` | `sha` and the committer date |
| `versions/appwrite-commits.json` | `https://api.github.com/repos/appwrite/specs/commits?path=specs&per_page=100&page=1` | the 40 newest commits; `sha`, `html_url`, the committer date |

Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.
