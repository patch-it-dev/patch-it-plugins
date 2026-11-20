// Package retirements is the old import path of the notices kind for a
// model provider's deprecations page, kept so code that imports it still
// builds.
//
// Deprecated: use plugins/notices: notices.Plugin{Page: retire.OpenAI}. This
// package only forwards there and registers nothing; it goes once nothing
// imports it.
package retirements

import "github.com/patch-it-dev/patch-it-plugins/plugins/notices"

// Plugin is notices.Plugin: set Page and Archiver, as before.
type Plugin = notices.Plugin
