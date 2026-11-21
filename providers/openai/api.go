package openai

import (
	"context"
	"fmt"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
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

var _ adapter.Adapter = OpenAI{}

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
	return adapter.Commits(ctx, openaiRepo, openaiPath)
}

// Fetch downloads the spec at commit sha, converts it from YAML to JSON (what
// diff.Parse reads) and moves the server's /v1 into the paths: see
// ServerBasePath. Version is the commit's date (YYYY-MM-DD): info.version
// ("2.3.0") isn't bumped with each change, and the API has no dated versions
// of its own.
func (OpenAI) Fetch(ctx context.Context, sha string) (adapter.Spec, error) {
	fullSHA, date, err := github.Commit(ctx, github.Get, fmt.Sprintf(openaiCommitURL, sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up openai spec commit %s: %w", sha, err)
	}
	if fullSHA == "" {
		return adapter.Spec{}, fmt.Errorf("github commit %s has no sha", sha)
	}
	url := fmt.Sprintf(openaiSpecURLTemplate, fullSHA)
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching openai spec %s: %w", sha, err)
	}
	content, err := adapter.YAMLToJSON(body)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("openai spec %s: %w", sha, err)
	}
	// The server is "https://api.openai.com/v1" and the paths don't repeat
	// /v1 (checked 2026-11-21): moved, an operation is named by the path a
	// caller sends ("POST /v1/audio/translations").
	if content, err = adapter.ServerBasePath(content); err != nil {
		return adapter.Spec{}, fmt.Errorf("openai spec %s: %w", sha, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (OpenAI) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var (
	_ adapter.Reshaper = OpenAI{}
	_ adapter.Lister   = OpenAI{}
)

// openaiDiff is how OpenAI's spec says what it says:
//
//   - Deprecations use the `deprecated` flag: in the spec of 2026-11-21, 17
//     operations carry it and none is called deprecated only in its summary
//     or description (one description mentions deprecated models, not the
//     operation). A flagged operation's description says why and when ("The
//     endpoint and model are deprecated and scheduled to shut down on
//     February 26, 2027"), so the event quotes the whole operation.
//   - Path parameters could be renamed without the URL changing; matching by
//     path template keeps a rename from reading as a removal.
var openaiDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// openaiKeep keeps operation deprecations, and removals of operations the old
// spec had marked deprecated. Everything else is dropped: the spec is
// committed several times a day, and most of what changes is the spec's own
// shape, not the API (enum lists rewritten as const, schemas regenerated),
// or work in progress pulled the day after it was added (the session
// spending-limit schemas, added on 2026-11-17 and gone on 2026-11-18: the
// drop test). New operations aren't something to fix either.
var openaiKeep = openapi.DeprecationsOnly(openaiDiff)
