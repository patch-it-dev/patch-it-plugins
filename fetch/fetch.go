// Package fetch is the old import path of github.com/patch-it-dev/patch-it-plugins/core/fetch.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/fetch
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package fetch

import moved "github.com/patch-it-dev/patch-it-plugins/core/fetch"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/core/fetch.
type (
	Cache       = moved.Cache
	StatusError = moved.StatusError
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/fetch.
var (
	Get = moved.Get
)
