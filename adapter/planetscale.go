package adapter

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
	"gopkg.in/yaml.v3"
)

const (
	// planetscaleRepo is PlanetScale's Terraform provider, which keeps a copy
	// of the PlanetScale API spec at planetscaleSpecPath.
	planetscaleRepo = "planetscale/terraform-provider-planetscale"
	// planetscaleSpecPath is the spec in that repo: a copy of
	// https://planetscale.com/docs/openapi.yaml, which has no versions,
	// committed when it changes.
	planetscaleSpecPath = "schemas/openapi.yaml"
)

// PlanetScale adapts the PlanetScale API spec for the openapi plugin. The
// repo tags no spec releases, so a tag here is the commit that last changed
// the spec, as for Supabase.
//
// The zero value reads GitHub. The fields point it at another server, for
// tests.
type PlanetScale struct {
	APIBase string // GitHub's REST API; "" is https://api.github.com
	RawBase string // files at a commit; "" is https://raw.githubusercontent.com
}

var _ Adapter = PlanetScale{}

func (PlanetScale) API() string { return "planetscale" }

func (p PlanetScale) api(path string) string {
	return cmp.Or(p.APIBase, "https://api.github.com") + "/repos/" + planetscaleRepo + path
}

// LatestTag is the SHA of the newest commit that changed the spec.
func (p PlanetScale) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, p.api("/commits?path="+planetscaleSpecPath+"&per_page=1"), githubHeaders())
	if err != nil {
		return "", fmt.Errorf("looking up planetscale's latest spec commit: %w", err)
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("parsing github commits response: %w", err)
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("github lists no commit that changed the planetscale spec")
	}
	return commits[0].SHA, nil
}

// Fetch downloads the spec at commit sha and converts it from YAML to JSON,
// which is what diff.Parse reads (as for Algolia).
//
// Version is the commit's date (YYYY-MM-DD): the API has no version of its
// own (`info.version` is a fixed "v1", checked 2026-11-19), so the date the
// spec changed is the closest thing to "the API version the change arrived
// in".
//
// A file that isn't a spec is an error, not an empty API: diffed against the
// last one, it would read as every operation removed. The commits that
// changed the file include one that deleted it (f33a0c3b35, 2025-12-04;
// it came back the same day), where the raw URL answers 404: fetch.Get's
// error. A file served 200 that has no paths (emptied, or a stub page like
// "404: Not Found", which YAML reads as a one-key mapping) is refused here.
func (p PlanetScale) Fetch(ctx context.Context, sha string) (Spec, error) {
	meta, err := fetch.Get(ctx, p.api("/commits/"+sha), githubHeaders())
	if err != nil {
		return Spec{}, fmt.Errorf("looking up planetscale spec commit %s: %w", sha, err)
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
	url := cmp.Or(p.RawBase, "https://raw.githubusercontent.com") + "/" + planetscaleRepo + "/" + commit.SHA + "/" + planetscaleSpecPath
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching planetscale spec %s: %w", sha, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return Spec{}, fmt.Errorf("parsing planetscale spec %s as yaml: %w", sha, err)
	}
	if paths, _ := doc["paths"].(map[string]any); len(paths) == 0 {
		return Spec{}, fmt.Errorf("planetscale spec %s has no paths: not a spec", sha)
	}
	content, err := json.Marshal(doc)
	if err != nil {
		return Spec{}, fmt.Errorf("converting planetscale spec %s to json: %w", sha, err)
	}
	// The server is "https://api.planetscale.com/v1" and the paths leave out
	// /v1 (all 32 versions, checked 2026-11-19). Moved into the paths, an
	// operation is named by the path a caller sends, and a spec that ever
	// moved it (as Sentry's did) can't read as every deprecated operation
	// removed.
	if content, err = ServerBasePath(content); err != nil {
		return Spec{}, fmt.Errorf("planetscale spec %s: %w", sha, err)
	}
	return Spec{Content: content, Version: commit.Commit.Committer.Date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (PlanetScale) SourcePointer(content []byte) func(string) string { return BasePathPointer(content) }

var _ Reshaper = PlanetScale{}
