package apisurface

import (
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// The PyPI packages whose public API this kind reads, one source each,
// registered with their provider. A package's Module is the top-level
// directory its wheel installs (checked by hand on each package's newest
// wheel, 2026-11-21).
var (
	OpenAIPython     = Package{ID: "openai/python-surface", Name: "openai", Ecosystem: "pypi", Module: "openai"}
	AnthropicPython  = Package{ID: "anthropic/python-surface", Name: "anthropic", Ecosystem: "pypi", Module: "anthropic"}
	StripePython     = Package{ID: "stripe/python-surface", Name: "stripe", Ecosystem: "pypi", Module: "stripe"}
	GoogleGenAI      = Package{ID: "google/genai-python-surface", Name: "google-genai", Ecosystem: "pypi", Module: "google"}
	SentryPython     = Package{ID: "sentry/python-surface", Name: "sentry-sdk", Ecosystem: "pypi", Module: "sentry_sdk"}
	PyMongo          = Package{ID: "mongodb/pymongo-surface", Name: "pymongo", Ecosystem: "pypi", Module: "pymongo"}
	RedisPy          = Package{ID: "redis/redis-py-surface", Name: "redis", Ecosystem: "pypi", Module: "redis"}
	NumPy            = Package{ID: "numpy/python-surface", Name: "numpy", Ecosystem: "pypi", Module: "numpy"}
	SupabasePython   = Package{ID: "supabase/python-surface", Name: "supabase", Ecosystem: "pypi", Module: "supabase"}
	AlgoliaPython    = Package{ID: "algolia/python-surface", Name: "algoliasearch", Ecosystem: "pypi", Module: "algoliasearch"}
	ElasticPython    = Package{ID: "elastic/python-surface", Name: "elasticsearch", Ecosystem: "pypi", Module: "elasticsearch"}
	DDTrace          = Package{ID: "datadog/ddtrace-python-surface", Name: "ddtrace", Ecosystem: "pypi", Module: "ddtrace"}
	CloudflarePython = Package{ID: "cloudflare/python-surface", Name: "cloudflare", Ecosystem: "pypi", Module: "cloudflare"}
	Textual          = Package{ID: "textual/python-surface", Name: "textual", Ecosystem: "pypi", Module: "textual"}
	URLLib3          = Package{ID: "urllib3/python-surface", Name: "urllib3", Ecosystem: "pypi", Module: "urllib3"}
)

func init() {
	pypi := func(p plugin.Provider, pkg Package, layer plugin.Layer, repo string) {
		_, src, _ := strings.Cut(pkg.ID, "/")
		p.Sources = []plugin.Source{{
			ID: src, Layer: layer, Page: "providers/" + p.ID + "-" + src + ".md",
			Watches:  "Classes, functions, methods and parameters removed or renamed in the public API of the " + pkg.Name + " Python package, release by release, read from each release's wheel",
			URLs:     []string{"https://pypi.org/project/" + pkg.Name + "/", repo},
			Packages: []string{"pypi:" + pkg.Name},
			Plugin:   Plugin{Package: pkg},
		}}
		plugin.RegisterProvider(p)
	}
	pypi(plugin.Provider{ID: "openai", Name: "OpenAI", Category: plugin.CategoryAI, Homepage: "https://openai.com"}, OpenAIPython, plugin.LayerSDK, "https://github.com/openai/openai-python")
	pypi(plugin.Provider{ID: "anthropic", Name: "Anthropic", Category: plugin.CategoryAI, Homepage: "https://www.anthropic.com"}, AnthropicPython, plugin.LayerSDK, "https://github.com/anthropics/anthropic-sdk-python")
	pypi(plugin.Provider{ID: "stripe", Name: "Stripe", Category: plugin.CategoryPayments, Homepage: "https://stripe.com"}, StripePython, plugin.LayerSDK, "https://github.com/stripe/stripe-python")
	pypi(plugin.Provider{ID: "google", Name: "Google", Category: plugin.CategoryAI, Homepage: "https://cloud.google.com"}, GoogleGenAI, plugin.LayerSDK, "https://github.com/googleapis/python-genai")
	pypi(plugin.Provider{ID: "sentry", Name: "Sentry", Category: plugin.CategoryMonitoring, Homepage: "https://sentry.io"}, SentryPython, plugin.LayerSDK, "https://github.com/getsentry/sentry-python")
	pypi(plugin.Provider{ID: "mongodb", Name: "MongoDB", Category: plugin.CategoryDatabases, Homepage: "https://www.mongodb.com"}, PyMongo, plugin.LayerSDK, "https://github.com/mongodb/mongo-python-driver")
	pypi(plugin.Provider{ID: "redis", Name: "Redis", Category: plugin.CategoryDatabases, Homepage: "https://redis.io"}, RedisPy, plugin.LayerSDK, "https://github.com/redis/redis-py")
	pypi(plugin.Provider{ID: "numpy", Name: "NumPy", Category: plugin.CategoryPackages, Homepage: "https://numpy.org"}, NumPy, plugin.LayerPackage, "https://github.com/numpy/numpy")
	pypi(plugin.Provider{ID: "supabase", Name: "Supabase", Category: plugin.CategoryDatabases, Homepage: "https://supabase.com"}, SupabasePython, plugin.LayerSDK, "https://github.com/supabase/supabase-py")
	pypi(plugin.Provider{ID: "algolia", Name: "Algolia", Category: plugin.CategorySearch, Homepage: "https://www.algolia.com"}, AlgoliaPython, plugin.LayerSDK, "https://github.com/algolia/algoliasearch-client-python")
	pypi(plugin.Provider{ID: "elastic", Name: "Elastic", Category: plugin.CategorySearch, Homepage: "https://www.elastic.co"}, ElasticPython, plugin.LayerSDK, "https://github.com/elastic/elasticsearch-py")
	pypi(plugin.Provider{ID: "datadog", Name: "Datadog", Category: plugin.CategoryMonitoring, Homepage: "https://www.datadoghq.com"}, DDTrace, plugin.LayerSDK, "https://github.com/DataDog/dd-trace-py")
	pypi(plugin.Provider{ID: "cloudflare", Name: "Cloudflare", Category: plugin.CategoryInfrastructure, Homepage: "https://www.cloudflare.com"}, CloudflarePython, plugin.LayerSDK, "https://github.com/cloudflare/cloudflare-python")
	pypi(plugin.Provider{ID: "textual", Name: "Textual", Category: plugin.CategoryPackages, Homepage: "https://textual.textualize.io"}, Textual, plugin.LayerPackage, "https://github.com/Textualize/textual")
	pypi(plugin.Provider{ID: "urllib3", Name: "urllib3", Category: plugin.CategoryPackages, Homepage: "https://urllib3.readthedocs.io"}, URLLib3, plugin.LayerPackage, "https://github.com/urllib3/urllib3")
}
