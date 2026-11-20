package kittest

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

func TestMemStoresAsIngestDoes(t *testing.T) {
	CheckSink(t, func(*testing.T) Sink { return NewMem() })
}

// Mem's ids are 1..n in save order with no gaps, whatever was skipped: a
// golden file's ids then depend only on the events it holds.
func TestMemIdsHaveNoGaps(t *testing.T) {
	m := NewMem()
	if _, err := m.SaveEvents([]event.Event{sinkEvent(1), sinkEvent(1), sinkEvent(2)}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SaveEvents([]event.Event{sinkEvent(2), sinkEvent(3)}); err != nil {
		t.Fatal(err)
	}
	all, _ := m.All()
	for i, e := range all {
		if e.ID != int64(i+1) {
			t.Errorf("event %d has id %d, want %d", i, e.ID, i+1)
		}
	}
}

// What LastSpec returns is a copy: a plugin changing it can't change the store.
func TestMemLastSpecIsACopy(t *testing.T) {
	m := NewMem()
	if err := m.SaveSpec(sinkSpec("body")); err != nil {
		t.Fatal(err)
	}
	got, _ := m.LastSpec("x")
	got.Content[0] = 'X'
	if again, _ := m.LastSpec("x"); string(again.Content) != "body" {
		t.Errorf("stored content changed to %q", again.Content)
	}
}

func sinkSpec(content string) plugin.Spec {
	return plugin.Baseline("x", "1", "v1", "https://example.test/1", []byte(content), Clock)
}
