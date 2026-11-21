package changelog

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	for _, f := range []File{Textual0860, Urllib3200} {
		kittest.Registered(t, Plugin{File: f})
	}
	kittest.Registered(t, Plugin{Log: MongoDBAtlas})
}
