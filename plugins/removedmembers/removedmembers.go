// Package removedmembers is the old import path of the migration-guide
// kind's removed-members reader, kept so code that imports it still builds.
//
// Deprecated: use plugins/migrationguide (the kind: Plugin) and
// plugins/migrationguide/members (the reader: Guide, NumPy2, Parse). This
// package only forwards to them and registers nothing; it goes once nothing
// imports it.
package removedmembers

import (
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/plugins/migrationguide/members"
)

type (
	// Plugin is migrationguide.Plugin: its Guide is now any reader, a
	// members.Guide among them.
	Plugin  = migrationguide.Plugin
	Guide   = members.Guide
	Format  = members.Format
	Removal = members.Removal
)

const (
	FormatRST      = members.FormatRST
	FormatMarkdown = members.FormatMarkdown
)

var (
	NumPy2     = members.NumPy2
	PostHog8   = members.PostHog8
	NewRelic11 = members.NewRelic11
)

// Parse is members.Parse.
func Parse(page []byte, g Guide) ([]Removal, error) { return members.Parse(page, g) }
