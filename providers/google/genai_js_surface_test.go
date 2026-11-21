package google

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @google/genai 1.0.0 removed LiveMusicSetClientContentParameters.
func TestGenaiJSSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: GenaiJSSurface, Golden: [2]string{"0.15.0", "1.0.0"}, Want: []string{
		"removed_symbol @google/genai/node.LiveMusicSetClientContentParameters ",
		"removed_symbol @google/genai/web.LiveMusicSetClientContentParameters ",
		"removed_symbol LiveMusicSetClientContentParameters ",
	}, Drop: [2]string{"1.0.0", "1.1.0"}})
}
