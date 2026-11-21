package notices

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/feed"
)

func TestFeedsRegisterThemselves(t *testing.T) {
	for _, f := range []feed.Feed{feed.GitHubActions, feed.VertexAIModels, feed.CloudflareAPI} {
		kittest.Registered(t, Plugin{Page: f})
	}
}
