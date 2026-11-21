# GitHub: saved copies and goldens

Real copies of what GitHub publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/github -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `github-2022-11-28.json`, `github-2026-03-10.json` | https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.2022-11-28.json (and `.2026-03-10.json`) | 2026-11-14 | MIT (github/rest-api-description) | "Remove deprecated rate property from rate limit endpoint" (GitHub's breaking-changes page for 2026-03-10) |

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/github-versions.json` | `https://api.github.com/versions` | whole |

## Saved feed

A real copy, saved once with `curl`; the golden test reads it through the whole `notices` plugin.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `github-changelog-actions.rss.xml` | https://github.blog/changelog/label/actions/feed/ (RSS 2.0; `ETag` and `Last-Modified` sent) | 2026-11-14 08:27 UTC | Excerpt for testing; © GitHub, Inc. | "The macOS 14 runner image will be retired on November 2, 2026." |

**Trimmed:** `github-changelog-actions.rss.xml` keeps the channel header and the three items the `github-actions` reader acts on (the macOS 14 retirement, which becomes events, and the two marked entries it drops: the ARM32 retirement and the `pull_request_target` change), copied unchanged. The seven unmarked items were left out; the reader ignores them, so its findings and the golden file are unchanged.

## Saved catalog snapshots

Read through the `catalog` kind; the golden file is what it stores from them.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `github-graphql-upcoming-changes-<sha>.json` (seven) | https://github.com/github/docs/blob/<sha>/src/graphql/data/fpt/upcoming-changes.json at fc26275f, 54c26c43, 2706f417 (2026-01-05 to 08) and fc7cedfe, 3a14e541, a934f06d, c5738e58 (2026-02-13 to 06-23) | 2026-11-21 | CC BY 4.0 (github/docs content). Excerpts for testing: each keeps the first entry under every date of the first file of its group, and every entry the group's commits add; every kept entry unchanged, in the file's order. The plugin finds the same additions between them as between the full files (2, 1, 1; and 17 re-added, past-dated) | `"location": "CheckAnnotation.databaseId", "description": "<p><code>databaseId</code> will be removed. Use <code>fullDatabaseId</code> instead.</p>"` |
| `github-commits-upcoming-changes-excerpt.json` | https://api.github.com/repos/github/docs/commits?path=src/graphql/data/fpt/upcoming-changes.json (shape only) | written for the test, 2026-11-21 | not a saved copy: three of the real commits (SHA, date, URL), newest first as GitHub lists them | for the version-listing test only |

## SDK release notes

Read with `Plugin{File: ...}` and `Notes: true` (`kinds/changelog/sdks.go`): an SDK whose changelog is its GitHub releases' notes. Each file is one release as GitHub's API returns it, trimmed to the fields the reader uses.

| File | URL | Tag commit | Saved | License; trimming | Use |
|---|---|---|---|---|---|
| `github-octokit-rest-18.0.0-release.json` | https://api.github.com/repos/octokit/rest.js/releases/tags/v18.0.0 | `b4ab0db` | 2026-11-20 | MIT (rest.js); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | golden |
| `github-octokit-rest-17.11.2-release.json` | https://api.github.com/repos/octokit/rest.js/releases/tags/v17.11.2 | `5306f37` | 2026-11-20 | MIT (rest.js); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | drop test |
| `github-octokit-net-0.5.2-release.json` | https://api.github.com/repos/octokit/octokit.net/releases/tags/v0.5.2 | `4b61105` | 2026-11-20 | MIT (octokit.net); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | golden |
| `github-octokit-net-0.5.1-release.json` | https://api.github.com/repos/octokit/octokit.net/releases/tags/v0.5.1 | `8a9e828` | 2026-11-20 | MIT (octokit.net); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | drop test |
