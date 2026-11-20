# PlanetScale (`planetscale`)

## What it watches

The PlanetScale API (`api.planetscale.com/v1`): operations PlanetScale marks deprecated, and removals of operations it had marked deprecated, between two versions of the API's OpenAPI spec. PlanetScale says it in the operation's description, never with the `deprecated` flag.

Kind: `openapi` (`plugins/openapi`), with `adapter.PlanetScale`.

## Why this source

- **Source:** `schemas/openapi.yaml` in [planetscale/terraform-provider-planetscale](https://github.com/planetscale/terraform-provider-planetscale), pinned to the commit that last changed it (`https://raw.githubusercontent.com/planetscale/terraform-provider-planetscale/<sha>/schemas/openapi.yaml`). The newest such commit comes from the GitHub commits API. Checked on 2026-11-19.
- **Why this one:**
  - It's PlanetScale's own spec, in PlanetScale's own GitHub organization: the repo's `make download-openapi` copies it from `https://planetscale.com/docs/openapi.yaml`, the spec the API reference is built from. A workflow runs that every Monday and opens a pull request when the file changed, and a maintainer merges it.
  - A commit can be cited and replayed. 33 commits changed the file from 2025-11-11 to 2026-11-09. One deleted it (`f33a0c3b35`, 2025-12-04) and it was back the same day, so there are 32 versions to compare.
- **Alternatives considered:**
  - **`https://planetscale.com/docs/openapi.yaml`.** The same spec served live (3.1 MB), a few days to a few months ahead of the repo. It has no versions to pin, cite or replay (`info.version` is always `v1`), and the Wayback Machine has one snapshot of it (2026-05-21).
  - **The [changelog](https://planetscale.com/changelog)** (Atom at `https://planetscale.com/changelog/feed.atom`). It holds only the newest 25 posts (2026-10-23 to 2026-11-19 on the day checked), and none of them says "deprecated". It's product news, not keyed by operation.
  - **[planetscale/planetscale-go](https://github.com/planetscale/planetscale-go).** The Go API client. Its release notes are GitHub's generated list of merged pull requests, with no breaking-change section.
  - **[planetscale/cli](https://github.com/planetscale/cli)** (`pscale`). It doesn't ship the spec.
  - **[planetscale/docs](https://github.com/planetscale/docs).** Archived; its last push was 2024-11-12.
  - **There's no deprecations page.**
- **Quote:** "Deprecated. Use GET /organizations/:organization/databases/:database/branches/:branch/change-requests instead." (`GET .../branches/{branch}/neki-changes`, deprecated in the spec on 2026-11-09)

## How the provider announces changes

- **No API version beyond `/v1`.** `info.version` is a fixed `v1`, so the event's `version` is the date of the spec commit, the closest thing to the API version a change arrived in.
- **The spec grows fast:** from 116 operations (2025-11-11) to 291 (2026-11-09) in the repo, and 298 live on 2026-11-19.
- **Deprecations are rare and said in words.** No version of the spec has ever set `deprecated: true`. The one deprecation in the history, five operations on 2026-11-09, put "Deprecated. Use ... instead." as the first line of each operation's description.
- **Operations leave without being marked.** 41 operations left the spec, none of them deprecated first.

## What the plugin reads and ignores

- **Reads:** the spec at the newest commit, converted from YAML to JSON with its values unchanged, and diffed against the last one. Two events come out of it:
  - **Deprecations:** an operation whose description now starts with "Deprecated." or "Deprecated:" (any case, after any leading whitespace). The event cites the description, so PlanetScale's replacement is in the excerpt. A description that mentions deprecation further on isn't a deprecation, nor is "Deprecated, storage shrinking is part of storage autoscaling.", which the spec writes on properties. If PlanetScale ever sets the flag, the event quotes the whole operation.
  - **Removals (breaking):** an operation that leaves the spec after the previous spec had marked it deprecated. None yet.
- **Names operations by the path a caller sends,** `GET /v1/organizations/…`: the server is `https://api.planetscale.com/v1` (in all 32 versions) and the paths leave out `/v1`, so the adapter moves it into the paths (as Sentry's does with `/api/0`). Quotes cite the spec as published, and PlanetScale's own "Use GET /organizations/…" text leaves out `/v1`. If PlanetScale ever moves `/v1` between the server and the paths, the operations stay the same, rather than every deprecated one reading as removed.
- **Matches operations by path template:** a renamed path parameter is the same URL to a caller.
- **Refuses a file that isn't a spec.** At the commit that deleted the file the raw URL answers `404`, and a file with no `paths` fails the fetch: diffed, it would read as every operation removed.
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** 41 in the history: 1 on 2026-06-01, 7 on 2026-10-03, 17 on 2026-10-14, 7 on 2026-10-23, 9 on 2026-11-09. On 2026-10-03, 6 of them were the read-only replica operations, renamed to dedicated read replicas, and on 2026-11-19 planetscale-go's `main` still sends `.../read-only-replicas` ("the deprecated read-only-replicas path", in its own words). Leaving the spec isn't the same as stopping working.
  - **Request and response property changes, and new operations.** The history has 216 new operations, and the property side is noisy: 226 type changes on 2025-12-06 alone, when whole numbers became `integer`.
- Over the 31 consecutive pairs this keeps 5 events, all deprecations on 2026-11-09 (`fc13e09d`): `GET` and `DELETE .../bouncers/{bouncer}/resizes`, `GET .../neki-changes`, `GET` and `DELETE .../neki-changes/{id}`. A pattern matching "deprecat" anywhere in the description finds the same 5.
- **Replays** a historical pair (`Replay`, from `e631095a` to `fc13e09d`) and recreates those 5. The drop test replays `b49c3540` to `f6d6ad4d` (the replica rename): no event, where the same pair unfiltered reports the 7 removals.

## Known gaps

- **Deprecations PlanetScale announces elsewhere aren't events.** The 12 `/organizations/{organization}/databases/{database}/workflows` operations left the spec on 2026-10-14 without ever being marked in it, and planetscale-go calls `read-only-replicas` deprecated while the spec never did.
- **Removals PlanetScale didn't mark first aren't events,** even if the endpoint really went away.
- **The repo lags the live spec.** A change arrives when the Monday pull request is merged, not when PlanetScale publishes it: on 2026-11-19 the live spec had 7 operations the repo's newest copy (2026-11-09) didn't, and the history has a three-month gap (2026-02-26 to 2026-06-01). The event's `version` is the commit date, not the day PlanetScale changed the API.
- **Some commits are hand edits** to the copy ("Fix type of `planetscale_organization` `invoice_budget_amount`", "Mark `parent_branch` as optional"), so a version can differ from what PlanetScale published. Only 13 of the 33 commits are the workflow's own.
- **Only the REST API is covered.** The SDKs and drivers aren't: `@planetscale/database` (database-js) 2.0.0 (2026-09-24) removed parameter interpolation, which this plugin can't see.
- **Two spec commits on one day share a `version`** (it happened on 2025-12-04, 2026-02-24, 2026-08-11 and 2026-09-10). The events stay distinct, because the operation is part of the fingerprint, but the second run's `previous_version` equals its `version`.
- **The patch-it client can't place these events yet.** Their `version` is a date (`2026-11-09`), because the API has no version. The client keeps a date-versioned API's events only if the API is in its list of date-versioned APIs, so `planetscale` needs an entry there.
