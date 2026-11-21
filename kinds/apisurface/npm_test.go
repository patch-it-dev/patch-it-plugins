package apisurface

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// npmCases are the npm sources' saved releases (testdata/npm/<package>/,
// testdata/README.md): a golden pair with the events it must store, and a
// drop pair that must store none. Both are consecutive releases on npm.
var npmCases = []pypiCase{
	// openai 7.60.0 dropped the agents' AgentSessionBudgetReachedEvent
	// type (the session spending limits pulled from the API).
	{OpenAINode, [2]string{"7.59.0", "7.60.0"}, []string{
		"removed_symbol openai/resources/beta.AgentSessionBudgetReachedEvent,openai/resources/beta/agents.AgentSessionBudgetReachedEvent,openai/resources/beta/agents/agents.AgentSessionBudgetReachedEvent ",
	}, [2]string{"7.60.0", "7.61.0"}},
	// @anthropic-ai/sdk 1.7.0 dropped the beta billing-frozen error.
	{AnthropicTS, [2]string{"1.6.0", "1.7.0"}, []string{
		"removed_symbol @anthropic-ai/sdk/resources/beta.BetaBillingFrozenErrorDetails,@anthropic-ai/sdk/resources/beta/organization.BetaBillingFrozenErrorDetails,@anthropic-ai/sdk/resources/beta/organization/organization.BetaBillingFrozenErrorDetails ",
		"removed_symbol Anthropic.Beta.BetaBillingFrozenError,Anthropic.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/client.Anthropic.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/client.Anthropic.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/beta.Beta.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/beta.Beta.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization.Organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization/organization.BetaBillingFrozenError,@anthropic-ai/sdk/resources/beta/organization/organization.Organization.BetaBillingFrozenError ",
	}, [2]string{"1.7.0", "1.8.0"}},
	// stripe 23.0.0 removed constructEventWithoutVerification and the
	// ErrorType namespace.
	{StripeNode, [2]string{"22.6.2", "23.0.0"}, []string{
		"removed_symbol Stripe.ErrorType ",
		"removed_symbol Stripe.constructEventWithoutVerification ",
	}, [2]string{"23.0.0", "23.0.1"}},
	// @sentry/node 10.0.0 stopped exporting systemErrorIntegration. 11.13.0
	// made the logger namespace a const object: its functions are still
	// there, as members the reader doesn't follow.
	{SentryNode, [2]string{"9.47.2", "10.0.0"}, []string{
		"removed_symbol systemErrorIntegration ",
	}, [2]string{"11.12.0", "11.13.0"}},
	// @neondatabase/serverless 1.0.0 dropped ClientBase.neonConfig and the
	// native export; Pool.connect, gone from Pool, is pg's base class's.
	{NeonServerless, [2]string{"0.10.4", "1.0.0"}, []string{
		"removed_symbol native ",
	}, [2]string{"1.0.0", "1.0.1"}},
	// @redis/client 5.0.0 removed commandOptions and RedisFlushModes.
	{RedisNode, [2]string{"1.6.1", "5.0.0"}, []string{
		"removed_symbol RedisFlushModes ",
		"removed_symbol commandOptions ",
	}, [2]string{"6.3.0", "6.3.1"}},
	// @supabase/supabase-js 2.0.0 removed the v1 subscription methods.
	{SupabaseJS, [2]string{"1.35.7", "2.0.0"}, []string{
		"removed_symbol SupabaseClient.getSubscriptions ",
		"removed_symbol SupabaseClient.removeAllSubscriptions ",
		"removed_symbol SupabaseClient.removeSubscription ",
	}, [2]string{"2.118.0", "2.119.0"}},
	// @google/genai 1.0.0 removed LiveMusicSetClientContentParameters.
	{GoogleGenAIJS, [2]string{"0.15.0", "1.0.0"}, []string{
		"removed_symbol @google/genai/node.LiveMusicSetClientContentParameters ",
		"removed_symbol @google/genai/web.LiveMusicSetClientContentParameters ",
		"removed_symbol LiveMusicSetClientContentParameters ",
	}, [2]string{"1.0.0", "1.1.0"}},
	// mongodb 7.0.0 removed CancellationToken, Transaction and more.
	{MongoDBNode, [2]string{"6.22.0", "7.0.0"}, []string{
		"removed_symbol AsyncDisposable ",
		"removed_symbol CancellationToken ",
		"removed_symbol ChangeStream.streamOptions ",
		"removed_symbol ClientMetadata ",
		"removed_symbol ClientMetadataOptions ",
		"removed_symbol ClientSession.transaction ",
		"removed_symbol CloseOptions ",
		"removed_symbol CursorStreamOptions ",
		"removed_symbol ReadPreference.minWireVersion ",
		"removed_symbol ResumeOptions ",
		"removed_symbol ServerCapabilities ",
		"removed_symbol Transaction ",
	}, [2]string{"7.0.0", "7.1.0"}},
	// convex 1.35.1 removed the react entry's QueryOptions and
	// UseQueryResult types.
	{ConvexJS, [2]string{"1.35.0", "1.35.1"}, []string{
		"removed_symbol convex/browser.QueryOptions ",
	}, [2]string{"1.42.2", "1.42.3"}},
}

