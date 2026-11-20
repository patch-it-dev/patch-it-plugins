// Package retire is the old import path of github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package retire

import moved "github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire.
type (
	Page       = moved.Page
	Retirement = moved.Retirement
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/plugins/notices/retire.
var (
	Events    = moved.Events
	First     = moved.First
	Parse     = moved.Parse
	Anthropic = moved.Anthropic
	OpenAI    = moved.OpenAI
)
