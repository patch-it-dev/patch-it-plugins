package google

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// google-genai 1.0.0 removed TuningJobOrOperation. 3.4.0 re-exports
// interactions' names from a private module: still importable.
func TestGenaiPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: GenaiPythonSurface, Golden: [2]string{"0.8.0", "1.0.0"}, Want: []string{
		"removed_symbol genai.types.TuningJobOrOperation ",
		"removed_symbol genai.types.TuningJobOrOperationDict ",
	}, Drop: [2]string{"3.3.0", "3.4.0"}})
}
