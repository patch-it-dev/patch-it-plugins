# OpenAI model retirements (`openai-models`)

## What it watches

OpenAI models that are being shut down: every text model name code sends to the Chat Completions or Responses API, with its shutdown date and the replacement OpenAI recommends. One breaking event per model name.

Format plugin: `retirements` (`plugins/retirements`), reading with `plugins/retirements/retire/`.

## Why this source

- **Source:** [developers.openai.com/api/docs/deprecations](https://developers.openai.com/api/docs/deprecations), read as markdown at `https://developers.openai.com/api/docs/deprecations.md`. Checked on 2026-10-13.
- **Why this one:** it's OpenAI's own list of every deprecation, each announcement a table of shutdown date, model and recommended replacement. The markdown version is machine-readable without scraping HTML. `robots.txt` allows it.
- **Alternatives considered:** the models list API (`GET /v1/models`: says what exists now, not when it stops); OpenAI's blog and email notices (announce only some changes, no table); the old `platform.openai.com/docs/deprecations` URL (redirects here).
- **Quote:** "| October 23, 2026 | `gpt-4-0613` \| `gpt-4`, … | `gpt-5.6-sol` |" (2026-04-22 announcement, "Legacy GPT model snapshots").

## How the provider announces changes

- An announcement every few weeks, shutdowns a few months later (at least 6 months for generally available models, 3 for specialized variants, possibly 2 weeks for previews, per the page).
- **Forced, by date**: a request naming a retired model fails on every SDK version. So the event's `version` is the shutdown date.
- The page keeps past announcements, newest first.

## What the plugin reads and ignores

- Reads every table whose first column is a shutdown or retirement date and which has a model column and a replacement column. Each backticked name in the model cell is one model (aliases after an escaped pipe too).
- A model listed twice keeps its first (newest) row; a newer "To be announced" withdraws an older date.
- A replacement that is itself retired is followed to one that still works (`gpt-3.5-turbo-0613` → `gpt-3.5-turbo` → `gpt-5.6-terra`), and the hint says so.
- Ignores image, audio, realtime, transcription, embedding and moderation models, fine-tuned bases (`ft-gpt-4`; code sends `ft:gpt-4-0613:org::id`) and the legacy `-completions` aliases: the events are for the operations that take text models.
- Every event cites a Wayback Machine snapshot of the page, never the live page, and is read from that snapshot. No snapshot, or one that differs from the live page, stores nothing that run. A page that suddenly lists far fewer retirements fails the run (a renamed column probably hid a table).

## Known gaps

- Replacements that need a setting (`gpt-5.6-sol` with `reasoning.mode: pro`) or name no model are passed on as written; a fix can't be generated from the second kind.
- Image, audio and embedding model retirements aren't events.
