package adapter

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// Daily keeps each day's last commit, in order.
func TestDailyKeepsEachDaysLastCommit(t *testing.T) {
	in := []plugin.Version{{ID: "2026-10-27", Ref: "a"}, {ID: "2026-10-28", Ref: "b"}, {ID: "2026-10-28", Ref: "c"}, {ID: "2026-10-30", Ref: "d"}}
	var got []string
	for _, v := range Daily(in) {
		got = append(got, v.ID+" "+v.Ref)
	}
	if want := "2026-10-27 a,2026-10-28 c,2026-10-30 d"; strings.Join(got, ",") != want {
		t.Errorf("got %v, want %s", got, want)
	}
}

func TestServerBasePathRefusesWhatItCantMove(t *testing.T) {
	if _, err := ServerBasePath([]byte(`{"servers":[{"url":"https://x/api"}],"paths":{"/a":{},"a":{}}}`)); err == nil {
		t.Error("a path without a leading /: want an error")
	}
	if _, err := ServerBasePath([]byte(`{"servers":[{"url":"https://x/api/0"},{"url":"https://y"}],"paths":{}}`)); err == nil {
		t.Error("servers with different paths: want an error")
	}
}

// A server whose whole origin is a variable (Convex's "{deployment-url}/api/v1")
// still has a path to move.
func TestServerPath(t *testing.T) {
	for in, want := range map[string]string{
		"{deployment-url}/api/v1":           "/api/v1",
		"{deployment-url}/api/v1/":          "/api/v1",
		"{deployment-url}":                  "",
		"https://{region}.sentry.io/api/0":  "/api/0",
		"https://{region}.sentry.io/api/0/": "/api/0",
		"https://{region}.sentry.io":        "",
		"https://sentry.io/":                "",
		"/api/0":                            "",
	} {
		if got := serverPath(in); got != want {
			t.Errorf("serverPath(%q) = %q, want %q", in, got, want)
		}
	}
}
