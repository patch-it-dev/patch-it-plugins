# Golden events

`golden/anthropic-1.0.0.json` is what the `anthropic` plugin stores from anthropic-sdk-python's MIGRATION.md at v1.0.0. The saved guide itself is shared with the `sdkguide` package: see [`sdkguide/testdata/README.md`](../../../sdkguide/testdata/README.md) for its URL, date and license.

Rewrite it after an intended change with `go test ./plugins/migrationguide -update`, and say in the PR why the new output is right.
