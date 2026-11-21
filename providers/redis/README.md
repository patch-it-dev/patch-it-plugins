# Redis

Every source patch-it reads for Redis, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

## Redis (`redis/commands`)

### What it watches

Redis's command surface: commands that disappear entirely between two Redis releases, read from the command definitions in the server's own source (`src/commands/`, one JSON file per command) at each release tag.

Kind: `catalog` (`kinds/catalog`), the `Commands` reader (`kinds/catalog/commands.go`).

### Why this source

- **Source:** [`src/commands/`](https://github.com/redis/redis/tree/unstable/src/commands) in redis/redis, read at each final release's tag (`X.Y.Z`, listed from [GitHub releases](https://github.com/redis/redis/releases) and ordered by version). A release is one listing (`GET https://api.github.com/repos/redis/redis/contents/src/commands?ref=<tag>`); a file that is gone at the next release is fetched once, at the older tag (`https://raw.githubusercontent.com/redis/redis/<tag>/src/commands/<file>`), for the command's name and the entry quoted as evidence. Checked on 2026-11-20: 460 files at 8.10.0, the directory there from 7.0.0.
- **Why this one:** it is what the server is built from: `commands.def`, the server's command table, is generated from these files, and so is redis.io's command reference. A release tag maps an event to the server version a team runs, and a tag never changes, so the file's URL is its own citation.
- **Alternatives considered:**
  - **`redis/redis-doc` `commands.json`** (what this source read before, until 2026-11-20): the repo is archived, the file last changed on 2023-10-24, 370 commands, so nothing from Redis 7.4 or 8.x was ever seen.
  - **`redis/docs` `data/commands.json`** (redis.io's maintained docs repo, 678 commands on 2026-11-20): one file, but a docs artifact. Its history (5 commits since 2024-02) shows editorial churn that would read as removals: commit `2f513f2` (2024-06-06) dropped 126 module commands (`BF.*`, `CF.*`, `FT.*`…) that Redis never removed, and later commits put them back; commit `9740114` (2026-07-30) deleted the file outright. Its versions are commit dates, not server releases. Every one of those would be a false event, so it isn't read.
  - **An OpenAPI spec.** Redis the database has none: it's a binary protocol, not a REST API.
  - **The client libraries' changelogs** (`redis-py`'s `CHANGES`, `node-redis`'s `CHANGELOG.md`): both stopped being updated years ago, and their bullets name client arguments, not server commands.
- **Quote:** at tag `8.8-m03`, `src/commands/gcra.json` is `{"GCRA": {"summary": "Rate limit via GCRA (Generic Cell Rate Algorithm).", … "since": "8.8.0", …}}`; at `8.8-rc1` the file is gone, and so is `gcrasetvalue.json`.

### How the provider announces changes

- **No API version**: Redis is versioned by server release (`7.4.0`, `8.10.0`). Several lines are patched at once (7.2.10 shipped after 8.0.0), which is why versions are ordered by number, not by date: a pair across lines would read the older line's missing commands as removals.
- **Commands are deprecated, essentially never removed.** Between every final release from 7.0.0 to 8.10.3, no command file was removed (checked 2026-11-20 across the tags); about twenty commands carry `deprecated_since` and still work. The one removal in that span is between pre-release tags: GCRA and GCRASETVALUE, added in the 8.8 milestone builds, gone at 8.8-rc1. The golden test pins that pair.

### What the plugin reads and ignores

- Reads: the list of `*.json` files in `src/commands/` at each release. A file in the older release's list and not in the newer's becomes a `breaking` `package_release` event (`change: removed_symbol`), its symbol the command's name (a subcommand with its container: `CLIENT KILL`), its version the newer release and its `previous_version` the older. The excerpt quotes the file at the older tag. A file in the newer release's list and not in the older's is an `added` event (`change: added_symbol`), quoting the file at the newer tag. A command whose name leaves one file and arrives in another is a file moved: neither.
- Ignores: what a command's file says (arguments, flags, complexity, ACL categories): only presence is compared.
- A release from before 7.0.0 has no `src/commands/`: nothing to read (the pair is skipped).
- Pre-release tags (`-rc`, `-m`) aren't listed: they aren't releases a team runs.
- The pinned drop test: 8.8.0 → 8.10.0 adds 17 command files and removes none: 17 `added` events, nothing breaking. The golden pair (8.8-m03 → 8.8-rc1) adds 19 (the array commands and INCREX) besides its two removals.

### Known gaps

- Commands gaining `deprecated_since` between two releases aren't turned into `deprecation` events yet: that needs the changed files' contents, not only the list.
- Module commands (`FT.*`, `JSON.*`, `TS.*`…) live in the modules' own repos, not in `src/commands/`.
- `node-redis` and `redis-py`'s own breaking changes aren't watched here: they're client SDKs, a separate source.

## ioredis (`redis/ioredis`)

### What it watches

Names removed or renamed in each release of ioredis, the Node.js Redis client, from that release's section of its `CHANGELOG.md` at its tag. Package `npm:ioredis`; releases listed from npm, each read at `vX.Y.Z`. A bullet under an "Added" or "Features" heading that plainly names what the release adds is an `added` event.

Kind: `changelog` (`kinds/changelog`, `sdks.go`).

### Why this source

- **Source:** https://github.com/redis/ioredis/blob/main/CHANGELOG.md, read raw at the release's tag; checked on 2026-11-20.
- **Why this one:** semantic-release writes every release's "BREAKING CHANGES" into this file; a major is headed `# [X.0.0](...)`, a minor or patch `## [X.Y.Z](...)`, and both are read.
- **Alternatives considered:** GitHub release notes (the same text); the README's upgrade notes (not versioned per release).
- **Quote:** "`Redis#serverInfo` is removed. This field is never documented so you very likely have never used it." (5.0.0).

### How the provider announces changes

A major every few years (v6.0.0 in 2026), with its breaking changes listed in the changelog.

### What the plugin reads and ignores

- `Class#member` is read as `Class.member`. Counts plain removals and renames.
- Drops (and logs): changed defaults, dropped Node.js versions, options removed (`allowUsernameInURI`), behavior changes.

### Known gaps

- Removed options aren't symbols: not events.

## node-redis client (`redis/node-redis-surface`)

### What it watches

The public API of the `@redis/client` package on npm, release by release, read from its TypeScript declarations: exported names removed, class members removed, parameters removed, and parameters that are no longer optional. One `package_release` event per change. Every final release is listed from the npm registry (pre-releases and deprecated versions left out), and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the TypeScript declaration reader (`kinds/apisurface/dtsurface`).

### Why this source

- **Source:** the release tarballs on the npm registry (<https://www.npmjs.com/package/@redis/client>), each version's `dist.tarball`, read as published; the code is at <https://github.com/redis/node-redis>. Checked on 2026-11-21.
- **Why this one:** the declarations in the tarball are exactly what TypeScript checks a customer's code against after `npm install @redis/client@<version>`, and what editors show JavaScript users. A published tarball never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** GitHub release notes; `docs/v4-to-v5.md` (majors only).
- **Quote:** `enum RedisFlushModes` (`dist/lib/commands/FLUSHALL.d.ts` line 1) at 1.6.1, gone at 5.0.0 (the client went from 1.x to 5.0.0 with redis 5).

### How the provider announces changes

- `redis` on npm re-exports `@redis/client` (and the module packages) wholesale; the client's declarations are where its API is, so this source reads `@redis/client` and covers `npm:redis` too.
- **Forced on upgrade**, for a repo whose lockfile reaches the release.

### What the plugin reads and ignores

- Reads `package.json` and the `.d.ts`, `.d.mts` and `.d.cts` files of each release's tarball from the npm registry, in memory, as text (`kinds/apisurface/dtsurface`): nothing in the package is run, no install script, no TypeScript compiler. The entry points are `package.json`'s `exports` (each subpath, wildcards matched against the files; for each, its `types` condition, else the `import` one), or its `types`/`typings`.
- For each entry point: every exported name (classes, functions, enums, variables, interfaces, type aliases, namespaces, and names re-exported from another package, by name only), and below a class or namespace what it holds: a class's public methods and properties, and its constructor's parameters; a namespace's classes, functions and namespaces. Symbols are named by the import path code writes: `@redis/client.createClient`.
- Parameters compare by position, as TypeScript passes them: a parameter gone off the end (with no rest parameter now) is removed; one optional before and required now is a `changed_default`. A renamed parameter is no change.
- Ignores names marked `@internal`, `private` and `protected` members, `#private` fields, setters, index signatures, interface and object-type members, interfaces and type aliases inside a namespace (an entry point's own are read), entry points under a private path (`_shims`, `internal`, `src`), the `./index` duplicate of the root, and what the default export holds (it's the same declaration as a named one).
- One declaration reachable at several import paths is one event, every path in its `symbols`.
- Drops (and logs) a removal that may not be one: a member of a name now re-exported from another package, now a value, an interface or a type (whose members aren't read); a method gone from a class that extends a class of another package (it may define it); a name an entry point may still get from a package it re-exports wholesale (`export * from "other"`).

### Known gaps

- What a property's type holds isn't followed: `client.chat.completions.create` is read as the `Completions` class's `create`, reached through namespaces and entry points, not through the `chat` property's type.
- Type changes (a parameter's type narrowed, a return type changed, an interface's fields) aren't events.
- A package that re-exports another wholesale (`export * from "@scope/core"`) hides that package's API: it's a source of its own, or not read.
- Classes are indexed by name across files to follow a method to its base class: two classes of the same name (in two namespaces) are read as one, which can only turn a removal into a note, never add one.
- A parameter added without being optional is a note (the change-event contract has no subject for it yet).
- `@redis/client`'s versions follow its own numbering; a lockfile's `redis` version maps to the client version it depends on.

## redis-py (`redis/redis-py-surface`)

### What it watches

The public API of the `redis` package on PyPI, release by release: classes, functions and methods removed or renamed, parameters removed, and parameters that lost their default (callers must now pass them). One `package_release` event per change. Every final release is listed from PyPI, and each consecutive pair is compared. What a release adds (a new module, class, function or method, or a new parameter with a default) is an `added` event.

Kind: `apisurface` (`kinds/apisurface`), with the Python reader (`kinds/apisurface/pysurface`).

### Why this source

- **Source:** the release files on PyPI (<https://pypi.org/project/redis/>), each release's wheel read as published; the code is at <https://github.com/redis/redis-py>. Checked on 2026-11-21.
- **Why this one:** the wheel is exactly what `pip install redis==<version>` installs, so its public names and signatures are what a customer's code calls. A published file never changes, so its URL is its own citation, and the line quoted is in it.
- **Alternatives considered:** GitHub release notes (the `CHANGES` file stops at 4.0).
- **Quote:** `def get_handler_for_connection(self)` (`OSSMaintNotificationsHandler`, `redis/maint_notifications.py` line 1073) at 8.0.1, gone at 8.1.0.

### How the provider announces changes

- The client releases minors every month or two; its newer subsystems (multi-database, maintenance notifications) change shape between minors.
- **Forced on upgrade**, for a repo whose pin reaches the release.

### What the plugin reads and ignores

- Reads the `.py` and `.pyi` files under `redis/` in the release's wheel (or, when a release has none, its sdist) as text, never importing or running them: module-level classes and functions, the methods of module-level classes, their parameters and defaults, `__all__`, an `__init__`'s re-exports from its own modules, and star imports. A `.pyi` stub is read instead of the `.py` beside it.
- Ignores private names: a leading underscore, a module under a private one or under one named `internal` or `vendor`, a name left out of `__all__`, and tests. Also classes nested in classes, a property's parameters (and its setter), and `self`/`cls` (callers never pass them).
- Drops (and logs): an overloaded function's parameters; a removed parameter when the function now takes `**kwargs`; a new parameter without a default (the change-event contract has no subject for it yet).
- A removal is a rename only when the same scope loses one and gains one of the same kind with identical parameters, at least one of them besides `self`/`cls`; anything less sure is a removal.
- Drops (and logs) a removal that may not be one: a name its module still binds in the newer release (an alias such as `DeadlineExceededError = GatewayTimeoutError`, an import from a private module, a star import from outside the package, a module-level `__getattr__` that hands out old names); a symbol whose file the newer release has too big to read (`MaxFileBytes`); a method whose class is still there but whose bases changed, which one of its bases now defines, or which the class now defines as an attribute.

### Known gaps

- A positional parameter renamed reads as the old one removed: a call that passes it by keyword breaks, one that passes it by position doesn't.
- A class with no `__init__` of its own (generated pydantic or dataclass-like types) has no parameters to compare, so its rename reads as a removal of the old name, and its fields aren't compared.
- A module that star-imports from outside the package, or defines `__getattr__`, hides every removal from it: those are notes, not events.
- Type changes (an annotation narrowed, a return type changed) aren't events.
- The wheel read is the first one by file name (a `-none-any` wheel when there is one); a release whose platform wheels differ in their Python files is read from that one.
