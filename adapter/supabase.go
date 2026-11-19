package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	// supabaseCommitsURL lists the commits that changed the Management API
	// spec in supabase/supabase, newest first; with per_page=1, the newest.
	supabaseCommitsURL = "https://api.github.com/repos/supabase/supabase/commits?path=apps/docs/spec/api_v1_openapi.json&per_page=1"
	// supabaseCommitURL is one commit, for its date.
	supabaseCommitURL = "https://api.github.com/repos/supabase/supabase/commits/%s"
	// supabaseSpecURLTemplate is the Management API spec (api.supabase.com)
	// at one commit. Supabase serves it live at
	// https://api.supabase.com/api/v1-json, which has no versions; the docs
	// copy in supabase/supabase is the same spec, committed when it changes
	// (90 commits from 2024-05-31 to 2026-11-18), and a commit never changes.
	supabaseSpecURLTemplate = "https://raw.githubusercontent.com/supabase/supabase/%s/apps/docs/spec/api_v1_openapi.json"
)

// Supabase adapts the Supabase Management API spec in supabase/supabase for
// the openapi plugin. The repo tags no spec releases, so a tag here is the
// commit that last changed the spec.
type Supabase struct{}

var _ Adapter = Supabase{}

func (Supabase) API() string { return "supabase" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Supabase) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, supabaseCommitsURL, githubHeaders())
	if err != nil {
		return "", fmt.Errorf("looking up supabase's latest spec commit: %w", err)
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("parsing github commits response: %w", err)
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("github lists no commit that changed the supabase spec")
	}
	return commits[0].SHA, nil
}

// Fetch downloads the spec at commit sha.
//
// Version is the commit's date (YYYY-MM-DD): the Management API has no
// version of its own (`/v1`, info.version a fixed "1.0.0", checked
// 2026-11-19), so the date the spec changed is the closest thing to "the API
// version the change arrived in".
func (Supabase) Fetch(ctx context.Context, sha string) (Spec, error) {
	meta, err := fetch.Get(ctx, fmt.Sprintf(supabaseCommitURL, sha), githubHeaders())
	if err != nil {
		return Spec{}, fmt.Errorf("looking up supabase spec commit %s: %w", sha, err)
	}
	var commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(meta, &commit); err != nil {
		return Spec{}, fmt.Errorf("parsing github commit %s: %w", sha, err)
	}
	if commit.SHA == "" || commit.Commit.Committer.Date.IsZero() {
		return Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := fmt.Sprintf(supabaseSpecURLTemplate, commit.SHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching supabase spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("supabase spec %s is not valid json", sha)
	}
	return Spec{Content: body, Version: commit.Commit.Committer.Date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}
