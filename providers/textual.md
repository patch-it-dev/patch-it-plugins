# Textual (`textual/textual`)

## What it watches

Removals and breaking changes in each Textual release, read from that release's section of its changelog: symbols removed (`App.dark`, …) or renamed. One `package_release` event per changelog bullet the plugin can read with certainty. The releases are listed from PyPI (final releases only) and each is read at its tag (`v<version>`); 0.86.0 is the release the golden test pins.

Kind: `changelog` (`plugins/changelog`).

## Why this source

- **Source:** [CHANGELOG.md at v0.86.0](https://github.com/Textualize/textual/blob/v0.86.0/CHANGELOG.md), read raw at the tag (`https://raw.githubusercontent.com/Textualize/textual/v0.86.0/CHANGELOG.md`). Checked on 2026-11-14.
- **Why this one:** it's the maintainers' own changelog in the [Keep a Changelog](https://keepachangelog.com) format, with `Removed` and `Changed` sections and explicit "Breaking change:" bullets. At a tag it never changes, so the URL is its own citation.
- **Alternatives considered:** GitHub release notes (the same text, but editable after the fact and not in one file); the docs site's changelog page (rendered from this file); diffing the package's code (finds removals but not the maintainers' wording or intent).
- **Quote:** "Breaking change: Removed `App.dark` reactive attribute" (0.86.0, Changed).

## How the provider announces changes

- Textual is pre-1.0 and releases often; breaking changes can land in any minor release and are marked in the changelog.
- **Forced on upgrade**, for a repo whose pin reaches the release.

## What the plugin reads and ignores

- Reads the release's section only; a release missing from the file is an error.
- An event needs a bullet whose first sentence is one plain template over code spans: `Removed X`, `Removed the X method from Y`, `Renamed X to Y`, `X property removed`, `Deprecated X in favor of Y`, or a bare name under `Removed`. Any bullet starting "Breaking change:" counts as breaking.
- Drops (and logs) everything else: behavior changes, signature changes, removed options and parameters (of which call?), dropped Python versions. 2 events from 0.86.0.

## Known gaps

- A release is read at its tag; one whose tag or changelog section is missing fails that pair, it isn't guessed.
- Breaking behavior changes written as prose aren't events.
