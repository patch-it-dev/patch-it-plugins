package openapi

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
)

// planetscaleDiff is how the PlanetScale API spec says what it says:
//
//   - Deprecations are only in the description's first words, "Deprecated.
//     Use GET /organizations/:organization/databases/:database/branches/:branch/change-requests
//     instead.": the spec has never set the `deprecated` flag (none in any
//     of the 32 versions of schemas/openapi.yaml in
//     planetscale/terraform-provider-planetscale, 2025-11-11 to 2026-11-09,
//     nor in the live https://planetscale.com/docs/openapi.yaml on
//     2026-11-19). If it ever does, the operation is quoted whole, so the
//     words come with it.
//   - Path parameters get renamed, and that isn't a change for callers.
var planetscaleDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true, DeprecatedDescription: planetscaleDeprecatedDescription}

// planetscaleDeprecatedWords is the way every deprecation in the history is
// written: the description starts "Deprecated." (the others start with what
// the operation does, or with an empty line, then "### Authorization"). Only
// at the start,
// so not a description that mentions deprecation further on; and with the
// period or a colon, so not "Deprecated, storage shrinking is part of
// storage autoscaling.", which the spec writes on properties.
var planetscaleDeprecatedWords = regexp.MustCompile(`(?i)^\s*deprecated[.:]`)

func planetscaleDeprecatedDescription(description string) bool {
	return planetscaleDeprecatedWords.MatchString(description)
}

// planetscaleKeep keeps operation deprecations, and removals of operations
// the old spec had marked deprecated. Over the 31 consecutive pairs of the
// spec's history that's 5 events, all on 2026-11-09 (fc13e09d): GET and
// DELETE .../bouncers/{bouncer}/resizes, GET .../neki-changes, GET and
// DELETE .../neki-changes/{id}. Everything else is dropped:
//
//   - 41 operations left the spec, none deprecated first. 6 of them, on
//     2026-10-03, were the read-only replica operations renamed to
//     dedicated read replicas, and planetscale/planetscale-go still sends
//     the old paths ("the deprecated read-only-replicas path", main on
//     2026-11-19): left the spec isn't stopped working.
//   - Property changes and new operations, as for vercel: 226 type changes
//     on 2025-12-06 alone, when the spec switched whole numbers to
//     `integer`.
var planetscaleKeep = deprecationsOnly(planetscaleDiff)
