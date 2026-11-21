package changelog

import "github.com/patch-it-dev/patch-it-plugins/core/plugin"

func init() {
	const page = "providers/redis-ioredis.md"
	plugin.RegisterProvider(plugin.Provider{
		ID: "redis", Name: "Redis", Category: plugin.CategoryDatabases, Homepage: "https://redis.io",
		Sources: []plugin.Source{sdk(page, "ioredis", RedisIoredis, "npm:ioredis")},
	})
}

// RedisIoredis is ioredis' semantic-release CHANGELOG.md: "## [X.Y.Z](...)"
// with "### BREAKING CHANGES"; tagged vX.Y.Z.
var RedisIoredis = File{API: "redis/ioredis", Package: "ioredis", Ecosystem: "npm",
	Repo: "redis/ioredis", TagPrefix: "v", Path: "CHANGELOG.md"}
