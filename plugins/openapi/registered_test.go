package openapi

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/adapter"
	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	for _, a := range []adapter.Adapter{adapter.Stripe{}, adapter.Datadog{}, adapter.GitLab{}, adapter.GitHub{}, adapter.Vercel{}, adapter.Supabase{}} {
		kittest.Registered(t, Plugin{Adapter: a})
	}
}
