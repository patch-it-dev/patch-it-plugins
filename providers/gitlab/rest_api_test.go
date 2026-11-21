package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/openapitest"
)

// gitlabPlugin is the plugin as registered for GitLab, its adapter reading the
// saved copies under testdata/ from a local server instead of gitlab.com.
func gitlabPlugin(t *testing.T) (openapi.Plugin, string) {
	t.Helper()
	srv := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(srv.Close)
	tmpl := srv.URL + "/gitlab-%s.yaml"
	return openapi.Plugin{Adapter: GitLab{SpecURLTemplate: tmpl}, Keep: openapi.RequestSide}, tmpl
}

// The golden test (testdata/golden/gitlab-rest-api.json) and the pair
// fixture in one: testdata/gitlab-v18.11.0-ee.yaml and gitlab-v19.0.0-ee.yaml
// are trimmed, unedited excerpts of GitLab's OpenAPI spec at those two tags
// (checked 2026-11-15). GitLab 19.0 removed app passwords from the Bitbucket
// Cloud import API ("Using app passwords with
// Bitbucket Cloud import GitLab API", announced 18.9, on
// docs.gitlab.com/update/deprecations): bitbucket_username and
// bitbucket_app_password are gone from POST /api/v4/import/bitbucket, and
// bitbucket_email and bitbucket_api_token became required. This is the change
// the pair v18.11.0-ee → v19.0.0-ee recreates.
//
// The excerpts also keep APIEntitiesBadge, whose id went from the generator's
// default "string" to "integer" in the same release while the API kept
// returning the same JSON: the response-side noise RequestSide drops.
func TestGitLabPairRecreatesTheBitbucketAppPasswordRemoval(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	p, tmpl := gitlabPlugin(t)

	rejected, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "v18.11.0-ee", "v19.0.0-ee")
	if err != nil || rejected {
		t.Fatalf("pair: rejected=%v err=%v", rejected, err)
	}
	events, err := s.All()
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}

	oldURL := strings.Replace(tmpl, "%s", "v18.11.0-ee", 1)
	newURL := strings.Replace(tmpl, "%s", "v19.0.0-ee", 1)
	type want struct{ description, url, pointer string }
	wants := map[string]want{
		"bitbucket_username": {
			"Property bitbucket_username removed from POST /api/v4/import/bitbucket.",
			oldURL, "/components/schemas/RequestBody_a98aaa3a441e/properties/bitbucket_username",
		},
		"bitbucket_app_password": {
			"Property bitbucket_app_password removed from POST /api/v4/import/bitbucket.",
			oldURL, "/components/schemas/RequestBody_a98aaa3a441e/properties/bitbucket_app_password",
		},
		"bitbucket_email": {
			"Property bitbucket_email is now required in POST /api/v4/import/bitbucket.",
			newURL, "/components/schemas/RequestBody_f0e6af1da635/properties/bitbucket_email",
		},
		"bitbucket_api_token": {
			"Property bitbucket_api_token is now required in POST /api/v4/import/bitbucket.",
			newURL, "/components/schemas/RequestBody_f0e6af1da635/properties/bitbucket_api_token",
		},
	}
	if len(events) != len(wants) {
		for _, e := range events {
			t.Logf("got: %s", e.Description)
		}
		t.Fatalf("stored %d events, want exactly %d", len(events), len(wants))
	}
	for _, e := range events {
		w, ok := wants[e.Subject.Property]
		if !ok {
			t.Errorf("unexpected event: %s", e.Description)
			continue
		}
		if e.API != "gitlab/rest-api" || e.Version != "19.0.0" || e.PreviousVersion != "18.11.0" {
			t.Errorf("%s: api=%s version=%s previous=%s", e.Subject.Property, e.API, e.Version, e.PreviousVersion)
		}
		if string(e.Change) != "breaking" || e.Change != event.ChangeBreaking {
			t.Errorf("%s: change=%s, want breaking", e.Subject.Property, e.Change)
		}
		if e.Subject.Operation != "POST /api/v4/import/bitbucket" || e.Subject.Side != "request" {
			t.Errorf("%s: subject %+v", e.Subject.Property, e.Subject)
		}
		if e.Description != w.description {
			t.Errorf("%s: description %q, want %q", e.Subject.Property, e.Description, w.description)
		}
		if e.SourceURL != w.url || e.SourcePointer != w.pointer {
			t.Errorf("%s: cited %s#%s, want %s#%s", e.Subject.Property, e.SourceURL, e.SourcePointer, w.url, w.pointer)
		}
		// The excerpt is exactly the node the pointer names in the published
		// YAML, not in the reshaped JSON the differ read.
		file := "testdata/gitlab-v18.11.0-ee.yaml"
		if e.SourceURL == newURL {
			file = "testdata/gitlab-v19.0.0-ee.yaml"
		}
		if got := openapitest.YAMLNodeJSON(t, file, e.SourcePointer); got != e.SourceExcerpt {
			t.Errorf("%s: excerpt %s, but %s#%s holds %s", e.Subject.Property, e.SourceExcerpt, file, e.SourcePointer, got)
		}
	}

	// The golden file names the spec where GitLab publishes it, not the local
	// server that stood in for it.
	published := map[string]string{
		oldURL: fmt.Sprintf(gitlabSpecURLTemplate, "v18.11.0-ee"),
		newURL: fmt.Sprintf(gitlabSpecURLTemplate, "v19.0.0-ee"),
	}
	golden := append([]event.Event(nil), events...)
	for i := range golden {
		golden[i].SourceURL = published[golden[i].SourceURL]
	}
	kittest.Golden(t, "testdata/golden/gitlab-rest-api.json", golden)

	// Reading the same pair again must not duplicate the events.
	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "v18.11.0-ee", "v19.0.0-ee"); err != nil {
		t.Fatalf("second read of the pair: %v", err)
	}
	if again, _ := s.All(); len(again) != len(events) {
		t.Errorf("reading the pair again added events: %d -> %d", len(events), len(again))
	}
}

// Without the filter, the same pair reports the response-side retyping too:
// the test above only passes because RequestSide drops it.
func TestGitLabResponseSideNoiseIsWhatTheFilterDrops(t *testing.T) {
	ctx := context.Background()
	s := kittest.NewMem()
	p, _ := gitlabPlugin(t)
	p.Keep = nil

	if _, err := kittest.Pair(ctx, openapitest.Env(s), s, p, "v18.11.0-ee", "v19.0.0-ee"); err != nil {
		t.Fatalf("pair: %v", err)
	}
	events, _ := s.All()
	var badge bool
	for _, e := range events {
		if e.Subject.Schema == "APIEntitiesBadge" && e.Subject.Property == "id" {
			badge = true
		}
	}
	if !badge {
		t.Error("unfiltered pair has no APIEntitiesBadge.id event; the fixture no longer shows what RequestSide is for")
	}
}
