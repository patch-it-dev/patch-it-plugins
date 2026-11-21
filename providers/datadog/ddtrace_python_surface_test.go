package datadog

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// ddtrace 4.13.0 moved AI Guard to ddtrace.aiguard: the old module
// forwards its names (__getattr__), so only the two integration helpers
// it dropped are gone.
func TestDdtracePythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: DdtracePythonSurface, Golden: [2]string{"4.12.3", "4.13.0"}, Want: []string{
		"removed_symbol appsec.ai_guard.integrations.resolve_strands_hook_provider ",
		"removed_symbol appsec.ai_guard.integrations.resolve_strands_plugin ",
	}, Drop: [2]string{"4.16.0", "4.16.1"}})
}
