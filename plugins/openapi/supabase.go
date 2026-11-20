package openapi

import "github.com/patch-it-dev/patch-it-plugins/core/diff"

// supabaseDiff is how the Supabase Management API spec says what it says:
//
//   - Deprecations use the `deprecated` flag, and the description sometimes
//     says why ("This endpoint is deprecated - use the deploy endpoint."), so
//     the event quotes the whole operation, not the bare `true`.
//   - Path parameters get renamed, and that isn't a change for callers:
//     /v1/branches/{branch_id}/merge became .../{branch_id_or_ref}/merge on
//     2025-09-11.
var supabaseDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// supabaseKeep keeps operation deprecations, and removals of operations the
// old spec had marked deprecated. In the spec's history (90 commits,
// 2024-05-31 to 2026-11-18) that's 2 events: POST /v1/projects/{ref}/functions
// deprecated on 2025-02-19, GET /v1/projects/{ref}/analytics/endpoints/logs.all
// on 2026-06-29. Everything else is dropped:
//
//   - 22 operations left the spec, none of them deprecated first: 14
//     analytics warehouse operations on 2024-07-15, 7 path-parameter renames
//     (which PathTemplates pairs up anyway), and one that came back 12
//     days later. Without credentials every route answers 401, so whether a
//     removed one still works can't be checked from outside.
//   - Request and response property changes, new operations: not changes a
//     caller has to fix, or not ones the spec's history shows reliably.
var supabaseKeep = deprecationsOnly(supabaseDiff)
