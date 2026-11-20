package openapi

import "github.com/patch-it-dev/patch-it-plugins/core/diff"

// neonDiff is how the Neon API spec says what it says:
//
//   - Deprecations use the `deprecated` flag, and only the flag: in the 23
//     commits of neondatabase/neon-pkgs's copy (2026-06-24 to 2026-11-17) no
//     operation is described as deprecated without it. (Two unflagged
//     operations deprecate a parameter in words; those aren't operation
//     deprecations.)
//     A flagged operation's description says why or when ("Deprecated: to
//     test a branch's already-saved configuration, use
//     `sendNeonAuthEmailProviderTest`", "Removal scheduled for March 1,
//     2026"), and every one carries an `x-sunset` date, so the event quotes
//     the whole operation, not the bare `true`.
//   - Path parameters could be renamed without the URL changing; none was in
//     this history, but matching by path template costs nothing here and
//     keeps a rename from reading as a removal.
var neonDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// neonKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. In the spec's history that's 1 event: POST
// /projects/{project_id}/branches/{branch_id}/auth/send_test_email,
// deprecated on 2026-08-23. Everything else is dropped:
//
//   - One operation left the spec, not deprecated first: POST
//     /projects/{project_id}/branches/{branch_id}/recover on 2026-08-09.
//     neon-pkgs's own PR for it (#408) says it "was not removed from the API.
//     It was removed from the published spec": it still routes.
//   - Request and response property changes, new operations: not changes a
//     caller has to fix, or not ones the spec's history shows reliably.
var neonKeep = deprecationsOnly(neonDiff)
