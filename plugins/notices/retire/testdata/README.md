# Saved model deprecations pages

The two providers' deprecations pages, as markdown (each page's URL plus `.md`). `retire`'s tests and the `openai-models` / `anthropic-models` golden tests (`plugins/notices`) read them.

**License:** excerpts for testing; © OpenAI and © Anthropic respectively. Not covered by this repo's license.

**Trimmed.** Each file keeps the page title and, for every retirement table that names at least one model the plugin keeps (text models code sends), the section heading above it and the whole table, copied unchanged. Prose, notice-period policy, endpoint tables and tables of only image, audio or embedding models were left out. `retire.Parse` returns exactly the same retirements from the trimmed and the full pages, so the golden files didn't change.

| File | URL | Fetched | Quote |
|---|---|---|---|
| `openai-deprecations.md` | https://developers.openai.com/api/docs/deprecations.md | 2026-10-13 | "\| October 23, 2026 \| `gpt-4-0613` \\\| `gpt-4`, … \| `gpt-5.6-sol` \|" |
| `anthropic-deprecations.md` | https://platform.claude.com/docs/en/about-claude/model-deprecations.md | 2026-10-13 | "\| November 30, 2026 \| `claude-sonnet-4-5-20250929` \| `claude-sonnet-5-5` \|" |
