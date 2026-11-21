# Google Vertex AI model retirements (`google/vertex-model-retirements`)

## What it watches

Generative AI models Google Cloud retires on Vertex AI (now called Gemini Enterprise Agent Platform): every model on Google Cloud's model versions page with a plain retirement date, one `model` event per model id, with the replacement Google names.

Kind: `notices` (`plugins/notices`), with the deprecations-table reader (`plugins/notices/retire`, `retire.Vertex`), reading HTML tables.

## Why this source

- **Source:** [Model versions and lifecycle](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions) (the old `cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versions` redirects there). Checked on 2026-11-21.
- **Why this one:** it's Google Cloud's own table of every model id with its release date, its retirement date and its replacement, including the "Retired models" table. The release-notes feed (`google/vertex-ai-models`) only announces some deprecations, in prose; this page is where the dates are kept.
- **Alternatives considered:**
  - **The Vertex AI release notes feed**: already a source (`google/vertex-ai-models`), for announcements; not every retirement is announced there with a table of ids.
  - **The deprecations page** (`docs.cloud.google.com/vertex-ai/generative-ai/docs/deprecations`): SDK modules and features, not model dates.
  - **The Gemini API deprecations page** (`google/gemini-models`): the Gemini API's own dates, which differ.
- **Quote:** "| `veo-3.0-generate-001` | July 29, 2025 | June 30, 2026 | `veo-3.1-generate-001` |" (Retired models).

## How the provider announces changes

- A GA model is kept for at least a year, written as a minimum ("October 13, 2027 or later", "No sooner than May 20, 2028"); when Google decides to retire it, the cell becomes a plain date, and the model later moves to "Retired models".

## What the plugin reads and ignores

- Reads every table with a "Retirement date", a "Model ID" and a "Replacement model ID" (or "Recommended upgrade") column: one `breaking` `model` event per model id with a plain date, `version` the date. `replacement` is set when the cell names exactly one model.
- Ignores a minimum lifetime ("… or later", "No sooner than …", "No retirement date announced"): it isn't a retirement notice. The pinned drop test: the Gemini image models' table, four rows with such dates and one dated retirement, gives one event.
- Ignores a cell that isn't a plain model id ("textembedding-gecko@003 *", with a footnote mark).

## Known gaps

- The page is about 430 KB of HTML; only its headings and tables are read, so a date given only in prose beside a table is missed.
- Partner models (Claude, Llama…) on Vertex AI are on other pages.
