package convex

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// convex 1.35.1 removed the react entry's QueryOptions and
// UseQueryResult types.
func TestJSSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: JSSurface, Golden: [2]string{"1.35.0", "1.35.1"}, Want: []string{
		"removed_symbol convex/browser.QueryOptions ",
	}, Drop: [2]string{"1.42.2", "1.42.3"}})
}
