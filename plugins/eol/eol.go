// Package eol is the old import path of the notices kind's end-of-life
// page reader, kept so code that imports it still builds.
//
// Deprecated: use plugins/notices (Plugin) and plugins/notices/eol (the
// reader: Page, Salesforce, Findings). This package only forwards to them
// and registers nothing; it goes once nothing imports it.
package eol

import (
	"github.com/patch-it-dev/patch-it-plugins/plugins/notices"
	moved "github.com/patch-it-dev/patch-it-plugins/plugins/notices/eol"
)

type (
	// Plugin is notices.Plugin: set Page and Archiver, as before.
	Plugin  = notices.Plugin
	Page    = moved.Page
	Row     = moved.Row
	Finding = moved.Finding
)

var (
	Salesforce = moved.Salesforce
	Parse      = moved.Parse
	Findings   = moved.Findings
)
