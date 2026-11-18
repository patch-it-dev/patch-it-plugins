// Package all imports every plugin, so each registers itself: a program that
// runs plugins imports this one package. A new plugin package is added here
// (TestEveryPluginIsImported fails until it is).
package all

import (
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/changelog"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/feed"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/openapi"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/removedmembers"
	_ "github.com/patch-it-dev/patch-it-plugins/plugins/retirements"
)
