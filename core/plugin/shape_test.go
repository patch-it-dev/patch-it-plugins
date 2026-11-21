package plugin

import (
	"errors"
	"testing"
)

func TestCheckCount(t *testing.T) {
	if err := (*Shape)(nil).CheckCount("items", 0.9, 77, 0); err != nil {
		t.Errorf("no shape, no guard: %v", err)
	}
	for _, c := range []struct {
		name        string
		shape       Shape
		before, now int
		changed     bool
	}{
		{"first read", Shape{}, -1, 0, false},
		{"nothing before", Shape{}, 0, 0, false},
		{"same", Shape{}, 77, 77, false},
		{"grew", Shape{}, 77, 90, false},
		{"a few removed", Shape{}, 77, 72, false},
		{"most gone", Shape{}, 77, 20, true},
		{"none left of a few", Shape{}, 3, 0, true},
		{"a few of a few: not compared", Shape{}, 4, 1, false},
		{"the source's own ratio", Shape{MinRatio: 0.2}, 77, 20, false},
	} {
		err := (&c.shape).CheckCount("items", 0.9, c.before, c.now)
		if got := errors.Is(err, ErrShapeChanged); got != c.changed {
			t.Errorf("%s: %v, want changed %v", c.name, err, c.changed)
		}
	}
}

func TestCheckAnchors(t *testing.T) {
	s := Shape{Anchors: []Anchor{{Entry: "gpt-4-0613", Checked: "2026-11-21"}}}
	has := func(e string) bool { return e == "gpt-4-0613" }
	if err := (*Shape)(nil).CheckAnchors(func(string) bool { return false }); err != nil {
		t.Errorf("no shape, no guard: %v", err)
	}
	if err := s.CheckAnchors(has); err != nil {
		t.Errorf("anchor present: %v", err)
	}
	if err := s.CheckAnchors(func(string) bool { return false }); !errors.Is(err, ErrShapeChanged) {
		t.Errorf("anchor missing: %v", err)
	}
}

// Nothing to read on the newest releases in a row, after one that had
// something: a shape signal, not ErrNothingToRead.
func TestNothingToRead(t *testing.T) {
	for _, c := range []struct {
		name    string
		read    []bool
		changed bool
	}{
		{"every release read", []bool{true, true, true}, false},
		{"one release without a section", []bool{true, false, true}, false},
		{"two in a row", []bool{true, true, false, false}, false},
		{"three in a row after a read", []bool{true, true, false, false, false}, true},
		{"never had anything", []bool{false, false, false, false}, false},
		{"recovered", []bool{true, false, false, false, true}, false},
	} {
		err := Shape{}.NothingToRead(c.read)
		if errors.Is(err, ErrShapeChanged) != c.changed || errors.Is(err, ErrNothingToRead) {
			t.Errorf("%s: %v, want changed %v", c.name, err, c.changed)
		}
	}
	if err := (Shape{NothingToReadAfter: 5}).NothingToRead([]bool{true, false, false, false}); err != nil {
		t.Errorf("the source's own count: %v", err)
	}
}
