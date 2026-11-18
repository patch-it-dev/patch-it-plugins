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
		{KindBreaking, "`Placeholder.reset_color_cycle`", "breaking removed_symbol Placeholder.reset_color_cycle"},
		{KindBreaking, "Property `Widget.focusable_children` https://x.test/1", "breaking removed_symbol Widget.focusable_children"},
		{KindBreaking, "Removed `Widget.reset_focus` (now called `Widget.blur`) https://x.test/1", "breaking renamed_symbol Widget.reset_focus>Widget.blur"},
		{KindBreaking, "Removed the ``_prepare_conn`` method from ``HTTPConnectionPool``. Previously…", "breaking removed_symbol HTTPConnectionPool._prepare_conn"},
		{KindBreaking, "Removed ``pkg.a`` and ``pkg.b()`` (`#1 <https://x.test/1>`__).", "breaking removed_symbol a,b"},
		{KindNone, "Breaking change: `SyntaxAwareDocument.language_name` property removed https://x.test/1", "breaking removed_symbol SyntaxAwareDocument.language_name"},
		{KindNone, "Breaking change: Renamed `App.namespace_bindings` to `active_bindings`", "breaking renamed_symbol App.namespace_bindings>active_bindings"},
		{KindDeprecation, "Deprecated ``A`` and ``B`` in favor of ``C``", "deprecation renamed_symbol A,B>C"},
		{KindDeprecation, "The `structlog.types` module is now deprecated in favor of the `structlog.typing` module.", "deprecation renamed_symbol types>typing"},
		{KindDeprecation, "Deprecated ``pkg.contrib.x`` module which will be removed in pkg v2.1.0 (#1)", "deprecation removed_symbol contrib.x"},
		{KindMixed, "Removed `X`", "breaking removed_symbol X"},
		{KindMixed, "Deprecated `Y`", "deprecation removed_symbol Y"},
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
		}
		if c.want == "deprecation renamed_symbol types>typing" {
			ch, _, _ = Classify(c.kind, c.text, "structlog")
			got = string(ch.ChangeType) + " " + string(ch.Change) + " " + strings.Join(ch.Symbols, ",") + ">" + ch.Replacement
		}
		if got != c.want {
			t.Errorf("%q: %q, want %q", c.text, got, c.want)
		}
	}
	if SectionKind("Deprecations and Removals") != KindMixed || SectionKind("Backwards incompatible changes") != KindBreaking ||
		SectionKind("Removed") != KindBreaking || SectionKind("Changed") != KindNone {
		t.Error("section kinds")
	}
}
