package openai

import (
	"context"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
)

const (
	wheel111 = "https://files.pythonhosted.org/packages/f7/86/ea99e3f966a639a3256e2f79d78cd63d45d5eb651a5457ccbdcfd9eed24a/openai-1.1.1-py3-none-any.whl"
	wheel112 = "https://files.pythonhosted.org/packages/dd/82/b92f73453ea318c0d46f31aeb56e9d94a42606c010fb72a513f4a3cd4bac/openai-1.1.2-py3-none-any.whl"
)

// The saved releases (testdata/README.md): PyPI's answers and the two
// wheels, trimmed to the openai.types.beta package.
var saved = kittest.Pages{
	"https://pypi.org/pypi/openai/json":       "testdata/pypi-openai-excerpt.json",
	"https://pypi.org/pypi/openai/1.1.1/json": "testdata/pypi-openai-1.1.1.json",
	"https://pypi.org/pypi/openai/1.1.2/json": "testdata/pypi-openai-1.1.2.json",
	wheel111: "testdata/openai-1.1.1-py3-none-any.trimmed.whl",
	wheel112: "testdata/openai-1.1.2-py3-none-any.trimmed.whl",
}

// pair lists the releases, reads 1.1.1 and 1.1.2 and extracts the pair, as
// patch-it's hosted ingestion service does, into the test kit's store.
func pair(t *testing.T) []event.Event {
	t.Helper()
	ctx, env := context.Background(), kittest.Env(t, saved)
	p := apisurface.Plugin{Package: PythonSurface}
	vs, err := p.ListVersions(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range vs {
		ids = append(ids, v.ID)
	}
	if got := strings.Join(ids, " "); got != "1.1.0 1.1.1 1.1.2" {
		t.Fatalf("versions %s", got)
	}
	last := versions.Pairs(vs)[1]
	return kittest.Versions(t, env, p, last[0], last[1])
}

// openai 1.1.2 renamed the misspelled AsssitantDeleted to AssistantDeleted,
// module and all: the old name is gone from both places code could import
// it. (Not read as a rename: the class has no parameters to tell it's the
// same class.) AssistantDeleted, new, is the pair's addition.
func TestPythonSurfaceGolden(t *testing.T) {
	got := pair(t)
	kittest.Golden(t, "testdata/golden/openai-python-surface.json", got)
	for _, e := range got {
		file := "openai/types/beta/asssitant_deleted.py line "
		if e.Change == event.ChangeAdded {
			file = "openai/types/beta/assistant_deleted.py line "
		}
		if e.PreviousVersion != "1.1.1" || e.Version != "1.1.2" || e.SourceURL != wheel112 || !strings.Contains(e.SourceExcerpt, file) {
			t.Errorf("%v: %s<-%s, excerpt %q", e.Subject.Symbols, e.Version, e.PreviousVersion, e.SourceExcerpt)
		}
	}
}

// The pinned drop test: 1.1.2 adds AssistantDeleted and its module. A name
// that appears is an addition, never a breaking change.
func TestPythonSurfaceAnAddedNameIsNoEvent(t *testing.T) {
	added := 0
	for _, e := range pair(t) {
		for _, s := range e.Subject.Symbols {
			if strings.Contains(s, "AssistantDeleted") || strings.Contains(s, "assistant_deleted") {
				if e.Change != event.ChangeAdded || e.Subject.Change != event.PackageAddedSymbol {
					t.Errorf("%s %s for %s, which 1.1.2 added", e.Change, e.Subject.Change, s)
				}
				added++
			}
		}
	}
	if added == 0 {
		t.Error("no added event for AssistantDeleted")
	}
}

func TestPythonSurfaceRegistersItself(t *testing.T) {
	kittest.Registered(t, apisurface.Plugin{Package: PythonSurface})
}
