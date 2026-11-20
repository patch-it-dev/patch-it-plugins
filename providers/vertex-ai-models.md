# Vertex AI generative models (`vertex-ai-models`)

## What it watches

Generative AI model endpoints on Google Cloud's Vertex AI that Google deprecates: one `model` event per model id in a "Discontinued endpoints" table, with the replacement when the row names exactly one.

Kind: `notices` (`plugins/notices`), with the feed reader (`plugins/notices/feed`) and its `VertexAIModels` feed.

## Why this source

- **Source:** the [Generative AI on Vertex AI release notes](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/release-notes) Atom feed, `https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml` (the `cloud.google.com/feeds/…` URL redirects here). Checked on 2026-11-14.
- **Why this one:** it's Google's own release notes as a feed, one entry per day, each change under a heading naming its kind (`Deprecated`), and deprecations come with a table of discontinued endpoints and replacements.
- **Alternatives considered:** the model versions and lifecycle docs page (HTML, rewritten in place, no history); the Google Cloud deprecations page (all of Cloud, prose); the models list API (what exists now, not when it stops).
- **Quote:** "We recommend updating your model endpoints before June 30, 2026, to avoid service disruption." (Deprecated: Imagen generation GA endpoints deprecation).

## How the provider announces changes

- Deprecations are announced in the release notes with a date to move by; shutdown dates aren't always given.
- Content is licensed CC BY 4.0.

## What the plugin reads and ignores

- Only `Deprecated` sections with a "Discontinued endpoints" table; one event per model id, `provider: google`. `replacement` only when the row names one other plain id.
- The date the section says to move by is the event's `version`; a section saying it "is deprecated" with only a "before …" date is a deprecation, not breaking.
- Drops (and logs) product names in prose (third-party models mentioned in a sentence) and non-model deprecations (Vertex AI Extensions). 35 events from the saved feed.
- Every event cites a Wayback Machine snapshot of the feed and is read from it.

## Known gaps

- Whether an "update before <date>" deprecation should wait for a stated shutdown date is open; today it's a deprecation with that date.
- Partner models (Claude, Llama on Vertex) are only named in prose, so they're dropped.
