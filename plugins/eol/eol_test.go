package eol

import (
	"fmt"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kit/kittest"
)

func TestRegistersItself(t *testing.T) {
	kittest.Registered(t, Plugin{Page: Salesforce})
}

// The golden test: Salesforce's own end-of-life policy page (checked
// 2026-11-20, saved 2026-10-31), real and unedited but for dropping the
// surrounding navigation chrome. It already holds two real, past retirements
// (versions 7.0 through 20.0, retired as of Summer '22; versions 21.0
// through 30.0, retired as of Summer '25) and one real, scheduled one
// (versions 31.0 through 40.0, deprecated from Summer '27, retired from
// Summer '28): 34 events, and the still-"Supported." 41.0 through 67.0 band
// names none, the pinned drop test.
func TestGolden(t *testing.T) {
	saved := kittest.Pages{Salesforce.URL: "testdata/salesforce-api-rest-eol-2026-10-31.html"}
	archive := &kittest.Archive{T: t, Pages: saved}
	got := kittest.IngestWith(t, kittest.Env(t, saved), Plugin{Page: Salesforce, Archiver: archive})
	if archive.Calls != 1 {
		t.Errorf("%d snapshots, want 1: the second run's page is unchanged", archive.Calls)
	}
	kittest.Golden(t, "testdata/golden/salesforce.json", got)
}

// The pinned drop test: "Versions 41.0 through 67.0: Supported." is a real
// row on the page, and it must name no version.
func TestDropsAStillSupportedBand(t *testing.T) {
	page := kittest.Read(t, "testdata/salesforce-api-rest-eol-2026-10-31.html")
	found, err := Findings(page)
	if err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{}
	for v := 41; v <= 67; v++ {
		supported[fmt.Sprintf("%d.0", v)] = true
	}
	var dropped bool
	for _, f := range found {
		if f.Dropped == "" && supported[f.Version] {
			t.Errorf("version %s got an event, want none: it's still Supported", f.Version)
		}
		if f.Dropped != "" && f.Row == "Versions 41.0 through 67.0" {
			dropped = true
		}
	}
	if !dropped {
		t.Error("the still-supported band wasn't found and dropped")
	}
}
