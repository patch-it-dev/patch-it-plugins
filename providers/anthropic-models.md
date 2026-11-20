# Anthropic model retirements (`anthropic-models`)

## What it watches

Claude models that are retired or have a retirement date on the Claude API, with the replacement Anthropic recommends. One breaking event per model name.

Kind: `notices` (`plugins/notices`), reading with the deprecations-page reader, `plugins/notices/retire/`.

## Why this source

- **Source:** [platform.claude.com/docs/en/about-claude/model-deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations), read as markdown at the same URL plus `.md`. Checked on 2026-10-13.
- **Why this one:** it's Anthropic's own deprecation history, one table per announcement (retirement date, deprecated model, recommended replacement), served as markdown. `robots.txt` disallows only `/api/`.
- **Alternatives considered:** the models API (`GET /v1/models`: what exists now, no dates); release notes and emails (not a complete list); the old docs.anthropic.com URL (redirects here).
- **Quote:** "| November 30, 2026 | `claude-sonnet-4-5-20250929` | `claude-sonnet-5-5` |" (Deprecation history, 2026-09-30: Claude Sonnet 4.5 model).

## How the provider announces changes

- A few announcements a year, at least 60 days before retirement (per the page).
- **Forced, by date**, on the Claude API: the event's `version` is the retirement date. Amazon Bedrock and Google Cloud set their own dates, so code calling Claude through them may be affected on a different day.

## What the plugin reads and ignores

- Same reader as [`openai-models`](openai-models.md): every dated table with a model and a replacement column, newest row wins, retired replacements followed, every event read from and citing a Wayback snapshot.
- Keeps every `claude-` name. Ignores the "Model status" table (active models with tentative dates) and any model without a date yet.

## Known gaps

- Retirements on Amazon Bedrock or Google Cloud aren't on this page.
- The page also documents SDK changes (the Python SDK 1.0 removing sampling parameters); those are the [`anthropic`](anthropic.md) plugin, not this one.
