# Version lists

What the kind's own tests read; each provider's saved changelogs are in its folder, `providers/<provider>/testdata/`.

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `versions/stripe-go-tags.json`, `versions/octokit-rest-releases.json` | https://api.github.com/repos/stripe/stripe-go/tags, https://api.github.com/repos/octokit/rest.js/releases (shape only) | written for the test, 2026-11-20 | not saved copies: a few real tag names and releases in the shape of GitHub's API (a pre-release, a draft and a tag of another series among them) | for the version-listing test only (tags for an ecosystem with no registry lister, releases for release notes); no golden reads them |
