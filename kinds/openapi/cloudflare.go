package openapi

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
)

// cloudflareDiff is how Cloudflare's spec says what it says:
//
//   - Most deprecations use the `deprecated` flag (329 operations in the spec
//     of 2026-11-20), and a few only the summary, written as Vercel's are: "List
//     datasets (deprecated)", "Deprecated: Retrieve legacy embed code HTML"
//     (11 of them). Other summaries and descriptions that say "deprecated"
//     are about a parameter or a model, not the operation, so the
//     description isn't read.
//   - Path parameters are renamed without the URL changing:
//     /zones/{zone_identifier}/analytics/dashboard became
//     /zones/{zone_id}/analytics/dashboard on 2026-10-30. Matching by path
//     template keeps that from reading as a removal.
var cloudflareDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedSummary: cloudflareDeprecatedSummary}

// cloudflareDeprecatedWords: a summary that starts "Deprecated:" or ends
// "(deprecated)". Not "deprecat" anywhere ("Omitting `datasetId` is
// deprecated" is about a parameter).
var cloudflareDeprecatedWords = regexp.MustCompile(`(?i)^deprecated:|\(deprecated\)\s*$`)

func cloudflareDeprecatedSummary(summary string) bool {
	return cloudflareDeprecatedWords.MatchString(summary)
}

// cloudflareKeep keeps operation deprecations, and removals of operations the
// old spec had marked deprecated. Everything else is dropped: a bot commits
// the 30 MB spec several times a day, and an operation can leave it without
// leaving the API or without notice (between 2026-10-29 and 2026-10-30 POST
// /accounts/move and four GET /tenants/{tenant_id} operations left it, never
// marked deprecated: the drop test), while property changes in the
// generated schemas are mostly the generator's.
var cloudflareKeep = deprecationsOnly(cloudflareDiff)
