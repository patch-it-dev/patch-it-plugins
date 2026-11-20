# urllib3 (`urllib3`)

## What it watches

Removals and deprecations in each urllib3 release, read from that release's section of its changelog: classes, methods and modules removed or renamed. One `package_release` event per bullet the plugin can read with certainty. The releases are listed from PyPI (final releases only) and each is read at its git tag (`<version>`); 2.0.0 is the release the golden test pins.

Format plugin: `changelog` (`plugins/changelog`).

## Why this source

- **Source:** [CHANGES.rst at 2.0.0](https://github.com/urllib3/urllib3/blob/2.0.0/CHANGES.rst), read raw at the tag (`https://raw.githubusercontent.com/urllib3/urllib3/2.0.0/CHANGES.rst`). Checked on 2026-11-14.
- **Why this one:** it's the maintainers' own changelog, with a "Removed" and a "Deprecated" section per release, each bullet linking its issue. At a tag it never changes.
- **Alternatives considered:** the [v2 migration guide](https://urllib3.readthedocs.io/en/stable/v2-migration-guide.html) (prose and examples, harder to read deterministically); GitHub release notes (same content as the changelog); diffing the package's code.
- **Quote:** "Removed ``urllib3.HTTPResponse.from_httplib``" (2.0.0, Removed).

## How the provider announces changes

- Semantic versions; 2.0.0 (2023-04-26) was the major release that removed long-deprecated APIs.
- **Forced on a fresh install** for an unpinned `urllib3`, which many packages pull in indirectly.

## What the plugin reads and ignores

- reStructuredText titles: release and section levels are told apart by their underline character.
- Same bullet templates as [`textual`](textual.md). A name the same release's "Added" section adds back isn't reported as removed: 2.0.0 removed the `urllib3.request` module and added a `urllib3.request()` function.
- Drops (and logs) dropped Python and OpenSSL versions and behavior changes. 12 events from 2.0.0.

## Known gaps

- A release is read at its tag; one whose tag or changelog section is missing fails that pair, it isn't guessed.
- Changed defaults (TLS versions, retries) are behavior changes and aren't events.
