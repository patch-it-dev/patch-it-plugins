// Package sdkguide is the old import path of github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package sdkguide

import moved "github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide.
type (
	Guide   = moved.Guide
	Removal = moved.Removal
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/sdkguide.
var (
	Events      = moved.Events
	Parse       = moved.Parse
	AnthropicV1 = moved.AnthropicV1
)
