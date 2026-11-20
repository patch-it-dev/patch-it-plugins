package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/plugins/openapi/adapter"
	"gopkg.in/yaml.v3"
)

// gitlabPlugin is the plugin as registered for GitLab, its adapter reading the
// saved copies under testdata/ from a local server instead of gitlab.com.
func gitlabPlugin(t *testing.T) (Plugin, string) {
	t.Helper()
	srv := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(srv.Close)
	tmpl := srv.URL + "/gitlab-%s.yaml"
	return Plugin{Adapter: adapter.GitLab{SpecURLTemplate: tmpl}, Keep: requestSide}, tmpl
}

// The golden test and the replay fixture in one: testdata/gitlab-v18.11.0-ee.yaml
// and gitlab-v19.0.0-ee.yaml are trimmed, unedited excerpts of GitLab's
// OpenAPI spec at those two tags (checked 2026-11-15). GitLab 19.0 removed app
// passwords from the Bitbucket Cloud import API ("Using app passwords with
// Bitbucket Cloud import GitLab API", announced 18.9, on
// docs.gitlab.com/update/deprecations): bitbucket_username and
// bitbucket_app_password are gone from POST /api/v4/import/bitbucket, and
// bitbucket_email and bitbucket_api_token became required. This is the pair
// the plugin's Replay from v18.11.0-ee to v19.0.0-ee recreates.
//
// The excerpts also keep APIEntitiesBadge, whose id went from the generator's
// default "string" to "integer" in the same release while the API kept
// returning the same JSON: the response-side noise requestSide drops.
func TestGitLabReplayRecreatesTheBitbucketAppPasswordRemoval(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	p, tmpl := gitlabPlugin(t)

	rejected, err := replayWith(ctx, s, p, "v18.11.0-ee", "v19.0.0-ee")
	if err != nil || rejected {
		t.Fatalf("replay: rejected=%v err=%v", rejected, err)
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
		if e.API != "gitlab" || e.Version != "19.0.0" || e.PreviousVersion != "18.11.0" {
			t.Errorf("%s: api=%s version=%s previous=%s", e.Subject.Property, e.API, e.Version, e.PreviousVersion)
		}
		if string(e.ChangeType) != "breaking" || !e.Breaking {
			t.Errorf("%s: change_type=%s breaking=%v, want breaking", e.Subject.Property, e.ChangeType, e.Breaking)
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
		if got := yamlNodeJSON(t, file, e.SourcePointer); got != e.SourceExcerpt {
			t.Errorf("%s: excerpt %s, but %s#%s holds %s", e.Subject.Property, e.SourceExcerpt, file, e.SourcePointer, got)
		}
	}

	// Replaying the same pair again must not duplicate the events.
	if _, err := replayWith(ctx, s, p, "v18.11.0-ee", "v19.0.0-ee"); err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if again, _ := s.All(); len(again) != len(events) {
		t.Errorf("repeating the replay added events: %d -> %d", len(events), len(again))
	}
}

// Without the filter, the same pair reports the response-side retyping too:
// the test above only passes because requestSide drops it.
func TestGitLabResponseSideNoiseIsWhatTheFilterDrops(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	p, _ := gitlabPlugin(t)
	p.Keep = nil

	if _, err := replayWith(ctx, s, p, "v18.11.0-ee", "v19.0.0-ee"); err != nil {
		t.Fatalf("replay: %v", err)
	}
	events, _ := s.All()
	var badge bool
	for _, e := range events {
		if e.Subject.Schema == "APIEntitiesBadge" && e.Subject.Property == "id" {
			badge = true
		}
	}
	if !badge {
		t.Error("unfiltered replay has no APIEntitiesBadge.id event; the fixture no longer shows what requestSide is for")
	}
}

// yamlNodeJSON resolves an RFC 6901 pointer in a YAML file and serializes the
// node the way normalize serializes excerpts.
func yamlNodeJSON(t *testing.T, file, pointer string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var node any
	if err := yaml.Unmarshal(data, &node); err != nil {
		t.Fatal(err)
	}
	for _, tok := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch n := node.(type) {
		case map[string]any:
			node = n[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i >= len(n) {
				return "<no node>"
			}
			node = n[i]
		default:
			return "<no node>"
		}
	}
	b, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
