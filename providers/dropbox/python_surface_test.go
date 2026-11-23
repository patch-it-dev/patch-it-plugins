package dropbox

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// dropbox 10.11.0 (the release where this really happened) was yanked on
// PyPI, so the real pair our plugin ever sees is the one on either side of
// it: 10.10.0, where dropbox/dropbox.py defines Dropbox, DropboxTeam and
// create_session (its __all__, checked 2026-11-22: RouteResult,
// RouteErrorResult and BadInputException are also defined there but left
// out of __all__, so this source doesn't read them either), to 11.0.0,
// where the same three names, unchanged, are defined in
// dropbox/dropbox_client.py instead (dropbox.py is gone) — the rename
// UPGRADING.md's "10.X.X to 11.0.0" section describes. Code that imported
// dropbox.dropbox.Dropbox directly breaks; `from dropbox import Dropbox`
// doesn't, since dropbox/__init__.py re-exports from wherever the class
// actually lives (this source doesn't read __init__.py, so it can't tell
// the two apart: see README). The same release also drops
// dropbox/stone_fixtures.py, a Stone-generated test-fixture module with no
// namespace of its own (EmptyContainer, MixedInternalOnlyContainer,
// MixedInternalOnlyEnum) — unrelated to the rename, but genuinely gone in
// the same real pair, so it's in Want too.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"10.10.0", "11.0.0"}, Want: []string{
		"removed_symbol dropbox.Dropbox ",
		"removed_symbol dropbox.DropboxTeam ",
		"removed_symbol dropbox.create_session ",
		"removed_symbol stone_fixtures.EmptyContainer ",
		"removed_symbol stone_fixtures.MixedInternalOnlyContainer ",
		"removed_symbol stone_fixtures.MixedInternalOnlyEnum ",
	}, Drop: [2]string{"11.0.0", "11.1.0"}})
}
