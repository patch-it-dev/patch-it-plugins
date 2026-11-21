# Saved changelogs

Real copies, saved once with `curl` from the release tag and never edited. The
golden files under `golden/` are what the plugin stores from them
(`go test ./plugins/changelog -update`).

| File | URL | Tag commit | Saved | License | Quote |
|---|---|---|---|---|---|
| `textual-v0.86.0-CHANGELOG.md` | https://raw.githubusercontent.com/Textualize/textual/v0.86.0/CHANGELOG.md | `5a738fa` | 2026-11-14 08:36 UTC | MIT (Textual), kept whole | "Breaking change: Removed `App.dark` reactive attribute" |
| `urllib3-2.0.0-CHANGES.rst` | https://raw.githubusercontent.com/urllib3/urllib3/2.0.0/CHANGES.rst | `303ba81` | 2026-11-14 08:35 UTC | MIT (urllib3), kept whole | "Removed support for Python 2.7, 3.5, and 3.6" |

## SDK changelogs

Read with `Plugin{File: ...}` (`sdks.go`); `golden/<provider>-<source>.json` is what each stores from its release (`sdks_test.go`), the pair from the release before it on the registry (or among the repo's tags). Every file is the changelog at the release's tag, saved with `curl`, **trimmed** to two releases' sections, each kept whole and unchanged: the release the golden pins and one more release from the same file whose notes remove nothing the reader can name (the drop test). Only the pinned release's own section is read for the golden, so the trimmed copy gives the same events as the whole file (checked when each copy was made).

| File | URL | Tag commit | Saved | License; trimming | Quote |
|---|---|---|---|---|---|
| `stripe-python-4.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-python/v4.0.0/CHANGELOG.md | `e65630a` | 2026-11-20 | MIT (stripe-python); trimmed to the sections of 4.0.0 (the golden) and 3.5.0 (the drop test), each unchanged | "Removed deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `IssuerFraudRecord`, `Recipient`, `Recip…" |
| `stripe-node-10.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-node/v10.0.0/CHANGELOG.md | `321d2fd` | 2026-11-20 | MIT (stripe-node); trimmed to the sections of 10.0.0 (the golden) and 9.16.0 (the drop test), each unchanged | "Remove `AlipayAccount` and `BitcoinReceiver` from `CustomerSource`." |
| `stripe-go-73.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-go/v73.0.0/CHANGELOG.md | `234502c` | 2026-11-20 | MIT (stripe-go); trimmed to the sections of 73.0.0 (the golden) and 72.122.0 (the drop test), each unchanged | "Remove `BalanceTransactionStatus`. It was meant to be an enum, but none of the enum values were defined, so it was ju…" |
| `stripe-java-21.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-java/v21.0.0/CHANGELOG.md | `cd2b023` | 2026-11-20 | MIT (stripe-java); trimmed to the sections of 21.0.0 (the golden) and 20.136.0 (the drop test), each unchanged | "Removed `CashBalance.retrieveCashBalance` method. Prefer `CashBalance.retrieve`." |
| `stripe-ruby-7.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-ruby/v7.0.0/CHANGELOG.md | `6081371` | 2026-11-20 | MIT (stripe-ruby); trimmed to the sections of 7.0.0 (the golden) and 6.5.0 (the drop test), each unchanged | "Remove `details` method from `Issuing.Card` resource. The method was not supported." |
| `stripe-dotnet-40.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-dotnet/v40.0.0/CHANGELOG.md | `e07ebe7` | 2026-11-20 | Apache-2.0 (stripe-dotnet); trimmed to the sections of 40.0.0 (the golden) and 39.126.0 (the drop test), each unchanged | "Remove `CustomerPaymentSettingsOptions`, `CustomerPaymentSettingsPaymentMethodOptionsBancontactOptions`, `CustomerPay…" |
| `stripe-php-9.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/stripe/stripe-php/v9.0.0/CHANGELOG.md | `e405c17` | 2026-11-20 | MIT (stripe-php); trimmed to the sections of 9.0.0 (the golden) and 8.12.0 (the drop test), each unchanged | "Remove deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `Recipient`, `RecipientTransfer`, and `Th…" |
| `sentry-python-changelog-2.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/getsentry/sentry-python/2.0.0/CHANGELOG.md | `38bbb8d` | 2026-11-20 | MIT (sentry-python); trimmed to the sections of 2.0.0 (the golden) and 1.45.0 (the drop test), each unchanged | "Removed `sentry_sdk.tracing.Span.new_span`. Use `sentry_sdk.tracing.Span.start_child` instead." |
| `sentry-javascript-changelog-10.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/getsentry/sentry-javascript/10.0.0/CHANGELOG.md | `19f754a` | 2026-11-20 | MIT (sentry-javascript); trimmed to the sections of 10.0.0 (the golden) and 9.40.0 (the drop test), each unchanged | "feat(core)!: Remove `BaseClient` ([#17071](https://github.com/getsentry/sentry-javascript/pull/17071))" |
| `sentry-go-0.31.0-CHANGELOG.md` | https://raw.githubusercontent.com/getsentry/sentry-go/v0.31.0/CHANGELOG.md | `17ceb6c` | 2026-11-20 | MIT (sentry-go); trimmed to the sections of 0.31.0 (the golden) and 0.30.0 (the drop test), each unchanged | "Remove `Segment` field from the `User` struct. This field is no longer used in the Sentry product. ([#928](https://gi…" |
| `sentry-dotnet-4.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/getsentry/sentry-dotnet/4.0.0/CHANGELOG.md | `d4d8af5` | 2026-11-20 | MIT (sentry-dotnet); trimmed to the sections of 4.0.0 (the golden) and 3.41.4 (the drop test), each unchanged | "`ITransaction` has been renamed to `ITransactionTracer`. You will need to update any references to these interfaces i…" |
| `cloudflare-typescript-7.1.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-typescript/v7.1.0/CHANGELOG.md | `f41ff03` | 2026-11-20 | Apache-2.0 (cloudflare-typescript); trimmed to the sections of 7.1.0 (the golden) and 7.0.0 (the drop test), each unchanged | "`cloudforceOne.threatEvents.indicatorTypes` has been renamed to `cloudforceOne.threatEvents.indicators`" |
| `cloudflare-python-5.6.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-python/v5.6.0/CHANGELOG.md | `02ab9a9` | 2026-11-20 | Apache-2.0 (cloudflare-python); trimmed to the sections of 5.6.0 (the golden) and 5.4.0 (the drop test), each unchanged | "accounts:** The `create()` and `get()` methods have been removed from `client.accounts.subscriptions`. The `Subscript…" |
| `cloudflare-go-7.15.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/cloudflare-go/v7.15.0/CHANGELOG.md | `d2f77a0` | 2026-11-20 | Apache-2.0 (cloudflare-go); trimmed to the sections of 7.15.0 (the golden) and 7.13.1 (the drop test), each unchanged | "rules:** the `ListCursor` and `ListCursorParam` types are removed." |
| `cloudflare-terraform-5.29.0-CHANGELOG.md` | https://raw.githubusercontent.com/cloudflare/terraform-provider-cloudflare/v5.29.0/CHANGELOG.md | `8e7a366` | 2026-11-20 | Apache-2.0 (terraform-provider-cloudflare); trimmed to the sections of 5.29.0 (the golden) and 5.28.1 (the drop test), each unchanged | "zone_dns_settings, account_dns_settings:** the deprecated `foundation_dns` attribute has been removed. On `cloudflare…" |
| `redis-ioredis-5.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/redis/ioredis/v5.0.0/CHANGELOG.md | `b8e99b4` | 2026-11-20 | MIT (ioredis); trimmed to the sections of 5.0.0 (the golden) and 5.0.0-beta.4 (the drop test), each unchanged | "you very likely have never used it." |
| `posthog-go-2.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/PostHog/posthog-go/v2.0.0/CHANGELOG.md | `ee5c496` | 2026-11-20 | MIT (posthog-go); trimmed to the sections of 2.0.0 (the golden) and 1.44.1 (the drop test), each unchanged | "fc05013: Remove `Capture.Library` and `Exception.Library`. Set `$lib` in the event's `Properties` instead, and set `$…" |

## SDK release notes

Read with `Plugin{File: ...}` and `Notes: true` (`sdks.go`): an SDK whose changelog is its GitHub releases' notes. Each file is one release as GitHub's API returns it, trimmed to the fields the reader uses.

| File | URL | Tag commit | Saved | License; trimming | Use |
|---|---|---|---|---|---|
| `github-octokit-rest-18.0.0-release.json` | https://api.github.com/repos/octokit/rest.js/releases/tags/v18.0.0 | `b4ab0db` | 2026-11-20 | MIT (rest.js); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | golden |
| `github-octokit-rest-17.11.2-release.json` | https://api.github.com/repos/octokit/rest.js/releases/tags/v17.11.2 | `5306f37` | 2026-11-20 | MIT (rest.js); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | drop test |
| `github-octokit-net-0.5.2-release.json` | https://api.github.com/repos/octokit/octokit.net/releases/tags/v0.5.2 | `4b61105` | 2026-11-20 | MIT (octokit.net); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | golden |
| `github-octokit-net-0.5.1-release.json` | https://api.github.com/repos/octokit/octokit.net/releases/tags/v0.5.1 | `8a9e828` | 2026-11-20 | MIT (octokit.net); trimmed to `html_url`, `tag_name`, `name`, `published_at` and `body`, re-indented, values unchanged | drop test |

## Version lists

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `versions/stripe-go-tags.json`, `versions/octokit-rest-releases.json` | https://api.github.com/repos/stripe/stripe-go/tags, https://api.github.com/repos/octokit/rest.js/releases (shape only) | written for the test, 2026-11-20 | not saved copies: a few real tag names and releases in the shape of GitHub's API (a pre-release, a draft and a tag of another series among them) | for the version-listing test only (tags for an ecosystem with no registry lister, releases for release notes); no golden reads them |
| `pypi-urllib3-excerpt.json` | https://pypi.org/pypi/urllib3/json (shape only) | written for the test, 2026-11-20 | not a saved copy: four releases in the shape of PyPI's JSON API (`releases` → files with `upload_time_iso_8601`, `yanked`), dates approximate, a pre-release among them | for the version-listing test only (ListVersions, consecutive pairs); no golden reads it |

## JSON changelogs

Read with `Plugin{Log: ...}` (json.go); `golden/mongodb-atlas.json` is what it stores from the first file.

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `mongodb-atlas-changelog-c0f99ca9.json` | https://raw.githubusercontent.com/mongodb/openapi/c0f99ca9145e5aa74d08ae4451020857e93b5f28/changelog/changelog.json (the "Release API Changelog" commit of 2026-11-18 20:15 UTC; the same file as `main` on 2026-11-19) | 2026-11-19 | Apache-2.0 (mongodb/openapi) | "endpoint with API Version '2023-01-01' was removed as it has reached its sunset date '2026-01-22'" |
| `mongodb-atlas-changelog-2026-11-18-day.json` | The same file's newest day (`2026-11-18`), kept whole and unchanged: the drop test | 2026-11-19 | Apache-2.0 (mongodb/openapi) | `"changeCode": "request-property-one-of-removed"`, `"backwardCompatible": false` |

**Trimmed, positions kept:** the real file is 10 MB. `mongodb-atlas-changelog-c0f99ca9.json` keeps every day, endpoint, API version and change in place. Every change the plugin doesn't read is replaced by `{}`: all but the three life-cycle codes and `endpoint-reactivated`. So is every endpoint or version left with none. So every JSON pointer in `golden/mongodb-atlas.json` names the same change in the real file. The 105 kept changes are copied unchanged. The plugin reads exactly the same 81 findings from this copy as from the real file (checked when the copy was made). The copy is re-indented.

| File | URL | Saved | License | Use |
|---|---|---|---|---|
| `log-history/commits.json`, `log-history/changelog-{1,2,3}.json` | Stand-ins for the GitHub commits list of `changelog/changelog.json` and the file at three commits, in the real file's shape | written for the test, 2026-11-20 | n/a (test fixtures) | for the pair-by-pair test only (three versions, two pairs); no golden reads them |
