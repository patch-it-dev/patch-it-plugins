package apisurface

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// pypiCase is one PyPI source's saved releases (testdata/pypi/<name>/,
// testdata/README.md): a golden pair, the events it must store, and a drop
// pair, a real release that must store none. Both pairs are consecutive
// releases on PyPI.
type pypiCase struct {
	pkg    Package
	golden [2]string
	want   []string // each event as "<change> <symbol> <parameter>"
	drop   [2]string
}

var pypiCases = []pypiCase{
	// anthropic 2.7.0 dropped the beta billing-frozen error types, module
	// and all. 2.5.0 made DeadlineExceededError an alias of
	// GatewayTimeoutError: still importable, no event.
	{AnthropicPython, [2]string{"2.6.0", "2.7.0"}, []string{
		"removed_symbol types.beta.BetaBillingFrozenError ",
		"removed_symbol types.beta.BetaBillingFrozenErrorDetails ",
		"removed_symbol types.beta.beta_billing_frozen_error.BetaBillingFrozenError ",
		"removed_symbol types.beta.beta_billing_frozen_error_details.BetaBillingFrozenErrorDetails ",
	}, [2]string{"2.4.0", "2.5.0"}},
	// stripe 16.0.0 removed StripeObject.request. 15.5.0 only added.
	{StripePython, [2]string{"15.6.2", "16.0.0"}, []string{"removed_symbol StripeObject.request "}, [2]string{"15.4.0", "15.5.0"}},
	// google-genai 1.0.0 removed TuningJobOrOperation. 3.4.0 re-exports
	// interactions' names from a private module: still importable.
	{GoogleGenAI, [2]string{"0.8.0", "1.0.0"}, []string{
		"removed_symbol genai.types.TuningJobOrOperation ",
		"removed_symbol genai.types.TuningJobOrOperationDict ",
	}, [2]string{"3.3.0", "3.4.0"}},
	// sentry-sdk 2.74.0 removed utils.to_timestamp. 2.68.0 renamed a
	// property setter's parameter: callers never pass it.
	{SentryPython, [2]string{"2.73.0", "2.74.0"}, []string{"removed_symbol utils.to_timestamp "}, [2]string{"2.67.1", "2.68.0"}},
	// pymongo 4.16.0's command cursors no longer take explicit_session.
	{PyMongo, [2]string{"4.15.5", "4.16.0"}, []string{
		"removed_parameter asynchronous.command_cursor.AsyncCommandCursor explicit_session",
		"removed_parameter asynchronous.command_cursor.AsyncRawBatchCommandCursor explicit_session",
		"removed_parameter command_cursor.CommandCursor explicit_session",
		"removed_parameter command_cursor.RawBatchCommandCursor explicit_session",
		"removed_parameter synchronous.command_cursor.CommandCursor explicit_session",
		"removed_parameter synchronous.command_cursor.RawBatchCommandCursor explicit_session",
	}, [2]string{"4.15.4", "4.15.5"}},
	// redis 8.1.0 removed OSSMaintNotificationsHandler.get_handler_for_connection.
	{RedisPy, [2]string{"8.0.1", "8.1.0"}, []string{"removed_symbol maint_notifications.OSSMaintNotificationsHandler.get_handler_for_connection "}, [2]string{"7.4.1", "8.0.0"}},
	// numpy 2.5.0 removed row_stack, deprecated since 2.0 (read from the
	// .pyi stubs).
	{NumPy, [2]string{"2.4.6", "2.5.0"}, []string{"removed_symbol row_stack "}, [2]string{"2.5.0", "2.5.1"}},
	// supabase 1.0.0's SupabaseAuthClient lost four parameters. 2.34.0 made
	// supabase.client a star re-export of the package: still importable.
	{SupabasePython, [2]string{"0.7.1", "1.0.0"}, []string{
		"removed_parameter SupabaseAuthClient api",
		"removed_parameter SupabaseAuthClient cookie_options",
		"removed_parameter SupabaseAuthClient local_storage",
		"removed_parameter SupabaseAuthClient replace_default_headers",
		"removed_parameter lib.auth_client.SupabaseAuthClient api",
		"removed_parameter lib.auth_client.SupabaseAuthClient cookie_options",
		"removed_parameter lib.auth_client.SupabaseAuthClient local_storage",
		"removed_parameter lib.auth_client.SupabaseAuthClient replace_default_headers",
		"removed_parameter lib.client_options.ClientOptions.replace local_storage",
	}, [2]string{"2.33.0", "2.34.0"}},
	// algoliasearch 4.44.3 dropped recommend's FacetFilters (4.44.4 brought
	// it back: an addition, no event).
	{AlgoliaPython, [2]string{"4.44.1", "4.44.3"}, []string{
		"removed_symbol recommend.models.FacetFilters ",
		"removed_symbol recommend.models.facet_filters.FacetFilters ",
	}, [2]string{"4.44.3", "4.44.4"}},
	// elasticsearch 9.1.3 made the rethrottle methods' requests_per_second
	// required.
	{ElasticPython, [2]string{"9.1.2", "9.1.3"}, []string{
		"changed_default Elasticsearch.delete_by_query_rethrottle requests_per_second",
		"changed_default Elasticsearch.reindex_rethrottle requests_per_second",
		"changed_default Elasticsearch.update_by_query_rethrottle requests_per_second",
		"changed_default client.Elasticsearch.delete_by_query_rethrottle requests_per_second",
		"changed_default client.Elasticsearch.reindex_rethrottle requests_per_second",
		"changed_default client.Elasticsearch.update_by_query_rethrottle requests_per_second",
	}, [2]string{"9.1.1", "9.1.2"}},
	// ddtrace 4.13.0 moved AI Guard to ddtrace.aiguard: the old module
	// forwards its names (__getattr__), so only the two integration helpers
	// it dropped are gone.
	{DDTrace, [2]string{"4.12.3", "4.13.0"}, []string{
		"removed_symbol appsec.ai_guard.integrations.resolve_strands_hook_provider ",
		"removed_symbol appsec.ai_guard.integrations.resolve_strands_plugin ",
	}, [2]string{"4.16.0", "4.16.1"}},
	// cloudflare 5.4.0 dropped the web crawler's crawl options from AI
	// Search instances, and moved the IAM OAuth scopes resource out of its
	// own package (the old package still binds the response wrappers: no
	// event for those).
	{CloudflarePython, [2]string{"5.3.0", "5.4.0"}, []string{
		"removed_symbol resources.iam.oauth_scopes.oauth_scopes.AsyncOAuthScopesResource ",
		"removed_symbol resources.iam.oauth_scopes.oauth_scopes.OAuthScopesResource ",
		"removed_symbol types.aisearch.instance_create_params.SourceParamsWebCrawlerCrawlOptions ",
	}, [2]string{"4.3.0", "4.3.1"}},
	// textual 8.2.6's walk_selectable_widgets no longer takes bounded.
	{Textual, [2]string{"8.2.5", "8.2.6"}, []string{"removed_parameter walk.walk_selectable_widgets bounded"}, [2]string{"8.2.6", "8.2.7"}},
	// urllib3 2.6.0 removed BaseHTTPResponse.getheader and getheaders,
	// deprecated since 2.0.
	{URLLib3, [2]string{"2.5.0", "2.6.0"}, []string{
		"removed_symbol BaseHTTPResponse.getheader ",
		"removed_symbol BaseHTTPResponse.getheaders ",
		"removed_symbol response.BaseHTTPResponse.getheader ",
		"removed_symbol response.BaseHTTPResponse.getheaders ",
	}, [2]string{"2.2.1", "2.2.2"}},
}

