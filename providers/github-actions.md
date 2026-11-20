# GitHub Actions runner images (`github-actions`)

## What it watches

GitHub-hosted runner images that GitHub retires: one `ci_runtime` event per `runs-on` label the retirement removes (`macos-14`, `macos-14-large`, …), dated by the removal date.

Kind: `notices` (`plugins/notices`), with the feed reader (`plugins/notices/feed`) and its `GitHubActions` feed.

## Why this source

- **Source:** the [GitHub changelog, Actions label](https://github.blog/changelog/label/actions/), as RSS: `https://github.blog/changelog/label/actions/feed/`. Checked on 2026-11-14.
- **Why this one:** it's GitHub's own changelog feed, built to be machine-read, with labels (`Retired`) as RSS categories. Retirement posts list the affected labels in a fixed shape and give the date.
- **Alternatives considered:** the [actions/runner-images](https://github.com/actions/runner-images) repo's announcement issues (detailed, but issue text varies and they're not a feed); the full GitHub changelog feed (every product, much noisier); the runner images README (says what exists now, not when it goes).
- **Quote:** "The macOS 14 runner image will be retired on November 2, 2026."

## How the provider announces changes

- A retirement post weeks or months ahead, often with scheduled "brownouts" where jobs on the old image fail temporarily.
- **Forced**: on the date, jobs using the label stop running.
- The feed carries only the latest 10 posts, so the plugin runs daily and cites a snapshot.

## What the plugin reads and ignores

- Only posts labelled `Retired` about a runner image that list `runs-on` labels.
- **Breaking needs GitHub to say it stops on a date** ("will be retired / removed on …"); softer words ("will be deprecated on", "before …") make it a deprecation; no dated sentence and the post is dropped.
- Drops (and logs) retirements with no runner label to name (the ARM32 Linux and Windows retirement) and behavior changes (`pull_request_target` blocked by default).
- Every event cites a Wayback Machine snapshot of the feed and is read from it.

## Known gaps

- The patch-it client doesn't fix `ci_runtime` events yet.
- Deprecations of actions themselves (`actions/upload-artifact@v3`, …) aren't read.
