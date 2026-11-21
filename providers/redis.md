# Redis (`redis/commands`)

## What it watches

Redis's command surface: commands that disappear entirely from `redis-doc`'s own catalog of every command, between two snapshots of it.

Kind: `catalog` (`plugins/catalog`), new in this change — no existing format plugin reads a flat JSON catalog keyed by name; see "Alternatives considered" for the formats checked and ruled out first.

## Why this source

- **Source:** `commands.json` in [redis/redis-doc](https://github.com/redis/redis-doc), fetched at a commit (`https://raw.githubusercontent.com/redis/redis-doc/<sha>/commands.json`); the newest commit from `GET https://api.github.com/repos/redis/redis-doc/commits/master`. Checked on 2026-11-18.
- **Why this one:** it's the real, machine-readable file redis.io's own command reference is generated from, one JSON object per command (370 of them as checked), each with the metadata (arity, flags, ACL categories, and for a few, `deprecated_since` / `replaced_by`) that drives the docs site. `redis-doc` has no tags, so a commit SHA stands in for a release; that SHA is still a permanent, immutable citation, the same property a tag gives elsewhere.
- **Alternatives considered:**
  - **An OpenAPI spec.** Redis the database has none: it's a binary protocol, not a REST API. (Redis Cloud, the separate hosted-management product, does publish a REST API, but only behind a live Swagger UI at `api.redislabs.com`, not an official, git-pinned spec repo — no stable tag to diff between releases the way Stripe's or GitHub's are.)
  - **The official client libraries' changelogs** (`redis-py`'s `CHANGES`, `node-redis`'s `CHANGELOG.md`): both exist, but both stopped being updated once each project moved past its early releases — `CHANGES` says outright "this file contains only the changes history before redis-py version 4.0.0 ... after ... all changes can be found in the Release notes published in GitHub"; `node-redis`'s file stops at v4.0.2 (Jan 2022) while the package itself is past v6. Checked their actual bullets against `plugins/changelog`'s classifier (which needs a plain "Removed `X`" sentence with a backtick'd name): across every version heading in both files, none matched — the prose names JS/Python arguments and config options, not Redis commands, and rarely in backticks.
  - **Redis's own deprecations/sunset prose** (`docs.github.com`-style breaking-changes page): Redis has no equivalent single page; deprecation notes live per-command, in the catalog itself (see below).
  - **An RSS/Atom feed:** none found for Redis's own release notes; `redis/redis`'s GitHub Releases has an Atom feed (`/releases.atom`), but its entries are prose changelogs with the same shape problem as the client libraries above, not a vendor-labelled "breaking" category the way `feed`'s existing readers (GitHub Actions, Vertex AI) depend on.
- **Quote:** `"FUNCTION INFO"` is in the catalog at commit `cc0427b`; by the very next commit touching the file, it and `FUNCTION CREATE` are gone, replaced by `FUNCTION LOAD` ([PR #1697](https://github.com/redis/redis-doc/pull/1697)).

## How the provider announces changes

- **No API version at all**: Redis is versioned by server release (`7.0.0`, `8.2.0`...), with no header, path or date scheme the way Stripe or GitHub have. A client picks a server to connect to; there is no "pin an API version" concept to gate against.
- **Commands are deprecated, essentially never removed.** Of the 370 commands in the live catalog, 21 carry `deprecated_since` (e.g. `HMSET` since 4.0.0, replaced by `HSET`); every one of them still works. Checked directly against the real source of truth, [redis/redis](https://github.com/redis/redis)'s own `src/commands/*.json`: diffing the full command file list between the `7.0.0` and `8.2.0` GA tags removed **zero** files. Redis's own backward-compatibility stance means a shipped command keeps working forever.
- **The one real removal found, `FUNCTION CREATE` / `FUNCTION INFO`, never shipped.** They existed only in `redis-doc`'s catalog during the pre-release design of the Functions feature; `src/commands/function-create.json` doesn't exist even at `7.0-rc1`, the first public release candidate, tagged a week *after* the catalog commit that dropped them. So this is a real, verifiable removal in the repo's history, but not one any real user's code could have depended on — unlike Stripe's, Datadog's or GitHub's demo changes, which all broke a real, shipped version. Recorded here rather than smoothed over: Redis's command surface, at least since 7.0 GA, has had no breaking removal to replay instead.

## What the plugin reads and ignores

- Reads: `commands.json` at the newest commit, compared with the commit read before it. A name present before and missing after becomes a `breaking` `package_release` event (`change: removed_symbol`); the excerpt quotes the entry's own JSON from the snapshot where it still existed.
- Ignores: everything else about a command (arguments, flags, complexity, ACL categories) — only presence is diffed. A name that's only *added* is not an event. A rename is read as one removal, since nothing in the catalog says a disappearance and an appearance are the same command; guessing that would trade precision for recall.
- The live `deprecated_since` / `replaced_by` fields on still-present commands aren't read yet (see "Known gaps"): they're deprecations, not removals, and every one of them already works forever regardless.
- Reads a historical pair (a version pair, commit `cc0427b` → `06f3598`) and recreates the real removal of `FUNCTION CREATE` and `FUNCTION INFO` (the golden test uses trimmed excerpts of both commits' `commands.json`, alongside `GET`, unchanged, as the drop-test fixture).

## Known gaps

- The 21 commands already marked `deprecated_since` in the live catalog aren't turned into `deprecation` events: there's no second snapshot showing the moment each one was marked that way conveniently paired with a trimmed fixture, and since none of them is ever actually removed, the user-facing urgency is low. A future change could diff for a command gaining `deprecated_since` between two snapshots the same way this one diffs for removal.
- `node-redis` and `redis-py`'s own breaking changes (argument shapes, defaults, RESP3 becoming the default response protocol) aren't watched: real breaking changes exist in both (checked: `node-redis` v4.0.0's "Breaking Changes" section, `redis-py` v8.0.0's RESP2/RESP3 unification), but neither client writes them as the plain, single-symbol sentences `plugins/changelog` already reads elsewhere. A client-library plugin for Redis would need its own, looser reader, not this one.
