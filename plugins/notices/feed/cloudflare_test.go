package feed

import (
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
)

const cloudflareFeed = "testdata/cloudflare-api-deprecations.rss.xml"

func cloudflareFindings(t *testing.T) (kept, dropped []Finding) {
	t.Helper()
	fs, err := CloudflareAPI.Findings(kittest.Read(t, cloudflareFeed))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.Dropped != "" {
			dropped = append(dropped, f)
		} else {
			kept = append(kept, f)
		}
	}
	return kept, dropped
}

// The historical breaking change: Cloudflare deprecated its legacy Registrar
// domain management endpoints on 2026-04-10 with an end of life of
// 2026-09-27 ("After this date, the legacy endpoints will no longer be
// available"), and they left Cloudflare's OpenAPI spec (cloudflare/api-schemas)
// in October 2026. The saved feed (2026-11-19) still carries the item.
func TestCloudflareRegistrarEndOfLife(t *testing.T) {
	kept, _ := cloudflareFindings(t)
	want := map[string]bool{
		"GET /accounts/{account_id}/registrar/domains":               true,
		"GET /accounts/{account_id}/registrar/domains/{domain_name}": true,
		"PUT /accounts/{account_id}/registrar/domains/{domain_name}": true,
	}
	var got int
	for _, f := range kept {
		if !want[f.Subject.Operation] {
			continue
		}
		got++
		if f.ChangeType != event.ChangeTypeBreaking || f.Version != "2026-09-27" {
			t.Errorf("%s: %s %s, want breaking 2026-09-27", f.Subject.Operation, f.ChangeType, f.Version)
		}
		if f.Subject.Type != event.SubjectAPIOperation || f.Subject.Kind != event.KindOperation || f.Subject.Side != event.SideRequest {
			t.Errorf("%s: subject %+v", f.Subject.Operation, f.Subject)
		}
		for _, line := range []string{"API deprecations - Legacy Registrar Domain Management API", "End of life date: September 27, 2026",
			"After this date, the legacy endpoints will no longer be available.", "Deprecated APIs:"} {
			if !strings.Contains(f.Excerpt, line) {
				t.Errorf("%s: excerpt %q lacks %q", f.Subject.Operation, f.Excerpt, line)
			}
		}
	}
	if got != len(want) {
		t.Errorf("%d registrar events, want %d", got, len(want))
	}
}

// The pinned drop test. "Gateway Audit SSH rules" lists two operations under
// "Deprecated APIs:", but what goes is one value of one field ("creating
// rules with action: \"audit_ssh\" is no longer accepted"): the operations
// stay. An event saying POST /accounts/{account_id}/gateway/rules is
// deprecated would be wrong, so the item makes none.
func TestCloudflareValueDeprecationIsNoOperation(t *testing.T) {
	kept, dropped := cloudflareFindings(t)
	for _, f := range kept {
		if strings.Contains(f.Subject.Operation, "/gateway/rules") {
			t.Errorf("event for %s: the item deprecates a value, not the operation", f.Subject.Operation)
		}
	}
	var n int
	for _, f := range dropped {
		if strings.Contains(f.Dropped, "Gateway Audit SSH rules") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d drops logged for the Gateway Audit SSH item, want its 2 list items", n)
	}
}

// On the saved feed: 114 operations from 27 items. Only the 5 operations of
// the 2 items that say the endpoints stop are breaking; the other 109 are
// deprecations. 27 drops are logged: 21 whole items (20 with no "Deprecated
// APIs:" list, 1 telling direct API callers no action is required) and 6
// list items that aren't one operation.
func TestCloudflareCounts(t *testing.T) {
	kept, dropped := cloudflareFindings(t)
	byType := map[event.ChangeType]int{}
	items := map[string]bool{}
	for _, f := range kept {
		byType[f.ChangeType]++
		items[f.Entry] = true
	}
	if byType[event.ChangeTypeBreaking] != 5 || byType[event.ChangeTypeDeprecation] != 109 || len(dropped) != 27 || len(items) != 27 {
		t.Errorf("breaking %d, deprecation %d, dropped %d, from %d items; want 5, 109, 27, 27",
			byType[event.ChangeTypeBreaking], byType[event.ChangeTypeDeprecation], len(dropped), len(items))
	}
	var noList int
	for _, f := range dropped {
		if strings.Contains(f.Dropped, `no "Deprecated APIs:" list`) {
			noList++
		}
	}
	if noList != 20 {
		t.Errorf("%d items dropped for having no list, want 20", noList)
	}
}

