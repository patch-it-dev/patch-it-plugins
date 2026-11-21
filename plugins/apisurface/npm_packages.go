package apisurface

import (
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// The npm packages whose public API this kind reads from their TypeScript
// declarations, one source each, registered with their provider. An npm
// package's Module is its own name: symbols are named from the import path
// code writes ("@anthropic-ai/sdk/resources/messages.Messages.create").
var (
	OpenAINode     = npmPackage("openai/node-surface", "openai")
	AnthropicTS    = npmPackage("anthropic/typescript-surface", "@anthropic-ai/sdk")
	StripeNode     = npmPackage("stripe/node-surface", "stripe")
	GoogleGenAIJS  = npmPackage("google/genai-js-surface", "@google/genai")
	SentryNode     = npmPackage("sentry/node-surface", "@sentry/node")
	MongoDBNode    = npmPackage("mongodb/node-surface", "mongodb")
	RedisNode      = npmPackage("redis/node-redis-surface", "@redis/client")
	SupabaseJS     = npmPackage("supabase/js-surface", "@supabase/supabase-js")
	NeonServerless = npmPackage("neon/serverless-surface", "@neondatabase/serverless")
	ConvexJS       = npmPackage("convex/js-surface", "convex")
)

func npmPackage(id, name string) Package {
	return Package{ID: id, Name: name, Ecosystem: "npm", Module: name}
}

func init() {
	npm := func(p plugin.Provider, pkg Package, repo string, also ...string) {
		_, src, _ := strings.Cut(pkg.ID, "/")
		p.Sources = []plugin.Source{{
			ID: src, Layer: plugin.LayerSDK, Page: "providers/" + p.ID + "-" + src + ".md",
			Watches:  "Exported classes, functions, methods, properties, types and parameters removed, or parameters no longer optional, in the public API of the " + pkg.Name + " npm package, release by release, read from each release's TypeScript declarations",
			URLs:     []string{"https://www.npmjs.com/package/" + pkg.Name, repo},
			Packages: append([]string{"npm:" + pkg.Name}, also...),
			Plugin:   Plugin{Package: pkg},
		}}
		plugin.RegisterProvider(p)
	}
	npm(plugin.Provider{ID: "openai", Name: "OpenAI", Category: plugin.CategoryAI, Homepage: "https://openai.com"}, OpenAINode, "https://github.com/openai/openai-node")
	npm(plugin.Provider{ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com"}, AnthropicTS, "https://github.com/anthropics/anthropic-sdk-typescript")
	npm(plugin.Provider{ID: "stripe", Name: "Stripe", Category: plugin.CategoryPayments, Homepage: "https://stripe.com"}, StripeNode, "https://github.com/stripe/stripe-node")
	npm(plugin.Provider{ID: "google", Name: "Google", Category: plugin.CategoryAI, Homepage: "https://cloud.google.com"}, GoogleGenAIJS, "https://github.com/googleapis/js-genai")
	npm(plugin.Provider{ID: "sentry", Name: "Sentry", Category: plugin.CategoryMonitoring, Homepage: "https://sentry.io"}, SentryNode, "https://github.com/getsentry/sentry-javascript")
	npm(plugin.Provider{ID: "mongodb", Name: "MongoDB", Category: plugin.CategoryDatabases, Homepage: "https://www.mongodb.com"}, MongoDBNode, "https://github.com/mongodb/node-mongodb-native")
	// redis on npm re-exports @redis/client and its siblings wholesale; the
	// client's declarations are where its API is.
	npm(plugin.Provider{ID: "redis", Name: "Redis", Category: plugin.CategoryDatabases, Homepage: "https://redis.io"}, RedisNode, "https://github.com/redis/node-redis", "npm:redis")
	npm(plugin.Provider{ID: "supabase", Name: "Supabase", Category: plugin.CategoryDatabases, Homepage: "https://supabase.com"}, SupabaseJS, "https://github.com/supabase/supabase-js")
	npm(plugin.Provider{ID: "neon", Name: "Neon", Category: plugin.CategoryDatabases, Homepage: "https://neon.com"}, NeonServerless, "https://github.com/neondatabase/serverless")
	npm(plugin.Provider{ID: "convex", Name: "Convex", Category: plugin.CategoryDatabases, Homepage: "https://www.convex.dev"}, ConvexJS, "https://github.com/get-convex/convex-js")
}
