# Providers

One page per plugin id: `providers/<api>.md`. Start from [`_template.md`](_template.md). Each page says:

- **What the plugin watches.**
- **Why this source:** the link the plugin reads, when it was checked, why it's the most reliable place this provider announces breaking changes, which other sources were considered and why they weren't used, and one short quote from it.
- **How often** it changes, and how the provider announces breaking changes.
- **What the plugin reads** from it, and what it deliberately ignores (and why).
- **Known gaps**: changes the provider makes that this plugin can't see yet.

| Page | Plugin | Format plugin |
|---|---|---|
| [stripe.md](stripe.md) | `stripe` | `openapi` |
| [datadog.md](datadog.md) | `datadog` | `openapi` |
| [github.md](github.md) | `github` | `openapi` |
| [gitlab.md](gitlab.md) | `gitlab` | `openapi` |
| [vercel.md](vercel.md) | `vercel` | `openapi` |
| [supabase.md](supabase.md) | `supabase` | `openapi` |
| [algolia.md](algolia.md) | `algolia-search`, `algolia-ingestion`, `algolia-recommend` | `openapi` |
| [openai-models.md](openai-models.md) | `openai-models` | `retirements` |
| [anthropic-models.md](anthropic-models.md) | `anthropic-models` | `retirements` |
| [anthropic.md](anthropic.md) | `anthropic` | `migrationguide` |
| [numpy.md](numpy.md) | `numpy` | `removedmembers` |
| [textual.md](textual.md) | `textual` | `changelog` |
| [urllib3.md](urllib3.md) | `urllib3` | `changelog` |
| [github-actions.md](github-actions.md) | `github-actions` | `feed` |
| [vertex-ai-models.md](vertex-ai-models.md) | `vertex-ai-models` | `feed` |
| [redis.md](redis.md) | `redis` | `catalog` |
| [cloudflare.md](cloudflare.md) | `cloudflare` | `feed` |
| [mongodb-atlas.md](mongodb-atlas.md) | `mongodb-atlas` | `apichangelog` |

The list of plugins, with their status, is [docs/plugins.md](../docs/plugins.md).
