# Saved feeds

Real copies, saved once with `curl`. The golden files under `golden/` are what
the plugin stores from them (`go test ./plugins/feed -update`).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `github-changelog-actions.rss.xml` | https://github.blog/changelog/label/actions/feed/ (RSS 2.0; `ETag` and `Last-Modified` sent) | 2026-11-14 08:27 UTC | Excerpt for testing; © GitHub, Inc. | "The macOS 14 runner image will be retired on November 2, 2026." |
| `vertex-ai-generative-ai.atom.xml` | https://docs.cloud.google.com/feeds/generative-ai-on-vertex-ai-release-notes.xml (Atom; `https://cloud.google.com/feeds/...` 301s here; `Last-Modified` sent) | 2026-11-14 08:27 UTC | CC BY 4.0 (Google Cloud documentation), kept whole | "Vertex AI Extensions is deprecated and will be shut down after November 26, 2026." |

**Trimmed:** `github-changelog-actions.rss.xml` keeps the channel header and the three items the `github-actions` reader acts on (the macOS 14 retirement, which becomes events, and the two marked entries it drops: the ARM32 retirement and the `pull_request_target` change), copied unchanged. The seven unmarked items were left out; the reader ignores them, so its findings and the golden file are unchanged.
