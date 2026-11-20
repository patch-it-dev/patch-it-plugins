package eol

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// Code written against the old path still builds and runs the same source.
func TestTheOldPathStillWorks(t *testing.T) {
	p := Plugin{Page: Salesforce}
	kittest.Registered(t, p)
	if p.Name() != "salesforce" {
		t.Errorf("Plugin{Page: Salesforce} is %q", p.Name())
	}
}
