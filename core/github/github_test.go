package github

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// pages serves canned API responses by URL.
func pages(m map[string]string) Getter {
	return func(_ context.Context, url string) ([]byte, error) {
		b, ok := m[url]
		if !ok {
			return nil, fmt.Errorf("no page %s", url)
		}
		return []byte(b), nil
	}
}

func TestLatestRelease(t *testing.T) {
	get := pages(map[string]string{"r": `{"tag_name":"v2468"}`, "empty": `{}`, "bad": `<html>`})
	if tag, err := LatestRelease(context.Background(), get, "r"); err != nil || tag != "v2468" {
		t.Errorf("got %q, %v", tag, err)
	}
	for _, url := range []string{"empty", "bad", "missing"} {
		if _, err := LatestRelease(context.Background(), get, url); err == nil {
			t.Errorf("%s: no error", url)
		}
	}
}

func TestLatestCommit(t *testing.T) {
	get := pages(map[string]string{"c": `[{"sha":"abc"},{"sha":"def"}]`, "none": `[]`})
	if sha, err := LatestCommit(context.Background(), get, "c"); err != nil || sha != "abc" {
		t.Errorf("got %q, %v", sha, err)
	}
	if _, err := LatestCommit(context.Background(), get, "none"); err == nil {
		t.Error("an empty list gave no error")
	}
}

func TestCommitAndHead(t *testing.T) {
	get := pages(map[string]string{
		"c":      `{"sha":"abc","commit":{"committer":{"date":"2026-11-19T10:00:00Z"}}}`,
		"nodate": `{"sha":"abc"}`,
	})
	sha, date, err := Commit(context.Background(), get, "c")
	if err != nil || sha != "abc" || !date.Equal(time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("got %q %v %v", sha, date, err)
	}
	if _, _, err := Commit(context.Background(), get, "nodate"); err == nil {
		t.Error("a commit with no date gave no error")
	}
	if sha, err := Head(context.Background(), get, "nodate"); err != nil || sha != "abc" {
		t.Errorf("Head: %q %v", sha, err)
	}
}
