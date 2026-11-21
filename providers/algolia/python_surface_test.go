package algolia

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// algoliasearch 4.44.3 dropped recommend's FacetFilters (4.44.4 brought
// it back: an addition, no event).
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"4.44.1", "4.44.3"}, Want: []string{
		"removed_symbol recommend.models.FacetFilters,recommend.models.facet_filters.FacetFilters ",
	}, Drop: [2]string{"4.44.3", "4.44.4"}})
}
