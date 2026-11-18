package hint

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
)

// Stripe reads docs.stripe.com's changelog, which every page serves as
// markdown at its URL plus ".md" (checked 2026-10-13). A version's entries are
// listed under a "## <version>" heading on the release's index page:
// /changelog/endive.md for 2026-09-30.endive (the flora release name after
// the dot), /changelog/2022-11-15.md for a version from before the names.
// Each entry's REST API table names endpoints the way the API reference does,
// which maps onto the spec's operations by Stripe's own conventions.
type Stripe struct {
	Base string                                                // default "https://docs.stripe.com"
	Get  func(ctx context.Context, url string) ([]byte, error) // default fetch.Get
}

const stripeDocs = "https://docs.stripe.com"

func (s Stripe) base() string {
	if s.Base != "" {
		return strings.TrimRight(s.Base, "/")
	}
	return stripeDocs
}

func (s Stripe) get(ctx context.Context, u string) ([]byte, error) {
	if s.Get != nil {
		return s.Get(ctx, u)
	}
	return fetch.Get(ctx, u, nil)
}

// ReleaseOf is the changelog page a version's entries live on: "endive" for
// "2026-09-30.endive", the date itself for "2022-11-15".
func ReleaseOf(version string) string {
	if i := strings.Index(version, "."); i >= 0 {
		return version[i+1:]
	}
	return version
}

var entryLink = regexp.MustCompile(`\((?:https?://[^/)]+)?(/changelog/[^)\s]+\.md)\)`)

// Index fetches the version's entries and keys them for lookup. An entry that
// can't be fetched or parsed is skipped with a log line: one bad page costs one
// hint, not the run. A version with no section on its release page is an error,
// so the caller can say the changelog has nothing for it.
func (s Stripe) Index(ctx context.Context, version string) (*Index, error) {
	indexURL := fmt.Sprintf("%s/changelog/%s.md", s.base(), ReleaseOf(version))
	page, err := s.get(ctx, indexURL)
	if err != nil {
		return nil, fmt.Errorf("changelog index %s: %w", indexURL, err)
	}
	heading := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(version) + `\s*$`)
	sec := section(string(page), heading, sectionRE)
	if sec == "" {
		return nil, fmt.Errorf("changelog index %s has no section for %s", indexURL, version)
	}
	seen := map[string]bool{}
	var entries []Entry
	for _, m := range entryLink.FindAllStringSubmatch(sec, -1) {
		path := m[1]
		if seen[path] {
			continue
		}
		seen[path] = true
		pageURL := s.base() + path
		body, err := s.get(ctx, pageURL)
		if err != nil {
			log.Printf("stripe changelog: skipping %s: %v", pageURL, err)
			continue
		}
		entry, err := Parse(strings.TrimSuffix(stripeDocs+path, ".md"), string(body), ResolveStripe)
		if err != nil {
			log.Printf("stripe changelog: skipping %s: %v", pageURL, err)
			continue
		}
		entries = append(entries, entry)
	}
	return NewIndex(entries), nil
}

// ResolveStripe reads one reference of a Stripe table row.
//
// The reference text is "Resource#method" or "Resource#method.parent.path"
// for an endpoint, "Resource" or "Resource.parent.path" for an object; the
// resource may be namespaced ("Checkout.Session"). The parent path is what a
// nested parameter sits under. When the reference is a link, its href names
// the place by the API reference's URL conventions, which is the exact form:
//
//	/api/payment_intents/create          -> POST /v1/payment_intents
//	/api/payment_intents/update          -> POST /v1/payment_intents/{}
//	/api/payment_intents/retrieve        -> GET /v1/payment_intents/{}
//	/api/payment_intents/list            -> GET /v1/payment_intents
//	/api/payment_intents/confirm         -> POST /v1/payment_intents/{}/confirm, and POST /v1/payment_intents/confirm
//	/api/checkout/sessions/create        -> POST /v1/checkout/sessions
//	/api/checkout/sessions/object        -> schema checkout.session
//	/api/v2/core/accounts/create         -> POST /v2/core/accounts
//
// A custom verb ("confirm", "create_preview") can't say whether it acts on one
// object or the collection, so both shapes are returned and the event's own
// operation picks. Without a link, the resource name is turned into its path
// ("PaymentIntent" -> payment_intents, "Checkout.Session" -> checkout/sessions)
// the same way. A reference that can't be read is skipped: a missing hint is
// the safe side.
func ResolveStripe(ref, href string) ([]Target, string, bool) {
	resource, method, parent := splitRef(ref)
	if resource == "" {
		return nil, "", false
	}
	if href != "" {
		targets, ok := fromHref(href)
		return targets, parent, ok
	}
	segments := strings.Split(resource, ".")
	if strings.EqualFold(segments[0], "v2") {
		return nil, "", false // v2 references carry the path only in their links
	}
	path := make([]string, len(segments))
	for i, seg := range segments {
		path[i] = snake(seg)
	}
	path[len(path)-1] = plural(path[len(path)-1])
	targets, ok := fromPath("v1", path, method)
	return targets, parent, ok
}

