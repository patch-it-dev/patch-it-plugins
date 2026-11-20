package migrationguide

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
)

// Unpinned, the SDK's versions are its releases on PyPI; only the pair that
// crosses into a new major reads a guide (at that major's tag) and has
// events, with previous_version set; the others read nothing.
func TestOnlyANewMajorHasAGuide(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://pypi.org/pypi/anthropic/json": "testdata/pypi-anthropic-excerpt.json",
		sdkguide.AnthropicV1.URL():             "sdkguide/testdata/anthropic-MIGRATION-v1.0.0.md",
	})
	p := Plugin{Guide: sdkguide.Anthropic}
	ctx := context.Background()
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, pair := range versions.Pairs(vs) {
		from, err := p.ReadAt(ctx, env, pair[0])
		if err != nil {
			t.Fatal(err)
		}
		to, err := p.ReadAt(ctx, env, pair[1])
		if err != nil {
			t.Fatal(err)
		}
		res, err := p.Extract(ctx, env, from, to)
		if err != nil || len(res.Rejected) > 0 {
			t.Fatalf("%v %v", err, res.Rejected)
		}
		var params []string
		for _, e := range res.Events {
			params = append(params, e.PreviousVersion+"->"+e.Version+":"+e.Subject.Parameter)
		}
		got = append(got, fmt.Sprintf("%s->%s [%s]", pair[0].ID, pair[1].ID, strings.Join(params, " ")))
	}
	want := "0.75.0->1.0.0 [0.75.0->1.0.0:temperature 0.75.0->1.0.0:top_p 0.75.0->1.0.0:top_k] | 1.0.0->1.0.1 []"
	if strings.Join(got, " | ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " | "), want)
	}
}

// Run on the unpinned guide reads the newest major's: the same events as the
// pinned golden.
func TestRunReadsTheNewestMajor(t *testing.T) {
	got := kittest.Ingest(t, Plugin{Guide: sdkguide.Anthropic}, kittest.Pages{
		"https://pypi.org/pypi/anthropic/json": "testdata/pypi-anthropic-excerpt.json",
		sdkguide.AnthropicV1.URL():             "sdkguide/testdata/anthropic-MIGRATION-v1.0.0.md",
	})
	kittest.CompareGolden(t, "testdata/golden/anthropic-1.0.0.json", got)
}
