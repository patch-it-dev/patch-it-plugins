package redis

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/apisurfacetest"
)

// @redis/client 5.0.0 removed commandOptions and RedisFlushModes.
func TestNodeRedisSurface(t *testing.T) {
	apisurfacetest.NPM(t, apisurfacetest.Case{Package: NodeRedisSurface, Golden: [2]string{"1.6.1", "5.0.0"}, Want: []string{
		"removed_symbol RedisFlushModes ",
		"removed_symbol commandOptions ",
	}, Drop: [2]string{"6.3.0", "6.3.1"}})
}
