package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"gopkg.in/yaml.v3"
)

// Variables, not constants, only so tests can point them at a local server.
var (
	openaiRepo = "openai/openai-openapi"
	openaiPath = "openapi.yaml"
	// openaiCommitsURL lists the commits that changed the spec, newest
	// first; with per_page=1, the newest.
	openaiCommitsURL = "https://api.github.com/repos/openai/openai-openapi/commits?path=openapi.yaml&per_page=1"
	// openaiCommitURL is one commit, for its date.
	openaiCommitURL = "https://api.github.com/repos/openai/openai-openapi/commits/%s"
	// openaiSpecURLTemplate is the OpenAPI description of the OpenAI API at
	// one commit of openai/openai-openapi (MIT), the spec OpenAI's own SDKs
	// are generated from. The repo commits it several times a day (589
	// commits that changed openapi.yaml from 2022-01-19 to 2026-11-21,
	// checked 2026-11-21) and tags it seldom (1.3.0, 2.0.0), so the commits
	// are the versions. openapi.json beside it only exists from 2026-07-16:
	// the YAML has the whole history.
	openaiSpecURLTemplate = "https://raw.githubusercontent.com/openai/openai-openapi/%s/openapi.yaml"
)

// OpenAI adapts the OpenAI API's spec in openai/openai-openapi for the
// openapi plugin. A tag here is a commit SHA.
type OpenAI struct{}

var _ Adapter = OpenAI{}

func (OpenAI) API() string { return "openai/api" }

// LatestTag is the SHA of the newest commit that changed the spec.
func (OpenAI) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, openaiCommitsURL)
	if err != nil {
		return "", fmt.Errorf("looking up openai's latest spec commit: %w", err)
	}
	return sha, nil
}

// Versions are the days the spec changed, each read at that day's last
// commit (Daily): several commits a day is the norm here, and a day is the
// version a caller can name.
func (OpenAI) Versions(ctx context.Context) ([]plugin.Version, error) {
	return commits(ctx, openaiRepo, openaiPath)
}

// Fetch downloads the spec at commit sha, converts it from YAML to JSON (what
// diff.Parse reads) and moves the server's /v1 into the paths: see
// ServerBasePath. Version is the commit's date (YYYY-MM-DD): info.version
// ("2.3.0") isn't bumped with each change, and the API has no dated versions
// of its own.
func (OpenAI) Fetch(ctx context.Context, sha string) (Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(openaiCommitURL, sha))
	if err != nil {
		return Spec{}, fmt.Errorf("looking up openai spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return Spec{}, fmt.Errorf("github commit %s has no sha", sha)
	}
	url := fmt.Sprintf(openaiSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching openai spec %s: %w", sha, err)
	}
	content, err := YAMLToJSON(body)
	if err != nil {
		return Spec{}, fmt.Errorf("openai spec %s: %w", sha, err)
	}
	// The server is "https://api.openai.com/v1" and the paths don't repeat
	// /v1 (checked 2026-11-21): moved, an operation is named by the path a
	// caller sends ("POST /v1/audio/translations").
	if content, err = ServerBasePath(content); err != nil {
		return Spec{}, fmt.Errorf("openai spec %s: %w", sha, err)
	}
	return Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (OpenAI) SourcePointer(content []byte) func(string) string { return BasePathPointer(content) }

var _ Reshaper = OpenAI{}

// YAMLToJSON converts a YAML spec to JSON: diff.Parse reads JSON only, and
// normalize quotes the node it cites re-serialized, never the original bytes,
// so nothing quoted changes.
func YAMLToJSON(body []byte) ([]byte, error) {
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return json.Marshal(doc)
}
