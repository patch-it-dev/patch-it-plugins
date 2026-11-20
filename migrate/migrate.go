// Package migrate is the old import path of github.com/patch-it-dev/patch-it-plugins/core/migrate.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/migrate
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package migrate

import moved "github.com/patch-it-dev/patch-it-plugins/core/migrate"

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/migrate.
var (
	Fresh = moved.Fresh
	ToV2  = moved.ToV2
)
