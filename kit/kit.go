// Package kit is the old import path of github.com/patch-it-dev/patch-it-plugins/core/kit.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/kit
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package kit

import (
	"context"

	moved "github.com/patch-it-dev/patch-it-plugins/core/kit"
)

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/core/kit.
type (
	Archiver = moved.Archiver
	Wayback  = moved.Wayback
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/kit.
var (
	Unchanged = moved.Unchanged
)

// Cite forwards to core/kit.Cite.
//
// Deprecated: use github.com/patch-it-dev/patch-it-plugins/core/kit.Cite.
func Cite[T any](ctx context.Context, a Archiver, pageURL string, read func([]byte) (T, error), want T) (snapURL string, snap []byte, got T, err error) {
	return moved.Cite(ctx, a, pageURL, read, want)
}
