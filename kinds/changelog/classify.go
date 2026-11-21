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
	KindBreaking         // Breaking changes, Backwards incompatible changes, Major Changes
	KindRemoved          // Removed: breaking, and a bullet of names alone is a removal
	KindDeprecation      // Deprecated, Deprecations
	KindMixed            // "Deprecations and Removals": the bullet's verb decides
	KindAdded            // Added, Features, New features
)

// SectionKind classifies a section title.
func SectionKind(title string) Kind {
	t := strings.ToLower(title)
	// "Major Changes" is a changesets changelog's section for what made the
	// release a new major.
	breaking := strings.Contains(t, "removed") || strings.Contains(t, "removal") ||
		strings.Contains(t, "breaking") || strings.Contains(t, "incompatible") ||
		strings.HasPrefix(t, "major change")
	deprecation := strings.Contains(t, "deprecat")
	switch {
	case breaking && deprecation:
		return KindMixed
	case breaking && !strings.Contains(t, "breaking") && (strings.Contains(t, "removed") || strings.Contains(t, "removal")):
		return KindRemoved
	case breaking:
		return KindBreaking
	case deprecation:
		return KindDeprecation
	case strings.Contains(t, "added") || strings.Contains(t, "feature"):
		return KindAdded
	}
	return KindNone
}

// A bullet in any section that starts with one of these is breaking: Keep a
// Changelog has no "Breaking" section, so projects mark it inline under
// "Changed" ("Breaking change: Removed `App.dark` reactive attribute").
// Stripe marks each breaking bullet with "⚠️", Neon with "**[BREAKING]**",
// conventional commits with "!" after the type ("remove!: ...").
var breakingPrefix = regexp.MustCompile(`(?i)^(?:\**(?:breaking( changes?)?|backwards?[- ]incompatible)\**\s*[:!-]\s*|⚠️?\s*|\**\[breaking\]\**\s*|[a-z]+(?:\([^)]*\))?!:\s*)`)

// What a bullet may start with that says nothing about the change, stripped
// before it's read: an area ("**client:**", "[datadog_monitor]", "Tracing -
// ", "Core: "), its issue or pull request ("[#1914](...)", "[[#51](...)]"),
// a changesets entry's commit and thanks ("abc1234: ", "[#12](...)
// [`abc1234`](...) Thanks [@me](...)! - ").
var noisePrefix = regexp.MustCompile("^(?:" + strings.Join([]string{
	`\*\*[\w ,./-]+:\*\*\s*`,
	`\[[\w ,./-]+\]\s+`,
	`\[\[#\d+\]\([^)]*\)\]\s*`,
	`\[#\d+\]\([^)]*\)\s*`,
	`\[` + "`" + `[0-9a-f]{7,40}` + "`" + `\]\([^)]*\)\s*`,
	"`?[0-9a-f]{7,40}`?:?\\s+",
	`Thanks \[@[^\]]+\]\([^)]*\)!\s*-\s*`,
	`(?:[A-Z][\w/]*(?: [A-Z][\w/]*)*) - `,
	`[A-Z][a-zA-Z]+: `,
}, "|") + ")")

var changeWord = regexp.MustCompile(`(?i)^(added|changed|deprecated|fixed|security|updated|improved)\b`)

// Change is what one bullet says, when it says it plainly enough.
type Change struct {
	ChangeType  event.Change
	Change      event.PackageChange // removed_symbol, renamed_symbol, added_symbol or added_parameter
	Symbols     []string            // as code writes them after the import
	Replacement string              // renamed_symbol only
	Parameter   string              // added_parameter only
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
	noun = `(?:(?:reactive |read-only |optional )?(?:module|method|function|class|attribute|property|constant|widget|exception|warning|classvar|alias|type|field|enum|interface|struct|resource|data source|sub-resource|definition)(?:es|s)?)`
	list = `(§(?:(?:,? and |, )§)*)`
)

var (
	removedActive = regexp.MustCompile(`(?i)^(?:removed|removes|remove|dropped|drops|drop) (?:the )?(?:previously )?(?:deprecated )?(?:` + noun + ` )?` + list +
		`(?: ` + noun + `)?(?: (?:from|in|on|of) (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?)?(?: \(now called (§)\))?$`)
	removedPassive = regexp.MustCompile(`(?i)^(?:the )?(?:deprecated )?(?:` + noun + ` )?` + list + `(?: ` + noun +
		`)? (?:has been |have been |was |were |is |are |is now |are now |now )?removed(?: from (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?)?$`)
	bare           = regexp.MustCompile(`(?i)^(?:` + noun + ` )?` + list + `(?: ` + noun + `)?$`)
	renamed        = regexp.MustCompile(`(?i)^(?:renamed|renames|rename) (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)? to (§)$`)
	renamedPassive = regexp.MustCompile(`(?i)^(?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)? (?:has been|have been|was|were|is|is now) renamed to (?:the )?(§)$`)
	deprecActive   = regexp.MustCompile(`(?i)^(?:deprecated|deprecates|deprecate) (?:the )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)?(?: (?:which|that) will be removed in .+| in favou?r of (?:the )?(§)(?: ` + noun + `)?)?$`)
	deprecPassive  = regexp.MustCompile(`(?i)^(?:the )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)? (?:is|are) (?:now )?deprecated(?: in favou?r of (?:the )?(§)(?: ` + noun + `)?)?$`)

	// constant is a bullet about a constant.
	constant = regexp.MustCompile(`(?i)\bconstants?\b`)

	// What an "Added" or "Features" bullet says plainly: names it adds
	// ("Added `Client.stream` method", "Add `a` and `b` to `Widget`", "New
	// `foo` function", "`bar` was added"), or a parameter it gives a call
	// ("Added `timeout` parameter to `Client.send`").
	addedActive  = regexp.MustCompile(`(?i)^(?:added|adds|add|introduced|introduces|introduce|new)(?: support for)? (?:the |a |an )?(?:new )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)?(?: (?:to|on|in) (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?)?$`)
	addedPassive = regexp.MustCompile(`(?i)^(?:the )?(?:new )?(?:` + noun + ` )?` + list + `(?: ` + noun + `)? (?:has been |have been |was |were |is |are |is now |are now )?added(?: to (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?)?$`)
	addedParam   = regexp.MustCompile(`(?i)^(?:added|adds|add|introduced|introduces|introduce) (?:the |a |an )?(?:new )?(?:optional )?(§) (?:parameter|argument|param|option|kwarg|keyword argument) (?:to|on|for|in) (?:the )?(?:` + noun + ` )?(§)(?: ` + noun + `)?$`)
)

