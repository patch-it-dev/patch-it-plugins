package changelog

import (
	"strings"
	"testing"
)

func titles(t *testing.T, file, release string) (string, map[string][]string) {
	t.Helper()
	heading, ss, err := ReleaseNotes([]byte(file), release)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, s := range ss {
		for _, b := range s.Bullets {
			out[s.Title] = append(out[s.Title], b.Text)
		}
	}
	return heading, out
}

func TestKeepAChangelogAndItsVariants(t *testing.T) {
	md := "# Changelog\n\n## [Unreleased]\n\n### Removed\n- `Later`\n\n## [2.0.0] - 2026-01-01\n\n### Removed\n\n- `A.b`\n  continued\n- `C`\n\n  a second paragraph\n\n### Added\n- `D`\n\n## [1.0.0]\n### Removed\n- `Old`\n"
	h, got := titles(t, md, "2.0.0")
	if h != "## [2.0.0] - 2026-01-01" || strings.Join(got["Removed"], "|") != "`A.b` continued|`C`" || len(got["Added"]) != 1 {
		t.Errorf("keep a changelog: %q %v", h, got)
	}
	// httpx's variant: "## 0.28.0 (date)" and bold section titles.
	_, got = titles(t, "## 0.28.0 (28th November, 2024)\n\n**Deprecations**:\n\n* The `cert` argument is now deprecated.\n\n## 0.27.2\n", "0.28.0")
	if len(got["Deprecations"]) != 1 {
		t.Errorf("httpx variant: %v", got)
	}
	// HISTORY.rst: "v2.0.0 (date)" over "+++", sections over "---".
	rst := "History\n-------\n\nv2.0.0 (22 March 2024)\n++++++++++++++++++++++\n\nRemoved\n~~~~~~~\n\n- ``OAuth2Session.old``\n  continued\n\nv1.4.0 (27 Feb 2024)\n++++++++++++++++++++\n\n- other\n"
	h, got = titles(t, rst, "2.0.0")
	if h != "v2.0.0 (22 March 2024)" || strings.Join(got["Removed"], "|") != "``OAuth2Session.old`` continued" {
		t.Errorf("rst: %q %v", h, got)
	}
	if _, _, err := ReleaseNotes([]byte(md), "3.0.0"); err == nil {
		t.Error("a release that isn't in the file: want an error, never zero events")
	}
}

