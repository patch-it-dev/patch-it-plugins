# Google Gemini API models (`google/gemini-models`)

## What it watches

Gemini API models Google shuts down: every model on Google's own Gemini API deprecations page with a shutdown date, one `model` event per model name, with the replacement Google recommends. Google gives each date as the earliest a model may be shut down; the event says so.

Kind: `notices` (`plugins/notices`), with the deprecations-table reader (`plugins/notices/retire`, `retire.Gemini`), reading Markdoc tables.

## Why this source

- **Source:** [Gemini deprecations](https://ai.google.dev/gemini-api/docs/deprecations), which redirects to `https://aistudio.google.com/docs/deprecations`; that URL serves the page as Markdown (Markdoc `{% table %}` blocks), and the plugin reads it there. Checked on 2026-11-21.
- **Why this one:** it's Google's own table of every Gemini API model with its release date, its shutdown date and the recommended replacement, one table per model family, the shut-down ones kept under "Deprecated models". It's where Google says deprecation dates are tracked: "Deprecation announcements are made on the Release notes page, and the announced earliest shutdown dates are tracked on this page."
- **Alternatives considered:**
  - **The Gemini API release notes** (`https://aistudio.google.com/docs/changelog`): the same deprecations, announced in prose, one entry per day. The table is the same facts, already structured.
  - **Vertex AI's model versions page** (`google/vertex-model-retirements`): the same models on Google Cloud, with Google Cloud's own retirement dates, which differ from the Gemini API's (gemini-2.5-pro: 2026-10-20 on Vertex AI, 2026-12-01 here). A separate source: a repo reaches one or the other.
  - **The Gemini API's models list** (`GET /v1beta/models`): what's served now, with no dates.
- **Quote:** "| `gemini-2.0-flash` | February 5, 2025 | June 1, 2026 | `gemini-3.6-flash` |" (Gemini 2.0 models, Deprecated models), and above the tables: "The shutdown dates listed in the table indicate the earliest possible dates on which a model might be retired".

## How the provider announces changes

- Models are named by version (`gemini-2.5-flash`, `gemini-2.5-flash-preview-09-2025`); a preview gets a shutdown date months after its stable model ships, a stable model about a year after its successor.
- A model is announced in the release notes, then its date appears here. Some shut-down models' requests are routed to a successor ("effective December 1, 2026, API requests to `gemini-2.5-pro` will automatically route to `gemini-3.8-flash`"); the replacement cell says so, and the hint quotes it.

## What the plugin reads and ignores

- Reads every Markdoc table with a "Shutdown date", a "Model" and a "Recommended replacement" column: one `breaking` `model` event per model name with a date, `version` the date. `replacement` is set when the cell names exactly one plain model; otherwise the hint quotes the cell.
- Ignores a row whose date isn't a date ("No shutdown date announced"), the managed agents' tables (an "Agent" column, not a model), and a cell that isn't a plain model id (a footnote mark). The pinned drop test: the Gemini 4 models and the agents, next to the two shut-down Gemini 3 previews, give only those two events.
- A read with far fewer models than the last one fails (the table changed shape), as do a page with no table and a block page.

## Known gaps

- Google's dates are the earliest possible; a later exact date only shows up when Google edits the row, which is then a new event.
- Models served only on Vertex AI are the other source's.
