package openapi

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
)

// atlasDays are the saved excerpts of mongodb/openapi's openapi/v2.json
// (testdata/README.md): day -> the day's last commit.
var atlasDays = map[string]string{
	"2024-07-25": "6ae1e79533409017d2c3251e645aae28aa54bd8b",
	"2024-08-07": "96b0555285636db831afeb5a0e9b2dfddbfab2c6",
	"2025-02-18": "f7b9298adb18b804230c8516cb533196538a8db9",
	"2025-02-20": "28b2c6fc35a44b3272d7afbcaae380e1d28f0e45",
}

func atlasURL(day string) string {
	return "https://raw.githubusercontent.com/mongodb/openapi/" + atlasDays[day] + "/openapi/v2.json"
}

// fakeAtlasAdapter serves the saved excerpts, cited by the real URL each was
// copied from, as adapter.MongoDBAtlas.Fetch does.
type fakeAtlasAdapter struct{}

func (fakeAtlasAdapter) API() string { return "mongodb/atlas-admin-spec" }

func (fakeAtlasAdapter) LatestTag(context.Context) (string, error) { return "2025-02-20", nil }

func (fakeAtlasAdapter) Fetch(_ context.Context, day string) (adapter.Spec, error) {
	if _, ok := atlasDays[day]; !ok {
		return adapter.Spec{}, fmt.Errorf("no saved copy for %s", day)
	}
	body, err := os.ReadFile("testdata/mongodb-atlas-" + day + ".json")
	if err != nil {
		return adapter.Spec{}, err
	}
	return adapter.Spec{Content: body, Version: day, SourceURL: atlasURL(day)}, nil
}

func atlasPlugin() Plugin {
	return Plugin{Adapter: fakeAtlasAdapter{}, Diff: atlasDiff, Keep: atlasKeep}
}

func TestMongoDBAtlasRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Adapter: adapter.MongoDBAtlas{}})
}

// The golden test: on 2025-02-20 Atlas deprecated 14 user-management
// operations at once, for the move to per-organization and per-project user
// endpoints. Three are saved (POST /users, GET /users/{userId}, GET
// /users/byName/{userName}), with GET /orgs/{orgId}/users, unchanged.
func TestMongoDBAtlasGolden(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if rejected, err := kittest.Pair(ctx, vercelEnv(s), s, atlasPlugin(), "2025-02-18", "2025-02-20"); err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	events := kittest.Stored(t, s)
	var ops []string
	for _, e := range events {
		ops = append(ops, e.Subject.Operation)
		if e.ChangeType != event.ChangeTypeDeprecation || e.Version != "2025-02-20" || e.PreviousVersion != "2025-02-18" || e.SourceURL != atlasURL("2025-02-20") || !strings.Contains(e.SourceExcerpt, `"deprecated":true`) {
			t.Errorf("%s: %s %s<-%s cites %s", e.Subject.Operation, e.ChangeType, e.Version, e.PreviousVersion, e.SourceURL)
		}
	}
	slices.Sort(ops)
	want := []string{"GET /api/atlas/v2/users/byName/{userName}", "GET /api/atlas/v2/users/{userId}", "POST /api/atlas/v2/users"}
	if !slices.Equal(ops, want) {
		t.Errorf("operations %v, want %v", ops, want)
	}
	kittest.Golden(t, "testdata/golden/mongodb-atlas-admin-spec.json", events)
}

// The pinned drop test: GET /rest/unauth/version left the spec on 2024-08-07
// without ever being marked deprecated. No event.
func TestMongoDBAtlasDropsRemovalsThatWerentDeprecated(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	if _, err := kittest.Pair(ctx, vercelEnv(s), s, atlasPlugin(), "2024-07-25", "2024-08-07"); err != nil {
		t.Fatal(err)
	}
	if events := kittest.Stored(t, s); len(events) != 0 {
		t.Fatalf("stored %d events, want none", len(events))
	}
}
