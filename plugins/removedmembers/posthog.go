package removedmembers

// PostHog8 is the posthog Python SDK's guide to 8.0.0,
// docs/migration-7.x-to-8.0.md at tag posthog-v8.0.0 (commit
// 6edff41425188056e48f6119457dc9bc0669ca99), and its "Fix removed APIs"
// table: "8.x.x removes these options and APIs". The guide isn't in the repo
// at posthog-v7.91.4, the release before, and is unchanged at posthog-v8.0.2.
// Checked 2026-11-19.
//
// The table is Markdown, | Removed | Use instead |. Names are kept as code
// writes them after `import posthog` or a `from posthog.<module> import`:
// `posthog.` is stripped (`posthog.capture_v1` is `capture_v1`), as the
// changelog plugin and NumPy's `np.` do; other prefixes are kept
// (`request.batch_post`). One event per name, as NumPy has one per member. Its
// 7 rows, each checked against the code at posthog-v7.91.4 and posthog-v8.0.0:
//
//   - `capture_mode`, `CaptureMode`, `POSTHOG_CAPTURE_MODE` | Nothing:
//     removed_symbol for capture_mode (the module setting and the client
//     argument) and CaptureMode (the enum, imported from posthog).
//     POSTHOG_CAPTURE_MODE is the environment variable capture_mode read
//     (EnvVars): no code imports it, so no event.
//   - `gzip=True` | `capture_compression`: dropped. It's an argument value, of
//     a call the row doesn't name (Client's gzip argument, per the code); a
//     removed_parameter needs the call, and guessing it isn't reading the
//     guide.
//   - `CaptureV1Error` | `CaptureError`: removed_symbol, with CaptureError
//     in the migration hint, not a renamed_symbol (NewReplacements):
//     CaptureError is new in 8.0.0, so a renamed import fails on 7.x, and the
//     module changes too, from posthog.capture_v1 to posthog, which renaming
//     the name alone wouldn't fix. The row doesn't say so; the guide does a
//     few paragraphs below ("Replace `from posthog.capture_v1 import
//     CaptureV1Error` with `from posthog import CaptureError`. There is no
//     alias.").
//   - `posthog.capture_v1` | `posthog.capture_event` and
//     `posthog.capture_send`: removed_symbol capture_v1 (the module; two
//     replacements, so no rename).
//   - `request.batch_post`, `async_batch_post`, `EVENTS_ENDPOINT`,
//     `AI_EVENTS_ENDPOINT` | Send events through a client: four removed_symbol
//     (advice, not a name).
//   - The `gzip` parameter of `request.post` and `request.flags` | Nothing:
//     dropped, prose. Not a removed_parameter either: both take **kwargs in
//     8.0.0, so gzip= raises nothing and is sent in the request body, as the
//     guide says.
//   - The `backoff` dependency | Your own requirements…: dropped, prose, and a
//     dependency rather than a member.
//
// 8 events. The guide's other tables (endpoints, AI lane arguments, error
// fields, retries, dropped events, options, and "Rename deprecated settings",
// whose old names still work) aren't under the heading, so they're never read.
var PostHog8 = Guide{
	Package: "posthog",
	Module:  "posthog",
	Release: "8.0.0",
	Tag:     "posthog-v8.0.0",
	Repo:    "PostHog/posthog-python",
	Path:    "docs/migration-7.x-to-8.0.md",
	Section: "Fix removed APIs",
	Format:  FormatMarkdown,
	Header:  "Removed",
	EnvVars: []string{"POSTHOG_CAPTURE_MODE"},
	// Every replacement in the table is new in 8.0.0.
	NewReplacements: true,
}
