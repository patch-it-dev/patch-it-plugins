package plugin

import (
	"context"
	"slices"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/event"
)

// testPlugin is a source defined only here: registering it and running it
// touches nothing outside this file.
type testPlugin struct {
	name string
	err  error
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Run(_ context.Context, env Env) (Result, error) {
	if p.err != nil {
		return Result{}, p.err
	}
	subject := event.Subject{Type: event.SubjectAPIOperation, Kind: event.AffectedKindOperation, Side: event.SideRequest, Operation: "POST /v1/things"}
	e := event.EventV2{
		Fingerprint:   event.FingerprintV2(p.name, "2", event.ChangeTypeNewFeature, subject, ""),
		API:           p.name,
		Version:       "2",
		ChangeType:    event.ChangeTypeNewFeature,
		Subject:       subject,
		Description:   "New operation POST /v1/things.",
		SourceURL:     "https://example.test/spec/2.json",
		SourceExcerpt: `{"post":{}}`,
		Confidence:    1,
		DetectedAt:    env.Now().UTC(),
	}
	content := []byte(`{"version":"2"}`)
	return Result{
		Label:     "v2",
		Events:    []event.EventV2{e},
		Baselines: []Spec{Baseline(p.name, "2", "v2", "https://example.test/spec/2.json", content, false, env.Now())},
	}, nil
}

func TestAPluginRegisteredInATestIsFound(t *testing.T) {
	register(t, testPlugin{name: "test-registered"})
	if p, ok := Lookup("test-registered"); !ok || p.Name() != "test-registered" || !slices.Contains(Names(), "test-registered") {
		t.Fatal("not registered")
	}
}

func TestNamesAreSorted(t *testing.T) {
	register(t, testPlugin{name: "test-b"})
	register(t, testPlugin{name: "test-a"})
	if !slices.IsSorted(Names()) {
		t.Errorf("Names() = %v, want sorted", Names())
	}
}

// register registers a test plugin for the rest of the test only, so the test
// can run again (-count).
func register(t *testing.T, p Plugin) {
	t.Helper()
	Register(p)
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		delete(registry, p.Name())
	})
}

func TestTwoPluginsCantShareAName(t *testing.T) {
	register(t, testPlugin{name: "test-dup"})
	defer func() {
		if recover() == nil {
			t.Error("a second plugin named test-dup was registered")
		}
	}()
	Register(testPlugin{name: "test-dup"})
}

func TestAPluginCantBeNamedAll(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error(`a plugin named "all" was registered: --api all would hide it`)
		}
	}()
	Register(testPlugin{name: "all"})
}