// npmDir is a package's directory under testdata/npm.
func npmDir(name string) string {
	return filepath.Join("testdata", "npm", strings.ReplaceAll(strings.TrimPrefix(name, "@"), "/", "-"))
}

// npmPages serves a case's saved files under the registry's URLs: the
// package document, each version's, and each trimmed tarball.
func npmPages(t *testing.T, name string) kittest.Pages {
	t.Helper()
	dir, safe := npmDir(name), strings.ReplaceAll(strings.TrimPrefix(name, "@"), "/", "-")
	reg := "https://registry.npmjs.org/" + strings.ReplaceAll(name, "/", "%2F")
	pages := kittest.Pages{reg: filepath.Join(dir, "npm-"+safe+"-excerpt.json")}
	files, err := filepath.Glob(filepath.Join(dir, "npm-"+safe+"-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		v := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "npm-"+safe+"-"), ".json")
		if v == "excerpt" {
			continue
		}
		pages[reg+"/"+v] = f
		var doc struct {
			Dist struct {
				Tarball string `json:"tarball"`
			} `json:"dist"`
		}
		if err := json.Unmarshal(kittest.Read(t, f), &doc); err != nil {
			t.Fatal(err)
		}
		pages[doc.Dist.Tarball] = filepath.Join(dir, strings.TrimSuffix(filepath.Base(doc.Dist.Tarball), ".tgz")+".trimmed.tgz")
	}
	return pages
}

func npmPair(t *testing.T, c pypiCase, pair [2]string) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, npmPages(t, c.pkg.Name))
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

// The golden test of every npm source.
func TestNPMGolden(t *testing.T) {
	for _, c := range npmCases {
		t.Run(c.pkg.ID, func(t *testing.T) {
			got := npmPair(t, c, c.golden)
			if len(got) == 0 {
				t.Fatal("no events")
			}
			var lines []string
			for _, e := range got {
				lines = append(lines, string(e.Subject.Change)+" "+strings.Join(e.Subject.Symbols, ",")+" "+e.Subject.Parameter)
				if e.Version != c.golden[1] || e.PreviousVersion != c.golden[0] || !strings.HasPrefix(e.SourceURL, "https://registry.npmjs.org/") || !strings.Contains(e.SourceExcerpt, c.pkg.Name+" "+c.golden[0]+" (") || e.Subject.Ecosystem != "npm" {
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

// The pinned drop test of every npm source: a real release that changes
// nothing a caller relies on stores no event.
func TestNPMDrop(t *testing.T) {
	for _, c := range npmCases {
		t.Run(c.pkg.ID, func(t *testing.T) {
			for _, e := range npmPair(t, c, c.drop) {
				t.Errorf("%s %v %s", e.Subject.Change, e.Subject.Symbols, e.Subject.Parameter)
			}
		})
	}
}

func TestNPMRegistersItself(t *testing.T) {
	for _, c := range npmCases {
		kittest.Registered(t, Plugin{Package: c.pkg})
	}
}
