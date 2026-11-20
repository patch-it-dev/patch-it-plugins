package openapi

import "github.com/patch-it-dev/patch-it-plugins/diff"

// convexDiff is how the Convex Deployment API spec says what it says:
//
//   - Deprecations use the `deprecated` flag, and the description says what
//     to use instead ("Deprecated: use `GET /integrations` instead."), so the
//     event quotes the whole operation, not the bare `true`. Convex's docs
//     build drops flagged operations, "since they stay served only for
//     existing callers" (npm-packages/docs/scripts/prepare-openapi-specs.js).
//   - PathTemplates, as for every spec here: a renamed path parameter isn't a
//     change for callers. None has been renamed in this spec yet.
var convexDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// convexKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. In the spec's history (68 commits, 2025-08-19
// to 2026-11-16) that's 6 events, all on 2026-11-05: the six log stream
// operations, deprecated for the /integrations ones added on 2026-10-30.
// No operation has left the spec. Everything else is dropped:
//
//   - Request and response property changes: 18 property removals, 16 type
//     changes and 10 schema removals over the history, most of them in the
//     data sync, usage-limit and S3 export APIs. On 2026-07-14 the optional
//     `name` left the usage-limit request and response, while the server
//     didn't refuse a request that still sent it: see the drop test.
//   - New operations: nobody has to fix code for a new endpoint.
var convexKeep = deprecationsOnly(convexDiff)
