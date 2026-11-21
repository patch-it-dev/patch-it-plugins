package migrationguide

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// guide is a migration guide at every major's tag, served from memory.
type guide struct{}

func (guide) Slug() string { return "acme/python-sdk" }
func (guide) Releases(context.Context, plugin.Env) ([]plugin.Version, error) {
	return nil, nil
}
func (guide) GuideURL(_ context.Context, _ plugin.Env, v plugin.Version) (string, error) {
	return "https://x.test/" + v.Ref + "/MIGRATION.md", nil
}
func (guide) Read([]byte, string, plugin.Version, plugin.Version, time.Time) ([]event.Event, []error, error) {
	return nil, nil, nil
}

// A guide that every new major used to have and the newest majors don't (it
// moved, or the repo dropped it) is a shape signal over the releases
// (Shape.NothingToRead), not three quiet majors.
func TestAGuideThatStoppedBeingFoundIsAShapeSignal(t *testing.T) {
	served := map[string]bool{"v1.0.0": true, "v2.0.0": true}
	env := plugin.Env{Now: time.Now, Get: func(_ context.Context, u string) ([]byte, error) {
		for ref := range served {
			if u == "https://x.test/"+ref+"/MIGRATION.md" {
				return []byte("# Migrating"), nil
			}
		}
		return nil, &fetch.StatusError{URL: u, StatusCode: 404}
	}}
	p := Plugin{Guide: guide{}}.WithShape(plugin.Shape{}).(plugin.Versioned)
	var read []bool
	for _, ref := range []string{"v1.0.0", "v2.0.0", "v3.0.0", "v4.0.0", "v5.0.0"} {
		_, err := p.ReadAt(context.Background(), env, plugin.Version{ID: ref[1:], Ref: ref})
		if err != nil && !errors.Is(err, plugin.ErrNothingToRead) {
			t.Fatalf("%s: %v", ref, err)
		}
		read = append(read, err == nil)
	}
	if err := (plugin.Shape{}).NothingToRead(read); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("%v: %v, want ErrShapeChanged", read, err)
	}
	if err := (plugin.Shape{}).NothingToRead(read[:4]); err != nil {
		t.Errorf("two majors without a guide: %v, not yet a signal", err)
	}
}
