package apichangelog

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

// Code written against the old path still builds and runs the same source.
func TestTheOldPathStillWorks(t *testing.T) {
	p := Plugin{Log: MongoDBAtlas}
	kittest.Registered(t, p)
	if p.Name() != "mongodb-atlas" {
		t.Errorf("Plugin{Log: MongoDBAtlas} is %q", p.Name())
	}
}