// Classify reads one bullet of a section of kind k. ok is false for a bullet
// that isn't breaking or a deprecation; why is set for one that is but
// doesn't say plainly which names it changes (dropped, never guessed).
func Classify(k Kind, text, module string) (c Change, ok bool, why string) {
	for {
		if m := breakingPrefix.FindString(text); m != "" {
			text, k = text[len(m):], KindBreaking
			continue
		}
		// "Deprecated: `X`" or "Added: `X`" says what the change is: its
		// word is not an area to read past.
		if m := noisePrefix.FindString(text); m != "" && !changeWord.MatchString(m) {
			text = text[len(m):]
			continue
		}
		break
	}
	if k == KindNone {
		return Change{}, false, ""
	}
	if k == KindAdded {
		return classifyAdded(text, module)
	}
	removedHeading := false
	sentence, codes := template(text)
	try := func(re *regexp.Regexp) []string { return re.FindStringSubmatch(sentence) }
	if k == KindRemoved {
		k = KindBreaking
		removedHeading = true
	}
	if k != KindDeprecation {
		if m := try(renamed); m != nil {
			return names(event.ChangeBreaking, codes, "§", "", "", "", module, codes[1])
		}
		if m := try(renamedPassive); m != nil {
			return names(event.ChangeBreaking, codes, "§", "", "", "", module, codes[1])
		}
		if m := try(removedActive); m != nil {
			return names(event.ChangeBreaking, codes, m[1], m[2], m[3], "", module, "")
		}
		if m := try(removedPassive); m != nil {
			return names(event.ChangeBreaking, codes, m[1], m[2], "", "", module, "")
		}
		// Only names, no verb ("`Placeholder.reset_color_cycle`"): counts
		// under a Removed heading, where the heading is the verb.
		if m := try(bare); m != nil && removedHeading {
			return names(event.ChangeBreaking, codes, m[1], "", "", "", module, "")
		}
	}
	if k != KindBreaking {
		if m := try(deprecActive); m != nil {
			return names(event.ChangeDeprecated, codes, m[1], "", "", m[2], module, "")
		}
		if m := try(deprecPassive); m != nil {
			return names(event.ChangeDeprecated, codes, m[1], "", "", m[2], module, "")
		}
	}
	return Change{}, true, "doesn't say plainly which names it removes or deprecates"
}

// classifyAdded reads a bullet of an "Added" or "Features" section: names it
// plainly adds, or a parameter it plainly gives a call. ok is false for any
// other bullet (a feature described in words): precision over recall, an
// addition that doesn't name what it adds isn't one.
func classifyAdded(text, module string) (Change, bool, string) {
	sentence, codes := template(text)
	if m := addedParam.FindStringSubmatch(sentence); m != nil {
		param, ok := cleanName(codes[0], module)
		sym, ok2 := cleanName(codes[1], module)
		if !ok || !ok2 || strings.Contains(param, ".") {
			return Change{}, false, ""
		}
		return Change{ChangeType: event.ChangeAdded, Change: event.PackageAddedParameter, Symbols: []string{sym}, Parameter: param}, true, ""
	}
	for _, re := range []*regexp.Regexp{addedActive, addedPassive} {
		m := re.FindStringSubmatch(sentence)
		if m == nil {
			continue
		}
		// A constant "to" a type is a member in one language and a
		// package-level name of that type in another (Go): which, the bullet
		// doesn't say.
		if m[2] != "" && constant.MatchString(sentence) {
			return Change{}, false, ""
		}
		c, ok, why := names(event.ChangeAdded, codes, m[1], m[2], "", "", module, "")
		if !ok || why != "" {
			return Change{}, false, ""
		}
		// "`A.b()`, `c()` and `d()`" leaves c and d's owner to the reader.
		dotted := 0
		for _, s := range c.Symbols {
			if strings.Contains(s, ".") {
				dotted++
			}
		}
		if m[2] == "" && dotted > 0 && dotted < len(c.Symbols) {
			return Change{}, false, ""
		}
		c.Change = event.PackageAddedSymbol
		return c, true, ""
	}
	return Change{}, false, ""
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

// cleanName is a code span as code writes the name after importing module:
// Ruby's and ioredis' "Class#method" is Class.method, a trailing "()" goes.
// ok is false for anything that isn't a name.
func cleanName(s, module string) (string, bool) {
	s = strings.ReplaceAll(strings.TrimSuffix(strings.TrimSpace(s), "()"), "#", ".")
	if module != "" {
		s = strings.TrimPrefix(s, module+".")
	}
	return s, ident.MatchString(s) && s != module
}

// names turns the § groups of a matched template into a Change. list is the
// matched list of §s; owner, when set, is the § the names were removed from;
// newName / favour the replacement's §.
func names(ct event.Change, codes []string, list, owner, newName, favour, module, renamedTo string) (Change, bool, string) {
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
	clean := func(s string) (string, bool) { return cleanName(s, module) }
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
