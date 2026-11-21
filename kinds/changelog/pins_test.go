package changelog

import (
	"maps"
	"testing"
)

// Two different versions in one release's section are none; a statement in
// a code block, or before the first release, is nobody's.
func TestAnUnsurePinIsNone(t *testing.T) {
	got := Pins([]byte("# Changelog\nThis release changes the pinned API version to `2020-01-01`.\n" +
		"## 2.0.0 - 2026-01-02\nThis release changes the pinned API version to `2026-01-01.a`.\nThis release changes the pinned API version to `2026-01-02.b`.\n" +
		"## 1.1.0 - 2026-01-01\n```\nThis release changes the pinned API version to `2025-12-01.a`.\n```\n" +
		"## 1.0.0 - 2025-12-01\n* This release changes the pinned API version to `2025-11-01.a`.\n"))
	if want := map[string]string{"1.0.0": "2025-11-01.a"}; !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// A "# comment" in a code example doesn't end the release's section.
	got = Pins([]byte("## 3.0.0 - 2026-02-01\n```sh\n# upgrade\n```\nThis release changes the pinned API version to `2026-02-01.c`.\n"))
	if want := map[string]string{"3.0.0": "2026-02-01.c"}; !maps.Equal(got, want) {
		t.Errorf("after a code block: got %v, want %v", got, want)
	}
}
