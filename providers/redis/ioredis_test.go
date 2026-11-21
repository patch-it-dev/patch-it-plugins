package redis

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestIoredisSDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: Ioredis, Release: "5.0.0", Prev: "4.31.0", Saved: "redis-ioredis-5.0.0-CHANGELOG.md", Dropped: "", Drop: "5.0.0-beta.4"},
	})
}
