# Normalizer golden cases

Small OpenAPI specs written for these tests, not copied from any provider. Each directory under `golden/` is one case: `old.json` and `new.json` in, `events.json` the exact events out.

**License:** written for this repo; Apache-2.0 like the rest of it.

Rewrite the expected events after an intended change with `go test ./normalize -run TestGolden -update`, and say in the PR why the new output is right.