// splitRef: "Subscription#update.trial_settings.end_behavior" ->
// ("Subscription", "update", "trial_settings.end_behavior");
// "Checkout.Session.payment_method_types" -> ("Checkout.Session", "object", "payment_method_types").
func splitRef(ref string) (resource, method, parent string) {
	ref = strings.TrimSpace(ref)
	if res, rest, found := strings.Cut(ref, "#"); found {
		method, parent, _ = strings.Cut(rest, ".")
		return res, method, parent
	}
	segments := strings.Split(ref, ".")
	n := 0
	for n < len(segments) && segments[n] != "" && unicode.IsUpper(rune(segments[n][0])) {
		n++
	}
	if n == 0 {
		return "", "", ""
	}
	return strings.Join(segments[:n], "."), "object", strings.Join(segments[n:], ".")
}

func fromHref(href string) ([]Target, bool) {
	u, err := url.Parse(href)
	if err != nil {
		return nil, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" {
		return nil, false
	}
	parts = parts[1:]
	version := "v1"
	if parts[0] == "v2" {
		version, parts = "v2", parts[1:]
	}
	if len(parts) < 2 {
		return nil, false
	}
	for _, seg := range parts {
		if seg == "" || strings.ContainsAny(seg, "{}") {
			return nil, false
		}
	}
	return fromPath(version, parts[:len(parts)-1], parts[len(parts)-1])
}

func fromPath(version string, resource []string, method string) ([]Target, bool) {
	collection := "/" + version + "/" + strings.Join(resource, "/")
	switch method {
	case "object":
		names := append([]string(nil), resource...)
		names[len(names)-1] = singular(names[len(names)-1])
		return []Target{{Schema: strings.Join(names, ".")}}, true
	case "create":
		return []Target{{Operation: "POST " + collection}}, true
	case "list":
		return []Target{{Operation: "GET " + collection}}, true
	case "retrieve":
		return []Target{{Operation: "GET " + collection + "/{}"}}, true
	case "update":
		return []Target{{Operation: "POST " + collection + "/{}"}}, true
	case "delete":
		return []Target{{Operation: "DELETE " + collection + "/{}"}}, true
	case "search":
		return []Target{{Operation: "GET " + collection + "/search"}}, true
	case "":
		return nil, false
	default:
		return []Target{
			{Operation: "POST " + collection + "/{}/" + method},
			{Operation: "POST " + collection + "/" + method},
		}, true
	}
}

// snake: "PaymentIntent" -> "payment_intent", "TrialOffer" -> "trial_offer".
func snake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// plural undoes singular: "payment_intent" -> "payment_intents", "capability" -> "capabilities".
func plural(name string) string {
	switch {
	case strings.HasSuffix(name, "y"):
		return name[:len(name)-1] + "ies"
	case strings.HasSuffix(name, "s"), strings.HasSuffix(name, "x"):
		return name + "es"
	default:
		return name + "s"
	}
}

// singular undoes the plural a collection name carries: "payment_intents" ->
// "payment_intent", "sessions" -> "session", "capabilities" -> "capability".
func singular(collection string) string {
	switch {
	case strings.HasSuffix(collection, "ies"):
		return collection[:len(collection)-3] + "y"
	case strings.HasSuffix(collection, "sses"), strings.HasSuffix(collection, "xes"):
		return collection[:len(collection)-2]
	default:
		return strings.TrimSuffix(collection, "s")
	}
}
