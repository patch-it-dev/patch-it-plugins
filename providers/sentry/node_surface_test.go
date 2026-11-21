package sentry

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @sentry/node 10.0.0 stopped exporting systemErrorIntegration. 11.13.0
// made the logger namespace a const object: its functions are still
// there, as members the reader doesn't follow.
func TestNodeSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: NodeSurface, Golden: [2]string{"9.47.2", "10.0.0"}, Want: []string{
		"removed_symbol systemErrorIntegration ",
	}, Drop: [2]string{"11.12.0", "11.13.0"}})
}
