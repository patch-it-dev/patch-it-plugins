package adapter

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
	"gopkg.in/yaml.v3"
)

const (
	// gitlabTagsURL lists gitlab-org/gitlab's tags, newest version first. One
	// page is enough: release candidates and older lines' patch releases share
	// it, but the newest stable release is always among the first hundred.
	gitlabTagsURL = "https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/repository/tags?order_by=version&sort=desc&per_page=100"
	// gitlabSpecURLTemplate is the OpenAPI 3.0 description GitLab generates
	// from its REST API code and commits to the repo as
	// doc/api/openapi/openapi_v3.yaml (checked 2026-11-15). It is present at
	// every release tag from v18.10.0-ee on; earlier tags carry only the partial
	// Swagger 2.0 file (openapi_v2.yaml), which this adapter doesn't read.
	//
	// Read through the Repository Files API, which returns the file's exact
	// bytes (compared with the /-/raw/ URL at v19.0.0-ee, 2026-11-15), rather
	// than /-/raw/: both are disallowed by gitlab.com's robots.txt, which is
	// written for search crawlers, but /api/v4 is the documented interface for
	// programs and has a published rate limit.
	gitlabSpecURLTemplate = "https://gitlab.com/api/v4/projects/gitlab-org%%2Fgitlab/repository/files/doc%%2Fapi%%2Fopenapi%%2Fopenapi_v3.yaml/raw?ref=%s"
)

// gitlabRelease is a stable GitLab release tag: "v19.0.0-ee". Release
// candidates ("v19.5.0-rc43-ee") and the FOSS mirror's tags don't match.
var gitlabRelease = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)-ee$`)

// GitLab adapts the OpenAPI spec in gitlab-org/gitlab for the openapi plugin.
// The REST API is versioned by path (/api/v4) and GitLab removes things from it
// only in a major release, so the spec at each release tag is what to diff.
//
// The zero value reads gitlab.com. The fields point it at another server, for
// tests.
type GitLab struct {
	TagsURL         string // the tags listing; "" is gitlabTagsURL
	SpecURLTemplate string // the spec, with %s for the tag; "" is gitlabSpecURLTemplate
}

var _ Adapter = GitLab{}

func (GitLab) API() string { return "gitlab" }

// LatestTag is the highest stable release tag, e.g. "v19.5.5-ee". The tags API
// sorts by version, but its handling of the "-ee" suffix and of release
// candidates isn't documented, so the tags are filtered and sorted here.
func (g GitLab) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, cmp.Or(g.TagsURL, gitlabTagsURL), nil)
	if err != nil {
		return "", fmt.Errorf("listing gitlab's tags: %w", err)
	}
	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &tags); err != nil {
		return "", fmt.Errorf("parsing gitlab tags response: %w", err)
	}
	type release struct {
		name string
		v    [3]int
	}
	var releases []release
	for _, t := range tags {
		m := gitlabRelease.FindStringSubmatch(t.Name)
		if m == nil {
			continue
		}
		var r release
		r.name = t.Name
		for i := range r.v {
			r.v[i], _ = strconv.Atoi(m[i+1])
		}
		releases = append(releases, r)
	}
	if len(releases) == 0 {
		return "", fmt.Errorf("gitlab tags response had no stable release tag")
	}
	newest := slices.MaxFunc(releases, func(a, b release) int {
		return cmp.Or(cmp.Compare(a.v[0], b.v[0]), cmp.Compare(a.v[1], b.v[1]), cmp.Compare(a.v[2], b.v[2]))
	})
	return newest.name, nil
}

// Fetch downloads the spec at tag and converts it from YAML to JSON, since
// diff.Parse only reads JSON. Excerpts are re-serialized from the node they
// quote, as for every adapter, so the conversion changes no provenance.
//
// Version is the GitLab version the tag names, "19.0.0" for "v19.0.0-ee",
// not info.version: at release tags that is the fixed path version "v4"
// (checked v18.11.0-ee and v19.0.0-ee, 2026-11-15), which can't tell two
// releases apart. The bare version is also what a self-managed instance
// reports and what patch-it's package gate can read; it refuses "v19.0.0-ee".
// A tag that isn't a release (a branch, for a replay) is its own version.
func (g GitLab) Fetch(ctx context.Context, tag string) (Spec, error) {
	url := fmt.Sprintf(cmp.Or(g.SpecURLTemplate, gitlabSpecURLTemplate), tag)
	body, err := fetch.Get(ctx, url, nil)
	if err != nil {
		return Spec{}, fmt.Errorf("fetching gitlab spec %s: %w", tag, err)
	}
	var doc any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return Spec{}, fmt.Errorf("parsing gitlab spec %s as yaml: %w", tag, err)
	}
	m, ok := stringKeys(doc).(map[string]any)
	if !ok {
		return Spec{}, fmt.Errorf("gitlab spec %s is not a yaml mapping", tag)
	}
	inlineRequestBodies(m)
	content, err := json.Marshal(m)
	if err != nil {
		return Spec{}, fmt.Errorf("converting gitlab spec %s to json: %w", tag, err)
	}
	return Spec{Content: content, Version: gitlabVersion(tag), SourceURL: url}, nil
}

// gitlabVersion is "19.0.0" for the release tag "v19.0.0-ee".
func gitlabVersion(tag string) string {
	if m := gitlabRelease.FindStringSubmatch(tag); m != nil {
		return m[1] + "." + m[2] + "." + m[3]
	}
	return tag
}

// gitlabBodyRef is a request body's schema as GitLab's generator names it: a
// hash of its content, "#/components/schemas/RequestBody_6ef4e663c0d4".
var gitlabBodyRef = regexp.MustCompile(`^#/components/schemas/(RequestBody_[0-9a-f]+)$`)

