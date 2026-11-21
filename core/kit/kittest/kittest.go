// Package kittest is the plugin test kit: saved copies of a source in, the
// events a plugin stores out, compared with a golden file. Nothing reaches the
// network or a database: a URL the test didn't save fails the test, events
// are kept in memory (Mem), and the clock is fixed, so the same pages always
// give byte-identical events. A source is read as patch-it's hosted ingestion
// service reads it, in its shape: a version pair of a versioned source
// (Versions, or Extracted when the test builds the snapshots), or a living
// page's first read (Page).
//
//	env := kittest.Env(t, kittest.Pages{url: "testdata/page.md"})
//	got := kittest.Versions(t, env, p, plugin.Version{ID: "1.26.15", Ref: "1.26.15"}, plugin.Version{ID: "2.0.0", Ref: "2.0.0"})
//	kittest.Golden(t, "testdata/golden/urllib3-2.0.0.json", got)
//
// `go test ./providers/<name> -update` rewrites a provider's golden files;
// review the diff like code.
package kittest

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

var update = flag.Bool("update", false, "rewrite the golden files from what the plugins store now")

// Clock is the time every kit run detects its changes at.
var Clock = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

// Pages are the saved copies a run may fetch: URL to file, relative to the
// test's package.
type Pages map[string]string

// Get serves Pages, and fails the test on any other URL.
func (p Pages) Get(t testing.TB) func(context.Context, string) ([]byte, error) {
	return func(_ context.Context, url string) ([]byte, error) {
		file, ok := p[url]
		if !ok {
			t.Errorf("fetched %s, which the test didn't save", url)
			return nil, fmt.Errorf("no saved copy of %s", url)
		}
		return Read(t, file), nil
	}
}

