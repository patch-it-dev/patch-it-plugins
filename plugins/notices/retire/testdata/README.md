# Saved model deprecations pages

The two providers' deprecations pages, as markdown (each page's URL plus `.md`). `retire`'s tests and the `openai-models` / `anthropic-models` golden tests (`plugins/notices`) read them.

**License:** excerpts for testing; © OpenAI and © Anthropic respectively. Not covered by this repo's license. Google's two pages are © Google, CC BY 4.0 (Google's developer documentation).

**Trimmed.** Each file keeps the page title and, for every retirement table that names at least one model the plugin keeps (text models code sends), the section heading above it and the whole table, copied unchanged. Prose, notice-period policy, endpoint tables and tables of only image, audio or embedding models were left out. `retire.Parse` returns exactly the same retirements from the trimmed and the full pages, so the golden files didn't change.

| File | URL | Fetched | Quote |
|---|---|---|---|
| `openai-deprecations.md` | https://developers.openai.com/api/docs/deprecations.md | 2026-10-13 | "\| October 23, 2026 \| `gpt-4-0613` \\\| `gpt-4`, … \| `gpt-5.6-sol` \|" |
| `anthropic-deprecations.md` | https://platform.claude.com/docs/en/about-claude/model-deprecations.md | 2026-10-13 | "\| November 30, 2026 \| `claude-sonnet-4-5-20250929` \| `claude-sonnet-5-5` \|" |
| `gemini-deprecations.md` | https://aistudio.google.com/docs/deprecations (where https://ai.google.dev/gemini-api/docs/deprecations redirects; served as Markdown). Kept whole, byte for byte | 2026-11-21 | "\| `gemini-2.0-flash` \| February 5, 2025 \| June 1, 2026 \| `gemini-3.6-flash` \|", as Markdoc `* cell` lines |
| `vertex-model-versions.html` | https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions. Excerpt: every `<h2>`/`<h3>` line and every line of every `<table>` (982 of 10,074 lines), unchanged; scripts, navigation and prose left out. `retire.Parse` finds the same 39 retirements in the excerpt as in the full page | 2026-11-21 | "&lt;td&gt;June 30, 2026&lt;/td&gt;" in the "Retired models" table |
