package removedmembers

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Guide: NumPy2})
}
