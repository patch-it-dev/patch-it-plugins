# README template for patch-it's public repos

Every public patch-it repo's README has the same shape, so a developer landing on any of them knows in ten seconds what it is and how to try it. This repo's [README](../README.md) is the worked example. Copy the skeleton at the bottom, then fill each section using the notes.

## The sections, in order

A README is **one screen**. Everything else goes in `docs/` and is linked.

| # | Section | What goes in it |
|---|---|---|
| 1 | **What** (two lines at most, bold first sentence, under the title) | What *this repo* is, in the product's words. For a part of patch-it, say which part of the story it is ("how patch-it notices, on its own, that something changed"). |
| 2 | **Badges** (one line of them) | The badge set below, in that order. |
| 3 | **Quickstart** (three steps) | The shortest path from a clean clone to the repo's main job, each step a command that runs as written. |
| 4 | **Links** | `docs/` pages (how-to, how it works), CONTRIBUTING, AGENTS, Discussions, good first issues, license, docs.patch-it.dev. |

**Never a list of what's covered** (providers, plugins, layers with statuses, counts): those change at run time and the code is their only source. Say how to find them (the docs site, `make plugins`) instead.

## The badge set

In this order; drop one only if it can't be honest for the repo (no release yet: no release badge):

1. **CI**: the main workflow's badge, `https://github.com/patch-it-dev/<repo>/actions/workflows/<workflow>.yml/badge.svg`.
2. **Package reference**, if it's a library: Go Reference (`https://pkg.go.dev/badge/<module>.svg`), PyPI or npm version.
3. **Language version**: e.g. `https://img.shields.io/github/go-mod/go-version/patch-it-dev/<repo>`.
4. **Latest release**, once there is one: `https://img.shields.io/github/v/release/patch-it-dev/<repo>`.
5. **License**: `https://img.shields.io/badge/license-<spdx>-3f5bf0`.
6. **No counts.** A badge that counts plugins, providers or anything else that changes at run time is a hand-kept list in disguise: link the docs instead.
7. **Docs**: `https://img.shields.io/badge/docs-docs.patch--it.dev-3f5bf0`.
8. **Discussions**: `https://img.shields.io/github/discussions/patch-it-dev/<repo>`.

Static badges use the brand accent, `3f5bf0` (`accent`, light theme, in the brand tokens).

## Words

- Say "patch-it" (lowercase), and refer to Patchy as "Patchy" or "patch-it", never "he".
- Always list the layers; never describe patch-it as API-only.
- Use: "on its own", "writes the fix", "the code change, not just the version", "you review and merge".
- Don't name version-bump bots in headlines, the first lines or repo descriptions, and don't write "better than", "killer" or "magic". A factual comparison belongs on the docs' comparison page only.
- A public README never mentions private repos, internal services or pricing that isn't public. The service that runs plugins is "patch-it's hosted ingestion service".

## Images

Use only images that are already public, linked by URL, never copied in: Patchy and the wordmark from the organization profile, `https://raw.githubusercontent.com/patch-it-dev/.github/main/profile/assets/<file>.svg` (`bot-asleep`, `bot-found-one`, `bot-merged`, `bot-excited`, `patchy-night-shift`, `wordmark-plain-on-light`, `wordmark-plain-on-dark`). Give every image alt text that says what it shows. No gradients, glows or starry skies.

## Skeleton

````markdown
<img align="right" src="https://raw.githubusercontent.com/patch-it-dev/.github/main/profile/assets/bot-found-one.svg" alt="Patchy, the patch-it bot" width="120">

# <repo>

**<What this repo is, in one line.>** <One more line at most.>

<badges, in the order above>

## <Quickstart: the repo's main job in three steps>

1. ...
2. ...
3. ...

## More

- <links to docs/, CONTRIBUTING, AGENTS, Discussions, license, docs.patch-it.dev>
````
