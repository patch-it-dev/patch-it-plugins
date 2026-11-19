package openapi

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/diff"
)

// vercelDiff is how Vercel's spec says what it says:
//
//   - Most deprecations are in the summary, not the flag. Of the 22 operations
//     deprecated between 2024-11 and 2026-11 in vercel/sdk's spec history, 16
//     said so only in words: "Check the price for a domain (deprecated)",
//     "Deprecated: Use DELETE /v1/drives/:name. ...".
//   - Renaming a path parameter isn't a change for callers, and Vercel does it:
//     DELETE /v2/sandboxes/drives/{name} became .../{nameOrId} in v1.28.65.
var vercelDiff = diff.Options{PathTemplates: true, DeprecatedSummary: vercelDeprecatedSummary}

// vercelDeprecatedWords are the two ways every word-only deprecation in the
// history is written: a summary that starts "Deprecated:" or ends
// "(deprecated)". Not "deprecat" anywhere, nor a bare leading "Deprecated",
// which would match an operation about deprecated things ("Deprecated
// runtimes for a project").
var vercelDeprecatedWords = regexp.MustCompile(`(?i)^deprecated:|\(deprecated\)\s*$`)

func vercelDeprecatedSummary(summary string) bool {
	return vercelDeprecatedWords.MatchString(summary)
}

// vercelKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. Everything else is dropped, because in
// vercel/sdk's spec history (207 releases, 2024-11 to 2026-11) it was mostly
// not a change to the API:
//
//   - 108 operations left the spec, and only 5 had been marked deprecated: the
//     four Legacy Domains endpoints, which Vercel did shut down on 2025-11-09
//     (they now answer "This endpoint has been sunsetted"), and the drives
//     rename PathTemplates already pairs up. Of the other 103, 16 came back in
//     a later release, and the edge-config ones were the product's rename to
//     Global Config.
//   - The response side reported 67,048 removed properties, about a thousand a
//     release, from the generated schemas reshuffling.
//   - The request side reported 478 breaking property changes, most of them
//     union members compared by position as their order moved; outside unions
//     there were still 142, and 27 of those reverted later.
//
// Additions are dropped too: nobody has to fix code for a new endpoint, and a
// re-added one isn't new.
func vercelKeep(c diff.Change) bool {
	switch c.Kind {
	case diff.OperationDeprecated:
		return true
	case diff.OperationRemoved:
		op, _ := c.Node.(map[string]any)
		return vercelDiff.OperationDeprecated(op)
	default:
		return false
	}
}
