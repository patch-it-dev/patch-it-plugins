package mongodb

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/lifecycle"
)

// Server is the MongoDB Server's end-of-life dates, from the "MongoDB
// Server" table of MongoDB's software lifecycle schedules (checked
// 2026-11-21: 14 releases, 3.0 to 9.0, each with its end-of-life date). The
// page's other tables (Atlas, Kubernetes controllers, Ops Manager, Search)
// are other products, and aren't read.
var Server = lifecycle.Page{
	API: "mongodb/server", Product: "MongoDB Server",
	URL:     "https://www.mongodb.com/legal/support-policy/lifecycles",
	Runtime: "mongodb",
	Rows:    serverRows,
}

var serverRelease = regexp.MustCompile(`^MongoDB (\d+\.\d+)$`)

// serverRows reads the table under the "MongoDB Server" heading whose header
// is Release, Release Date, End of Life Date.
func serverRows(page []byte) ([]lifecycle.Row, error) {
	want := []string{"Release", "Release Date", "End of Life Date"}
	for _, t := range lifecycle.HTMLTables(page) {
		if t.Heading != "MongoDB Server" || len(t.Rows) == 0 || !slices.Equal(t.Rows[0], want) {
			continue
		}
		var out []lifecycle.Row
		for _, r := range t.Rows[1:] {
			if len(r) != 3 {
				continue
			}
			m := serverRelease.FindStringSubmatch(r[0])
			if m == nil {
				continue
			}
			out = append(out, lifecycle.Row{Version: m[1], EOL: lifecycle.Date(r[2]), Excerpt: t.Excerpt(r)})
		}
		return out, nil
	}
	return nil, fmt.Errorf("no \"MongoDB Server\" table with the header %v (the page's shape may have changed): %w", want, plugin.ErrNothingToRead)
}
