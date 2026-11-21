# Stripe: saved copies and goldens

Real copies of what Stripe publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/stripe -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

Stripe's spec isn't saved here: its tests use small specs written inline.

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/stripe-openapi-releases.json` | `https://api.github.com/repos/stripe/openapi/releases/tags/<tag>`, one per release, in the order `releases?per_page=100` lists them | 21 of 2,591: the first and the last release of each of the 10 API versions their notes name (v2191 to v2602), and v2190, the newest whose notes name none; `tag_name`, `name`, `draft`, `prerelease`, `published_at`, `html_url`, `body` |

## Saved releases

Each golden file, `golden/<provider>-<source>.json`, is what the `apisurface` kind stores from the source's golden pair, read from `pypi/<package>/` or `npm/<package>/` (the openai Python package's pair, 1.1.1 → 1.1.2, from the files at the top level).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `pypi/stripe/`: `stripe-15.4.0-py3-none-any.trimmed.whl`, `stripe-15.5.0-py3-none-any.trimmed.whl`, `stripe-15.6.2-py3-none-any.trimmed.whl`, `stripe-16.0.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/00/e1/3047b4e244cc78bb598b0d5fcab6632b8bf8a4d526f2e840c4f4de02c989/stripe-15.4.0-py3-none-any.whl; https://files.pythonhosted.org/packages/d5/20/4c8377c570697ebe105ad5e073538e498d58e4985ca6e2944bdc917a5034/stripe-15.5.0-py3-none-any.whl; https://files.pythonhosted.org/packages/25/e9/251eb26d9ceae6c44adb3a54cb81ec717dc469452c09756b53a117726277/stripe-15.6.2-py3-none-any.whl; https://files.pythonhosted.org/packages/b5/de/5141ae3990862cc3d61a1cdd4acff72bcb587afb7b4d31c52eaddbb6bc8f/stripe-16.0.0-py3-none-any.whl | 2026-11-21 | MIT (the wheel's own license files are kept in it). Trimmed: only `stripe/__init__.py`, `stripe/_stripe_object.py` and the dist-info `METADATA`, `WHEEL` and license files, each copied byte for byte | see the source's page under `providers/` |
| `pypi/stripe/`: `pypi-stripe-excerpt.json` and `pypi-stripe-15.4.0.json`, `pypi-stripe-15.5.0.json`, `pypi-stripe-15.6.2.json`, `pypi-stripe-16.0.0.json` | https://pypi.org/pypi/stripe/json, https://pypi.org/pypi/stripe/&lt;version&gt;/json | 2026-11-21 | PyPI metadata. Trimmed: `releases` to the saved versions, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked`; each version's `urls` to `filename`, `packagetype`, `url`, `yanked` | |
| `npm/stripe/`: `stripe-22.6.2.trimmed.tgz`, `stripe-23.0.0.trimmed.tgz`, `stripe-23.0.1.trimmed.tgz` | https://registry.npmjs.org/stripe/-/stripe-22.6.2.tgz; https://registry.npmjs.org/stripe/-/stripe-23.0.0.tgz; https://registry.npmjs.org/stripe/-/stripe-23.0.1.tgz | 2026-11-21 | MIT (the package's own license file is kept in each tarball). Trimmed: only `package.json`, the license and `esm/stripe.esm.node.d.ts`, each copied byte for byte | see the source's page under `providers/` |
| `npm/stripe/`: `npm-stripe-excerpt.json` and `npm-stripe-22.6.2.json`, `npm-stripe-23.0.0.json`, `npm-stripe-23.0.1.json` | https://registry.npmjs.org/stripe, …/&lt;version&gt; | 2026-11-21 | npm registry metadata. Trimmed: `versions` to the saved versions (name and version only) and their `time`; each version's document to `name`, `version`, `dist.tarball` | |

## SDK changelogs

