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
	// neonCommitsURL lists the commits that changed the Neon API spec in
	// neondatabase/neon-pkgs, newest first; with per_page=1, the newest.
	neonCommitsURL = "https://api.github.com/repos/neondatabase/neon-pkgs/commits?path=packages/sdk/spec/neon-openapi.json&per_page=1"
	// neonCommitURL is one commit, for its date.
	neonCommitURL = "https://api.github.com/repos/neondatabase/neon-pkgs/commits/%s"
	// neonSpecURLTemplate is the Neon API spec (console.neon.tech/api/v2) at
	// one commit. Neon serves it live at
	// https://neon.com/api_spec/release/v2.json, which has no versions; the
	// copy in neondatabase/neon-pkgs is pulled from there and committed when
	// a daily drift check sees it change (23 commits from 2026-06-24 to
	// 2026-11-17), and a commit never changes.
	neonSpecURLTemplate = "https://raw.githubusercontent.com/neondatabase/neon-pkgs/%s/packages/sdk/spec/neon-openapi.json"
)

// Neon adapts the Neon API spec in neondatabase/neon-pkgs for the openapi
// plugin. The repo tags no spec releases, so a tag here is the commit that
// last changed the spec.
type Neon struct{}

var _ Adapter = Neon{}

func (Neon) API() string { return "neon" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (Neon) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, neonCommitsURL, githubHeaders())
	if err != nil {
		return "", fmt.Errorf("looking up neon's latest spec commit: %w", err)
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("parsing github commits response: %w", err)
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("github lists no commit that changed the neon spec")
	}
	return commits[0].SHA, nil
}

// Fetch downloads the spec at commit sha and moves the server URL's path into
// the paths: see ServerBasePath.
//
// Version is the commit's date (YYYY-MM-DD): the API's only version is the
// `/api/v2` in its URL (info.version a fixed "v2" in all 23 commits, checked
// 2026-11-19), so the date the spec changed is the closest thing to "the API
// version the change arrived in".
func (Neon) Fetch(ctx context.Context, sha string) (Spec, error) {
	meta, err := fetch.Get(ctx, fmt.Sprintf(neonCommitURL, sha), githubHeaders())
	if err != nil {
		return Spec{}, fmt.Errorf("looking up neon spec commit %s: %w", sha, err)
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
	url := fmt.Sprintf(neonSpecURLTemplate, commit.SHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching neon spec %s: %w", sha, err)
	}
	if !json.Valid(body) {
		return Spec{}, fmt.Errorf("neon spec %s is not valid json", sha)
	}
	// The server is "https://console.neon.tech/api/v2" and the paths don't
	// repeat /api/v2 (all 23 commits, checked 2026-11-19). Moved into the
	// paths, an operation is named by the path a caller sends, and a spec
	// that ever moved it (as Sentry's did) can't read as every deprecated
	// operation removed.
	content, err := ServerBasePath(body)
	if err != nil {
		return Spec{}, fmt.Errorf("neon spec %s: %w", sha, err)
	}
	return Spec{Content: content, Version: commit.Commit.Committer.Date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Neon) SourcePointer(content []byte) func(string) string { return BasePathPointer(content) }

var _ Reshaper = Neon{}
