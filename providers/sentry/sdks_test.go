package sentry

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestSentrySDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: PythonChangelog, Release: "2.0.0", Prev: "1.45.1", Saved: "sentry-python-changelog-2.0.0-CHANGELOG.md", Dropped: "", Drop: "1.45.0"},
		{File: JavascriptChangelog, Release: "10.0.0", Prev: "9.47.2", Saved: "sentry-javascript-changelog-10.0.0-CHANGELOG.md", Dropped: "", Drop: "9.40.0"},
		{File: Go, Release: "0.31.0", Prev: "0.30.0", Saved: "sentry-go-0.31.0-CHANGELOG.md", Dropped: "", Drop: "0.30.0"},
		{File: Dotnet, Release: "4.0.0", Prev: "3.41.4", Saved: "sentry-dotnet-4.0.0-CHANGELOG.md", Dropped: "", Drop: "3.41.4"},
	})
}
