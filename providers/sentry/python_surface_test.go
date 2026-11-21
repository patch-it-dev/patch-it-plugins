package sentry

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// sentry-sdk 2.74.0 removed utils.to_timestamp. 2.68.0 renamed a
// property setter's parameter: callers never pass it.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"2.73.0", "2.74.0"}, Want: []string{"removed_symbol utils.to_timestamp "}, Drop: [2]string{"2.67.1", "2.68.0"}})
}
