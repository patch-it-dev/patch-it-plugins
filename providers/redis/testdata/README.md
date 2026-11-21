# Redis: saved copies and goldens

Real copies of what Redis publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/redis -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved catalog snapshots

Read through the `catalog` kind; the golden file is what it stores from them.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `redis-contents-src-commands-8.8-m03.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8-m03 | 2026-11-20 | BSD-3-Clause / RSALv2 / SSPLv1 / AGPLv3 (redis/redis); the listing cut to the two fields read, `name` and `type`, every entry kept | `{"name": "gcra.json", "type": "file"}` |
| `redis-contents-src-commands-8.8-rc1.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8-rc1 | 2026-11-20 | as above | no `gcra.json`, no `gcrasetvalue.json` |
| `redis-contents-src-commands-8.8.0.json`, `redis-contents-src-commands-8.10.0.json` | https://api.github.com/repos/redis/redis/contents/src/commands?ref=8.8.0 and `?ref=8.10.0` | 2026-11-20 | as above | the drop test: 17 files added, none removed |
| `redis-8.8-m03-gcra.json` | https://raw.githubusercontent.com/redis/redis/8.8-m03/src/commands/gcra.json | 2026-11-20 | as above, kept whole, byte for byte | `"GCRA": {"summary": "Rate limit via GCRA (Generic Cell Rate Algorithm).", … "since": "8.8.0"` |
| `redis-8.8-m03-gcrasetvalue.json` | https://raw.githubusercontent.com/redis/redis/8.8-m03/src/commands/gcrasetvalue.json | 2026-11-20 | as above, kept whole, byte for byte | `"GCRASETVALUE": {"summary": "An internal command for recording a GCRA TAT value during AOF rewrite and replication."` |
| `redis-8.8-rc1-<command>.json` (19: `arcount.json` … `arset.json`, `increx.json`) and `redis-8.10.0-<command>.json` (17) | https://raw.githubusercontent.com/redis/redis/8.8-rc1/src/commands/<command>.json and …/8.10.0/… | 2026-11-21 | as above, kept whole, byte for byte | the files 8.8-rc1 has and 8.8-m03 doesn't, and 8.10.0 has and 8.8.0 doesn't: `"INCREX": {` … |
| `redis-releases-excerpt.json` | https://api.github.com/repos/redis/redis/releases (shape only) | written for the test, 2026-11-20 | not a saved copy: five entries in the shape of GitHub's releases API (`tag_name`, `draft`, `prerelease`, `published_at`, `html_url`), dates approximate, a pre-release and a patch to an older line published later among them | for the version-listing test only |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/redis/`: `redis-7.4.1-py3-none-any.trimmed.whl`, `redis-8.0.0-py3-none-any.trimmed.whl`, `redis-8.0.1-py3-none-any.trimmed.whl`, `redis-8.1.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/4a/2e/2677f3f93dae0497e7e33b6637302e7f3744efc553f34231183e32584885/redis-7.4.1-py3-none-any.whl; https://files.pythonhosted.org/packages/27/e3/b519734372d305bd547534a9f32e4ce9f98552af753dce72cf3483a0ff0b/redis-8.0.0-py3-none-any.whl; https://files.pythonhosted.org/packages/fd/0a/c2345ebf1ebe70840ce3f6c6ee612f8fa749cfbd1b03069c53bf0c62aaad/redis-8.0.1-py3-none-any.whl; https://files.pythonhosted.org/packages/66/9d/c5731f6e3608663d4d3656fd8d3aecee8b509c3082818f5a13eae925baea/redis-8.1.0-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `redis/__init__.py`, `redis/maint_notifications.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/redis/`: `pypi-redis-excerpt.json` and `pypi-redis-7.4.1.json`, `pypi-redis-8.0.0.json`, `pypi-redis-8.0.1.json`, `pypi-redis-8.1.0.json` | https://pypi.org/pypi/redis/json, https://pypi.org/pypi/redis/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/redis-client/`: `client-1.6.1.trimmed.tgz`, `client-5.0.0.trimmed.tgz`, `client-6.3.0.trimmed.tgz`, `client-6.3.1.trimmed.tgz` | https://registry.npmjs.org/@redis/client/-/client-1.6.1.tgz; https://registry.npmjs.org/@redis/client/-/client-5.0.0.tgz; https://registry.npmjs.org/@redis/client/-/client-6.3.0.tgz; https://registry.npmjs.org/@redis/client/-/client-6.3.1.tgz | 2026-11-21 | MIT (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `dist/index.d.ts`, `dist/lib/client/index.d.ts`, `dist/lib/command-options.d.ts`, `dist/lib/commands/FLUSHALL.d.ts`, `dist/lib/errors.d.ts`, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `npm/redis-client/`: `npm-redis-client-excerpt.json` and `npm-redis-client-1.6.1.json`, `npm-redis-client-5.0.0.json`, `npm-redis-client-6.3.0.json`, `npm-redis-client-6.3.1.json` | https://registry.npmjs.org/@redis%2Fclient, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |

## SDK changelogs

Read with `Plugin{File: ...}` (`kinds/changelog/sdks.go`); `golden/<provider>-<source>.json` is what each stores from its release (`changelogtest.CheckSDKs`), the pair from the release before it on the registry (or among the repo's tags). Every file is the changelog at the release's tag, saved with `curl`, **trimmed** to two releases' sections, each kept whole and unchanged: the release the golden pins and one more release from the same file whose notes remove nothing the reader can name (the drop test). Only the pinned release's own section is read for the golden, so the trimmed copy gives the same events as the whole file (checked when each copy was made).

| File | URL | Tag commit | Saved | License; trimming | Quote |
|---|---|---|---|---|---|
| `redis-ioredis-5.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/redis/ioredis/v5.0.0/CHANGELOG.md | `b8e99b4` | 2026-11-20 | MIT (ioredis); trimmed to the sections of 5.0.0 (the golden) and 5.0.0-beta.4 (the drop test), each unchanged | "you very likely have never used it." |
