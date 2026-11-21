package changelog

import "testing"

func TestIoredisSDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{RedisIoredis, "5.0.0", "4.31.0", "redis-ioredis-5.0.0-CHANGELOG.md", "", "5.0.0-beta.4"},
	})
}
