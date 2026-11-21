package cloudflare

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// cloudflare 5.4.0 dropped the web crawler's crawl options from AI
// Search instances, and moved the IAM OAuth scopes resource out of its
// own package (the old package still binds the response wrappers: no
// event for those).
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"5.3.0", "5.4.0"}, Want: []string{
		"removed_symbol resources.iam.oauth_scopes.oauth_scopes.AsyncOAuthScopesResource ",
		"removed_symbol resources.iam.oauth_scopes.oauth_scopes.OAuthScopesResource ",
		"removed_symbol types.aisearch.instance_create_params.SourceParamsWebCrawlerCrawlOptions ",
	}, Drop: [2]string{"4.3.0", "4.3.1"}})
}
