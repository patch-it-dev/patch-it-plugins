package retirements

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/retire"
)

func TestRegistersItself(t *testing.T) {
	for _, p := range []retire.Page{retire.OpenAI, retire.Anthropic} {
		kittest.Registered(t, Plugin{Page: p})
	}
}