func cfEntry(dates, items string) Entry {
	return Entry{Title: "API deprecations - X", Link: "https://x.test/#x", Published: "Mon, 29 Jun 2026 00:00:00 GMT",
		HTML: dates + "<p>Deprecated APIs:</p>\n<ul>\n" + items + "</ul>\n<p>Replacement:</p><ul><li><code>GET /new</code></li></ul>"}
}

// How each way the page writes an operation is read, old and new.
func TestCloudflareOperationForms(t *testing.T) {
	for item, want := range map[string]string{
		"<li><code>GET /zones/{zone_id}/settings</code></li>":                          "GET /zones/{zone_id}/settings",
		"<li><code>PUT /accounts/{account_id}/x/{name}</code> — Update domain</li>":    "PUT /accounts/{account_id}/x/{name}",
		"<li><code>GET zones/:zone_identifier/analytics/dashboard</code></li>":         "GET /zones/{zone_identifier}/analytics/dashboard",
		"<li><code>GET/zones/{zone_id}/dns_analytics/</code> (DNS Analytics API)</li>": "GET /zones/{zone_id}/dns_analytics",
		"<li><code>GET</code> and <code>POST</code> /accounts/{account_id}/ns</li>":    "dropped",
		"<li>DoH resolver on resolver.cloudflare-eth.com</li>":                         "dropped",
		"<li><code>POST /a/rules</code> — rules with <code>audit_ssh</code></li>":      "dropped",
		"<li>Other similar endpoints for these datasets: DNS, HTTP</li>":               "dropped",
		"<li><code>FETCH /zones</code></li>":                                           "dropped",
	} {
		fs := readCloudflareAPI(cfEntry("<p>Deprecation date: April 10, 2026</p>", item))
		got := "dropped"
		if len(fs) == 1 && fs[0].Dropped == "" {
			got = fs[0].Subject.Operation
		}
		if got != want {
			t.Errorf("%s: %s, want %s", item, got, want)
		}
	}
}

