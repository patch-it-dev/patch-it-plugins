package urllib3

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// urllib3 2.6.0 removed BaseHTTPResponse.getheader and getheaders,
// deprecated since 2.0.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"2.5.0", "2.6.0"}, Want: []string{
		"removed_symbol BaseHTTPResponse.getheader,response.BaseHTTPResponse.getheader ",
		"removed_symbol BaseHTTPResponse.getheaders,response.BaseHTTPResponse.getheaders ",
	}, Drop: [2]string{"2.2.1", "2.2.2"}})
}
