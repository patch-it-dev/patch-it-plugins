package redis

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// redis 8.1.0 removed OSSMaintNotificationsHandler.get_handler_for_connection.
func TestRedisPySurface(t *testing.T) {
	apisurfacetest.PyPI(t, apisurfacetest.Case{Package: RedisPySurface, Golden: [2]string{"8.0.1", "8.1.0"}, Want: []string{"removed_symbol maint_notifications.OSSMaintNotificationsHandler.get_handler_for_connection "}, Drop: [2]string{"7.4.1", "8.0.0"}})
}
