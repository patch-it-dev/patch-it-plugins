package retirements

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/retirements/retire"
)

// Code written against the old paths still builds and runs the same source.
func TestTheOldPathStillWorks(t *testing.T) {
	p := Plugin{Page: retire.OpenAI}
	kittest.Registered(t, p)
	if p.Name() != "openai-models" {
		t.Errorf("Plugin{Page: retire.OpenAI} is %q", p.Name())
	}
}
