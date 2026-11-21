package mongodb

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// mongodb 7.0.0 removed CancellationToken, Transaction and more.
func TestNodeSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: NodeSurface, Golden: [2]string{"6.22.0", "7.0.0"}, Want: []string{
		"removed_symbol AsyncDisposable ",
		"removed_symbol CancellationToken ",
		"removed_symbol ChangeStream.streamOptions ",
		"removed_symbol ClientMetadata ",
		"removed_symbol ClientMetadataOptions ",
		"removed_symbol ClientSession.transaction ",
		"removed_symbol CloseOptions ",
		"removed_symbol CursorStreamOptions ",
		"removed_symbol ReadPreference.minWireVersion ",
		"removed_symbol ResumeOptions ",
		"removed_symbol ServerCapabilities ",
		"removed_symbol Transaction ",
	}, Drop: [2]string{"7.0.0", "7.1.0"}})
}
