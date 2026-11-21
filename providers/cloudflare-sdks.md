# Cloudflare SDKs and Terraform provider (`cloudflare/typescript`, `cloudflare/python`, `cloudflare/go`, `cloudflare/terraform`)

## What it watches

Methods, sub-resources, types and attributes removed or renamed in each release of Cloudflare's TypeScript, Python and Go SDKs and its Terraform provider, from that release's section of the repo's `CHANGELOG.md` at its tag. One `package_release` event per bullet that names what it removes plainly. Cloudflare's API notices are a separate source (`cloudflare/api`).

Kind: `changelog` (`plugins/changelog`, `sdks.go`).

| Source | Package | Releases listed from | Layer |
|---|---|---|---|
| `cloudflare/typescript` | `npm:cloudflare` | npm | sdk |
| `cloudflare/python` | `pypi:cloudflare` | PyPI | sdk |
| `cloudflare/go` | `go:github.com/cloudflare/cloudflare-go/v7` (events name the module at each release's major) | the repo's `vX.Y.Z` tags | sdk |
| `cloudflare/terraform` | none (Terraform registry `cloudflare/cloudflare`; events carry ecosystem `terraform`) | the repo's `vX.Y.Z` tags | infra |

## Why this source

- **Source:** https://github.com/cloudflare/cloudflare-typescript/blob/main/CHANGELOG.md, https://github.com/cloudflare/cloudflare-python/blob/main/CHANGELOG.md, https://github.com/cloudflare/cloudflare-go/blob/main/CHANGELOG.md, https://github.com/cloudflare/terraform-provider-cloudflare/blob/main/CHANGELOG.md, each read raw at the release's `vX.Y.Z` tag; checked on 2026-11-20.
- **Why this one:** the SDKs are generated and breaking changes ship in minors; release-please puts each under "⚠ BREAKING CHANGES", one bullet per area (`**accounts:** …`), naming what was removed.
- **Alternatives considered:** GitHub release notes (the same text); the per-major upgrade guides (`docs/guides/version-5-upgrade.md` for Terraform: one entry per resource, no plain removal sentences); diffing the published packages (`apisurface`, when it's built).
- **Quote:** "**ssl:** The `Host`, `Status`, and `ValidationMethod` types have been removed from `cloudflare.types.ssl`." (cloudflare-python 5.6.0).

## How the provider announces changes

Frequent minors, several a month; breaking bullets in a minor's "⚠ BREAKING CHANGES" section.

## What the plugin reads and ignores

- Reads the release's section; the area prefix (`**ssl:**`) is read past. Counts `X has been removed (from Y)`, `X and Y types removed`, `Remove X`, `X has been renamed to Y`, and bare names under a removal heading.
- Drops (and logs): changed return types, changed import IDs, simplified responses, renamed enum values, anything that removes a list of names "along with" others in prose.

## Known gaps

- A Terraform attribute removed from a block is named by the attribute only when the bullet doesn't name its resource plainly.
- A release whose section is missing at its tag is skipped as nothing to read.
