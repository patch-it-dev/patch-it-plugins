package supabase

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @supabase/supabase-js 2.0.0 removed the v1 subscription methods.
func TestJSSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: JSSurface, Golden: [2]string{"1.35.7", "2.0.0"}, Want: []string{
		"removed_symbol SupabaseClient.getSubscriptions ",
		"removed_symbol SupabaseClient.removeAllSubscriptions ",
		"removed_symbol SupabaseClient.removeSubscription ",
	}, Drop: [2]string{"2.118.0", "2.119.0"}})
}