// Breaking only when the item says the endpoints stop and gives a full
// end-of-life date; otherwise a deprecation dated by
// the deprecation date, else the day the item was published. The
// replacement list after "Deprecated APIs:" is never read as deprecated.
func TestCloudflareDates(t *testing.T) {
	item := "<li><code>GET /old</code></li>"
	for dates, want := range map[string]string{
		"<p>Deprecation date: April 10, 2026</p><p>End of life date: September 27, 2026</p><p>After this date, the legacy endpoints will no longer be available.</p>": "breaking 2026-09-27",
		"<p>Deprecation date: April 10, 2026</p><p>End of life date: September 27, 2026</p>":                                                                          "deprecation 2026-04-10",
		"<p>End of life date: September 27, 2026</p><p>After this date, the endpoints will no longer be available.</p>":                                               "breaking 2026-09-27",
		"<p>Deprecation date: April 10, 2026</p><p>The endpoints will no longer be available after the migration.</p>":                                                "deprecation 2026-04-10",
		"<p>Deprecation date: November 2025</p><p>End of life date: January 2026</p>":                                                                                 "deprecation 2026-06-29",
		"<p>Deprecation date: April 10, 2026</p><p>End of life date: January 2026</p>":                                                                                "deprecation 2026-04-10",
		"<p>Deprecation date: April 10, 2026</p>":                                                                                                                     "deprecation 2026-04-10",
		"": "deprecation 2026-06-29",
	} {
		fs := readCloudflareAPI(cfEntry(dates, item))
		if len(fs) != 1 || fs[0].Dropped != "" {
			t.Errorf("%q: %+v, want one finding", dates, fs)
			continue
		}
		if got := string(fs[0].ChangeType) + " " + fs[0].Version; got != want {
			t.Errorf("%q: %s, want %s", dates, got, want)
		}
	}
	// Mirage's shape with a full date: the only "will be removed" is a list
	// item under "Affected APIs:", about a field, so it stays a deprecation.
	mirage := cfEntry("<p>Deprecation date: November 3, 2025</p><p>End of life date: January 15, 2026</p>", item)
	mirage.HTML += "<p>Affected APIs:</p><ul><li><code>POST /zones/{zone_id}/pagerules</code> - Mirage parameter will be removed.</li></ul>"
	if fs := readCloudflareAPI(mirage); len(fs) != 1 || fs[0].ChangeType != event.ChangeTypeDeprecation || fs[0].Version != "2025-11-03" {
		t.Errorf("a field's removal under Affected APIs: %+v, want deprecation 2025-11-03", fs)
	}
	// A paragraph that says something is removed, but not the endpoints.
	field := cfEntry("<p>Deprecation date: November 3, 2025</p><p>End of life date: January 15, 2026</p><p>The mirage setting will be removed from the dashboard.</p>", item)
	if fs := readCloudflareAPI(field); len(fs) != 1 || fs[0].ChangeType != event.ChangeTypeDeprecation {
		t.Errorf("a setting's removal: %+v, want deprecation", fs)
	}
	// A paragraph that mentions the API, but what's removed is something else
	// (the Service Key item, given a list and a full date).
	auth := cfEntry("<p>Deprecation date: March 19, 2026</p><p>End of life date: September 30, 2026</p><p>Service Key authentication for the Cloudflare API is deprecated and will be removed on September 30, 2026.</p>", item)
	if fs := readCloudflareAPI(auth); len(fs) != 1 || fs[0].ChangeType != event.ChangeTypeDeprecation {
		t.Errorf("an authentication method's removal: %+v, want deprecation", fs)
	}
	// The endpoints as what stops, as the Registrar and Zone Settings Batch
	// items say it, and as Privacy Pass says it of the API.
	for _, stop := range []string{
		"After this date, the legacy endpoints will no longer be available.",
		"After this date, the batch endpoints will no longer be available.",
		"By the end of March 2024, the current public-facing API will be removed.",
	} {
		fs := readCloudflareAPI(cfEntry("<p>End of life date: March 31, 2027</p><p>"+stop+"</p>", item))
		if len(fs) != 1 || fs[0].ChangeType != event.ChangeTypeBreaking {
			t.Errorf("%q: %+v, want breaking", stop, fs)
		}
	}
	noDates := cfEntry("", item)
	noDates.Published = ""
	if fs := readCloudflareAPI(noDates); len(fs) != 1 || fs[0].Dropped == "" {
		t.Errorf("no date at all: %+v, want dropped", fs)
	}
	if fs := readCloudflareAPI(Entry{Title: "API deprecations - Y", HTML: "<p>Affected APIs:</p><ul><li><code>GET /a</code></li></ul>"}); len(fs) != 1 || fs[0].Dropped == "" {
		t.Errorf("only affected APIs: %+v, want dropped", fs)
	}
}

// A second pinned drop test. "Cloudflare Radar: Summary and Timeseries Groups
// Endpoints" has an end-of-life date (April 15, 2026) and lists four
// operations, but says "For users calling the API directly (not via the
// Cloudflare SDK), no action is required": /radar/http/summary/device_type
// still fits the new /radar/http/summary/{dimension}. Only an SDK method
// moves, so there's no API change to report.
func TestCloudflareNoActionForAPICallersIsNoEvent(t *testing.T) {
	kept, dropped := cloudflareFindings(t)
	for _, f := range kept {
		if strings.HasPrefix(f.Subject.Operation, "GET /radar/http/summary/") || strings.HasPrefix(f.Subject.Operation, "GET /radar/http/timeseries_groups/") {
			t.Errorf("event for %s: Cloudflare says direct API callers need no action", f.Subject.Operation)
		}
	}
	var n int
	for _, f := range dropped {
		if strings.Contains(f.Dropped, "Summary and Timeseries Groups") && strings.Contains(f.Dropped, "no action is required") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d drops logged for the Radar summary item, want 1", n)
	}
}