func TestOnlyAPlainSentenceNamesItsSymbols(t *testing.T) {
	for _, c := range []struct {
		kind Kind
		text string
		want string // "ct change symbols>replacement", "" = not breaking/deprecation, "drop"
	}{
		{KindBreaking, "Removed `App.dark` reactive attribute https://x.test/pull/1", "breaking removed_symbol App.dark"},
		{KindRemoved, "`Placeholder.reset_color_cycle`", "breaking removed_symbol Placeholder.reset_color_cycle"},
		{KindBreaking, "`Placeholder.reset_color_cycle`", "drop"}, // names alone count only under Removed
		{KindRemoved, "Deprecated: `X`", "drop"},                  // a change word is not an area
		{KindRemoved, "Property `Widget.focusable_children` https://x.test/1", "breaking removed_symbol Widget.focusable_children"},
		{KindBreaking, "Removed `Widget.reset_focus` (now called `Widget.blur`) https://x.test/1", "breaking renamed_symbol Widget.reset_focus>Widget.blur"},
		{KindBreaking, "Removed the ``_prepare_conn`` method from ``HTTPConnectionPool``. Previously…", "breaking removed_symbol HTTPConnectionPool._prepare_conn"},
		{KindBreaking, "Removed ``pkg.a`` and ``pkg.b()`` (`#1 <https://x.test/1>`__).", "breaking removed_symbol a,b"},
		{KindNone, "Breaking change: `SyntaxAwareDocument.language_name` property removed https://x.test/1", "breaking removed_symbol SyntaxAwareDocument.language_name"},
		{KindNone, "Breaking change: Renamed `App.namespace_bindings` to `active_bindings`", "breaking renamed_symbol App.namespace_bindings>active_bindings"},
		{KindDeprecation, "Deprecated ``A`` and ``B`` in favor of ``C``", "deprecated renamed_symbol A,B>C"},
		{KindDeprecation, "The `structlog.types` module is now deprecated in favor of the `structlog.typing` module.", "deprecated renamed_symbol types>typing"},
		{KindDeprecation, "Deprecated ``pkg.contrib.x`` module which will be removed in pkg v2.1.0 (#1)", "deprecated removed_symbol contrib.x"},
		{KindMixed, "Removed `X`", "breaking removed_symbol X"},
		{KindMixed, "Deprecated `Y`", "deprecated removed_symbol Y"},
		// Additions: names plainly added, a parameter plainly given, and
		// nothing for a feature described in words.
		{KindAdded, "Added `Client.stream` method", "added added_symbol Client.stream"},
		{KindAdded, "Add `a` and `b()` to `Widget`", "added added_symbol Widget.a,Widget.b"},
		{KindAdded, "New `foo` function (#12)", "added added_symbol foo"},
		{KindAdded, "`Bar` class was added", "added added_symbol Bar"},
		{KindAdded, "`Placeholder.cycle`", ""},                   // names alone: added to what?
		{KindAdded, "Add `A.b()`, `c()` and `d()` methods", ""},  // c and d's owner is the reader's guess
		{KindAdded, "Add `SOURCE_X` constant to `Transfer`", ""}, // a member, or a package-level name?
		{KindAdded, "Added `timeout` parameter to `Client.send`", "added added_parameter Client.send(timeout)"},
		{KindAdded, "Support streaming responses", ""},
		{KindAdded, "Added `x` when `y` is set", ""},
		{KindAdded, "Removed `Z`", ""}, // an Added section's removal isn't read as one
		// The SDK changelogs' shapes: a mark that makes a bullet breaking, and
		// an area, issue or commit read past.
		{KindNone, "⚠️ [#1914](https://x.test/1914) Remove deprecated `StripeObject.request()` method", "breaking removed_symbol StripeObject.request"},
		{KindNone, "⚠️ Remove `RedirectUrl` from `LoginLinkCreateOptions`. The property was deprecated.", "breaking removed_symbol LoginLinkCreateOptions.RedirectUrl"},
		{KindNone, "**[BREAKING]** Removed the attribute `default_branch_protected` from the resource `neon_project`.", "breaking removed_symbol neon_project.default_branch_protected"},
		{KindNone, "feat(core)!: Remove `BaseClient` ([#17071](https://x.test/17071))", "breaking removed_symbol BaseClient"},
		{KindBreaking, "**ssl:** The `Host`, `Status`, and `ValidationMethod` types have been removed from `pkg.types.ssl`.", "breaking removed_symbol types.ssl.Host,types.ssl.Status,types.ssl.ValidationMethod"},
		{KindBreaking, "fc05013: Remove `Capture.Library` and `Exception.Library`. Set `$lib` instead.", "breaking removed_symbol Capture.Library,Exception.Library"},
		{KindBreaking, "`octokit.users.listEmails()` has been renamed to `octokit.users.listEmailsForAuthenticated()`", "breaking renamed_symbol octokit.users.listEmails>octokit.users.listEmailsForAuthenticated"},
		{KindBreaking, "`Redis#serverInfo` is removed. This field is never documented.", "breaking removed_symbol Redis.serverInfo"},
		{KindDeprecation, "Tracing - Deprecates `ddtrace.filters.FilterRequestsOnUrl`.", "deprecated removed_symbol ddtrace.filters.FilterRequestsOnUrl"},
		{SectionKind("Major Changes"), "Remove `X`", "breaking removed_symbol X"},
		{SectionKind("⚠ BREAKING CHANGES"), "**api:** add support for new API version 2026-10-21", "drop"},
		// Not plain: dropped.
		{KindBreaking, "Drop support for Python 3.7. (#2813)", "drop"},
		{KindBreaking, "Removed `limit_rules` from Stylesheet.apply", "drop"},
		{KindBreaking, "Breaking change: `Document.query_syntax_tree` signature changed", "drop"},
		{KindDeprecation, "`X`", "drop"}, // only names, under Deprecated: deprecated how?
		{KindBreaking, "Removed `not a name`", "drop"},
		// Not breaking or a deprecation at all.
		{KindNone, "Removed `X`", ""},
		{SectionKind("Fixed"), "Fixed `X`", ""},
	} {
		ch, ok, why := Classify(c.kind, c.text, "pkg")
		got := ""
		switch {
		case !ok:
		case why != "":
			got = "drop"
		default:
			got = string(ch.ChangeType) + " " + string(ch.Change) + " " + strings.Join(ch.Symbols, ",")
			if ch.Replacement != "" {
				got += ">" + ch.Replacement
			}
			if ch.Parameter != "" {
				got += "(" + ch.Parameter + ")"
			}
		}
		if c.want == "deprecated renamed_symbol types>typing" {
			ch, _, _ = Classify(c.kind, c.text, "structlog")
			got = string(ch.ChangeType) + " " + string(ch.Change) + " " + strings.Join(ch.Symbols, ",") + ">" + ch.Replacement
		}
		if got != c.want {
			t.Errorf("%q: %q, want %q", c.text, got, c.want)
		}
	}
	if SectionKind("Deprecations and Removals") != KindMixed || SectionKind("Backwards incompatible changes") != KindBreaking ||
		SectionKind("Removed") != KindRemoved || SectionKind("Changed") != KindNone ||
		SectionKind("Added") != KindAdded || SectionKind("New Features") != KindAdded {
		t.Error("section kinds")
	}
}
