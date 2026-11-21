package openapi

import "github.com/patch-it-dev/patch-it-plugins/core/diff"

// openaiDiff is how OpenAI's spec says what it says:
//
//   - Deprecations use the `deprecated` flag: in the spec of 2026-11-21, 17
//     operations carry it and none is called deprecated only in its summary
//     or description (one description mentions deprecated models, not the
//     operation). A flagged operation's description says why and when ("The
//     endpoint and model are deprecated and scheduled to shut down on
//     February 26, 2027"), so the event quotes the whole operation.
//   - Path parameters could be renamed without the URL changing; matching by
//     path template keeps a rename from reading as a removal.
var openaiDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// openaiKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. Everything else is dropped: the spec is
// committed several times a day, and most of what changes is the spec's own
// shape, not the API (enum lists rewritten as const, schemas regenerated),
// or work in progress pulled the day after it was added (the session
// spending-limit schemas, added on 2026-11-17 and gone on 2026-11-18: the
// drop test). New operations aren't something to fix either.
var openaiKeep = deprecationsOnly(openaiDiff)
