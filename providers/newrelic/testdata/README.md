# New Relic: saved copies and goldens

Real copies of what New Relic publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/newrelic -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved removed-members guide

Read through the `migrationguide` kind's removed-members reader; the golden file is what it stores from the guide.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `python-agent-110000-f0b4f817.mdx` | https://github.com/newrelic/docs-website/blob/f0b4f817bf9a7b76a78e26609369a8a7ce764c8e/src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx (the repo has no tags; the file's latest commit) | 2026-11-19 | Excerpt for testing; © New Relic, CC BY-NC-SA 4.0. Kept: the "## Removed features" heading, the bullets that introduce its two tables, the "Legacy API" table and the "Legacy Setting" table (lines 30-31 and 36-66 of 136), every line unchanged. Left out: the front matter, "Notes", the Python 3.7 bullet, and the "Deprecated features", "New features", "Bug fixes" and "Support statement" sections. The plugin finds the same 8 rows and 7 events in the excerpt as in the full file | "The following APIs have been removed:", with rows such as "\| `record_exception` \| [`notice_error()`](/docs/agents/python-agent/python-agent-api/notice_error) \|" |
| `github-commits-python-agent-110000.json` | https://api.github.com/repos/newrelic/docs-website/commits?path=src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx&per_page=1 (shape only) | written for the test, 2026-11-20 | not a saved copy: GitHub's commit list cut to the one field read (`sha`), naming the commit the note above was fetched at | for the guide's ref: the newest commit that changed the note |
| `github-commits-none.json` | the same request for a major with no note (shape only) | written for the test, 2026-11-20 | GitHub's answer for a path no commit changed: an empty list | for the "no guide, nothing to read" test |
