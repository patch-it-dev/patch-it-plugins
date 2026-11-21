package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func snap(id string, v any) plugin.Snapshot {
	b, _ := json.Marshal(v)
	return plugin.Snapshot{Version: plugin.Version{ID: id, Ref: id}, Content: b, SourceURL: "https://x.test/" + id}
}

func files(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("cmd%d.json", i))
	}
	return out
}

// A release whose command directory lost most of its files moved or
// reorganized it: read as removals it would be most of the catalog, so it
// fails the shape guard before a single file is fetched.
func TestACatalogThatLostMostCommandsChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now, Get: func(_ context.Context, u string) ([]byte, error) {
		return nil, fmt.Errorf("fetched %s", u)
	}}
	c := Commands{API: "acme/commands", Ecosystem: "acme", Package: "acme", Repo: "acme/acme", Dir: "commands", Release: regexp.MustCompile(`^v(.*)$`)}
	res, err := c.WithShape(plugin.Shape{}).Extract(context.Background(), env, snap("8.0.0", files(200)), snap("8.2.0", files(20)))
	if !errors.Is(err, plugin.ErrShapeChanged) || len(res.Events) != 0 {
		t.Errorf("200 -> 20 command files: %v, %d events", err, len(res.Events))
	}
}

// A file of upcoming changes that suddenly lists nothing (reformatted, or
// emptied by a bad commit) fails the guard; entries leaving in a batch when
// their date passes don't.
func TestUpcomingChangesThatVanishedChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	entries := func(n int) map[string][]map[string]string {
		out := map[string][]map[string]string{}
		for i := 0; i < n; i++ {
			d := fmt.Sprintf("2027-%02d-01", i%12+1)
			out[d] = append(out[d], map[string]string{"location": fmt.Sprintf("Thing.field%d", i), "date": d + "T00:00:00+00:00", "criticality": "breaking"})
		}
		return out
	}
	u := UpcomingChanges{API: "acme/graphql", Vendor: "Acme", Repo: "acme/acme", Path: "upcoming.json"}.WithShape(plugin.Shape{})
	if _, err := u.Extract(context.Background(), env, snap("2026-11-01", entries(40)), snap("2026-11-02", map[string]any{})); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("40 -> 0 entries: %v", err)
	}
	if _, err := u.Extract(context.Background(), env, snap("2026-11-01", entries(40)), snap("2026-11-02", entries(20))); err != nil {
		t.Errorf("40 -> 20 entries, a batch shipped: %v", err)
	}
}
