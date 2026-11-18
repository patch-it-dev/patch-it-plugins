package migrationguide

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/sdkguide"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Guide: sdkguide.AnthropicV1})
}
