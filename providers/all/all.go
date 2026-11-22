// Package all imports every provider, so each registers itself: a program
// that runs sources imports this one package. A new provider's package is
// added here (TestEveryProviderIsImported fails until it is).
package all

import (
	_ "github.com/patch-it-dev/patch-it-plugins/providers/algolia"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/anthropic"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/appwrite"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/cloudflare"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/convex"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/datadog"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/dropbox"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/elastic"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/flagsmith"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/github"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/gitlab"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/google"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/mongodb"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/neon"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/newrelic"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/numpy"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/openai"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/planetscale"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/posthog"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/redis"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/salesforce"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/sentry"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/stripe"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/supabase"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/textual"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/urllib3"
	_ "github.com/patch-it-dev/patch-it-plugins/providers/vercel"
)
