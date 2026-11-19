# Saved catalog snapshots

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `redis-doc-cc0427b.json` | https://github.com/redis/redis-doc/blob/cc0427b3c3671e90b92f1ad2fe4e835fd32cd3c9/commands.json | 2026-11-18 | BSD-3-Clause (redis-doc); trimmed to `FUNCTION CREATE`, `FUNCTION INFO` and `GET` | `"FUNCTION CREATE": {"summary": "Create a function with the given arguments (name, code, description)", "since": "7.0.0", ...}` |
| `redis-doc-06f3598.json` | https://github.com/redis/redis-doc/blob/06f3598f39c7ca86dbdb0aa1668cfdd1b82b908d/commands.json | 2026-11-18 | BSD-3-Clause (redis-doc); trimmed to `FUNCTION LOAD` and `GET` | `"FUNCTION LOAD": {"summary": "Create a function with the given arguments (name, code, description)", "since": "7.0.0", ...}` |
| `commits-master.json` | n/a: a stand-in for `GET https://api.github.com/repos/redis/redis-doc/commits/master`'s response shape, not a saved page. The plugin never cites this lookup as an event's evidence, only the two files above. | — | — | — |

Both `commands.json` snapshots are real, un-invented excerpts: `06f3598` ([PR #1697](https://github.com/redis/redis-doc/pull/1697), 2022-01-24) is the actual commit that removed `FUNCTION CREATE` and `FUNCTION INFO` and added `FUNCTION LOAD`; `cc0427b` is its immediate parent. `GET` is kept whole and unmodified in both as the drop-test fixture: an unrelated, decades-old command that must never produce an event just because the file around it changed.
