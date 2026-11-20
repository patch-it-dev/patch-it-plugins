// Package feed is the old import path of the notices kind's RSS and Atom
// feed reader, kept so code that imports it still builds.
//
// Deprecated: use plugins/notices (notices.Plugin{Page: feed.GitHubActions})
// and plugins/notices/feed (the reader). This package only forwards to them
// and registers nothing; it goes once nothing imports it.
package feed

import (
	"context"

	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices"
	moved "github.com/patch-it-dev/patch-it-plugins/plugins/notices/feed"
)

type (
	Feed    = moved.Feed
	Reader  = moved.Reader
	Finding = moved.Finding
	Entry   = moved.Entry
	Section = moved.Section
)

var (
	GitHubActions  = moved.GitHubActions
	VertexAIModels = moved.VertexAIModels
	CloudflareAPI  = moved.CloudflareAPI
	Parse          = moved.Parse
	Text           = moved.Text
	Lines          = moved.Lines
	SplitBy        = moved.SplitBy
	Date           = moved.Date
	Codes          = moved.Codes
)

// Plugin reads one feed: notices.Plugin{Page: Feed, Archiver: Archiver}.
type Plugin struct {
	Feed     Feed
	Archiver kit.Archiver
}

func (p Plugin) notices() notices.Plugin { return notices.Plugin{Page: p.Feed, Archiver: p.Archiver} }

func (p Plugin) Name() string { return p.notices().Name() }

// Run is notices.Plugin's Run.
func (p Plugin) Run(ctx context.Context, env plugin.Env) (plugin.Result, error) {
	return p.notices().Run(ctx, env)
}
