package changelog

import "testing"

func TestSentrySDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{SentryPython, "2.0.0", "1.45.1", "sentry-python-changelog-2.0.0-CHANGELOG.md", "", "1.45.0"},
		{SentryJavaScript, "10.0.0", "9.47.2", "sentry-javascript-changelog-10.0.0-CHANGELOG.md", "", "9.40.0"},
		{SentryGo, "0.31.0", "0.30.0", "sentry-go-0.31.0-CHANGELOG.md", "", "0.30.0"},
		{SentryDotnet, "4.0.0", "3.41.4", "sentry-dotnet-4.0.0-CHANGELOG.md", "", "3.41.4"},
	})
}
