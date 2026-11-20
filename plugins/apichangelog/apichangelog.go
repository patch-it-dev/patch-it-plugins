// Package apichangelog is the old import path of the changelog kind's JSON
// parser, kept so code that imports it still builds.
//
// Deprecated: use plugins/changelog: Plugin{Log: changelog.MongoDBAtlas}.
// This package only forwards there and registers nothing; it goes once
// nothing imports it.
package apichangelog

import "github.com/patch-it-dev/patch-it-plugins/plugins/changelog"

type (
	// Plugin is changelog.Plugin: set Log, as before.
	Plugin  = changelog.Plugin
	Log     = changelog.Log
	Finding = changelog.LogFinding
)

var MongoDBAtlas = changelog.MongoDBAtlas
