package neon

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @neondatabase/serverless 1.0.0 dropped ClientBase.neonConfig and the
// native export; Pool.connect, gone from Pool, is pg's base class's.
func TestServerlessSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: ServerlessSurface, Golden: [2]string{"0.10.4", "1.0.0"}, Want: []string{
		"removed_symbol native ",
	}, Drop: [2]string{"1.0.0", "1.0.1"}})
}
