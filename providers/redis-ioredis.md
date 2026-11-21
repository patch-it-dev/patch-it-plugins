# ioredis (`redis/ioredis`)

## What it watches

Names removed or renamed in each release of ioredis, the Node.js Redis client, from that release's section of its `CHANGELOG.md` at its tag. Package `npm:ioredis`; releases listed from npm, each read at `vX.Y.Z`.

Kind: `changelog` (`plugins/changelog`, `sdks.go`).

## Why this source

- **Source:** https://github.com/redis/ioredis/blob/main/CHANGELOG.md, read raw at the release's tag; checked on 2026-11-20.
- **Why this one:** semantic-release writes every release's "BREAKING CHANGES" into this file; a major is headed `# [X.0.0](...)`, a minor or patch `## [X.Y.Z](...)`, and both are read.
- **Alternatives considered:** GitHub release notes (the same text); the README's upgrade notes (not versioned per release).
- **Quote:** "`Redis#serverInfo` is removed. This field is never documented so you very likely have never used it." (5.0.0).

## How the provider announces changes

A major every few years (v6.0.0 in 2026), with its breaking changes listed in the changelog.

## What the plugin reads and ignores

- `Class#member` is read as `Class.member`. Counts plain removals and renames.
- Drops (and logs): changed defaults, dropped Node.js versions, options removed (`allowUsernameInURI`), behavior changes.

## Known gaps

- Removed options aren't symbols: not events.
