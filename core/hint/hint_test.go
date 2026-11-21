package hint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// fakeDocs serves the saved changelog pages under testdata the way
// docs.stripe.com does: every page at its path plus ".md".
func fakeDocs(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func request(op, property string, ct event.Change, hint string) event.Event {
	return event.Event{
		Change: ct,
		Subject: event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindRequestParameter, Side: event.SideRequest,
			Operation: op, Property: property},
		MigrationHint: hint,
	}
}

func TestResolveStripeFollowsTheApiReferenceConventions(t *testing.T) {
	type want struct {
		ops    []string
		schema string
		parent string
	}
	cases := []struct {
		ref, href string
		want      want
	}{
		{"PaymentIntent#create", "/api/payment_intents/create?api-version=2026-09-30.endive#create_payment_intent", want{ops: []string{"POST /v1/payment_intents"}}},
		{"PaymentIntent#update", "/api/payment_intents/update", want{ops: []string{"POST /v1/payment_intents/{}"}}},
		{"PaymentIntent#retrieve", "/api/payment_intents/retrieve", want{ops: []string{"GET /v1/payment_intents/{}"}}},
		{"PaymentIntent#list", "/api/payment_intents/list", want{ops: []string{"GET /v1/payment_intents"}}},
		{"PaymentIntent#confirm", "/api/payment_intents/confirm", want{ops: []string{"POST /v1/payment_intents/{}/confirm", "POST /v1/payment_intents/confirm"}}},
		{"Checkout.Session#create", "/api/checkout/sessions/create", want{ops: []string{"POST /v1/checkout/sessions"}}},
		{"Checkout.Session", "/api/checkout/sessions/object?api-version=2026-09-30.endive", want{schema: "checkout.session"}},
		{"Capability", "/api/capabilities/object", want{schema: "capability"}},
		{"V2.Core.Account#create", "/api/v2/core/accounts/create", want{ops: []string{"POST /v2/core/accounts"}}},
		{"Charge#search", "/api/charges/search", want{ops: []string{"GET /v1/charges/search"}}},
		// The parent path of a nested parameter is in the reference text.
		{"Subscription#update.trial_settings.end_behavior", "/api/subscriptions/update#x", want{ops: []string{"POST /v1/subscriptions/{}"}, parent: "trial_settings.end_behavior"}},
		{"Subscription.trial_settings.end_behavior", "/api/subscriptions/object#x", want{schema: "subscription", parent: "trial_settings.end_behavior"}},
		// No link at all: the resource name gives the path.
		{"Subscription#update.billing_cycle_anchor", "", want{ops: []string{"POST /v1/subscriptions/{}"}, parent: "billing_cycle_anchor"}},
		{"Invoice#create_preview.subscription_details.billing_cycle_anchor", "", want{ops: []string{"POST /v1/invoices/{}/create_preview", "POST /v1/invoices/create_preview"}, parent: "subscription_details.billing_cycle_anchor"}},
		{"Checkout.Session#create.payment_method_types", "", want{ops: []string{"POST /v1/checkout/sessions"}, parent: "payment_method_types"}},
		{"Capability#retrieve", "", want{ops: []string{"GET /v1/capabilities/{}"}}},
	}
	for _, c := range cases {
		targets, parent, ok := ResolveStripe(c.ref, c.href)
		if !ok {
			t.Errorf("%s %s: not resolved", c.ref, c.href)
			continue
		}
		var ops []string
		var schema string
		for _, tg := range targets {
			if tg.Operation != "" {
				ops = append(ops, tg.Operation)
			}
			schema = tg.Schema
		}
		if strings.Join(ops, ",") != strings.Join(c.want.ops, ",") || schema != c.want.schema || parent != c.want.parent {
			t.Errorf("%s %s: got ops=%v schema=%q parent=%q, want %+v", c.ref, c.href, ops, schema, parent, c.want)
		}
	}
	for _, bad := range []struct{ ref, href string }{
		{"dynamic payment methods", "https://docs.stripe.com/payments/dynamic-payment-methods.md"},
		{"x", "/api/object"}, {"x", "/docs/x"}, {"lowercase", ""},
	} {
		if _, _, ok := ResolveStripe(bad.ref, bad.href); ok {
			t.Errorf("%+v: resolved, want skipped", bad)
		}
	}
}

