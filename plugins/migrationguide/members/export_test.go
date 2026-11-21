package members

// What the external tests (package members_test, which can import the kind
// that imports this package) reach of the unexported helpers.
var (
	PipeCells = pipeCells
	SoleName  = soleName
)

func (g Guide) TagOf(release string) string   { return g.tagOf(release) }
func (g Guide) CommitsURL(major int) string   { return g.commitsURL(major) }
func (g Guide) AbsoluteLinks(s string) string { return g.absoluteLinks(s) }
