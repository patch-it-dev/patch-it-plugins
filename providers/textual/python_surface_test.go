package textual

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// textual 8.2.6's walk_selectable_widgets no longer takes bounded.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"8.2.5", "8.2.6"}, Want: []string{"removed_parameter walk.walk_selectable_widgets bounded"}, Drop: [2]string{"8.2.6", "8.2.7"}})
}