// gitlabInlined is the root key Fetch records, for each request body it
// inlined, where the body now is and where it is in the published file. The
// differ reads only paths and components, so the key is never diffed.
const gitlabInlined = "x-patch-it-inlined"

// inlineRequestBodies puts every operation's request body schema back under
// its operation, where Stripe's spec has it and the differ reads request
// sides, and drops the RequestBody_<hash> components.
//
// This isn't following $ref to diff: the name is a hash of the body, not a
// type anyone wrote, so any change to a body renames it. Left as a ref, adding
// one optional parameter would read as "the body's type changed" plus a
// schema removed and another added, and the parameter itself would never be
// named. Only these refs are replaced, one level deep; every $ref inside a
// body still points where it did and is compared as a ref.
func inlineRequestBodies(doc map[string]any) {
	schemas := asMap(asMap(doc["components"])["schemas"])
	if schemas == nil {
		return
	}
	inlined := map[string]any{}
	for path, item := range asMap(doc["paths"]) {
		for method, op := range asMap(item) {
			for mediaType, media := range asMap(asMap(asMap(op)["requestBody"])["content"]) {
				mm := asMap(media)
				ref := asMap(mm["schema"])
				r, _ := ref["$ref"].(string)
				g := gitlabBodyRef.FindStringSubmatch(r)
				if g == nil || len(ref) != 1 {
					continue
				}
				body, ok := schemas[g[1]]
				if !ok {
					continue
				}
				mm["schema"] = deepCopy(body)
				at := "/paths/" + pointerToken(path) + "/" + pointerToken(method) +
					"/requestBody/content/" + pointerToken(mediaType) + "/schema"
				inlined[at] = "/components/schemas/" + pointerToken(g[1])
			}
		}
	}
	for name := range schemas {
		if gitlabBodyRef.MatchString("#/components/schemas/" + name) {
			delete(schemas, name)
		}
	}
	if len(inlined) > 0 {
		doc[gitlabInlined] = inlined
	}
}

// SourcePointer turns a pointer into an inlined request body back into one
// into the RequestBody_<hash> schema it came from, which is where the evidence
// is in the published YAML. Every other pointer is already right.
func (GitLab) SourcePointer(content []byte) func(string) string {
	var doc struct {
		Inlined map[string]string `json:"x-patch-it-inlined"`
	}
	_ = json.Unmarshal(content, &doc) // unreadable content: pointers stay as they are
	return func(ptr string) string {
		// The longest recorded prefix wins; one ends at a token boundary, so
		// ".../schema" never matches ".../schemaX".
		best := ""
		for at := range doc.Inlined {
			if (ptr == at || strings.HasPrefix(ptr, at+"/")) && len(at) > len(best) {
				best = at
			}
		}
		if best == "" {
			return ptr
		}
		return doc.Inlined[best] + ptr[len(best):]
	}
}

var _ Reshaper = GitLab{}

// pointerToken encodes one JSON Pointer token (RFC 6901), as diff does.
func pointerToken(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

// asMap is v as an object, or nil, which reads and ranges as empty.
func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// deepCopy copies a decoded YAML node, so two operations sharing a body don't
// share one map in the document.
func deepCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			m[k] = deepCopy(e)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, e := range t {
			s[i] = deepCopy(e)
		}
		return s
	default:
		return v
	}
}

// stringKeys turns every YAML mapping into one JSON can hold. yaml.v3 decodes a
// mapping with any non-string key (an unquoted response code, 200:) as
// map[any]any, which json.Marshal refuses.
func stringKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			t[k] = stringKeys(e)
		}
		return t
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			m[fmt.Sprint(k)] = stringKeys(e)
		}
		return m
	case []any:
		for i, e := range t {
			t[i] = stringKeys(e)
		}
		return t
	default:
		return v
	}
}
