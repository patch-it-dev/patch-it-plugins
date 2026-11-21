// Package all imports every plugin, so each registers itself: a program that
// runs plugins imports this one package. A new plugin package is added here
// (TestEveryPluginIsImported fails until it is).
package all

import (
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/apisurface"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/catalog"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/changelog"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	_ "github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)
