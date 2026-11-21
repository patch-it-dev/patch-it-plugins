package numpy

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// numpy 2.5.0 removed row_stack, deprecated since 2.0 (read from the
// .pyi stubs).
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"2.4.6", "2.5.0"}, Want: []string{"removed_symbol row_stack "}, Drop: [2]string{"2.5.0", "2.5.1"}})
}
