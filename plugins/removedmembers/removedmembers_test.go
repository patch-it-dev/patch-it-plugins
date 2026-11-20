package removedmembers

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// Code written against the old path still builds and runs the same source.
func TestTheOldPathStillWorks(t *testing.T) {
	p := Plugin{Guide: NumPy2}
	kittest.Registered(t, p)
	if p.Name() != "numpy" || NumPy2.URL() == "" {
		t.Errorf("Plugin{Guide: NumPy2} is %q at %q", p.Name(), NumPy2.URL())
	}
}
