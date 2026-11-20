package changelog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// Kind is what a section (or a bullet's own prefix) says its changes are.
type Kind int

const (
	KindNone        Kind = iota
	KindBreaking         // Removed, Breaking changes, Backwards incompatible changes
	KindDeprecation      // Deprecated, Deprecations
	KindMixed            // "Deprecations and Removals": the bullet's verb decides
)

// SectionKind classifies a section title.
func SectionKind(title string) Kind {
	t := strings.ToLower(title)
	breaking := strings.Contains(t, "removed") || strings.Contains(t, "removal") ||
		strings.Contains(t, "breaking") || strings.Contains(t, "incompatible")
	deprecation := strings.Contains(t, "deprecat")
	switch {
	case breaking && deprecation:
		return KindMixed
	case breaking:
		return KindBreaking
	case deprecation:
		return KindDeprecation
	}
	return KindNone
}

// A bullet in any section that starts with one of these is breaking: Keep a
// Changelog has no "Breaking" section, so projects mark it inline under
// "Changed" ("Breaking change: Removed `App.dark` reactive attribute").
var breakingPrefix = regexp.MustCompile(`(?i)^\**(breaking( changes?)?|backwards?[- ]incompatible)\**\s*[:!-]\s*`)

// Change is what one bullet says, when it says it plainly enough.
type Change struct {
	ChangeType  event.ChangeType
	Change      event.PackageChange // removed_symbol or renamed_symbol
	Symbols     []string            // as code writes them after the import
	Replacement string              // renamed_symbol only
}

