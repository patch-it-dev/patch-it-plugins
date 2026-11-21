package kittest

import (
	"fmt"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// CheckSink checks that the Sinks open returns store as patch-it's hosted
// ingestion service does: dedupe by fingerprint, ids in save order, an invalid
// event saving nothing. Mem passes it; the
// hosted service runs it against its own store too, so the two can't drift
// apart. open must return an empty Sink.
func CheckSink(t *testing.T, open func(t *testing.T) Sink) {
	t.Run("dedupe and ids", func(t *testing.T) {
		s := open(t)
		first := []event.Event{sinkEvent(1), sinkEvent(2), sinkEvent(1)}
		if n, err := s.SaveEvents(first); err != nil || n != 2 {
			t.Fatalf("first save: %d new, %v; want 2 (the repeat in the batch is skipped)", n, err)
		}
		if n, err := s.SaveEvents([]event.Event{sinkEvent(2), sinkEvent(3)}); err != nil || n != 1 {
			t.Fatalf("second save: %d new, %v; want 1", n, err)
		}
		all, err := s.All()
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for i, e := range all {
			got = append(got, e.Subject.Property)
			if i > 0 && e.ID <= all[i-1].ID {
				t.Errorf("id %d after %d: ids must grow in save order", e.ID, all[i-1].ID)
			}
		}
		if want := "[field_1 field_2 field_3]"; fmt.Sprint(got) != want {
			t.Errorf("stored %v, want %s", got, want)
		}
	})
	t.Run("an invalid event saves nothing", func(t *testing.T) {
		s := open(t)
		bad := sinkEvent(2)
		bad.SourceExcerpt = ""
		if _, err := s.SaveEvents([]event.Event{sinkEvent(1), bad}); err == nil {
			t.Fatal("saved an event without provenance")
		}
		if all, _ := s.All(); len(all) != 0 {
			t.Errorf("%d events stored from a rejected batch", len(all))
		}
	})
}

func sinkEvent(n int) event.Event {
	subject := event.Subject{
		Type: event.SubjectAPIOperation, Kind: event.KindSchemaProperty, Side: event.SideResponse,
		Schema: "thing", Property: fmt.Sprintf("field_%d", n),
	}
	return event.Event{
		Fingerprint:     event.Fingerprint("example/x", "2", event.ChangeTypeBreaking, subject, ""),
		API:             "example/x",
		PreviousVersion: "1",
		Version:         "2",
		ChangeType:      event.ChangeTypeBreaking,
		Breaking:        true,
		Subject:         subject,
		Description:     fmt.Sprintf("Property field_%d removed.", n),
		SourceURL:       "https://example.test/2",
		SourceExcerpt:   `{"type":"string"}`,
		Confidence:      1,
		DetectedAt:      Clock,
	}
}
