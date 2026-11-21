package migrationguide

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Guide: anthropicGuideV1})
}
