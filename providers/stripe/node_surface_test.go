package stripe

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// stripe 23.0.0 removed constructEventWithoutVerification and the
// ErrorType namespace.
func TestNodeSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: NodeSurface, Golden: [2]string{"22.6.2", "23.0.0"}, Want: []string{
		"removed_symbol Stripe.ErrorType ",
		"removed_symbol Stripe.constructEventWithoutVerification ",
	}, Drop: [2]string{"23.0.0", "23.0.1"}})
}
