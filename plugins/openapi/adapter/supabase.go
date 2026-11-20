package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
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

func (Supabase) API() string { return "supabase/management-api" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Supabase) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, supabaseCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up supabase's latest spec commit: %w", err)
	}
	return sha, nil
}

// Fetch downloads the spec at commit sha.
//
// Version is the commit's date (YYYY-MM-DD): the Management API has no
// version of its own (`/v1`, info.version a fixed "1.0.0", checked
// 2026-11-19), so the date the spec changed is the closest thing to "the API
// version the change arrived in".
func (Supabase) Fetch(ctx context.Context, sha string) (Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(supabaseCommitURL, sha))
	if err != nil {
		return Spec{}, fmt.Errorf("looking up supabase spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return Spec{}, fmt.Errorf("github commit %s has no sha or date", sha)
	}
	// Pin to the full SHA even when given a short one: the citation must
	// name exactly one document.
	url := fmt.Sprintf(supabaseSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching supabase spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("supabase spec %s is not valid json", sha)
	}
	return Spec{Content: body, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}
