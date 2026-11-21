// Package newrelic is what patch-it reads for New Relic: every source, registered
// below with its metadata, each read by its kind (kinds/<kind>). README.md
// says why each source and what it reads.
package newrelic

import (
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide"
	"github.com/patch-it-dev/patch-it-plugins/kinds/migrationguide/members"
)

func init() {
	plugin.RegisterProvider(plugin.Provider{
		ID: "newrelic", Name: "New Relic", Category: plugin.CategoryMonitoring, Homepage: "https://newrelic.com",
		Sources: []plugin.Source{
			{
				ID: "python-agent", Layer: plugin.LayerSDK,
				Watches:  "Members a new major of the New Relic Python agent removes, from the major's release notes (Removed features)",
				URLs:     []string{"https://docs.newrelic.com/docs/release-notes/agent-release-notes/python-release-notes/", "https://pypi.org/project/newrelic/"},
				Packages: []string{"pypi:newrelic"},
				Plugin:   migrationguide.Plugin{Guide: PythonAgent},
			},
		},
	})
}

// PythonAgent is New Relic's release note for each major of the Python agent
// (the `newrelic` package on PyPI), in newrelic/docs-website as
// python-agent-<major>0000.mdx (python-agent-110000.mdx for 11.0.0; 9.0.0
// to 13.0.0 are there, checked 2026-11-20). That repo has no tags, so the
// newest commit that changed the note stands in for one (Untagged). Its
// "Removed features" section's first table, | Legacy API | Replacement API |:
// "The following APIs have been removed". A note without that section has
// nothing to read. 11.0.0's, at commit f0b4f817 (checked 2026-11-19), is
// the one described below.
//
// 11.0.0 is yanked on PyPI ("Issues with server-side configuration");
// 11.0.1 has the same public API (newrelic/agent.py and newrelic/api/log.py
// are byte-identical at v11.0.0 and v11.0.1). The version stays 11.0.0, the
// release that made the change: Releases keeps yanked releases.
//
// Names are kept as code writes them after `import newrelic`: the table
// writes them bare, and Paths gives each its module. Every row was checked
// against the agent's code (newrelic/newrelic-python-agent, Apache-2.0) at
// v10.17.0, the release before, and v11.0.0. The six functions were
// re-exported by newrelic/agent.py, the documented way to call the agent's
// API (`import newrelic.agent`), and are gone from it and from their
// newrelic.api modules in 11.0.0. NewRelicLogHandler was only ever
// newrelic.api.log's, never re-exported by newrelic.agent.
//
// Rename or removal: the client applies a renamed_symbol as a rename, so only
// a drop-in replacement is one (DropIns). Its 8 rows:
//
//   - `add_custom_parameter` | [`add_custom_attribute`](…): renamed_symbol
//     agent.add_custom_parameter -> agent.add_custom_attribute. At v10.17.0
//     add_custom_parameter(key, value) only warns and returns
//     add_custom_attribute(key, value), which newrelic.agent exports too.
//   - `add_custom_parameters` | [`add_custom_attributes`](…): renamed_symbol
//     agent.add_custom_parameters -> agent.add_custom_attributes, the same:
//     add_custom_parameters(items) returns add_custom_attributes(items).
//   - `accept_distributed_trace_payload` | [`accept_distributed_trace_headers`](…):
//     removed_symbol. It takes New Relic's own trace payload (what
//     create_distributed_trace_payload made); the replacement takes the
//     request's headers (traceparent, tracestate, newrelic). Same argument
//     positions, different data.
//   - `create_distributed_trace_payload` | [`insert_distributed_trace_headers`](…):
//     removed_symbol. It takes nothing and returns the payload; the
//     replacement fills a list of headers passed in.
//   - `get_browser_timing_footer` | The functionality of this has been wrapped
//     into [`get_browser_timing_header()`](…): removed_symbol, prose. At
//     v10.17.0 the footer already returns "" (and warns); renaming it to the
//     header would put the header in a page twice.
//   - `record_exception` | [`notice_error()`](…): removed_symbol.
//     record_exception(exc, value, tb, params, ignore_errors, application)
//     maps to notice_error(error=(exc, value, tb), attributes=params,
//     ignore=ignore_errors, application=application): different arguments.
//   - `NewRelicLogHandler` | `NewRelicLogForwardingHandler`: removed_symbol
//     api.log.NewRelicLogHandler. The old handler takes a license key, a host
//     and proxy settings and sends logs itself; the new one takes none of
//     them and hands records to the agent's log forwarding.
//   - `lambda_handler` & `LambdaHandlerWrapper` | These APIs have moved to the
//     [`newrelic-lambda`](…) package: dropped (logged), as "&" isn't one of
//     the list separators the reader takes. Both were in newrelic.agent at
//     v10.17.0 and are gone at v11.0.0 (newrelic/api/lambda_handler.py with
//     them), but the replacement is another package, which a symbol event
//     can't say.
//
// 7 events. The section's second table, | Legacy Setting | Replacement
// Setting |, lists configuration settings (newrelic.ini keys), not members
// of the package: only the section's first table is read, so it never is.
//
// Paths and DropIns were checked against 11.0.0's table. A later major's
// rows with one replacement are removed_symbol unless listed in DropIns, so
// a new note is read conservatively.
var PythonAgent = members.Guide{
	API:      "newrelic/python-agent",
	Package:  "newrelic",
	Module:   "newrelic",
	Untagged: true,
	Repo:     "newrelic/docs-website",
	Path:     "src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-{major}0000.mdx",
	Section:  "Removed features",
	Format:   members.FormatMarkdown,
	Header:   "Legacy API",
	Doc:      "release notes",
	// The cells link to docs.newrelic.com by path ("/docs/agents/…").
	LinkBase: "https://docs.newrelic.com",
	Paths: map[string]string{
		"accept_distributed_trace_payload": "agent.accept_distributed_trace_payload",
		"add_custom_parameter":             "agent.add_custom_parameter",
		"add_custom_parameters":            "agent.add_custom_parameters",
		"create_distributed_trace_payload": "agent.create_distributed_trace_payload",
		"get_browser_timing_footer":        "agent.get_browser_timing_footer",
		"record_exception":                 "agent.record_exception",
		"NewRelicLogHandler":               "api.log.NewRelicLogHandler",
		// The two drop-in replacements, in newrelic.agent at v10.17.0.
		"add_custom_attribute":  "agent.add_custom_attribute",
		"add_custom_attributes": "agent.add_custom_attributes",
	},
	DropIns: []string{"agent.add_custom_parameter", "agent.add_custom_parameters"},
}
