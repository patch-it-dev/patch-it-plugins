package stripe

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// stripe 16.0.0 removed StripeObject.request. 15.5.0 only added.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"15.6.2", "16.0.0"}, Want: []string{"removed_symbol StripeObject.request "}, Drop: [2]string{"15.4.0", "15.5.0"}})
}
