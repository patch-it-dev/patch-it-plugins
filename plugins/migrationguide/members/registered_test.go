package members_test

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"

	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	. "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members" // the reader under test
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, migrationguide.Plugin{Guide: NumPy})
}
