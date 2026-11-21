package posthog

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestPostHogGoSDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: Go, Release: "2.0.0", Prev: "1.44.1", Saved: "posthog-go-2.0.0-CHANGELOG.md", Dropped: "", Drop: "1.44.1"},
	})
}