// Read is a saved file, or the test fails.
func Read(t testing.TB, file string) []byte {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Env is Pages for the network and Clock for the time. Nothing outside the
// test is touched: no database, no network.
func Env(t testing.TB, pages Pages) plugin.Env {
	t.Helper()
	return plugin.Env{Get: pages.Get(t), Now: func() time.Time { return Clock }}
}

// Versions is the version pair from -> to of p, as the hosted ingestion
// service extracts it: ReadAt both sides, Extract. A zero from is read as
// nothing (no version before). It returns the events stored; any error or
// rejected event fails the test, and so does extracting the same pair again
// storing anything new: the fingerprints must dedupe.
func Versions(t testing.TB, env plugin.Env, p plugin.Versioned, from, to plugin.Version) []event.Event {
	t.Helper()
	ctx := context.Background()
	var a plugin.Snapshot
	if from != (plugin.Version{}) {
		var err error
		if a, err = p.ReadAt(ctx, env, from); err != nil {
			t.Fatalf("%s: reading %s: %v", p.Name(), from.ID, err)
		}
	}
	b, err := p.ReadAt(ctx, env, to)
	if err != nil {
		t.Fatalf("%s: reading %s: %v", p.Name(), to.ID, err)
	}
	return Extracted(t, env, p, a, b)
}

// Extracted is Versions over snapshots the test made: for a pair whose older
// side the test names without reading it (a changelog's or a guide's Extract
// reads only the newer side's own section).
func Extracted(t testing.TB, env plugin.Env, p plugin.Plugin, from, to plugin.Snapshot) []event.Event {
	t.Helper()
	sink := NewMem()
	if rejected, err := Extract(context.Background(), env, sink, p, from, to); err != nil || rejected {
		t.Fatalf("%s: rejected=%v err=%v", p.Name(), rejected, err)
	}
	first := Stored(t, sink)
	if rejected, err := Extract(context.Background(), env, sink, p, from, to); err != nil || rejected {
		t.Fatalf("%s, the same pair again: rejected=%v err=%v", p.Name(), rejected, err)
	}
	if again := Stored(t, sink); len(again) != len(first) {
		t.Errorf("%s: the same pair again stored %d more events", p.Name(), len(again)-len(first))
	}
	return first
}

// Page is a living page's first read, as the hosted ingestion service reads
// it: Read with nothing kept, Extract from nothing, the events stored. A
// second Read over the same pages must find the page unchanged (the snapshot
// kept, by CompareHash) and store nothing; any error or rejected event fails
// the test.
func Page(t testing.TB, env plugin.Env, p plugin.Living) []event.Event {
	t.Helper()
	ctx := context.Background()
	sink := NewMem()
	cur, rejected, err := ReadPage(ctx, env, sink, p, plugin.Snapshot{})
	if err != nil || rejected {
		t.Fatalf("%s: rejected=%v err=%v", p.Name(), rejected, err)
	}
	first := Stored(t, sink)
	again, rejected, err := ReadPage(ctx, env, sink, p, cur)
	if err != nil || rejected {
		t.Fatalf("%s, second read: rejected=%v err=%v", p.Name(), rejected, err)
	}
	if again.CompareHash != cur.CompareHash || again.Version != cur.Version {
		t.Errorf("%s: a second read of the same page is a new snapshot (%s), want the kept one (%s)", p.Name(), again.Version.ID, cur.Version.ID)
	}
	if n := len(Stored(t, sink)); n != len(first) {
		t.Errorf("%s: a second read of the same page stored %d more events", p.Name(), n-len(first))
	}
	return first
}

// Stored is every event in s, in feed order, or the test fails.
func Stored(t testing.TB, s Sink) []event.Event {
	t.Helper()
	all, err := s.All()
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// Golden compares events with the JSON file at path, or rewrites it with
// -update. Every event must also be valid, or the golden would pin a bug.
func Golden(t testing.TB, path string, events []event.Event) {
	t.Helper()
	golden(t, path, events, *update)
}

// CompareGolden is Golden that never rewrites the file, even with -update:
// for a test that checks someone else's golden file.
func CompareGolden(t testing.TB, path string, events []event.Event) {
	t.Helper()
	golden(t, path, events, false)
}

func golden(t testing.TB, path string, events []event.Event, rewrite bool) {
	t.Helper()
	for _, e := range events {
		if err := e.Validate(); err != nil {
			t.Errorf("event %d (%s) is invalid: %v", e.ID, e.Description, err)
		}
	}
	if events == nil {
		events = []event.Event{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(events); err != nil {
		t.Fatal(err)
	}
	if rewrite {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run the test with -update to create it)", err)
	}
	if !bytes.Equal(want, buf.Bytes()) {
		t.Errorf("%s: the stored events differ from the golden file (%d events now); if the change is intended, rerun with -update and review the diff\n%s",
			path, len(events), firstDiff(want, buf.Bytes()))
	}
}

// firstDiff is the first line that differs, for the failure message.
func firstDiff(want, got []byte) string {
	w, g := bytes.Split(want, []byte("\n")), bytes.Split(got, []byte("\n"))
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl []byte
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if !bytes.Equal(wl, gl) {
			return "line " + strconv.Itoa(i+1) + ":\n  want " + string(wl) + "\n  got  " + string(gl)
		}
	}
	return ""
}

// Archive is the Wayback Machine without the network: it serves a saved copy
// as the snapshot of a page, under a fixed timestamp, and counts the
// snapshots it was asked for.
type Archive struct {
	T         testing.TB
	Pages     Pages  // the page's URL to the file the snapshot holds
	Timestamp string // 14 digits; default 20261114120000
	Calls     int
}

func (a *Archive) Snapshot(_ context.Context, page string) (string, []byte, error) {
	a.Calls++
	file, ok := a.Pages[page]
	if !ok {
		return "", nil, fmt.Errorf("no snapshot of %s in the test archive", page)
	}
	ts := a.Timestamp
	if ts == "" {
		ts = "20261114120000"
	}
	return "https://web.archive.org/web/" + ts + "/" + page, Read(a.T, file), nil
}

// Registered fails the test unless p's package registered a plugin of p's
// type under p's name: each plugin package checks its own registration.
func Registered(t testing.TB, p plugin.Plugin) {
	t.Helper()
	got, ok := plugin.Lookup(p.Name())
	if !ok {
		t.Errorf("no plugin registered as %q", p.Name())
		return
	}
	if reflect.TypeOf(got) != reflect.TypeOf(p) {
		t.Errorf("%q is a %T, want a %T", p.Name(), got, p)
	}
}