var (
	// md links, rst links (`text <url>`_), bare URLs, issue refs.
	mdLink   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	rstLink  = regexp.MustCompile("`[^`<]*<[^>]+>`_{1,2}")
	bareURL  = regexp.MustCompile(`https?://\S+`)
	issueRef = regexp.MustCompile(`\(\s*(#\d+[,\s]*)+\)|\(\s*\)|\s*,\s*$`)
	codeSpan = regexp.MustCompile("`([^`]+)`")
	ident    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

// Templates over a bullet's first sentence with every code span replaced by
// §. Each says one thing plainly; a bullet matching none is dropped.
const (
	noun = `(?:(?:reactive )?(?:module|method|function|class|attribute|property|constant|widget|exception|warning|classvar|alias)s?)`
	list = `(§(?:(?:,? and |, )§)*)`
)

var (
	removedActive = regexp.MustCompile(`(?i)^(?:removed|remove|dropped|drop) (?:the )?(?:deprecated )?(?:` + noun + ` )?` + list +
		`(?: ` + noun + `)?(?: from (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?)?(?: \(now called (§)\))?$`)
	removedPassive = regexp.MustCompile(`(?i)^(?:the )?(?:deprecated )?(?:` + noun + ` )?` + list + `(?: ` + noun +
		`)? (?:has been |have been |was |were |is |are |is now |are now |now )?removed$`)
	bare          = regexp.MustCompile(`(?i)^(?:` + noun + ` )?` + list + `(?: ` + noun + `)?$`)
	renamed       = regexp.MustCompile(`(?i)^renamed (§)(?: ` + noun + `)? to (§)$`)
	deprecActive  = regexp.MustCompile(`(?i)^(?:deprecated|deprecate) (?:the )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)?(?: (?:which|that) will be removed in .+| in favou?r of (?:the )?(§)(?: ` + noun + `)?)?$`)
	deprecPassive = regexp.MustCompile(`(?i)^(?:the )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)? (?:is|are) (?:now )?deprecated(?: in favou?r of (?:the )?(§)(?: ` + noun + `)?)?$`)
)

// Classify reads one bullet of a section of kind k. ok is false for a bullet
// that isn't breaking or a deprecation; why is set for one that is but
// doesn't say plainly which names it changes (dropped, never guessed).
func Classify(k Kind, text, module string) (c Change, ok bool, why string) {
	if m := breakingPrefix.FindString(text); m != "" {
		text, k = text[len(m):], KindBreaking
	}
	if k == KindNone {
		return Change{}, false, ""
	}
	sentence, codes := template(text)
	try := func(re *regexp.Regexp) []string { return re.FindStringSubmatch(sentence) }
	if k != KindDeprecation {
		if m := try(renamed); m != nil {
			return names(event.ChangeTypeBreaking, codes, "§", "", "", "", module, codes[1])
		}
		if m := try(removedActive); m != nil {
			return names(event.ChangeTypeBreaking, codes, m[1], m[2], m[3], "", module, "")
		}
		if m := try(removedPassive); m != nil {
			return names(event.ChangeTypeBreaking, codes, m[1], "", "", "", module, "")
		}
		// Only names, no verb ("`Placeholder.reset_color_cycle`"): counts
		// under a Removed heading, where the heading is the verb.
		if m := try(bare); m != nil && k == KindBreaking {
			return names(event.ChangeTypeBreaking, codes, m[1], "", "", "", module, "")
		}
	}
	if k != KindBreaking {
		if m := try(deprecActive); m != nil {
			return names(event.ChangeTypeDeprecation, codes, m[1], "", "", m[2], module, "")
		}
		if m := try(deprecPassive); m != nil {
			return names(event.ChangeTypeDeprecation, codes, m[1], "", "", m[2], module, "")
		}
	}
	return Change{}, true, "doesn't say plainly which names it removes or deprecates"
}

// template is the bullet's first sentence, links and issue references
// dropped, with every code span replaced by §, and the spans in order.
func template(text string) (string, []string) {
	s := rstLink.ReplaceAllString(text, "")
	s = strings.ReplaceAll(s, "``", "`")
	s = mdLink.ReplaceAllString(s, "$1")
	s = bareURL.ReplaceAllString(s, "")
	var codes []string
	s = codeSpan.ReplaceAllStringFunc(s, func(m string) string {
		codes = append(codes, codeSpan.FindStringSubmatch(m)[1])
		return "§"
	})
	// The first sentence: code spans are out of the way, so a ". " ends it.
	if i := strings.Index(s, ". "); i >= 0 {
		s = s[:i]
	}
	s = issueRef.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimRight(s, ". "), codes
}

// names turns the § groups of a matched template into a Change. list is the
// matched list of §s; owner, when set, is the § the names were removed from;
// newName / favour the replacement's §.
func names(ct event.ChangeType, codes []string, list, owner, newName, favour, module, renamedTo string) (Change, bool, string) {
	// Which codes are in which group: count §s before each group.
	idx := 0
	take := func(group string) []string {
		n := strings.Count(group, "§")
		out := codes[idx : idx+n]
		idx += n
		return out
	}
	syms := take(list)
	var own, repl []string
	if owner != "" {
		own = take(owner)
	}
	if newName != "" {
		repl = take(newName)
	}
	if favour != "" {
		repl = take(favour)
	}
	if renamedTo != "" {
		repl = []string{renamedTo}
	}
	clean := func(s string) (string, bool) {
		s = strings.TrimSuffix(strings.TrimSpace(s), "()")
		if module != "" {
			s = strings.TrimPrefix(s, module+".")
		}
		return s, ident.MatchString(s) && s != module
	}
	c := Change{ChangeType: ct, Change: event.PackageRemovedSymbol}
	prefix := ""
	if len(own) == 1 {
		o, ok := clean(own[0])
		if !ok {
			return Change{}, true, fmt.Sprintf("%q isn't a name code writes", own[0])
		}
		prefix = o + "."
	}
	for _, s := range syms {
		n, ok := clean(s)
		if !ok {
			return Change{}, true, fmt.Sprintf("%q isn't a name code writes", s)
		}
		if !slices.Contains(c.Symbols, prefix+n) {
			c.Symbols = append(c.Symbols, prefix+n)
		}
	}
	if len(repl) == 1 {
		r, ok := clean(repl[0])
		if !ok {
			return Change{}, true, fmt.Sprintf("replacement %q isn't a name code writes", repl[0])
		}
		if !slices.Contains(c.Symbols, r) {
			c.Change, c.Replacement = event.PackageRenamedSymbol, r
		}
	}
	return c, true, ""
}
