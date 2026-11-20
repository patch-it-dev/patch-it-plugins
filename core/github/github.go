// Package github is the one place a plugin asks GitHub's REST API where a
// source's newest version is: the newest release, the newest commit that
// changed a path, and one commit's full SHA and date. Every plugin that
// follows a GitHub repo calls these instead of decoding GitHub's responses
// itself.
//
// Each function takes how to fetch (Get, or a plugin's env.Get) and the full
// API URL, so a test serves it from saved pages or a local server.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
)

// API is GitHub's REST API.
const API = "https://api.github.com"

// Raw serves a repo's files at a ref, with no auth.
const Raw = "https://raw.githubusercontent.com"

// Headers adds GITHUB_TOKEN as a bearer token when set, which raises the
// GitHub API's rate limit. Never required.
func Headers() map[string]string {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

// Getter fetches one URL: Get, or plugin.Env.Get.
type Getter func(ctx context.Context, url string) ([]byte, error)

// Get is fetch.Get with Headers: GITHUB_TOKEN, when set, raises the rate limit
// (never required).
func Get(ctx context.Context, url string) ([]byte, error) {
	return fetch.Get(ctx, url, Headers())
}

// LatestRelease reads a repo's releases/latest (url) and returns its tag.
// GitHub's "latest" skips drafts and prereleases.
func LatestRelease(ctx context.Context, get Getter, url string) (string, error) {
	body, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing github release response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("github release response had no tag_name")
	}
	return release.TagName, nil
}

// LatestCommit reads a commit list (url: commits?path=<file>&per_page=1, newest
// first) and returns the newest commit's SHA, or an error when it lists none.
func LatestCommit(ctx context.Context, get Getter, url string) (string, error) {
	body, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("parsing github commits response: %w", err)
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("github lists no commit")
	}
	return commits[0].SHA, nil
}

// Commit reads one commit (url: commits/<ref>) and returns its full SHA and
// its committer date. A response with no date is an error; one with no SHA
// returns "" (a caller that pins by SHA checks it).
func Commit(ctx context.Context, get Getter, url string) (sha string, date time.Time, err error) {
	body, err := get(ctx, url)
	if err != nil {
		return "", time.Time{}, err
	}
	var c struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return "", time.Time{}, fmt.Errorf("parsing github commit: %w", err)
	}
	if c.Commit.Committer.Date.IsZero() {
		return "", time.Time{}, fmt.Errorf("github commit has no date")
	}
	return c.SHA, c.Commit.Committer.Date, nil
}

// Head reads one commit (url: commits/<branch or ref>) and returns its full
// SHA: the newest commit on a branch, when the source has no releases.
func Head(ctx context.Context, get Getter, url string) (string, error) {
	body, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	var c struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return "", fmt.Errorf("parsing github commit: %w", err)
	}
	if c.SHA == "" {
		return "", fmt.Errorf("the commit lookup named no sha")
	}
	return c.SHA, nil
}
