package openapi

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

func TestRegistersItself(t *testing.T) {
	for _, a := range []adapter.Adapter{adapter.Stripe{}, adapter.Datadog{}, adapter.GitLab{}, adapter.GitHub{}, adapter.Vercel{}, adapter.Supabase{}, adapter.Algolia{Spec: "search"}, adapter.Algolia{Spec: "ingestion"}, adapter.Algolia{Spec: "recommend"}, adapter.Elasticsearch{}, adapter.Sentry{}, adapter.Neon{}, adapter.Convex{}, adapter.PlanetScale{}, adapter.Appwrite{}, adapter.OpenAI{}, adapter.Cloudflare{}, adapter.MongoDBAtlas{}} {
		kittest.Registered(t, Plugin{Adapter: a})
	}
}
