package openapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// openaiDays are the saved excerpts of openai/openai-openapi's openapi.yaml
// (testdata/README.md): day -> the day's last commit, the one Daily reads.
var openaiDays = map[string]string{
	"2026-10-27": "54640e1ce5465c6f2e597ac4b030f9b92ba6d044",
	"2026-10-28": "aa880040fa7b23c8a7e3618b5ca757ecec71de95",
	"2026-11-17": "b2fe5a5d9eb3a580da5aed1d3ff2a03b3853e4f8",
	"2026-11-18": "d4eae433ac22ac1b1f9af8ecfdbd529d6cea082e",
}

func openaiURL(day string) string {
	return "https://raw.githubusercontent.com/openai/openai-openapi/" + openaiDays[day] + "/openapi.yaml"
}

// fakeOpenAIAdapter serves the saved excerpts, cited by the real URL each was
// copied from and converted and reshaped as adapter.OpenAI.Fetch does.
type fakeOpenAIAdapter struct{}

func (fakeOpenAIAdapter) API() string { return "openai/api" }

func (fakeOpenAIAdapter) LatestTag(context.Context) (string, error) { return "2026-11-18", nil }

func (fakeOpenAIAdapter) Fetch(_ context.Context, day string) (adapter.Spec, error) {
	if _, ok := openaiDays[day]; !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", day)
	}
	body, err := os.ReadFile("testdata/openai-" + day + ".yaml")
	if err != nil {
		return adapter.Spec{}, err
	}
	content, err := adapter.YAMLToJSON(body)
	if err != nil {
		return adapter.Spec{}, err
	}
	if content, err = adapter.ServerBasePath(content); err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: content, Version: day, SourceURL: openaiURL(day)}, nil
}

func (fakeOpenAIAdapter) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

func openaiPlugin() Plugin {
	return Plugin{Adapter: fakeOpenAIAdapter{}, Diff: openaiDiff, Keep: openaiKeep}
}

func TestOpenAIRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Adapter: adapter.OpenAI{}})
}

// The golden test: on 2026-10-28 ("Mark /audio/translations endpoint as
// deprecated", ff80a233) POST /audio/translations gained the flag and "The
// endpoint and model are deprecated and scheduled to shut down on February
// 26, 2027". The same day added GET and POST /organization/external_keys
// (dropped: new) and left /audio/transcriptions as it was.
func TestOpenAIGolden(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, openaiPlugin(), "2026-10-27", "2026-10-28"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	if len(events) != 1 {
		t.Fatalf("stored %d events, want 1", len(events))
	}
	e := events[0]
	if e.Subject.Operation != "POST /v1/audio/translations" || e.ChangeType != event.ChangeTypeDeprecation || e.Version != "2026-10-28" || e.PreviousVersion != "2026-10-27" {
		t.Errorf("got %s %s %s (after %s)", e.ChangeType, e.Subject.Operation, e.Version, e.PreviousVersion)
	}
	if e.SourceURL != openaiURL("2026-10-28") || e.SourcePointer != "/paths/~1audio~1translations/post" {
		t.Errorf("cites %s#%s", e.SourceURL, e.SourcePointer)
	}
	if !strings.Contains(e.SourceExcerpt, "scheduled to shut down on February 26, 2027") {
		t.Errorf("excerpt lacks the shutdown date: %.300s", e.SourceExcerpt)
	}
	kittest.Golden(t, "testdata/golden/openai-api.json", events)
}

// The pinned drop test: the session spending-limit schemas added on
// 2026-11-17 were gone on 2026-11-18 ("Remove session spending limit
// features"), never released as anything callers relied on. Three schema
// removals unfiltered, no event.
func TestOpenAIDropsWorkInProgressPulledBack(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, openaiPlugin(), "2026-11-17", "2026-11-18"); err != nil {
		t.Fatal(err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("stored %d events, want none", len(events))
	}
	s = openStore(t)
	p := openaiPlugin()
	p.Keep = nil
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, p, "2026-11-17", "2026-11-18"); err != nil {
		t.Fatal(err)
	}
	if n := len(kittest.Stored(t, s)); n != 3 {
		t.Errorf("unfiltered: %d events, want the 3 schema removals", n)
	}
}
