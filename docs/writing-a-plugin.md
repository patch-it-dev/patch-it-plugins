# Writing a plugin

> The plugin code is still moving into this repo, so this is the short version. The full guide, with the Go interfaces and a worked example, lands with the code.

A plugin watches one provider source and turns what changed into change events. Most new providers don't need a new kind of plugin: they are a new entry for an existing format plugin (a new OpenAPI spec, a new deprecations page, a new feed).

## Steps

1. **Pick the format plugin** that reads your source: `openapi`, `retirements`, `migrationguide`, `removedmembers`, `changelog` or `feed` (see the [columns in plugins.md](plugins.md#the-columns)). Only write a new one if none fits.
2. **Register it** with a unique lowercase id (the `--api` value).
3. **Fetch only through the plugin environment** (`env.Get`, or the helpers in `kit/`). Plugins never open their own HTTP connections, read files outside `testdata/`, or touch a database.
4. **Save a real copy of the source** under the plugin's `testdata/`, with a `testdata/README.md` row: URL, date fetched, license, and a short quote of the part your plugin reads. Trim vendor pages to what the test needs; keep openly licensed files whole.
5. **Write a golden test**: the source in, the exact events out (`testdata/golden/<api>.json`).
6. **Pin one drop test**: a real-looking change your plugin must *not* report (a renamed heading, an unrelated section, a non-breaking addition). Precision over recall: a missed change is a bug, a wrong one costs a user a bad pull request.
7. **Write `providers/<api>.md`**: what the provider publishes, where, how often, and what the plugin reads from it.
8. **Add a row to [plugins.md](plugins.md).** Not optional: the docs site's list of supported APIs is built from it.

## What makes an event

Every event names its source (a URL the user can open) and quotes the line it came from. No source, no event. The event format is defined in [patch-it's contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract).
