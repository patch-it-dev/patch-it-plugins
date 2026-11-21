package openapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// spec is an OpenAPI document with n operations, GET /things/0 to n-1.
func spec(version string, n int) []byte {
	var paths []string
	for i := 0; i < n; i++ {
		paths = append(paths, fmt.Sprintf(`"/things/%d": {"get": {"operationId": "get%d", "responses": {"200": {"description": "ok"}}}}`, i, i))
	}
	return []byte(`{"openapi": "3.0.0", "info": {"version": "` + version + `"}, "paths": {` + strings.Join(paths, ",") + `}}`)
}

func specPair(fromN, toN int) (plugin.Snapshot, plugin.Snapshot) {
	return plugin.Snapshot{Version: plugin.Version{ID: "1"}, Content: spec("1", fromN), SourceURL: "https://x.test/1"},
		plugin.Snapshot{Version: plugin.Version{ID: "2"}, Content: spec("2", toN), SourceURL: "https://x.test/2"}
}

// A spec that lost most of its operations in one release (split into files,
// moved, truncated) fails the shape guard: no removal is read from it.
func TestASpecThatLostMostOperationsChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	guarded := Plugin{Adapter: fakeAdapter{}}.WithShape(plugin.Shape{})
	from, to := specPair(40, 4)
	res, err := guarded.Extract(context.Background(), env, from, to)
	if !errors.Is(err, plugin.ErrShapeChanged) || len(res.Events) != 0 {
		t.Errorf("40 -> 4 operations: err %v, %d events; want ErrShapeChanged and none", err, len(res.Events))
	}
	from, to = specPair(40, 0)
	if _, err := guarded.Extract(context.Background(), env, from, to); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("40 -> 0 operations: %v", err)
	}
	// A release that removes a few is a release.
	from, to = specPair(40, 37)
	if _, err := guarded.Extract(context.Background(), env, from, to); err != nil {
		t.Errorf("40 -> 37 operations: %v", err)
	}
	// The source's own ratio.
	from, to = specPair(40, 4)
	if _, err := (Plugin{Adapter: fakeAdapter{}}).WithShape(plugin.Shape{MinRatio: 0.05}).Extract(context.Background(), env, from, to); err != nil {
		t.Errorf("with MinRatio 0.05: %v", err)
	}
}
