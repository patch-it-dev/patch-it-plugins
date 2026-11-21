package plugin

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// testPlugin is a source defined only here: registering it and running it
// touches nothing outside this file.
type testPlugin struct {
	name string
	err  error
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) WithShape(Shape) Plugin { return p }

func (p testPlugin) Read(_ context.Context, env Env, _ Snapshot) (Snapshot, error) {
	if p.err != nil {
		return Snapshot{}, p.err
	}
	return Snapshot{Version: Version{ID: env.Now().UTC().Format(time.RFC3339)}, Content: []byte(`{"version":"2"}`), SourceURL: "https://example.test/spec/2.json"}, nil
}

func (p testPlugin) Extract(_ context.Context, env Env, _, _ Snapshot) (Result, error) {
	subject := event.Subject{Type: event.SubjectModel, Provider: "test", Name: "model-1"}
	e := event.Event{
		Fingerprint:   event.Fingerprint(p.name, "2026-12-01", event.ChangeBreaking, subject, ""),
		API:           p.name,
		Version:       "2026-12-01",
		Change:        event.ChangeBreaking,
		Subject:       subject,
		Description:   "model-1 shuts down on 2026-12-01.",
		SourceURL:     "https://example.test/spec/2.json",
		SourceExcerpt: "model-1 | 2026-12-01",
		Confidence:    1,
		DetectedAt:    env.Now().UTC(),
	}
	return Result{Events: []event.Event{e}}, nil
}

func TestASourceRegisteredInATestIsFound(t *testing.T) {
	register(t, "test-registered")
	if p, ok := Lookup("test-registered/api"); !ok || p.Name() != "test-registered/api" || !slices.Contains(Names(), "test-registered/api") {
		t.Fatal("not registered")
	}
}

func TestNamesAreSorted(t *testing.T) {
	register(t, "test-b")
	register(t, "test-a")
	if !slices.IsSorted(Names()) {
		t.Errorf("Names() = %v, want sorted", Names())
	}
}

// register registers a test provider with one source, api, for the rest of
// the test only, so the test can run again (-count).
func register(t *testing.T, provider string) {
	t.Helper()
	RegisterProvider(Provider{ID: provider, Name: provider, Category: CategoryDevTools, Homepage: "https://example.test",
		Sources: []Source{{ID: "api", Layer: LayerAPI, Watches: "w", URLs: []string{"https://example.test/spec"}, Plugin: testPlugin{name: provider + "/api"}}}})
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		delete(registry, provider+"/api")
		delete(sources, provider+"/api")
		delete(providers, provider)
	})
}

func TestTwoSourcesCantShareAnID(t *testing.T) {
	register(t, "test-dup")
	defer func() {
		if recover() == nil {
			t.Error("a second source test-dup/api was registered")
		}
	}()
	register(t, "test-dup")
}

func TestAVersionPinsNothingUntilSet(t *testing.T) {
	if !(Version{ID: "12.0.0"}).MapsTo.IsZero() {
		t.Error("a version with no MapsTo pins something")
	}
	if (Pin{API: "example/api", Version: "2026-03-31"}).IsZero() {
		t.Error("a set pin is zero")
	}
}
