# Golden events

`golden/openai-models.json` and `golden/anthropic-models.json` are what the `openai-models` and `anthropic-models` plugins store from the saved deprecations pages. The pages themselves are shared with the `retire` package: see [`retire/testdata/README.md`](../../../retire/testdata/README.md) for their URLs, dates and licenses.

Rewrite them after an intended change with `go test ./plugins/retirements -update`, and say in the PR why the new output is right.
