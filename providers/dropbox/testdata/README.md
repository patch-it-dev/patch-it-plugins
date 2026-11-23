# Dropbox: saved copies and goldens

Real copies of what Dropbox publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/dropbox -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved `files.stone` commits

Each file is the real commit's `files.stone`, trimmed to its `namespace` line and every `^route ` declaration line, each copied byte for byte in the file's own order; everything else (structs, unions, docstrings, route attributes) is left out, since `Routes` reads only the declaration lines. Checked that the trimmed copy's route names match the full file's exactly, for every commit saved (66 routes at `c8364404`, 67 at `6c68656c` and `a4fc0169`, 50 at `222ba8c0`, 51 at `ff540e4a`).

| File | URL | Committer date | License | Quote |
|---|---|---|---|---|
| `dropbox-files-c8364404.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/c8364404953d875801d496a81f786c5545f78223/files.stone | 2023-04-26 | MIT (dropbox/dropbox-api-spec `LICENSE`): excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route properties/remove(file_properties.RemovePropertiesArg, Void, file_properties.RemovePropertiesError) deprecated` |
| `dropbox-files-6c68656c.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/6c68656caafdff65a30943fe186a1800b9f35bbc/files.stone | 2026-05-12 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route tags/add (AddTagArg, Void, AddTagError)` (moved here from `file_tagging.stone`, which this commit deletes; `properties/remove` and its two siblings are gone) |
| `dropbox-files-a4fc0169.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/a4fc0169eda7dc981facd4208ebc7b66a4db00ee/files.stone | 2026-06-07 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | same 67 route names as `dropbox-files-6c68656c.stone`: the drop test |
| `dropbox-files-222ba8c0.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/222ba8c06cbf005e38f63c253b44b60d82200f17/files.stone | 2018-05-17 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route copy_v2 (RelocationArg, RelocationResult, RelocationError)` |
| `dropbox-files-ff540e4a.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/ff540e4a194313d394ca506f6a6aefa2f34bf968/files.stone | 2018-08-14 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route copy:2 (RelocationArg, RelocationResult, RelocationError)`: same operation as `copy_v2` above, a renamed spelling, not a removal — the rename test |

## Saved directory listings and other files of the namespace

`Routes` checks a name that looks removed or added against the other `.stone` files declaring the same namespace at that commit, so the pairs with a change need GitHub's listing of the repo's top directory at the commit they look at.

| File | URL | Recorded | Trimmed to | Quote |
|---|---|---|---|---|
| `dropbox-spec-listing-c8364404.json` | https://api.github.com/repos/dropbox/dropbox-api-spec/contents?ref=c8364404953d875801d496a81f786c5545f78223 | 2026-11-22 | 2 of 46 entries, `name`, `path`, `type`: `files.stone` and `file_tagging.stone`, the only two of its 40 `.stone` files declaring `namespace files` (checked against every one) | |
| `dropbox-spec-listing-6c68656c.json` | https://api.github.com/repos/dropbox/dropbox-api-spec/contents?ref=6c68656caafdff65a30943fe186a1800b9f35bbc | 2026-11-22 | 1 of 29: `files.stone`, the only one of its 23 `.stone` files declaring `namespace files` | |
| `dropbox-spec-listing-222ba8c0.json` | https://api.github.com/repos/dropbox/dropbox-api-spec/contents?ref=222ba8c06cbf005e38f63c253b44b60d82200f17 | 2026-11-22 | 1 of 30: `files.stone`, the only one of its 29 `.stone` files declaring `namespace files` | |
| `dropbox-file_tagging-c8364404.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/c8364404953d875801d496a81f786c5545f78223/file_tagging.stone | 2026-11-22 | its `namespace` line and its 3 `^route ` lines, byte for byte; MIT, as above | `namespace files`, `route tags/add(AddTagArg, Void, AddTagError)`: why the golden pair's `tags/*` in `files.stone` aren't additions |

The golden file, `golden/dropbox-files-api.json`, is every event `TestFilesAPIGolden` stores from `dropbox-files-c8364404.stone` and `dropbox-files-6c68656c.stone`, checked against the listings and the `file_tagging.stone` copy above.

The rename pair (`dropbox-files-222ba8c0.stone`, `dropbox-files-ff540e4a.stone`) was found by walking the plugin against the real, untrimmed history of `files.stone` outside this test suite (all 69 real commits, not just the saved ones here): a naive name diff read 5 real routes as removed and re-added under Dropbox's newer `:N` spelling, which is why `Routes.Extract` matches a removed and an added name by the `Operation` they compute to, not by name, before deciding either is real.

## Saved PyPI releases (`python-surface`)

Each `.trimmed.whl` is the real release's wheel (the only wheel each release has), trimmed to `dropbox/base.py`, copied byte for byte: every method the tests compare is in it. The `pypi-dropbox-*.json` files are PyPI's own metadata, trimmed to the fields the adapter reads, and the excerpt's `releases` to the four saved versions. Checked 2026-11-22.

| File | URL | License | Quote |
|---|---|---|---|
| `dropbox-12.2.0-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/03/52/a9deb96572656bd773955be3b86e58ce7488a883cc8b56ae09f3f07e6437/dropbox-12.2.0-py3-none-any.whl | MIT (dropbox/dropbox-sdk-python `LICENSE`) | the drop test's older side: 169 of `base.py`'s 180 methods have their signature split one parameter per line |
| `dropbox-12.2.1-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/f4/8d/4dfdb50aa7378cfe8be2820ed375d840340a9dfb4971e76c780883c860d5/dropbox-12.2.1-py3-none-any.whl | MIT (dropbox/dropbox-sdk-python `LICENSE`) | the drop test's newer side: the same 180 methods, 34 of them split; no name or parameter changed |
| `dropbox-12.2.3-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/de/01/c8cb45346fd2df249d6f6a311ec25cd84c37c8f549f5f204d09586a1debf/dropbox-12.2.3-py3-none-any.whl | MIT (dropbox/dropbox-sdk-python `LICENSE`) | `def sharing_remove_file_member(self, file, member)` (line 6054), already warning `"remove_file_member is deprecated."` |
| `dropbox-12.2.4-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/3e/c3/442b4dd1360c5d4a3b146de0732949c71527fd5282ef7a1e7711cd27037d/dropbox-12.2.4-py3-none-any.whl | MIT (dropbox/dropbox-sdk-python `LICENSE`) | `sharing_remove_file_member` gone; `sharing_remove_file_member_2` still there |
| `pypi-dropbox-excerpt.json`, `pypi-dropbox-12.2.0.json`, `pypi-dropbox-12.2.1.json`, `pypi-dropbox-12.2.3.json`, `pypi-dropbox-12.2.4.json` | https://pypi.org/pypi/dropbox/json, https://pypi.org/pypi/dropbox/&lt;version&gt;/json | PyPI metadata | |

The golden file, `golden/dropbox-python-surface.json`, is every event `TestPythonSurface` stores from 12.2.3 → 12.2.4 (one, the removal; with only `base.py` saved, nothing is added).
