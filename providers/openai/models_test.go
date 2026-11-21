package openai

import (
	"context"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/noticestest"
)

// OpenAI's saved page (2026-11-21, testdata/README.md) in, every event out.
func TestModelsGolden(t *testing.T) {
	noticestest.Golden(t, Models, "testdata/openai-deprecations.md")
}

// The page's layout before 2026-11 (one table per notice, plain dates, no
// Status or Notice date column) still reads to the same events it did.
func TestModelsGoldenBefore202611(t *testing.T) {
	noticestest.GoldenFile(t, Models, "testdata/openai-deprecations-2026-10-13.md", "testdata/layouts/openai-models-2026-10-13.json")
}

// The drop test: a realistic update of today's page that must produce no
// event. A feature or endpoint row (no model column), a new image model and
// a text model whose Status isn't a retirement add nothing a client sends to
// the Chat Completions or Responses APIs, and a row with no date yet isn't a
// retirement.
func TestModelsDropsWhatIsNotATextModelRetirement(t *testing.T) {
	before := read(t, "openai-deprecations.md")
	after := []byte(string(before) + `
## Upcoming deprecations

| Shutdown date    | Affected model         | Status     | Recommended replacement   | Notice date  |
| ---------------- | ---------------------- | ---------- | ------------------------- | ------------ |
| **Mar 1, 2027**  | ` + "`gpt-image-2.1`" + `        | Deprecated | ` + "`gpt-image-2.6-flare`" + `    | Nov 20, 2026 |
| **Mar 1, 2027**  | ` + "`gpt-5.5-nova`" + `         | Postponed  | ` + "`gpt-6-luna`" + `             | Nov 20, 2026 |
| To be announced  | ` + "`gpt-5.5-ember`" + `        | Deprecated | ` + "`gpt-6-luna`" + `             | Nov 20, 2026 |

## Feature and endpoint deprecations

| Shutdown date    | Affected feature or endpoint | Status     | Recommended replacement   | Notice date  |
| ---------------- | ---------------------------- | ---------- | ------------------------- | ------------ |
| **Mar 1, 2027**  | ` + "`/v1/assistants`" + `               | Deprecated | ` + "`/v1/responses`" + `          | Nov 20, 2026 |
`)
	env := plugin.Env{Now: func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }}
	from := plugin.Snapshot{Version: plugin.Version{ID: "2026-11-21T12:00:00Z"}, Content: before, SourceURL: "https://web.archive.org/web/20261121120000/" + Models.URL}
	to := plugin.Snapshot{Version: plugin.Version{ID: "2026-11-22T12:00:00Z"}, Content: after, SourceURL: "https://web.archive.org/web/20261122120000/" + Models.URL}
	res, err := notices.Plugin{Page: Models}.Extract(context.Background(), env, from, to)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Events {
		t.Errorf("unexpected event: %s %s", e.Subject.Name, e.Description)
	}
}
