# GitHub

Every source patch-it reads for GitHub, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## GitHub REST API (`github/rest-api`)

### What it watches

GitHub's REST API between dated API versions (`2022-11-28` → `2026-03-10`): properties, schemas, parameters and operations removed or changed in a new version.

Kind: `openapi` (`kinds/openapi`), with its adapter, `GitHub` ([`rest_api.go`](rest_api.go)).

### Why this source

- **Source:** [github/rest-api-description](https://github.com/github/rest-api-description), one file per supported dated version: `descriptions/api.github.com/api.github.com.<version>.json` on `main`. The supported versions come from `GET https://api.github.com/versions`. Checked on 2026-11-14.
- **Why this one:** it's GitHub's own OpenAPI description, generated from the same source as docs.github.com's reference and the Octokit SDKs. A dated version's file is frozen, so two of them diff exactly.
- **Alternatives considered:** the undated `api.github.com.json` (the in-progress spec, moves several times a day with additive changes; not a version anyone pins); the `dereferenced/` variants (every `$ref` inlined, so one schema change would show up on every endpoint using it); the repo's `CHANGELOG.md` (one section per dated version listing affected endpoints; good prose, not wired as a hint source yet); the [Breaking changes](https://docs.github.com/en/rest/about-the-rest-api/breaking-changes) docs page (prose).
- **Quote:** "Remove deprecated `rate` property from rate limit endpoint." ([CHANGELOG.md](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/CHANGELOG.md), Version 2026-03-10).

### How the provider announces changes

- Clients pick a version per request with `X-GitHub-Api-Version: <date>`; without it GitHub uses `2022-11-28`.
- **Breaking changes only ship in a new dated version**, which is rare: `2022-11-28` was the only one for over three years.
- **Opt-in now, forced later**: a version keeps working for 24 months after a newer one ships, then returns `410 Gone` (with `Deprecation` / `Sunset` headers before that).
- The event's `version` is the dated version string, not `info.version` (a fixed package version, `1.1.4`).

### What the plugin reads and ignores

- Reads: the newest dated version's spec, diffed against the last one it saw. `GET /versions`' order isn't documented, so the adapter sorts the versions itself.
- The real `2022-11-28` → `2026-03-10` diff is 108 breaking events. The golden test replays a trimmed pair that keeps one of them: `rate` removed from `rate-limit-overview`.

### Known gaps

- Additive changes between dated versions aren't events (they never get a version).
- GraphQL isn't watched.

## GitHub Actions runner images (`github/actions`)

### What it watches

GitHub-hosted runner images that GitHub retires: one `runtime` event (`platform: github-actions`) per `runs-on` label the retirement removes (`macos-14`, `macos-14-large`, …), dated by the removal date.

Kind: `notices` (`kinds/notices`), with the feed reader (`kinds/notices/feed`) and its `Actions` feed ([`actions.go`](actions.go)).

### Why this source

- **Source:** the [GitHub changelog, Actions label](https://github.blog/changelog/label/actions/), as RSS: `https://github.blog/changelog/label/actions/feed/`. Checked on 2026-11-14.
- **Why this one:** it's GitHub's own changelog feed, built to be machine-read, with labels (`Retired`) as RSS categories. Retirement posts list the affected labels in a fixed shape and give the date.
- **Alternatives considered:** the [actions/runner-images](https://github.com/actions/runner-images) repo's announcement issues (detailed, but issue text varies and they're not a feed); the full GitHub changelog feed (every product, much noisier); the runner images README (says what exists now, not when it goes).
- **Quote:** "The macOS 14 runner image will be retired on November 2, 2026."
- **Anchors** (checked 2026-11-21, declared in `provider.go`): the feed's own title, "Use Case: actions - GitHub Changelog" (its 10 posts roll off). A read without them fails as a change of layout, never as nothing new.

### How the provider announces changes

- A retirement post weeks or months ahead, often with scheduled "brownouts" where jobs on the old image fail temporarily.
- **Forced**: on the date, jobs using the label stop running.
- The feed carries only the latest 10 posts, so the plugin runs daily and cites a snapshot.

### What the plugin reads and ignores

- Only posts labelled `Retired` about a runner image that list `runs-on` labels.
- **Breaking needs GitHub to say it stops on a date** ("will be retired / removed on …"); softer words ("will be deprecated on", "before …") make it a deprecation; no dated sentence and the post is dropped.
- Drops (and logs) retirements with no runner label to name (the ARM32 Linux and Windows retirement) and behavior changes (`pull_request_target` blocked by default).
- Every event cites a Wayback Machine snapshot of the feed and is read from it.

### Known gaps

- The patch-it client doesn't fix `runtime` events yet: it lists them.
- Deprecations of actions themselves (`actions/upload-artifact@v3`, …) aren't read.

## GitHub GraphQL API (`github/graphql`)

### What it watches

Removals GitHub announces for its GraphQL API: a field, an argument, an input field or an enum value, with the date it goes. One `deprecation` event per newly announced removal, its `version` the removal date.

Kind: `catalog` (`kinds/catalog`), the `UpcomingChanges` reader (`kinds/catalog/upcoming.go`).

### Why this source

- **Source:** [`src/graphql/data/fpt/upcoming-changes.json`](https://github.com/github/docs/blob/main/src/graphql/data/fpt/upcoming-changes.json) in github/docs, read at every commit that changed it (listed from GitHub's commits API, `?path=`). It's the file [Breaking changes](https://docs.github.com/en/graphql/overview/breaking-changes) on docs.github.com is built from. Checked on 2026-11-21: 1,291 entries, every one `"criticality": "breaking"`, 29 commits since 2023-05-01.
- **Why this one:** one JSON object keyed by removal date, each entry naming its location (`RepositoryPermission.TRIAGE_PLUS`), what to do and why, from GitHub's own schema tooling. A commit never changes, so the raw file at it is its own citation.
- **Alternatives considered:**
  - **The Breaking changes page** (HTML): the same entries, rendered; the JSON is what it's made from.
  - **The GraphQL schema** (`schema.docs.graphql`, `@deprecated(reason: "… Removal on 2027-04-01 UTC.")`): the same dates inside a GraphQL SDL, which needs a schema reader no kind has.
  - **The GitHub changelog**: announces some of these, in prose, with no location.
- **Quote:** `{"location": "RepositoryPermission.TRIAGE_PLUS", "description": "<p><code>TRIAGE_PLUS</code> will be removed. Use MODERATE instead.</p>", "date": "2027-04-01", "criticality": "breaking"}`.

### How the provider announces changes

- GitHub's GraphQL API has no versions: a change is announced here with its removal date, a quarter's first day (`2026-10-01`, `2027-01-01`).
- Announcements come a few at a time (one or two a month in 2026), sometimes in a batch (1,109 entries on 2025-04-14, the audit-log fields).

### What the plugin reads and ignores

- Reads every entry of the file at each commit. An entry (location and date) in the newer commit's file that the older one didn't have is a `deprecation` event: `version` the removal date (the API has no versions; the day it breaks is what a gate compares, as a model's shutdown date is), no `previous_version` (a dated notice), the hint the entry's description and reason as text. The excerpt is the entry's fields as the file writes them, its `owner` left out.
- Subject (`graphql_field`): the location's type, field and, when it names one, argument: `Query.sponsorables.dependencyEcosystem` is `type_name: Query`, `field: sponsorables`, `argument: dependencyEcosystem`; an input object's field or an enum value is its type and the field (`RepositoryPermission` / `TRIAGE_PLUS`). A location with more than three parts isn't read. A dated notice: `version` is the removal date, no `previous_version`.
- An entry a commit drops while still upcoming and a later commit restores is the same event (same removal date, same fingerprint), so it isn't announced twice.
- Ignores an entry dated before the commit that adds it: a revert putting back a removal already made, not an announcement. On 2026-01-05 the file lost 17 such entries, got them back the next day and lost them again on 2026-01-08: no event (the pinned drop test).
- Ignores an entry leaving the file: the removal shipped, or GitHub withdrew it, and the file doesn't say which.
- A first read, with no commit before it, has nothing to compare: no events.

### Known gaps

- The removal itself (the day it ships) isn't a separate `breaking` event; the deprecation names the date.
- GitHub Enterprise Server's own files (`ghes-*`) aren't read.
- A location GitHub moves to a new date is a new event (a new `version`), alongside the first.

## Octokit SDKs (`github/octokit-rest`, `github/octokit-net`)

### What it watches

Methods removed or renamed in each release of `@octokit/rest` (JavaScript) and Octokit (.NET), from each GitHub release's notes: neither keeps a changelog file. Packages `npm:@octokit/rest` and `nuget:Octokit`; releases listed from each repo's GitHub releases (`vX.Y.Z`, drafts and pre-releases left out). GitHub's REST API itself is a separate source (`github/rest-api`). A bullet under an "Added" or "Features" heading that plainly names what the release adds is an `added` event.

Kind: `changelog` (`kinds/changelog`, `sdks.go`, with `Notes: true`).

### Why this source

- **Source:** https://github.com/octokit/rest.js/releases and https://github.com/octokit/octokit.net/releases, each release read through GitHub's API (`/repos/{repo}/releases/tags/{tag}`) and cited by its release page; checked on 2026-11-20.
- **Why this one:** it's the only per-release record either project publishes. rest.js's semantic-release notes list every removal and rename under "BREAKING CHANGES" (`` `octokit.users.listEmails()` has been renamed to `octokit.users.listEmailsForAuthenticated()` ``).
- **Alternatives considered:** the docs sites (current version only); diffing the published packages (`apisurface`, when it's built).
- **Quote:** "deprecated `octokit.reactions.deleteLegacy()` has been removed" (@octokit/rest 18.0.0).

### How the provider announces changes

rest.js ships a major when GitHub removes or renames REST endpoints; octokit.net's notes list "Breaking changes" in prose more often than by name.

### What the plugin reads and ignores

- A rename whose new name the same release renames again or removes points to where the chain ends (rest.js 18.0.0: `issues.replaceLabels` → `issues.setLabels`, not the intermediate `replaceAllLabels`), or to nothing when the chain ends in a removal.
- Every heading in a release's notes starts a section; bullets under a breaking or deprecation heading count when they name what they remove or rename plainly.
- Drops (and logs): signature changes (`ISshKeysClient.Get(int id)`), dropped Node.js versions, "remove deprecated endpoints" without names.

### Known gaps

- Release notes can be edited after the release: what's read is the notes when the pair is extracted.
- octokit.net's releases describe breaking changes in prose or as signatures: the last plainly named removal is 0.5.2's (`MilestoneUpdate.Number`), the release its golden pins.