Read with `Plugin{File: ...}` (`kinds/changelog/sdks.go`); `golden/<provider>-<source>.json` is what each stores from its release (`changelogtest.CheckSDKs`), the pair from the release before it on the registry (or among the repo's tags). Every file is the changelog at the release's tag, saved with `curl`, **trimmed** to two releases' sections, each kept whole and unchanged: the release the golden pins and one more release from the same file whose notes remove nothing the reader can name (the drop test). Only the pinned release's own section is read for the golden, so the trimmed copy gives the same events as the whole file (checked when each copy was made).

| File | URL | Tag commit | Saved | License; trimming | Quote |
|---|---|---|---|---|---|
| `stripe-python-4.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-python/v4.0.0/CHANGELOG.md | `e65630a` | 2026-11-20 | MIT (stripe-python); trimmed to the sections of 4.0.0 (the golden) and 3.5.0 (the drop test), each unchanged | "Removed deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `IssuerFraudRecord`, `Recipient`, `Recip…" |
| `stripe-node-10.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-node/v10.0.0/CHANGELOG.md | `321d2fd` | 2026-11-20 | MIT (stripe-node); trimmed to the sections of 10.0.0 (the golden) and 9.16.0 (the drop test), each unchanged | "Remove `AlipayAccount` and `BitcoinReceiver` from `CustomerSource`." |
| `stripe-go-73.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-go/v73.0.0/CHANGELOG.md | `234502c` | 2026-11-20 | MIT (stripe-go); trimmed to the sections of 73.0.0 (the golden) and 72.122.0 (the drop test), each unchanged | "Remove `BalanceTransactionStatus`. It was meant to be an enum, but none of the enum values were defined, so it was ju…" |
| `stripe-java-21.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-java/v21.0.0/CHANGELOG.md | `cd2b023` | 2026-11-20 | MIT (stripe-java); trimmed to the sections of 21.0.0 (the golden) and 20.136.0 (the drop test), each unchanged | "Removed `CashBalance.retrieveCashBalance` method. Prefer `CashBalance.retrieve`." |
| `stripe-ruby-7.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-ruby/v7.0.0/CHANGELOG.md | `6081371` | 2026-11-20 | MIT (stripe-ruby); trimmed to the sections of 7.0.0 (the golden) and 6.5.0 (the drop test), each unchanged | "Remove `details` method from `Issuing.Card` resource. The method was not supported." |
| `stripe-dotnet-40.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-dotnet/v40.0.0/CHANGELOG.md | `e07ebe7` | 2026-11-20 | Apache-2.0 (stripe-dotnet); trimmed to the sections of 40.0.0 (the golden) and 39.126.0 (the drop test), each unchanged | "Remove `CustomerPaymentSettingsOptions`, `CustomerPaymentSettingsPaymentMethodOptionsBancontactOptions`, `CustomerPay…" |
| `stripe-php-9.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-php/v9.0.0/CHANGELOG.md | `e405c17` | 2026-11-20 | MIT (stripe-php); trimmed to the sections of 9.0.0 (the golden) and 8.12.0 (the drop test), each unchanged | "Remove deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `Recipient`, `RecipientTransfer`, and `Th…" |

## What a release pins

Read by `kinds/changelog/pins.go`; `pins_api_test.go` and `pins_test.go` check each release's stated pin. Every file is the SDK's `CHANGELOG.md` at its newest release tag, saved with `curl` on 2026-11-20, **trimmed** to the sections listed, each kept whole and unchanged.

| File | URL | Tag commit | Sections kept | License | Quote |
|---|---|---|---|---|---|
| `pins/stripe-python-17.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-python/v17.1.0/CHANGELOG.md | `2bbff43` | 17.1.0 (changes the pin), 17.0.0 (says nothing) | MIT (stripe-python) | "This release changes the pinned API version to `2026-11-18.endive`." |
| `pins/stripe-node-23.2.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-node/v23.2.0/CHANGELOG.md | `dd92429` | 23.2.0, 23.0.1 (says nothing), 22.0.0 ("uses the same pinned API version to") | MIT (stripe-node) | "This release uses the same pinned API version to `2026-03-25.dahlia` as the last major release." |
| `pins/stripe-go-87.2.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-go/v87.2.0/CHANGELOG.md | `c97713a` | 87.2.0, 86.4.2 (says nothing), 86.0.0 ("doesn't change") | MIT (stripe-go) | "This release **doesn't** change the pinned API version; it still uses `2026-05-27.dahlia`." |
| `pins/stripe-java-34.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-java/v34.1.0/CHANGELOG.md | `d1dcf1a` | 34.1.0, 33.4.2 (says nothing), 33.0.0 ("doesn't change") | MIT (stripe-java) | "This release changes the pinned API version to `2026-10-28.endive`." |
| `pins/stripe-ruby-21.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-ruby/v21.1.0/CHANGELOG.md | `6605e66` | 21.1.0, 21.0.0 (says nothing) | MIT (stripe-ruby) | "This release changes the pinned API version to `2026-11-18.endive`." |
| `pins/stripe-dotnet-53.2.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-dotnet/v53.2.0/CHANGELOG.md | `a32d3eb` | 53.2.0, 52.4.2 (says nothing), 52.0.0 ("doesn't change") | Apache-2.0 (stripe-dotnet) | "This release changes the pinned API version to `2026-11-18.endive`." |
| `pins/stripe-php-23.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-php/v23.1.0/CHANGELOG.md | `6694b4d` | 23.1.0, 23.0.0 (says nothing), 21.0.0 ("does not change") | MIT (stripe-php) | "This release **does not** change the pinned API version. It's still `2026-06-24.dahlia`." |

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `versions/stripe-go-tags-pins.json` | https://api.github.com/repos/stripe/stripe-go/tags (shape only) | written for the test, 2026-11-20 | not a saved copy: a few real tag names in the shape of GitHub's API | for the pins tests only; no golden reads it |
