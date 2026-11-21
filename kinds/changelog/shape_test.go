package changelog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// changelogFile is a Keep a Changelog file heading releases 1.0.0 to 1.<n-1>.0, newest first.
func changelogFile(n int) []byte {
	var b strings.Builder
	b.WriteString("# Changelog\n\n")
	for i := n - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "## [1.%d.0] - 2026-01-%02d\n\n### Fixed\n\n- a fix\n\n", i, i%28+1)
	}
	return []byte(b.String())
}

func at(release string, content []byte) plugin.Snapshot {
	return plugin.Snapshot{Version: plugin.Version{ID: release, Ref: "v" + release}, Content: content, SourceURL: "https://x.test/" + release}
}

// A changelog moved elsewhere leaves a stub at the old path: far fewer
// release headings than the release before is a shape change, not "nothing
// to read" for this release.
func TestAMovedChangelogChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	p := Plugin{File: File{API: "acme/acme", Package: "acme", Ecosystem: "pypi", Module: "acme", Repo: "acme/acme", TagPrefix: "v", Path: "CHANGELOG.md"}}
	guarded := p.WithShape(plugin.Shape{})
	stub := []byte("# Changelog\n\nThe changelog moved to docs/changes.md.\n")
	_, err := guarded.Extract(context.Background(), env, at("1.11.0", changelogFile(12)), at("1.12.0", stub))
	if !errors.Is(err, plugin.ErrShapeChanged) || errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("moved: %v, want ErrShapeChanged only", err)
	}
	// Reformatted: the same releases under headings the reader no longer knows.
	reformatted := []byte(strings.ReplaceAll(string(changelogFile(13)), "## [", "**Release ["))
	if _, err := guarded.Extract(context.Background(), env, at("1.11.0", changelogFile(12)), at("1.12.0", reformatted)); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("reformatted: %v", err)
	}
	// The next release, one more heading: read as before.
	if _, err := guarded.Extract(context.Background(), env, at("1.11.0", changelogFile(12)), at("1.12.0", changelogFile(13))); err != nil {
		t.Errorf("next release: %v", err)
	}
	// Built by hand, no Shape: no guard, the stub is only nothing to read.
	if _, err := p.Extract(context.Background(), env, at("1.11.0", changelogFile(12)), at("1.12.0", stub)); !errors.Is(err, plugin.ErrNothingToRead) {
		t.Errorf("unguarded: %v", err)
	}
}

// A JSON changelog keeps every day: one with far fewer days changed shape.
func TestAJSONChangelogThatLostItsDaysChangedShape(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	days := func(n int) []byte {
		var ds []string
		for i := 0; i < n; i++ {
			ds = append(ds, fmt.Sprintf(`{"date": "2026-01-%02d", "paths": []}`, i+1))
		}
		return []byte("[" + strings.Join(ds, ",") + "]")
	}
	p := Plugin{Log: Log{API: "acme/api", Vendor: "Acme", Name: "Acme API", Repo: "acme/api", Path: "changelog.json"}}.WithShape(plugin.Shape{})
	if _, err := p.Extract(context.Background(), env, at("2026-01-20", days(20)), at("2026-01-21", days(2))); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("20 -> 2 days: %v", err)
	}
	if _, err := p.Extract(context.Background(), env, at("2026-01-20", days(20)), at("2026-01-21", days(21))); err != nil {
		t.Errorf("20 -> 21 days: %v", err)
	}
}

// Release notes on GitHub are one release each: a run of releases with no
// notes after ones that had some is the runner's shape signal
// (Shape.NothingToRead), never the kind's.
func TestReleasesWithNoNotesInARowAreAShapeSignal(t *testing.T) {
	env := plugin.Env{Now: time.Now}
	p := Plugin{File: File{API: "acme/acme", Package: "acme", Ecosystem: "pypi", Module: "acme", Repo: "acme/acme", TagPrefix: "v", Notes: true}}.WithShape(plugin.Shape{})
	notes := func(tag, body string) []byte {
		return []byte(fmt.Sprintf(`{"tag_name": "v%s", "body": %q}`, tag, body))
	}
	var read []bool
	prev := at("1.0.0", notes("1.0.0", "## Fixed\n\n- a fix\n"))
	for i, body := range []string{"## Fixed\n\n- a fix\n", "", "", ""} {
		release := fmt.Sprintf("1.%d.0", i+1)
		cur := at(release, notes(release, body))
		_, err := p.Extract(context.Background(), env, prev, cur)
		if err != nil && !errors.Is(err, plugin.ErrNothingToRead) {
			t.Fatalf("%s: %v", release, err)
		}
		read = append(read, err == nil)
		prev = cur
	}
	if err := (plugin.Shape{}).NothingToRead(read); !errors.Is(err, plugin.ErrShapeChanged) {
		t.Errorf("%v: %v, want ErrShapeChanged", read, err)
	}
}
