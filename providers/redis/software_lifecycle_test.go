package redis

import (
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/lifecycle"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

const softwarePage = "testdata/redis-software-lifecycle.html"

// The golden test: every cluster version with an end-of-life date.
func TestSoftwareLifecycleGolden(t *testing.T) {
	noticestest.Golden(t, SoftwareLifecycle, softwarePage)
}

// The pinned drop test: the newest version's end of life is "-" (8.2 on
// 2026-11-21), and the page's database version table ("Redis 8.0 | STS |
// December 1, 2026") is another runtime with another header: neither is an
// event.
func TestSoftwareLifecycleDropsNoDateAndTheDatabaseTable(t *testing.T) {
	b, err := os.ReadFile(softwarePage)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := softwareRows(b)
	if err != nil {
		t.Fatal(err)
	}
	var newest lifecycle.Row
	for _, r := range rows {
		if r.Version == "8.2" {
			newest = r
		}
		if strings.HasPrefix(r.Excerpt, "Redis 8.0") || strings.Contains(r.Excerpt, "STS") {
			t.Errorf("a database version row read: %q", r.Excerpt)
		}
	}
	if newest.Version == "" || newest.EOL != "" {
		t.Errorf("8.2: %+v, want listed with no date", newest)
	}
	found, err := SoftwareLifecycle.Find(b)
	if err != nil {
		t.Fatal(err)
	}
	if found.Has("8.2") {
		t.Error("8.2 counts as dated")
	}
}
