package openapi

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
)

// sentryDiff is how Sentry's spec says what it says:
//
//   - Deprecations are mostly in the operation's description, not the flag.
//     Over the spec's history (the 1,130 commits that changed it in
//     getsentry/sentry-api-schema, 2020-08-25 to 2026-11-14, then the 19
//     api@ releases to api@2.16.0) the flag deprecates 3 operations and the
//     description 13, those 3 included, never later than the flag: the 10
//     issue and metric alert-rule operations on 2026-01-22, never flagged;
//     POST .../user-feedback/ on 2024-07-31 and GET .../issues/ on
//     2025-07-25, both flagged on 2026-06-16; GET .../stats/, words and flag
//     together on 2022-03-01.
//   - The flag, when it's set, comes with the reason in the description, so
//     a flagged operation is quoted whole.
//   - Path parameters get renamed ({organization_slug} became
//     {organization_id_or_slug} on 2024-05-10), and that isn't a change for
//     callers.
var sentryDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedDescription: sentryDeprecatedDescription}

// sentryDeprecatedWords are the three ways the descriptions say it, over the
// whole history: a "## Deprecated" heading with nothing else on its line, a
// line starting "**Deprecated**:", and "This endpoint is deprecated" in any
// case ("*This endpoint is DEPRECATED."). Also "**Deprecated:**", the colon
// inside the bold: never used for a deprecation yet, but how Sentry writes
// its other notes ("**Experimental:**", 45 times in api@2.16.0). Not a heading
// with more words ("## Deprecated fields"), which would be about part of the
// operation. Not "deprecat" anywhere: that also matches
// three descriptions that aren't deprecations, GET .../events/'s "**Eventsv2
// Deprecation Note**" (about another endpoint; 2022-06-16, and again under
// the 2024-05-10 rename) and GET .../trace-items/attributes/'s "examples, and
// deprecation details" (2026-10-15).
var sentryDeprecatedWords = regexp.MustCompile(`(?im)^\s*#+\s*deprecated[^\p{L}\p{N}\n]*$|^\s*\*\*deprecated(\*\*:|:\*\*)|\bthis endpoint is deprecated\b`)

func sentryDeprecatedDescription(description string) bool {
	return sentryDeprecatedWords.MatchString(description)
}

// sentryKeep keeps operation deprecations only: not even a deprecated
// operation's removal. An operation leaving Sentry's spec means Sentry
// stopped documenting it, and the history doesn't show when, or whether, it
// stopped working.
//
//   - 157 operations left the spec (by path template, with /api/0 in every
//     path: see adapter.Sentry). 11 had been deprecated first: GET
//     /api/0/organizations/{organization_slug}/stats/, gone on 2024-01-22
//     and still routed in getsentry/sentry's src/sentry/api/urls.py on
//     2026-11-19, and the 10 alert-rule operations, gone on 2026-05-08 (the
//     day after Sentry's "Monitors & Alerts — Now Generally Available"),
//     with nothing from Sentry saying they stopped answering then.
//   - Compared as published, 289 more left on 2026-11-13, when /api/0
//     moved into the server URL, GET .../issues/ and POST .../user-feedback/
//     among them, both flagged deprecated.
//
// Property changes and new operations are dropped too, as for vercel.
func sentryKeep(c diff.Change) bool { return c.Kind == diff.OperationDeprecated }
