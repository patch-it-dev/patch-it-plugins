// Package gitlab is what patch-it reads for GitLab: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package gitlab

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "gitlab", Name: "GitLab", Category: plugin.CategoryDevTools, Homepage: "https://about.gitlab.com",
		Sources: []plugin.Source{
			// No Hints: GitLab's deprecations are prose per feature, not a changelog
			// keyed by API version, so there's nothing for hint.Source to look up.
			{
				ID: "rest-api", Layer: plugin.LayerAPI,
				Watches: "REST API changes between releases, request side only (what callers send)",
				URLs:    []string{"https://gitlab.com/gitlab-org/gitlab/-/tree/master/doc/api/openapi"},
				Plugin:  openapi.Plugin{Adapter: GitLab{}, Keep: openapi.RequestSide},
			},
		},
	})
}
