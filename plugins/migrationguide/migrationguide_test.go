package migrationguide

import (
	"strings"
	"testing"
)

func TestAMigrationGuideStoresOneEventPerRemovedParameter(t *testing.T) {
	// Only the v1.0.0 tag is saved: fetching anything else fails the test, and
	// kittest's second extract of the pair must store nothing new.
	var params []string
	for _, e := range anthropicV1(t, anthropicGuideV1) {
		params = append(params, e.Subject.Parameter)
	}
	if strings.Join(params, ",") != "temperature,top_p,top_k" {
		t.Errorf("stored %v", params)
	}
}
