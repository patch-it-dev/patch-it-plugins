// Package redis is what patch-it reads for Redis: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package redis

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/catalog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "redis", Name: "Redis", Category: plugin.CategoryDatabases, Homepage: "https://redis.io",
		Sources: []plugin.Source{
			{
				ID: "commands", Layer: plugin.LayerRuntime, Watches: "Commands removed entirely between two Redis releases, from the command definitions in the server's own source (src/commands/) at each release tag",
				URLs:   []string{"https://github.com/redis/redis/tree/unstable/src/commands", "https://github.com/redis/redis/releases"},
				Plugin: Commands,
			},
			changelog.SDK("ioredis", Ioredis, "npm:ioredis"),
			apisurface.PyPI(RedisPySurface, plugin.LayerSDK, "https://github.com/redis/redis-py"),
			// redis on npm re-exports @redis/client and its siblings wholesale; the
			// client's declarations are where its API is.
			apisurface.NPM(NodeRedisSurface, "https://github.com/redis/node-redis", "npm:redis"),
		},
	})
}

// Ioredis is ioredis' semantic-release CHANGELOG.md: "## [X.Y.Z](...)"
// with "### BREAKING CHANGES"; tagged vX.Y.Z.
var Ioredis = changelog.File{API: "redis/ioredis", Package: "ioredis", Ecosystem: "npm",
	Repo: "redis/ioredis", TagPrefix: "v", Path: "CHANGELOG.md"}

// Commands is redis/redis's src/commands/ (checked 2026-11-20: 460 files at
// 8.10.0, every final release a GitHub release tagged X.Y.Z). The directory
// exists from 7.0.0; an older release has nothing to read.
var Commands = catalog.Commands{
	API: "redis/commands", Ecosystem: "redis", Package: "redis",
	Repo: "redis/redis", Dir: "src/commands",
	Release: regexp.MustCompile(`^(\d+\.\d+\.\d+)$`),
}

// The packages whose public API the apisurface sources read.
var (
	NodeRedisSurface = apisurface.NPMPackage("redis/node-redis-surface", "@redis/client")
	RedisPySurface   = apisurface.Package{ID: "redis/redis-py-surface", Name: "redis", Ecosystem: "pypi", Module: "redis"}
)
