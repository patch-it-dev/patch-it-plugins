// Package salesforce is what patch-it reads for Salesforce: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package salesforce

import (
	"github.com/patch-it-dev/patch-it-plugins/core/kit"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices"
	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/eol"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "salesforce", Name: "Salesforce", Category: plugin.CategoryCRM, Homepage: "https://www.salesforce.com",
		Sources: []plugin.Source{
			{
				ID: "rest-api", Layer: plugin.LayerAPI,
				Watches: "REST API version ranges deprecated and retired, from Salesforce's own API end-of-life policy page",
				URLs:    []string{"https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html"},
				Plugin:  notices.Plugin{Page: RESTAPI, Archiver: kit.Wayback{}},
			},
		},
	})
}

// RESTAPI is Salesforce's REST API end-of-life policy page (checked
// 2026-11-20):
// a table of version ranges, each "Supported.", already retired ("As of
// Summer '25, these versions are retired and unavailable."), or scheduled
// ("Deprecated and unsupported from Summer '27. Retired from Summer '28.").
// Salesforce gives no exact date, only the named release ("Summer '27"); the
// policy text above the table says the notice is at least a year ahead of
// retirement.
var RESTAPI = eol.Page{
	API: "salesforce/rest-api", Vendor: "Salesforce",
	URL:       "https://developer.salesforce.com/docs/platform/api-rest/guide/api-rest-eol.html",
	Operation: func(v string) string { return "GET /services/data/v" + v + "/" },
}
