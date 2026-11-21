package kittest

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
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
