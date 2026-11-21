# GitHub GraphQL API (`github/graphql`)

## What it watches

Removals GitHub announces for its GraphQL API: a field, an argument, an input field or an enum value, with the date it goes. One `deprecation` event per newly announced removal, its `version` the removal date.

Kind: `catalog` (`plugins/catalog`), the `UpcomingChanges` reader (`plugins/catalog/graphql.go`).

## Why this source

- **Source:** [`src/graphql/data/fpt/upcoming-changes.json`](https://github.com/github/docs/blob/main/src/graphql/data/fpt/upcoming-changes.json) in github/docs, read at every commit that changed it (listed from GitHub's commits API, `?path=`). It's the file [Breaking changes](https://docs.github.com/en/graphql/overview/breaking-changes) on docs.github.com is built from. Checked on 2026-11-21: 1,291 entries, every one `"criticality": "breaking"`, 29 commits since 2023-05-01.
- **Why this one:** one JSON object keyed by removal date, each entry naming its location (`RepositoryPermission.TRIAGE_PLUS`), what to do and why, from GitHub's own schema tooling. A commit never changes, so the raw file at it is its own citation.
- **Alternatives considered:**
  - **The Breaking changes page** (HTML): the same entries, rendered; the JSON is what it's made from.
  - **The GraphQL schema** (`schema.docs.graphql`, `@deprecated(reason: "… Removal on 2027-04-01 UTC.")`): the same dates inside a GraphQL SDL, which needs a schema reader no kind has.
  - **The GitHub changelog**: announces some of these, in prose, with no location.
- **Quote:** `{"location": "RepositoryPermission.TRIAGE_PLUS", "description": "<p><code>TRIAGE_PLUS</code> will be removed. Use MODERATE instead.</p>", "date": "2027-04-01", "criticality": "breaking"}`.

## How the provider announces changes

- GitHub's GraphQL API has no versions: a change is announced here with its removal date, a quarter's first day (`2026-10-01`, `2027-01-01`).
- Announcements come a few at a time (one or two a month in 2026), sometimes in a batch (1,109 entries on 2025-04-14, the audit-log fields).

## What the plugin reads and ignores

- Reads every entry of the file at each commit. An entry (location and date) in the newer commit's file that the older one didn't have is a `deprecation` event: `version` the removal date (the API has no versions; the day it breaks is what a gate compares, as a model's shutdown date is), `previous_version` the older commit's date, the hint the entry's description and reason as text. The excerpt is the entry's fields as the file writes them, its `owner` left out.
- Subject (`api_operation`): an input object's field (`AddPullRequestReviewInput.comments`) or a root field's argument (`Query.sponsorables.dependencyEcosystem`) is a `request_parameter` of `POST /graphql`, its `property` the location; any other location (a root field itself, a type's field or its argument, an enum value) is a `schema_property`, its `schema` the type and its `property` the rest (`RepositoryPermission` / `TRIAGE_PLUS`).
- An entry a commit drops while still upcoming and a later commit restores is the same event (same removal date, same fingerprint), so it isn't announced twice.
- Ignores an entry dated before the commit that adds it: a revert putting back a removal already made, not an announcement. On 2026-01-05 the file lost 17 such entries, got them back the next day and lost them again on 2026-01-08: no event (the pinned drop test).
- Ignores an entry leaving the file: the removal shipped, or GitHub withdrew it, and the file doesn't say which.
- A first read, with no commit before it, has nothing to compare: no events.

## Known gaps

- The removal itself (the day it ships) isn't a separate `breaking` event; the deprecation names the date.
- GitHub Enterprise Server's own files (`ghes-*`) aren't read.
- A location GitHub moves to a new date is a new event (a new `version`), alongside the first.
