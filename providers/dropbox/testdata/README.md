# Dropbox: saved copies and goldens

Real copies of what Dropbox publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/dropbox -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved `files.stone` commits

Each file is the real commit's `files.stone`, trimmed to its `namespace` line and every `^route ` declaration line, each copied byte for byte in the file's own order; everything else (structs, unions, docstrings, route attributes) is left out, since `Routes` reads only the declaration lines. Checked that the trimmed copy's route names match the full file's exactly, for all three commits.

| File | URL | Committer date | License | Quote |
|---|---|---|---|---|
| `dropbox-files-c8364404.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/c8364404953d875801d496a81f786c5545f78223/files.stone | 2023-04-26 | MIT (dropbox/dropbox-api-spec `LICENSE`): excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route properties/remove(file_properties.RemovePropertiesArg, Void, file_properties.RemovePropertiesError) deprecated` |
| `dropbox-files-6c68656c.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/6c68656caafdff65a30943fe186a1800b9f35bbc/files.stone | 2026-05-12 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route tags/add (AddTagArg, Void, AddTagError)` (new; `properties/remove` and its two siblings are gone) |
| `dropbox-files-a4fc0169.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/a4fc0169eda7dc981facd4208ebc7b66a4db00ee/files.stone | 2026-06-07 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | same 68 route names as `dropbox-files-6c68656c.stone`: the drop test |
| `dropbox-files-222ba8c0.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/222ba8c06cbf005e38f63c253b44b60d82200f17/files.stone | 2018-05-17 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route copy_v2 (RelocationArg, RelocationResult, RelocationError)` |
| `dropbox-files-ff540e4a.stone` | https://raw.githubusercontent.com/dropbox/dropbox-api-spec/ff540e4a194313d394ca506f6a6aefa2f34bf968/files.stone | 2018-08-14 | MIT: excerpt for testing, © Dropbox, Inc.; maintainers, please confirm | `route copy:2 (RelocationArg, RelocationResult, RelocationError)`: same operation as `copy_v2` above, a renamed spelling, not a removal — the rename test |

The golden file, `golden/dropbox-files-api.json`, is every event `TestFilesAPIGolden` stores from the first two files above.

The rename pair (`dropbox-files-222ba8c0.stone`, `dropbox-files-ff540e4a.stone`) was found by walking the plugin against the real, untrimmed history of `files.stone` outside this test suite (all 69 real commits, not just the saved ones here): a naive name diff read 5 real routes as removed and re-added under Dropbox's newer `:N` spelling, which is why `Routes.Extract` matches a removed and an added name by the `Operation` they compute to, not by name, before deciding either is real.

## Saved PyPI releases (`python-surface`)

Each `.trimmed.whl` is the real release's wheel (`https://files.pythonhosted.org/packages/.../dropbox-<version>-py2-none-any.whl`, the artifact `pickArtifact` actually picks: both a `py2` and a `py3` wheel exist for every version here, and `py2` sorts first), trimmed to only the `.py` files the test needs, each copied byte for byte; the `pypi-dropbox-*.json` files are PyPI's own release metadata, trimmed to the fields the adapter reads. Checked 2026-11-22.

| Files | Source | License | Quote |
|---|---|---|---|
| `dropbox-10.10.0-py2-none-any.trimmed.whl`: `dropbox/dropbox.py`, `dropbox/stone_fixtures.py` | https://files.pythonhosted.org/packages/d3/34/03373f4dac2b5dbd83511ffdef15c30ef9c1be01452f3113384dbc2b8f4d/dropbox-10.10.0-py2-none-any.whl | MIT (dropbox/dropbox-sdk-python `LICENSE`) | `__all__ = ['Dropbox', 'DropboxTeam', 'create_session']` — why `RouteResult`, `RouteErrorResult` and `BadInputException`, also in this file, aren't read |
| `dropbox-11.0.0-py2-none-any.trimmed.whl`: `dropbox/dropbox_client.py` | https://files.pythonhosted.org/packages/11/38/99906b7d664db658591a30a7f8357d9e13111c8ba44112a46730428f95d0/dropbox-11.0.0-py2-none-any.whl | MIT | `class Dropbox(_DropboxTransport, DropboxBase)`, unchanged from `dropbox.py`, now here — the golden test's pair with `10.10.0` |
| `dropbox-11.1.0-py2-none-any.trimmed.whl`: `dropbox/dropbox_client.py` | https://files.pythonhosted.org/packages/04/f7/b4df9588a01aaec86cd0198370c6925c765a467f175f7c902601ea0096aa/dropbox-11.1.0-py2-none-any.whl | MIT | adds `USER_AUTH`/`TEAM_AUTH`/`APP_AUTH`/`NO_AUTH` module constants and an app-auth branch; no class, function or parameter changes — the drop test's pair with `11.0.0` |
| `pypi-dropbox-excerpt.json`, `pypi-dropbox-10.10.0.json`, `pypi-dropbox-11.0.0.json`, `pypi-dropbox-11.1.0.json` | https://pypi.org/pypi/dropbox/json, https://pypi.org/pypi/dropbox/\<version\>/json | PyPI metadata | `10.11.0` — the release `UPGRADING.md`'s "10.X.X to 11.0.0" section describes the rename at — is marked `"yanked": true` on both its files; kept in the excerpt to show why the real pair this source ever compares is `10.10.0` → `11.0.0`, not `10.10.0` → `10.11.0` |

The `10.10.0` → `11.0.0` pair was found the same way as the Stone rename above: by first checking `UPGRADING.md`'s own account of the `dropbox.dropbox` → `dropbox.dropbox_client` move against the real wheels on PyPI, which is where the yank turned up.
