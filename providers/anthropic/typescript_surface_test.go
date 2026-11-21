package anthropic

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @anthropic-ai/sdk 1.7.0 dropped the beta billing-frozen error.
func TestTypescriptSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: TypescriptSurface, Golden: [2]string{"1.6.0", "1.7.0"}, Want: []string{
		"removed_symbol @anthropic-ai/sdk/resources/beta.BetaBillingFrozenErrorDetails,@anthropic-ai/sdk/resources/beta/organization.BetaBillingFrozenErrorDetails,@anthropic-ai/sdk/resources/beta/organization/organization.BetaBillingFrozenErrorDetails ",
		"removed_symbol Anthropic.Beta.BetaBillingFrozenError,Anthropic.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/client.Anthropic.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/client.Anthropic.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/beta.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/beta.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization/organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization/organization.Organization.BetaBillingFrozenError ",
	}, Drop: [2]string{"1.7.0", "1.8.0"}})
}
