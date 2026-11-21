# Redis (`redis/commands`)

## What it watches

Redis's command surface: commands that disappear entirely between two Redis releases, read from the command definitions in the server's own source (`src/commands/`, one JSON file per command) at each release tag.

Kind: `catalog` (`plugins/catalog`), the `Commands` reader (`plugins/catalog/redis.go`).

## Why this source

- **Source:** [`src/commands/`](https://github.com/redis/redis/tree/unstable/src/commands) in redis/redis, read at each final release's tag (`X.Y.Z`, listed from [GitHub releases](https://github.com/redis/redis/releases) and ordered by version). A release is one listing (`GET https://api.github.com/repos/redis/redis/contents/src/commands?ref=<tag>`); a file that is gone at the next release is fetched once, at the older tag (`https://raw.githubusercontent.com/redis/redis/<tag>/src/commands/<file>`), for the command's name and the entry quoted as evidence. Checked on 2026-11-20: 460 files at 8.10.0, the directory there from 7.0.0.
- **Why this one:** it is what the server is built from: `commands.def`, the server's command table, is generated from these files, and so is redis.io's command reference. A release tag maps an event to the server version a team runs, and a tag never changes, so the file's URL is its own citation.
- **Alternatives considered:**
  - **`redis/redis-doc` `commands.json`** (what this source read before, until 2026-11-20): the repo is archived, the file last changed on 2023-10-24, 370 commands, so nothing from Redis 7.4 or 8.x was ever seen.
  - **`redis/docs` `data/commands.json`** (redis.io's maintained docs repo, 678 commands on 2026-11-20): one file, but a docs artifact. Its history (5 commits since 2024-02) shows editorial churn that would read as removals: commit `2f513f2` (2024-06-06) dropped 126 module commands (`BF.*`, `CF.*`, `FT.*`…) that Redis never removed, and later commits put them back; commit `9740114` (2026-07-30) deleted the file outright. Its versions are commit dates, not server releases. Every one of those would be a false event, so it isn't read.
  - **An OpenAPI spec.** Redis the database has none: it's a binary protocol, not a REST API.
  - **The client libraries' changelogs** (`redis-py`'s `CHANGES`, `node-redis`'s `CHANGELOG.md`): both stopped being updated years ago, and their bullets name client arguments, not server commands.
- **Quote:** at tag `8.8-m03`, `src/commands/gcra.json` is `{"GCRA": {"summary": "Rate limit via GCRA (Generic Cell Rate Algorithm).", … "since": "8.8.0", …}}`; at `8.8-rc1` the file is gone, and so is `gcrasetvalue.json`.

## How the provider announces changes

- **No API version**: Redis is versioned by server release (`7.4.0`, `8.10.0`). Several lines are patched at once (7.2.10 shipped after 8.0.0), which is why versions are ordered by number, not by date: a pair across lines would read the older line's missing commands as removals.
- **Commands are deprecated, essentially never removed.** Between every final release from 7.0.0 to 8.10.3, no command file was removed (checked 2026-11-20 across the tags); about twenty commands carry `deprecated_since` and still work. The one removal in that span is between pre-release tags: GCRA and GCRASETVALUE, added in the 8.8 milestone builds, gone at 8.8-rc1. The golden test pins that pair.

## What the plugin reads and ignores

- Reads: the list of `*.json` files in `src/commands/` at each release. A file in the older release's list and not in the newer's becomes a `breaking` `package_release` event (`change: removed_symbol`), its symbol the command's name (a subcommand with its container: `CLIENT KILL`), its version the newer release and its `previous_version` the older. The excerpt quotes the file at the older tag.
- Ignores: what a command's file says (arguments, flags, complexity, ACL categories): only presence is compared. An added command is not an event. A file renamed with the same command in it would read as a removal; none was renamed in the span checked.
- A release from before 7.0.0 has no `src/commands/`: nothing to read (the pair is skipped).
- Pre-release tags (`-rc`, `-m`) aren't listed: they aren't releases a team runs.
- The pinned drop test: 8.8.0 → 8.10.0 adds 17 command files and removes none, so no event.

## Known gaps

- Commands gaining `deprecated_since` between two releases aren't turned into `deprecation` events yet: that needs the changed files' contents, not only the list.
- Module commands (`FT.*`, `JSON.*`, `TS.*`…) live in the modules' own repos, not in `src/commands/`.
- `node-redis` and `redis-py`'s own breaking changes aren't watched here: they're client SDKs, a separate source.
