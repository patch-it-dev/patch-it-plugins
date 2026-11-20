// Package kittest is the old import path of github.com/patch-it-dev/patch-it-plugins/core/kit/kittest.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/kit/kittest
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package kittest

import moved "github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/core/kit/kittest.
type (
	Archive = moved.Archive
	Mem     = moved.Mem
	Pages   = moved.Pages
	Sink    = moved.Sink
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/kit/kittest.
var (
	CheckSink     = moved.CheckSink
	CompareGolden = moved.CompareGolden
	Env           = moved.Env
	Golden        = moved.Golden
	Ingest        = moved.Ingest
	IngestWith    = moved.IngestWith
	NewMem        = moved.NewMem
	Read          = moved.Read
	Registered    = moved.Registered
	Replay        = moved.Replay
	Run           = moved.Run
	Stored        = moved.Stored
	Clock         = moved.Clock
)