func TestParseReadsWhatsNewAndTheRestTable(t *testing.T) {
	md, err := os.ReadFile("testdata/remove-payment-method-types-checkout-sessions.md")
	if err != nil {
		t.Fatal(err)
	}
	e, err := Parse("https://docs.stripe.com/x", string(md), ResolveStripe)
	if err != nil {
		t.Fatal(err)
	}
	if e.Title != "Removes the payment method types parameter from Checkout Sessions" {
		t.Errorf("title: %q", e.Title)
	}
	if !strings.HasPrefix(e.WhatsNew, "Removes the [payment_method_types]") || strings.Contains(e.WhatsNew, "## Why") {
		t.Errorf("what's new is wrong: %q", e.WhatsNew)
	}
	if len(e.Rows) != 2 {
		t.Fatalf("rows: got %d, want 2: %+v", len(e.Rows), e.Rows)
	}
	added, removed := e.Rows[0], e.Rows[1]
	if added.Parameter != "allowed_payment_method_types" || added.Change != "Added" {
		t.Errorf("row 0: %+v", added)
	}
	if removed.Parameter != "payment_method_types" || removed.Change != "Removed" {
		t.Errorf("row 1: %+v", removed)
	}
	if len(removed.Targets) != 1 || removed.Targets[0] != (Target{Operation: "POST /v1/checkout/sessions", Path: "payment_method_types"}) {
		t.Errorf("row 1 targets: %+v", removed.Targets)
	}
}

func TestParseReadsNestedPathsFromReferencesAndBareRows(t *testing.T) {
	md, _ := os.ReadFile("testdata/trial-offer-ga.md")
	e, err := Parse("https://x/trial", string(md), ResolveStripe)
	if err != nil {
		t.Fatal(err)
	}
	var nested *Row
	for i := range e.Rows {
		if e.Rows[i].Parameter == "billing_cycle_anchor" {
			nested = &e.Rows[i]
		}
	}
	if nested == nil {
		t.Fatalf("no billing_cycle_anchor row: %+v", e.Rows)
	}
	for _, tg := range nested.Targets {
		if tg.Path != "trial_settings.end_behavior.billing_cycle_anchor" {
			t.Errorf("the parent path from the reference text is missing: %+v", tg)
		}
	}

	md, _ = os.ReadFile("testdata/polymorphic-billing-cycle-anchor.md")
	e, err = Parse("https://x/poly", string(md), ResolveStripe)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Rows) != 3 {
		t.Fatalf("rows: got %d, want 3: %+v", len(e.Rows), e.Rows)
	}
	update := e.Rows[2]
	if update.Parameter != "billing_cycle_anchor" || update.Change != "Changed" {
		t.Errorf("row: %+v", update)
	}
	if update.Targets[0] != (Target{Operation: "POST /v1/subscriptions/{}", Path: "billing_cycle_anchor"}) {
		t.Errorf("bare reference target: %+v", update.Targets)
	}
}

func TestIndexFetchesAVersionsEntriesAndAttachesHints(t *testing.T) {
	srv := fakeDocs(t)
	ix, err := Stripe{Base: srv.URL}.Index(context.Background(), "2026-09-30.endive")
	if err != nil {
		t.Fatal(err)
	}
	if len(ix.Entries) != 5 {
		t.Fatalf("entries: got %d, want 5", len(ix.Entries))
	}

	breaking := event.ChangeBreaking
	events := []event.Event{
		request("POST /v1/checkout/sessions", "payment_method_types", breaking, "Stop sending payment_method_types."),
		request("POST /v1/payment_intents/{intent}/confirm", "payment_method_types", breaking, ""),
		request("POST /v1/payment_methods/{payment_method}", "payto", breaking, ""),
		// Same parameter name, an operation no entry names: keeps its placeholder.
		request("POST /v1/invoices", "payment_settings.payment_method_types[]", breaking, "placeholder"),
		{Change: breaking, Subject: event.Subject{Type: event.SubjectAPIOperation, Kind: event.KindSchemaProperty, Side: event.SideResponse,
			Schema: "payment_intent", Property: "charges"}},
		// A collision found in review: the top-level billing_cycle_anchor type
		// change on Subscription#update, and the nested one trial offers added.
		// Each gets its own entry; neither gets the other's.
		request("POST /v1/subscriptions/{subscription_exposed_id}", "billing_cycle_anchor", breaking, "Handle billing_cycle_anchor as object."),
		request("POST /v1/subscriptions/{subscription_exposed_id}", "trial_settings.end_behavior.billing_cycle_anchor", event.ChangeAdded, ""),
		request("POST /v1/subscriptions/{subscription}/resume", "billing_cycle_anchor", breaking, "Handle billing_cycle_anchor as object."),
	}
	if n := ix.Attach(events); n != 6 {
		t.Fatalf("attached %d, want 6", n)
	}
	checkout := events[0].MigrationHint
	if !strings.HasPrefix(checkout, "Stop sending payment_method_types.\n\n") {
		t.Errorf("the normalizer's hint must stay first: %q", checkout)
	}
	if !strings.Contains(checkout, "allowed_payment_method_types") || !strings.HasSuffix(checkout,
		"Source: https://docs.stripe.com/changelog/endive/2026-09-30/remove-payment-method-types-checkout-sessions") {
		t.Errorf("checkout hint: %q", checkout)
	}
	if strings.Contains(checkout, "](https://") {
		t.Errorf("link URLs must be stripped from the prose: %q", checkout)
	}
	if !strings.Contains(events[1].MigrationHint, "Payment Intents") {
		t.Errorf("confirm hint: %q", events[1].MigrationHint)
	}
	if !strings.Contains(events[2].MigrationHint, "PayTo") {
		t.Errorf("payto hint: %q", events[2].MigrationHint)
	}
	if events[3].MigrationHint != "placeholder" || events[4].MigrationHint != "" {
		t.Errorf("unmatched events changed: %q / %q", events[3].MigrationHint, events[4].MigrationHint)
	}
	top, nested, resume := events[5].MigrationHint, events[6].MigrationHint, events[7].MigrationHint
	if !strings.HasPrefix(top, "Handle billing_cycle_anchor as object.\n\n") || !strings.HasSuffix(top, "/polymorphic-billing-cycle-anchor") {
		t.Errorf("top-level billing_cycle_anchor: %q", top)
	}
	if !strings.HasSuffix(nested, "/trial-offer-ga") {
		t.Errorf("nested billing_cycle_anchor: %q", nested)
	}
	if !strings.HasSuffix(resume, "/polymorphic-billing-cycle-anchor") {
		t.Errorf("resume billing_cycle_anchor: %q", resume)
	}
	for _, e := range events {
		if n := len([]rune(e.MigrationHint)); n > hintLimit {
			t.Errorf("hint over the cap: %d", n)
		}
	}
}

