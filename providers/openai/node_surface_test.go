package openai

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// openai 7.60.0 dropped the agents' AgentSessionBudgetReachedEvent
// type (the session spending limits pulled from the API).
func TestNodeSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: NodeSurface, Golden: [2]string{"7.59.0", "7.60.0"}, Want: []string{
		"removed_symbol openai/resources/beta.AgentSessionBudgetReachedEvent,openai/resources/beta/agents.AgentSessionBudgetReachedEvent,openai/resources/beta/agents/agents.AgentSessionBudgetReachedEvent ",
	}, Drop: [2]string{"7.60.0", "7.61.0"}})
}
