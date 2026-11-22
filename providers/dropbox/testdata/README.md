# Dropbox: saved copies and goldens

Real copies of what Dropbox publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/dropbox -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved `files.stone` commits

Each file is the real commit's `files.stone`, trimmed to its `namespace` line and every `^route ` declaration line, each copied byte for byte in the file's own order; everything else (structs, unions, docstrings, route attributes) is left out, since `Routes` reads only the declaration lines. Checked that the trimmed copy's route names match the full file's exactly, for all three commits.

| File | URL | Committer date | License | Quote |
|---|---|---|---|---|
| `dropbox-files-c8364404.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/c8364404953d875801d496a81f786c5545f78223/files.stone | 2023-04-26 | MIT (dropbox/dropbox-api-spec `LICENSE`): excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route properties/remove(file_properties.RemovePropertiesArg, Void, file_properties.RemovePropertiesError) deprecated` |
| `dropbox-files-6c68656c.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/6c68656caafdff65a30943fe186a1800b9f35bbc/files.stone | 2026-05-12 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route tags/add (AddTagArg, Void, AddTagError)` (new; `properties/remove` and its two siblings are gone) |
| `dropbox-files-a4fc0169.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/a4fc0169eda7dc981facd4208ebc7b66a4db00ee/files.stone | 2026-06-07 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | same 68 route names as `dropbox-files-6c68656c.stone`: the drop test |

The golden file, `golden/dropbox-files-api.json`, is every event `TestFilesAPIGolden` stores from the first two files above.
