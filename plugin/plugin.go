// Package plugin is the old import path of github.com/patch-it-dev/patch-it-plugins/core/plugin.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/plugin
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package plugin

import moved "github.com/patch-it-dev/patch-it-plugins/core/plugin"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/core/plugin.
type (
	BaselineReader = moved.BaselineReader
	Env            = moved.Env
	Plugin         = moved.Plugin
	Replayer       = moved.Replayer
	Result         = moved.Result
	Spec           = moved.Spec
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/plugin.
var (
	Baseline = moved.Baseline
	Lookup   = moved.Lookup
	Names    = moved.Names
	Register = moved.Register
	SHA256   = moved.SHA256
)
