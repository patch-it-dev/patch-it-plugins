package mongodb

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// pymongo 4.16.0's command cursors no longer take explicit_session.
func TestPymongoSurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: PymongoSurface, Golden: [2]string{"4.15.5", "4.16.0"}, Want: []string{
		"removed_parameter asynchronous.command_cursor.AsyncCommandCursor explicit_session",
		"removed_parameter asynchronous.command_cursor.AsyncRawBatchCommandCursor explicit_session",
		"removed_parameter command_cursor.CommandCursor,synchronous.command_cursor.CommandCursor explicit_session",
		"removed_parameter command_cursor.RawBatchCommandCursor,synchronous.command_cursor.RawBatchCommandCursor explicit_session",
	}, Drop: [2]string{"4.15.4", "4.15.5"}})
}
