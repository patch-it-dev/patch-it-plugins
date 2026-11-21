// Package mongodb is what patch-it reads for MongoDB: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package mongodb

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "mongodb", Name: "MongoDB", Category: plugin.CategoryDatabases, Homepage: "https://www.mongodb.com",
		Sources: []plugin.Source{
			// No Hints: the Atlas changelog is its own source (mongodb/atlas-admin-api,
			// kind changelog). Only operation deprecations and removals: see
			// mongodb.go.
			{
				ID: "atlas-admin-spec", Layer: plugin.LayerAPI, Watches: "Atlas Administration API v2 operations deprecated, and deprecated operations removed, day by day in MongoDB's published spec",
				URLs:   []string{"https://github.com/mongodb/openapi/blob/main/openapi/v2.json"},
				Plugin: openapi.Plugin{Adapter: MongoDBAtlas{}, Diff: atlasDiff, Keep: atlasKeep},
			},
			{
				ID: "atlas-admin-api", Layer: plugin.LayerAPI, Watches: "Atlas Administration API versions deprecated and removed at their sunset date, per endpoint, from MongoDB's own API changelog",
				URLs:   []string{"https://github.com/mongodb/openapi/blob/main/changelog/changelog.json"},
				Plugin: changelog.Plugin{Log: AtlasAdminAPI},
			},
			apisurface.PyPI(PymongoSurface, plugin.LayerSDK, "https://github.com/mongodb/mongo-python-driver"),
			apisurface.NPM(NodeSurface, "https://github.com/mongodb/node-mongodb-native"),
		},
	})
}

// AtlasAdminAPI is the Atlas Administration API's changelog,
// changelog/changelog.json in mongodb/openapi (checked 2026-11-19: 188 days
// from 2023-02-01 to 2026-11-18, released by MongoDB's CI, "Release API
// Changelog", a few times a month; the docs page
// https://www.mongodb.com/docs/atlas/reference/api-resources-spec/changelog/
// is built from it). Atlas versions each resource by date (`2023-01-01`),
// chosen per request in the Accept header.
var AtlasAdminAPI = changelog.Log{
	API: "mongodb/atlas-admin-api", Vendor: "MongoDB", Name: "Atlas Administration API",
	Repo: "mongodb/openapi", Path: "changelog/changelog.json",
}

// The packages whose public API the apisurface sources read.
var (
	NodeSurface    = apisurface.NPMPackage("mongodb/node-surface", "mongodb")
	PymongoSurface = apisurface.Package{ID: "mongodb/pymongo-surface", Name: "pymongo", Ecosystem: "pypi", Module: "pymongo"}
)
