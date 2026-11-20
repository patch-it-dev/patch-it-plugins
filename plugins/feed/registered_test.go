package feed

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	for _, f := range []Feed{GitHubActions, VertexAIModels, CloudflareAPI} {
		kittest.Registered(t, Plugin{Feed: f})
	}
}
