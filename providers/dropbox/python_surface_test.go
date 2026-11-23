package dropbox

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// dropbox 12.2.4 removed Dropbox.sharing_remove_file_member, which 12.2.3
// still had (already warning "remove_file_member is deprecated."); its
// route is gone from sharing.stone too, remove_file_member_2 left in its
// place. The sharing namespace isn't one files-api reads. The drop pair,
// 12.2.0 to 12.2.1, reformats every method signature in base.py and
// changes none of them.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"12.2.3", "12.2.4"}, Want: []string{
		"removed_symbol base.DropboxBase.sharing_remove_file_member ",
	}, Drop: [2]string{"12.2.0", "12.2.1"}})
}
