# Stripe SDKs (`stripe/python`, `stripe/node`, `stripe/go`, `stripe/java`, `stripe/ruby`, `stripe/dotnet`, `stripe/php`)

## What it watches

Classes, methods, properties, constants and resources removed or renamed in each release of Stripe's seven official server SDKs, read from that release's section of the SDK's own `CHANGELOG.md`. One `package_release` event per bullet that names what it removes plainly. Stripe's API itself is a separate source (`stripe/api`, kind `openapi`).

Kind: `changelog` (`plugins/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Tag |
|---|---|---|---|
| `stripe/python` | `pypi:stripe` | PyPI | `vX.Y.Z` |
| `stripe/node` | `npm:stripe` | npm | `vX.Y.Z` |
| `stripe/go` | `go:github.com/stripe/stripe-go/v87` (events name the module at each release's major: `…/v73`) | the repo's `vX.Y.Z` tags | `vX.Y.Z` |
| `stripe/java` | `maven:com.stripe:stripe-java` | the repo's tags | `vX.Y.Z` |
| `stripe/ruby` | `rubygems:stripe` | the repo's tags | `vX.Y.Z` |
| `stripe/dotnet` | `nuget:Stripe.net` | the repo's tags | `vX.Y.Z` |
| `stripe/php` | `packagist:stripe/stripe-php` | the repo's tags | `vX.Y.Z` |

## Why this source

- **Source:** each SDK's `CHANGELOG.md`, read raw at the release's tag, e.g. https://github.com/stripe/stripe-python/blob/master/CHANGELOG.md (checked on 2026-11-20; `https://raw.githubusercontent.com/stripe/stripe-python/v4.0.0/CHANGELOG.md` for the golden).
- **Why this one:** it's Stripe's own record of each release, and every breaking bullet is marked `⚠️`, in all seven SDKs. At a tag it never changes, so the URL is its own citation.
- **Alternatives considered:** GitHub release notes (the same text, release by release, editable afterwards); the SDKs' wiki migration guides (one page per major, prose); Stripe's API changelog (about the API, not the SDK's names); diffing the published packages (the `apisurface` kind, a separate source when it's built).
- **Quote:** "⚠️ Removed the deprecated `Charge.refund` method. Prefer [Refund.create](https://stripe.com/docs/api/refunds/create)" (stripe-python 4.0.0).

## How the provider announces changes

Stripe ships a new major of every SDK when it pins a new API version (twice a year or so), and marks each breaking bullet `⚠️`. Older changelogs head releases `## X.Y.Z - date`, newer ones `## <a id="X-Y-Z"></a>X.Y.Z - date`; both are read.

## What the plugin reads and ignores

- Reads the release's section only. A bullet counts when it's marked `⚠️` or sits under a breaking or deprecation heading, and its first sentence is a plain template over code spans: `Remove X`, `Remove X from Y`, `Rename X to Y`, `X has been removed`, `Deprecated X in favor of Y`. The pull-request link it starts with is read past.
- Drops (and logs) everything else: type changes ("Change type of …"), nullable fields, pinned API versions, dropped language versions, bullets that list names inside prose or parentheses, and "Generated changes from …" roll-ups.

## Known gaps

- Stripe's big generated roll-ups ("⚠️ Generated changes from #…") list their removals in nested bullets the reader doesn't open: those releases give no event.
- A release whose tag or section is missing is skipped as nothing to read, never guessed.
