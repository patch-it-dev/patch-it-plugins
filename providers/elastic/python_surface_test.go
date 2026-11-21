package elastic

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// elasticsearch 9.1.3 made the rethrottle methods' requests_per_second
// required.
func TestPythonSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PythonSurface, Golden: [2]string{"9.1.2", "9.1.3"}, Want: []string{
		"changed_default Elasticsearch.delete_by_query_rethrottle,client.Elasticsearch.delete_by_query_rethrottle requests_per_second",
		"changed_default Elasticsearch.reindex_rethrottle,client.Elasticsearch.reindex_rethrottle requests_per_second",
		"changed_default Elasticsearch.update_by_query_rethrottle,client.Elasticsearch.update_by_query_rethrottle requests_per_second",
	}, Drop: [2]string{"9.1.1", "9.1.2"}})
}
