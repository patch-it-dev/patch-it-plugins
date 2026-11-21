package anthropic

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// anthropic 2.7.0 dropped the beta billing-frozen error types, module
// and all. 2.5.0 made DeadlineExceededError an alias of
// GatewayTimeoutError: still importable, no event.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"2.6.0", "2.7.0"}, Want: []string{
		"removed_symbol types.beta.BetaBillingFrozenError,types.beta.beta_billing_frozen_error.BetaBillingFrozenError ",
		"removed_symbol types.beta.BetaBillingFrozenErrorDetails,types.beta.beta_billing_frozen_error_details.BetaBillingFrozenErrorDetails ",
	}, Drop: [2]string{"2.4.0", "2.5.0"}})
}
