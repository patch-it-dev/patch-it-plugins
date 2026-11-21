# Cloudflare: saved copies and goldens

Real copies of what Cloudflare publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/cloudflare -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved feed

A real copy, saved once with `curl`; the golden test reads it through the whole `notices` plugin.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `cloudflare-api-deprecations.rss.xml` | https://developers.cloudflare.com/fundamentals/api/reference/deprecations/index.xml (RSS 2.0; `ETag` sent) | 2026-11-19 15:44 UTC | CC BY 4.0 (Cloudflare's documentation, cloudflare/cloudflare-docs), kept whole | "End of life date: September 27, 2026" (Legacy Registrar Domain Management API) |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/`.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/cloudflare/`: `cloudflare-4.3.0-py3-none-any.trimmed.whl`, `cloudflare-4.3.1-py3-none-any.trimmed.whl`, `cloudflare-5.3.0-py3-none-any.trimmed.whl`, `cloudflare-5.4.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/83/7f/0f9370a1d2a2e425056134ccfc6b5b8f12ed43be600223bd1612f9f56c32/cloudflare-4.3.0-py3-none-any.whl; https://files.pythonhosted.org/packages/3a/8f/c6c543565efd3144da4304efa5917aac06b6416a8663a6defe0e9b2b7569/cloudflare-4.3.1-py3-none-any.whl; https://files.pythonhosted.org/packages/c5/6f/b7624d3a8f0637b04e883f289f530c88e1c4f683c70d8ebe291e4b430ec1/cloudflare-5.3.0-py3-none-any.whl; https://files.pythonhosted.org/packages/c2/bf/20002f7398cb5dc2f289cccd19777aa8869d40d5b01961ccbe8e25e094a6/cloudflare-5.4.0-py3-none-any.whl | 2026-11-21 | Apache-2.0 (the wheel's own license files are kept in it). Trimmed: only `cloudflare/resources/iam/__init__.py`, `cloudflare/resources/iam/oauth_scopes.py`, `cloudflare/resources/iam/oauth_scopes/__init__.py`, `cloudflare/resources/iam/oauth_scopes/oauth_scopes.py`, `cloudflare/types/aisearch/__init__.py`, `cloudflare/types/aisearch/instance_create_params.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's section of [`../README.md`](../README.md) |
| `pypi/cloudflare/`: `pypi-cloudflare-excerpt.json` and `pypi-cloudflare-4.3.0.json`, `pypi-cloudflare-4.3.1.json`, `pypi-cloudflare-5.3.0.json`, `pypi-cloudflare-5.4.0.json` | https://pypi.org/pypi/cloudflare/json, https://pypi.org/pypi/cloudflare/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |

## SDK changelogs

Read with `Plugin{File: ...}` (`kinds/changelog/sdks.go`); `golden/<provider>-<source>.json` is what each stores from its release (`changelogtest.CheckSDKs`), the pair from the release before it on the registry (or among the repo's tags). Every file is the changelog at the release's tag, saved with `curl`, **trimmed** to two releases' sections, each kept whole and unchanged: the release the golden pins and one more release from the same file whose notes remove nothing the reader can name (the drop test). Only the pinned release's own section is read for the golden, so the trimmed copy gives the same events as the whole file (checked when each copy was made).

| File | URL | Tag commit | Saved | License; trimming | Quote |
|---|---|---|---|---|---|
| `cloudflare-typescript-7.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-typescript/v7.1.0/CHANGELOG.md | `f41ff03` | 2026-11-20 | Apache-2.0 (cloudflare-typescript); trimmed to the sections of 7.1.0 (the golden) and 7.0.0 (the drop test), each unchanged | "`cloudforceOne.threatEvents.indicatorTypes` has been renamed to `cloudforceOne.threatEvents.indicators`" |
| `cloudflare-python-5.6.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-python/v5.6.0/CHANGELOG.md | `02ab9a9` | 2026-11-20 | Apache-2.0 (cloudflare-python); trimmed to the sections of 5.6.0 (the golden) and 5.4.0 (the drop test), each unchanged | "accounts:** The `create()` and `get()` methods have been removed from `client.accounts.subscriptions`. The `Subscript…" |
| `cloudflare-go-7.15.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-go/v7.15.0/CHANGELOG.md | `d2f77a0` | 2026-11-20 | Apache-2.0 (cloudflare-go); trimmed to the sections of 7.15.0 (the golden) and 7.13.1 (the drop test), each unchanged | "rules:** the `ListCursor` and `ListCursorParam` types are removed." |
| `cloudflare-terraform-5.29.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/terraform-provider-cloudflare/v5.29.0/CHANGELOG.md | `8e7a366` | 2026-11-20 | Apache-2.0 (terraform-provider-cloudflare); trimmed to the sections of 5.29.0 (the golden) and 5.28.1 (the drop test), each unchanged | "zone_dns_settings, account_dns_settings:** the deprecated `foundation_dns` attribute has been removed. On `cloudflare…" |
