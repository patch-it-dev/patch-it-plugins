package changelog

import "testing"

func TestPostHogGoSDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{PostHogGo, "2.0.0", "1.44.1", "posthog-go-2.0.0-CHANGELOG.md", "", "1.44.1"},
	})
}
