package migrationguide

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Guide: sdkguide.AnthropicV1})
}
