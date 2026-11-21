package members_test

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"

	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

// The guide as published at tag v2.0.0 (fetched 2026-11-04).
func guide(t *testing.T) []byte {
	t.Helper()
	return kittest.Read(t, "testdata/numpy_2_0_migration_guide-v2.0.0.rst")
}

func byMember(t *testing.T) map[string]Removal {
	t.Helper()
	rs, err := Parse(guide(t), NumPy)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Removal{}
	for _, r := range rs {
		out[r.Member] = r
	}
	return out
}

func TestEveryRemovedMemberOfTheMainNamespaceIsRead(t *testing.T) {
	got := byMember(t)
	if len(got) != 56 {
		t.Errorf("%d removed members, want the table's 56", len(got))
	}
	// A guideline continued over several lines is read whole.
	if r := got["find_common_type"]; !strings.Contains(r.Guideline, "``0.0``, or ``0j``") {
		t.Errorf("find_common_type: %q", r.Guideline)
	}
	// The table's escaped underscore is the member's own.
	if _, ok := got["float_"]; !ok {
		t.Error(`no "float_": the \_ escape wasn't undone`)
	}
}

// Only a guideline naming exactly one member of np is a rename; anything else
// (a builtin, an expression, advice) is a removal the patch-it client
// opens as an issue.
func TestOnlyOnePlainReplacementInNpIsARename(t *testing.T) {
	got := byMember(t)
	for member, want := range map[string]string{
		"float_":     "float64",
		"NaN":        "nan",
		"product":    "prod",
		"cumproduct": "cumprod",
		"string_":    "bytes_",
		"Inf":        "inf",
		"alltrue":    "", // "Use ``all`` instead.": the builtin, or np.all? Not said
		"NINF":       "", // "Use ``-np.inf`` instead.": an expression
		"asfarray":   "", // "Use ``np.asarray`` with a float dtype instead."
		"compat":     "", // no replacement
		"who":        "",
	} {
		if r, ok := got[member]; !ok || r.Replacement != want {
			t.Errorf("%s: replacement %q, want %q", member, r.Replacement, want)
		}
	}
}

func TestTheGuideStoresOneEventPerMemberAndARerunNothing(t *testing.T) {
	// The guide at the v2.0.0 tag, and nothing else; kittest extracts the
	// pair again and fails on anything new.
	events := release(t, NumPy, "testdata/numpy_2_0_migration_guide-v2.0.0.rst", "1.26.4", "2.0.0")
	if len(events) != 56 {
		t.Fatalf("%d events, want 56", len(events))
	}
	var float *event.Event
	for i := range events {
		if events[i].Subject.Symbols[0] == "float_" {
			float = &events[i]
		}
	}
	if float == nil || float.Subject.Change != event.PackageRenamedSymbol || float.Subject.Replacement != "float64" ||
		float.API != "numpy/numpy" || float.Version != "2.0.0" || !strings.Contains(float.SourceExcerpt, "float\\_") {
		t.Errorf("float_: %+v", float)
	}
}
