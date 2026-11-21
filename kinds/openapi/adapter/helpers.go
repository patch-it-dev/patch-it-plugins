package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"gopkg.in/yaml.v3"
)

// ServerBasePath puts the server URL's path back at the front of every path,
// so an operation is named by the path a caller sends ("GET /api/0/…"), and
// a spec that moves the server's path into its paths (or out of them) reads
// as the same operations, not as every operation removed and added again.
// Used by every adapter whose API has a path in its server URL.
//
// Sentry did move it:
// Until sentry-api-schema 2.0.1 the server was "https://{region}.sentry.io"
// and the paths started "/api/0/"; from 2.1.0 (2026-11-13) the server is
// "https://{region}.sentry.io/api/0" and the paths don't. The URLs a caller
// sends are the same, but compared as published, every operation (289 of
// them) reads as removed and added again. With the server's path moved into
// the paths, every spec names an operation by the path a caller sends,
// "GET /api/0/organizations/", as Sentry's API reference does.
//
// A spec whose server has no path is returned byte for byte. The values are
// otherwise unchanged (numbers kept as written); the path moved is recorded
// for SourcePointer.
func ServerBasePath(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing the spec: %w", err)
	}
	base := ""
	for i, s := range AsSlice(doc["servers"]) {
		u, _ := AsMap(s)["url"].(string)
		p := serverPath(u)
		if i > 0 && p != base {
			return nil, fmt.Errorf("servers with different paths (%q, %q)", base, p)
		}
		base = p
	}
	if base == "" {
		return body, nil
	}
	paths := AsMap(doc["paths"])
	moved := make(map[string]any, len(paths))
	for p, item := range paths {
		if !strings.HasPrefix(p, "/") {
			return nil, fmt.Errorf("path %q doesn't start with /", p)
		}
		moved[base+p] = item
	}
	doc["paths"] = moved
	doc[basePathKey] = base
	return json.Marshal(doc)
}

// serverPath is the path of a server URL ("/api/0" for
// "https://{region}.sentry.io/api/0", "/api/v1" for Convex's
// "{deployment-url}/api/v1", whose whole origin is a variable), without a
// trailing slash; "" for none. Read by hand: url.Parse refuses a variable in
// the host.
func serverPath(u string) string {
	_, rest, ok := strings.Cut(u, "://")
	if !ok {
		if !strings.HasPrefix(u, "{") || !strings.Contains(u, "}") {
			return ""
		}
		_, rest, _ = strings.Cut(u, "}")
		return strings.TrimRight(rest, "/")
	}
	i := strings.Index(rest, "/")
	if i < 0 {
		return ""
	}
	return strings.TrimRight(rest[i:], "/")
}

// BasePathPointer turns a pointer into a path ServerBasePath prefixed with
// the server's path back into one into the path as published. Every other
// pointer, and every pointer into a spec ServerBasePath didn't change, is
// already right.
func BasePathPointer(content []byte) func(string) string {
	var doc struct {
		Base string `json:"x-patch-it-base-path"`
	}
	_ = json.Unmarshal(content, &doc) // unreadable content: pointers stay as they are
	if doc.Base == "" {
		return func(ptr string) string { return ptr }
	}
	prefix := "/paths/" + PointerToken(doc.Base)
	return func(ptr string) string {
		rest, ok := strings.CutPrefix(ptr, prefix)
		// Every path starts with /, so what follows is "~1..."; anything
		// else is a pointer into a path that wasn't moved.
		if !ok || !strings.HasPrefix(rest, "~1") {
			return ptr
		}
		return "/paths/" + rest
	}
}

// AsSlice is v as an array, or nil.
func AsSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

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

// PointerToken encodes one JSON Pointer token (RFC 6901), as diff does.
func PointerToken(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

// AsMap is v as an object, or nil, which reads and ranges as empty.
func AsMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// Releases are a GitHub repo's releases, oldest first, the tags keep returns
// true for (nil keeps every one).
func Releases(ctx context.Context, repo string, keep func(string) bool) ([]plugin.Version, error) {
	return versions.GitHubReleases(ctx, github.Get, repo, keep)
}

// Commits are the days a file changed, each read at the day's last commit
// (Daily).
func Commits(ctx context.Context, repo, path string) ([]plugin.Version, error) {
	vs, err := versions.GitHubCommits(ctx, github.Get, repo, path)
	if err != nil {
		return nil, err
	}
	return Daily(vs), nil
}

// Daily keeps one version per day of vs (oldest first, IDs dates, as
// versions.GitHubCommits lists them): the day's last commit, so a spec
// committed several times a day has one version per day, each ID once.
func Daily(vs []plugin.Version) []plugin.Version {
	var out []plugin.Version
	for _, v := range vs {
		if n := len(out); n > 0 && out[n-1].ID == v.ID {
			out[n-1] = v
			continue
		}
		out = append(out, v)
	}
	return out
}

// basePathKey is the root key ServerBasePath records the server path it moved
// into the paths under. The differ reads only paths and components, so the
// key is never diffed.
const basePathKey = "x-patch-it-base-path"
