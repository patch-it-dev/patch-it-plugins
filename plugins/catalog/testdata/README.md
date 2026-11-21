# Saved catalog snapshots

The golden file, `golden/redis-commands.json`, is what the catalog kind stores from the pair below (`go test ./plugins/catalog -update`).

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `redis-contents-src-commands-8.8-m03.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8-m03 | 2026-11-20 | BSD-3-Clause / RSALv2 / SSPLv1 / AGPLv3 (redis/redis); the listing cut to the two fields read, `name` and `type`, every entry kept | `{"name": "gcra.json", "type": "file"}` |
| `redis-contents-src-commands-8.8-rc1.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8-rc1 | 2026-11-20 | as above | no `gcra.json`, no `gcrasetvalue.json` |
| `redis-contents-src-commands-8.8.0.json`, `redis-contents-src-commands-8.10.0.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8.0 and `?ref=8.10.0` | 2026-11-20 | as above | the drop test: 17 files added, none removed |
| `redis-8.8-m03-gcra.json` | https://raw.githubusercontent.com/redis/redis/8.8-m03/src/commands/gcra.json | 2026-11-20 | as above, kept whole, byte for byte | `"GCRA": {"summary": "Rate limit via GCRA (Generic Cell Rate Algorithm).", … "since": "8.8.0"` |
| `redis-8.8-m03-gcrasetvalue.json` | https://raw.githubusercontent.com/redis/redis/8.8-m03/src/commands/gcrasetvalue.json | 2026-11-20 | as above, kept whole, byte for byte | `"GCRASETVALUE": {"summary": "An internal command for recording a GCRA TAT value during AOF rewrite and replication."` |
| `redis-releases-excerpt.json` | https://api.github.com/repos/redis/redis/releases (shape only) | written for the test, 2026-11-20 | not a saved copy: five entries in the shape of GitHub's releases API (`tag_name`, `draft`, `prerelease`, `published_at`, `html_url`), dates approximate, a pre-release and a patch to an older line published later among them | for the version-listing test only |
