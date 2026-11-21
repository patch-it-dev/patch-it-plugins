package notices

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"
)

func TestRetirementsRegisterThemselves(t *testing.T) {
	for _, p := range []retire.Page{retire.OpenAI, retire.Anthropic, retire.Gemini} {
		kittest.Registered(t, Plugin{Page: p})
	}
}
