# Salesforce: saved copies and goldens

Real copies of what Salesforce publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/salesforce -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved end-of-life page

A real copy; the golden test reads it through the whole `notices` plugin.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `salesforce-api-rest-eol-2026-10-31.html` | https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html | 2026-11-20 (the page itself as crawled 2026-10-31; the live page 403s a plain fetch, so this is the newest Wayback Machine snapshot, checked 2026-11-20) | Excerpt for testing; © Salesforce. Trimmed to the policy paragraphs and the version table, every line copied unchanged (its internal links' Wayback rewrite prefix removed, to read as the live page serves them); the surrounding navigation and page chrome left out. | "As of Summer ’25, these versions are retired and unavailable." (Versions 21.0 through 30.0) |
