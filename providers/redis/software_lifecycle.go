package redis

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/lifecycle"
)

// SoftwareLifecycle is Redis Software's (Redis Enterprise's) cluster version
// end-of-life schedule (checked 2026-11-21: 12 versions, 5.2 to 8.2, each
// with its end-of-life date but the newest, "-"). Redis Open Source has no
// lifecycle page; the same page's database version table is left for later
// (see README.md).
var SoftwareLifecycle = lifecycle.Page{
	API: "redis/software-lifecycle", Product: "Redis Software",
	URL:     "https://redis.io/docs/latest/operate/rs/installing-upgrading/product-lifecycle/",
	Runtime: "redis-software",
	Rows:    softwareRows,
}

// softwareVersion is a row's first cell: "8.0 – October 2025".
var softwareVersion = regexp.MustCompile(`^(\d+\.\d+) – `)

// softwareRows reads the table whose header is "Version - Release date",
// "End of Life (EOL)".
func softwareRows(page []byte) ([]lifecycle.Row, error) {
	want := []string{"Version - Release date", "End of Life (EOL)"}
	for _, t := range lifecycle.HTMLTables(page) {
		if len(t.Rows) == 0 || !slices.Equal(t.Rows[0], want) {
			continue
		}
		var out []lifecycle.Row
		for _, r := range t.Rows[1:] {
			if len(r) != 2 {
				continue
			}
			m := softwareVersion.FindStringSubmatch(r[0])
			if m == nil {
				continue
			}
			out = append(out, lifecycle.Row{Version: m[1], EOL: lifecycle.Date(r[1]), Excerpt: t.Excerpt(r)})
		}
		return out, nil
	}
	return nil, fmt.Errorf("no table with the header %v (the page's shape may have changed): %w", want, plugin.ErrNothingToRead)
}
