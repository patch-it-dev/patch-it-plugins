# PostHog: saved copies and goldens

Real copies of what PostHog publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/posthog -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved removed-members guide

Read through the `migrationguide` kind's removed-members reader; the golden file is what it stores from the guide.

| File | URL | Fetched | License | Quote |
|---|---|---|---|---|
| `migration-7.x-to-8.0-posthog-v8.0.0.md` | https://github.com/PostHog/posthog-python/blob/posthog-v8.0.0/docs/migration-7.x-to-8.0.md (commit 6edff41425188056e48f6119457dc9bc0669ca99) | 2026-11-19 | MIT (the repo's LICENSE is the MIT text, with two copyright lines, so GitHub's detector says NOASSERTION; PyPI says MIT), kept whole, byte for byte | "8.x.x removes these options and APIs:" (Fix removed APIs), with rows such as "\| `CaptureV1Error` \| `CaptureError` \|" |

## SDK changelogs

Read with `Plugin{File: ...}` (`kinds/changelog/sdks.go`); `golden/<provider>-<source>.json` is what each stores from its release (`changelogtest.CheckSDKs`), the pair from the release before it on the registry (or among the repo's tags). Every file is the changelog at the release's tag, saved with `curl`, **trimmed** to two releases' sections, each kept whole and unchanged: the release the golden pins and one more release from the same file whose notes remove nothing the reader can name (the drop test). Only the pinned release's own section is read for the golden, so the trimmed copy gives the same events as the whole file (checked when each copy was made).

| File | URL | Tag commit | Saved | License; trimming | Quote |
|---|---|---|---|---|---|
| `posthog-go-2.0.0-CHANGELOG.md` | https://raw.githubusercontent.com/PostHog/posthog-go/v2.0.0/CHANGELOG.md | `ee5c496` | 2026-11-20 | MIT (posthog-go); trimmed to the sections of 2.0.0 (the golden) and 1.44.1 (the drop test), each unchanged | "fc05013: Remove `Capture.Library` and `Exception.Library`. Set `$lib` in the event's `Properties` instead, and set `$…" |