// pypiPages serves a case's saved files: PyPI's answers, and each picked
// wheel, trimmed, under its real URL.
func pypiPages(t *testing.T, name string) kittest.Pages {
	t.Helper()
	dir := filepath.Join("testdata", "pypi", name)
	pages := kittest.Pages{"https://pypi.org/pypi/" + name + "/json": filepath.Join(dir, "pypi-"+name+"-excerpt.json")}
	files, err := filepath.Glob(filepath.Join(dir, "pypi-"+name+"-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		v := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "pypi-"+name+"-"), ".json")
		if v == "excerpt" {
			continue
		}
		pages["https://pypi.org/pypi/"+name+"/"+v+"/json"] = f
		var doc struct {
			URLs []struct {
				URL      string `json:"url"`
				Filename string `json:"filename"`
			} `json:"urls"`
		}
		if err := json.Unmarshal(kittest.Read(t, f), &doc); err != nil {
			t.Fatal(err)
		}
		for _, u := range doc.URLs {
			trimmed := filepath.Join(dir, strings.TrimSuffix(u.Filename, ".whl")+".trimmed.whl")
			if _, err := os.Stat(trimmed); err == nil {
				pages[u.URL] = trimmed
			}
		}
	}
	return pages
}

// pypiPair stores one pair of a case's releases, listed as PyPI lists them.
func pypiPair(t *testing.T, c pypiCase, pair [2]string) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, pypiPages(t, c.pkg.Name))
	p := Plugin{Package: c.pkg}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]plugin.Version{}
	for _, v := range vs {
		byID[v.ID] = v
	}
	from, ok1 := byID[pair[0]]
	to, ok2 := byID[pair[1]]
	if !ok1 || !ok2 {
		t.Fatalf("%s: %v not both listed in %v", c.pkg.Name, pair, vs)
	}
	return kittest.Versions(t, env, p, from, to)
}

// The golden test of every PyPI source: its golden pair stores exactly
// the events listed, each citing the newer wheel and quoting both releases.
func TestPyPIGolden(t *testing.T) {
	for _, c := range pypiCases {
		t.Run(c.pkg.ID, func(t *testing.T) {
			got := pypiPair(t, c, c.golden)
			var lines []string
			for _, e := range got {
				lines = append(lines, string(e.Subject.Change)+" "+strings.Join(e.Subject.Symbols, ",")+" "+e.Subject.Parameter)
				if e.Version != c.golden[1] || e.PreviousVersion != c.golden[0] || !strings.Contains(e.SourceURL, "files.pythonhosted.org") || !strings.Contains(e.SourceExcerpt, c.pkg.Name+" "+c.golden[0]+" (") {
					t.Errorf("%s: %s<-%s cites %s, excerpt %q", e.Subject.Symbols, e.Version, e.PreviousVersion, e.SourceURL, e.SourceExcerpt)
				}
			}
			slices.Sort(lines)
			if !slices.Equal(lines, c.want) {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(c.want, "\n"))
			}
			provider, source, _ := strings.Cut(c.pkg.ID, "/")
			kittest.Golden(t, "testdata/golden/"+provider+"-"+source+".json", got)
		})
	}
}

// The pinned drop test of every PyPI source: a real release that changes
// the package's files but nothing a caller relies on stores no event.
func TestPyPIDrop(t *testing.T) {
	for _, c := range pypiCases {
		t.Run(c.pkg.ID, func(t *testing.T) {
			for _, e := range pypiPair(t, c, c.drop) {
				t.Errorf("%s %v %s", e.Subject.Change, e.Subject.Symbols, e.Subject.Parameter)
			}
		})
	}
}

func TestPyPIRegistersItself(t *testing.T) {
	for _, c := range pypiCases {
		kittest.Registered(t, Plugin{Package: c.pkg})
	}
}
