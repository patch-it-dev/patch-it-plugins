package supabase

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// supabase 1.0.0's SupabaseAuthClient lost four parameters. 2.34.0 made
// supabase.client a star re-export of the package: still importable.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"0.7.1", "1.0.0"}, Want: []string{
		"removed_parameter SupabaseAuthClient,lib.auth_client.SupabaseAuthClient api",
		"removed_parameter SupabaseAuthClient,lib.auth_client.SupabaseAuthClient cookie_options",
		"removed_parameter SupabaseAuthClient,lib.auth_client.SupabaseAuthClient local_storage",
		"removed_parameter SupabaseAuthClient,lib.auth_client.SupabaseAuthClient replace_default_headers",
		"removed_parameter lib.client_options.ClientOptions.replace local_storage",
	}, Drop: [2]string{"2.33.0", "2.34.0"}})
}
