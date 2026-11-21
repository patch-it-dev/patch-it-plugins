# <Provider> (`<provider>/<source>`)

Copy this file to `providers/<provider>-<source>.md` and fill every section. Delete these instructions.

## What it watches

One or two sentences: which API, SDK, package or runtime, and which kinds of change (removed fields, model shutdowns, major releases…).

## Why this source

The link (or links) the plugin reads, and **why this one**:

- **Source:** `<URL>`, checked on `<YYYY-MM-DD>`.
- **Why this one:** why it's the most reliable place this provider announces breaking changes (an official spec beats a changelog; a changelog beats a blog post; a blog post beats a forum).
- **Alternatives considered:** other places the provider publishes changes, and why the plugin doesn't read them (incomplete, not machine-readable, published late, marketing only…).
- **Quote:** one short line copied from the source that shows the kind of change the plugin turns into an event.

## How the provider announces changes

How often it changes, how it versions its API, and how much notice it gives before something breaks.

## What the plugin reads and ignores

What becomes an event, and what is deliberately dropped (and why). Precision over recall: when unsure, drop.

## Known gaps

Changes this provider makes that the plugin can't see yet.
