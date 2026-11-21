# <Provider>

Every source patch-it reads for <Provider>, one section each: what it watches, why this source, what it reads and ignores, its known gaps. Each is registered in [`provider.go`](provider.go) and read by its kind ([`kinds/`](../../kinds/)); the saved copies its tests replay, and the golden files, are in [`testdata/`](testdata/README.md).

<!--
This is providers/<provider>/README.md: one page per provider, one section per source. Copy this file there for a new provider (make new-plugin does), or add a section for a new source of a provider that has a page. Fill every part of the section and delete these instructions. `go test ./providers/all` fails while a registered source isn't named on its provider's page as `<provider>/<source>`.
-->

## <What the source is> (`<provider>/<source>`)

### What it watches

One or two sentences: which API, SDK, package or runtime, and which kinds of change (removed fields, model shutdowns, major releases…).

Kind: `<kind>` (`kinds/<kind>`), and the reader or adapter it uses.

### Why this source

The link (or links) the source reads, and **why this one**:

- **Source:** `<URL>`, checked on `<YYYY-MM-DD>`.
- **Why this one:** why it's the most reliable place this provider announces breaking changes (an official spec beats a changelog; a changelog beats a blog post; a blog post beats a forum).
- **Alternatives considered:** other places the provider publishes changes, and why the source doesn't read them (incomplete, not machine-readable, published late, marketing only…).
- **Quote:** one short line copied from the source that shows the kind of change the source turns into an event.

### How the provider announces changes

How often it changes, how it versions its API, and how much notice it gives before something breaks.

### What the plugin reads and ignores

What becomes an event, and what is deliberately dropped (and why). Precision over recall: when unsure, drop.

### Known gaps

Changes this provider makes that the source can't see yet.
