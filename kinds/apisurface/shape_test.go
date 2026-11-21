package apisurface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/pysurface"
)

func release(id string, n int) plugin.Snapshot {
	s := pysurface.Surface{}
	for i := 0; i < n; i++ {
		s[fmt.Sprintf("acme.f%d", i)] = sym("function")
	}
	b, _ := json.Marshal(Release{Artifact: "acme-" + id + ".whl", Surface: s})
	return plugin.Snapshot{Version: plugin.Version{ID: id}, Content: b, SourceURL: "https://x.test/acme-" + id + ".whl"}
}

// A release whose public API reads as a fraction of the one before (the
// code moved to a new top-level module, or the declarations aren't where the
// reader looks any more) fails the shape guard: no symbol is read as removed.
func TestASurfaceThatShrankChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	p := Plugin{Package: Package{ID: "acme/python-surface", Name: "acme", Ecosystem: "pypi", Module: "acme"}}
	res, err := p.WithShape(plugin.Shape{}).Extract(context.Background(), env, release("1.0.0", 60), release("2.0.0", 3))
	if !errors.Is(err, plugin.ErrShapeChanged) || len(res.Events) != 0 {
		t.Errorf("60 -> 3 names: %v, %d events", err, len(res.Events))
	}
	if _, err := p.WithShape(plugin.Shape{}).Extract(context.Background(), env, release("1.0.0", 60), release("2.0.0", 0)); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("60 -> 0 names: %v", err)
	}
	res, err = p.WithShape(plugin.Shape{}).Extract(context.Background(), env, release("1.0.0", 60), release("2.0.0", 55))
	if err != nil || len(res.Events) != 5 {
		t.Errorf("60 -> 55 names: %v, %d events, want 5 removals", err, len(res.Events))
	}
}
