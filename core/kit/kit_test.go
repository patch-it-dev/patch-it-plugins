package kit

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type archive struct {
	body string
	err  error
}

func (a archive) Snapshot(_ context.Context, page string) (string, []byte, error) {
	return "https://web.archive.org/web/20261114120000/" + page, []byte(a.body), a.err
}

func lines(b []byte) ([]string, error) {
	if len(b) == 0 {
		return nil, errors.New("empty")
	}
	return strings.Split(string(b), "\n"), nil
}

func TestCiteReturnsASnapshotThatHoldsWhatThePageSaid(t *testing.T) {
	want := []string{"a", "b"}
	url, snap, got, err := Cite(context.Background(), archive{body: "a\nb"}, "https://x.test/p", lines, want)
	if err != nil || url != "https://web.archive.org/web/20261114120000/https://x.test/p" || string(snap) != "a\nb" || len(got) != 2 {
		t.Errorf("%s %q %v %v", url, snap, got, err)
	}
}

// No snapshot, an unreadable one, or one that says something else: no
// provenance, so no events.
func TestCiteRefusesWhatItCantCite(t *testing.T) {
	for name, a := range map[string]archive{
		"refused":    {err: errors.New("429")},
		"unreadable": {body: ""},
		"changed":    {body: "a\nc"},
	} {
		if _, _, _, err := Cite(context.Background(), a, "https://x.test/p", lines, []string{"a", "b"}); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
