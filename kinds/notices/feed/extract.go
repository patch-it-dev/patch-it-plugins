package feed

import "github.com/patch-it-dev/patch-it-plugins/core/extract"

// The feed and HTML helpers live in core/extract, for every reader; these
// names keep the readers here (and code that imports them) as they were.
type (
	Entry   = extract.Entry
	Section = extract.Section
)

var (
	Parse   = extract.ParseFeed
	Text    = extract.Text
	Lines   = extract.Lines
	SplitBy = extract.SplitBy
	Date    = extract.Date
	Codes   = extract.Codes
)
