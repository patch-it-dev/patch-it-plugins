package mongodb

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/lifecycle"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

const lifecycles = "testdata/mongodb-lifecycles.html"

// The golden test: every MongoDB Server release in the table, one runtime
// event per end-of-life date.
func TestServerGolden(t *testing.T) {
	noticestest.Golden(t, Server, lifecycles)
}

// The pinned drop test: the page's Atlas tables have the same header and
// list the same releases, with other dates (MongoDB 8.0: June 20, 2025 on
// Atlas auto-upgrade, October 31, 2029 for the server), and one with no date
// ("TBD, no sooner than October 2026"). Without the "MongoDB Server" table
// nothing is read, and a cell that isn't a plain date gives no date.
func TestServerDropsTheAtlasTables(t *testing.T) {
	b, err := os.ReadFile(lifecycles)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	i := strings.Index(page, ">MongoDB Server<")
	if i < 0 {
		t.Fatal("the saved page's shape changed")
	}
	atlas := page[:i]
	rows, err := Server.Rows([]byte(atlas))
	if !errors.Is(err, plugin.ErrNothingToRead) || len(rows) != 0 {
		t.Errorf("the Atlas tables alone: %d rows, %v; want none read", len(rows), err)
	}
	var tbd bool
	for _, tb := range lifecycle.HTMLTables([]byte(atlas)) {
		for _, r := range tb.Rows {
			if len(r) == 3 && strings.HasPrefix(r[2], "TBD") {
				tbd = true
				if d := lifecycle.Date(r[2]); d != "" {
					t.Errorf("%q read as the date %s", r[2], d)
				}
			}
		}
	}
	if !tbd {
		t.Error("no TBD row in the Atlas tables: the drop proves nothing")
	}
}