func TestAnAddedRowNeverExplainsABreakingChange(t *testing.T) {
	ix := NewIndex([]Entry{{URL: "https://x/added", WhatsNew: "Adds p.",
		Rows: []Row{{Parameter: "p", Change: "Added", Targets: []Target{{Operation: "POST /v1/x", Path: "p"}}}}}})
	events := []event.Event{
		request("POST /v1/x", "p", event.ChangeBreaking, "Stop sending p."),
		request("POST /v1/x", "p", event.ChangeAdded, ""),
	}
	if n := ix.Attach(events); n != 1 {
		t.Fatalf("attached %d, want 1", n)
	}
	if events[0].MigrationHint != "Stop sending p." || !strings.HasSuffix(events[1].MigrationHint, "Source: https://x/added") {
		t.Errorf("%q / %q", events[0].MigrationHint, events[1].MigrationHint)
	}
}

// The mirror: a parameter removed in one place and added in another must not hand
// the removal's text to the addition.
func TestARemovedRowNeverExplainsAnAddition(t *testing.T) {
	ix := NewIndex([]Entry{{URL: "https://x/removed", WhatsNew: "Removes p.",
		Rows: []Row{{Parameter: "p", Change: "Removed", Targets: []Target{{Operation: "POST /v1/x", Path: "p"}}}}}})
	events := []event.Event{
		request("POST /v1/x", "p", event.ChangeBreaking, ""),
		request("POST /v1/x", "p", event.ChangeAdded, ""),
	}
	if n := ix.Attach(events); n != 1 {
		t.Fatalf("attached %d, want 1", n)
	}
	if !strings.HasSuffix(events[0].MigrationHint, "Source: https://x/removed") || events[1].MigrationHint != "" {
		t.Errorf("%q / %q", events[0].MigrationHint, events[1].MigrationHint)
	}
}

func TestIndexSaysWhenTheChangelogHasNothingForTheVersion(t *testing.T) {
	srv := fakeDocs(t)
	if _, err := (Stripe{Base: srv.URL}).Index(context.Background(), "2026-10-14.endive"); err == nil {
		t.Fatal("want an error for a version the page has no section for")
	}
	if _, err := (Stripe{Base: srv.URL}).Index(context.Background(), "2022-11-15"); err == nil {
		t.Fatal("want an error when the release page doesn't exist")
	}
}

func TestReleaseOf(t *testing.T) {
	if ReleaseOf("2026-09-30.endive") != "endive" || ReleaseOf("2022-11-15") != "2022-11-15" {
		t.Fatal("release names")
	}
}

func TestAHintIsCappedFromTheFrontAndKeepsItsSource(t *testing.T) {
	ix := NewIndex([]Entry{{URL: "https://x", WhatsNew: strings.Repeat("a", 5000),
		Rows: []Row{{Parameter: "p", Change: "Changed", Targets: []Target{{Operation: "POST /v1/x", Path: "p"}}}}}})
	events := []event.Event{request("POST /v1/x", "p", event.ChangeBreaking, "Handle p as q.")}
	ix.Attach(events)
	got := events[0].MigrationHint
	if n := len([]rune(got)); n > hintLimit {
		t.Fatalf("got %d runes, want <= %d", n, hintLimit)
	}
	if !strings.HasPrefix(got, "Handle p as q.\n\n") || !strings.HasSuffix(got, "…\n\nSource: https://x") {
		t.Fatalf("the precise hint first, the source last, the prose cut between: %q", got[:40]+"…"+got[len(got)-30:])
	}
}

func TestPlainLinksKeepsTheTextAndDropsTheUrl(t *testing.T) {
	got := plainLinks("Use [dynamic payment methods](https://docs.stripe.com/x.md) or `a`.")
	if got != "Use dynamic payment methods or `a`." {
		t.Fatalf("%q", got)
	}
}
