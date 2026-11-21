# Octokit SDKs (`github/octokit-rest`, `github/octokit-net`)

## What it watches

Methods removed or renamed in each release of `@octokit/rest` (JavaScript) and Octokit (.NET), from each GitHub release's notes: neither keeps a changelog file. Packages `npm:@octokit/rest` and `nuget:Octokit`; releases listed from each repo's GitHub releases (`vX.Y.Z`, drafts and pre-releases left out). GitHub's REST API itself is a separate source (`github/rest-api`).

Kind: `changelog` (`plugins/changelog`, `sdks.go`, with `Notes: true`).

## Why this source

- **Source:** https://github.com/octokit/rest.js/releases and https://github.com/octokit/octokit.net/releases, each release read through GitHub's API (`/repos/{repo}/releases/tags/{tag}`) and cited by its release page; checked on 2026-11-20.
- **Why this one:** it's the only per-release record either project publishes. rest.js's semantic-release notes list every removal and rename under "BREAKING CHANGES" (`` `octokit.users.listEmails()` has been renamed to `octokit.users.listEmailsForAuthenticated()` ``).
- **Alternatives considered:** the docs sites (current version only); diffing the published packages (`apisurface`, when it's built).
- **Quote:** "deprecated `octokit.reactions.deleteLegacy()` has been removed" (@octokit/rest 18.0.0).

## How the provider announces changes

rest.js ships a major when GitHub removes or renames REST endpoints; octokit.net's notes list "Breaking changes" in prose more often than by name.

## What the plugin reads and ignores

- Every heading in a release's notes starts a section; bullets under a breaking or deprecation heading count when they name what they remove or rename plainly.
- Drops (and logs): signature changes (`ISshKeysClient.Get(int id)`), dropped Node.js versions, "remove deprecated endpoints" without names.

## Known gaps

- Release notes can be edited after the release: what's read is the notes when the pair is extracted.
- octokit.net's recent releases describe breaking changes in prose: few of them are events.
