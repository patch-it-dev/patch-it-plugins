package openapi

import "github.com/patch-it-dev/patch-it-plugins/core/diff"

// atlasDiff is how the Atlas Administration API's spec says what it says:
//
//   - Deprecations use the `deprecated` flag (53 operations on 2026-11-19).
//     Fourteen operations mention "deprecated" in their description without
//     it, each about the deprecated invitation flow ("This resource cannot
//     be used to view details about users invited via the deprecated
//     Invite One MongoDB Cloud User to Join One Project endpoint"), not the
//     operation itself, so the description isn't read.
//   - Matching by path template keeps a renamed path parameter from reading
//     as a removal.
var atlasDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// atlasKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated (the tenant backup operations, deprecated and
// then gone between 2025-10 and 2026-11). Everything else is dropped:
// operations that leave the spec unmarked (GET /rest/unauth/version on
// 2024-08-07: the drop test), and schema and property changes, which the
// API's own changelog (mongodb/atlas-admin-api, kind changelog) reports per
// resource version.
var atlasKeep = deprecationsOnly(atlasDiff)
