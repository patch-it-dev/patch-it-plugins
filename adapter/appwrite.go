package adapter

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/fetch"
)

var (
	// appwriteCommit is a full appwrite/specs commit SHA.
	appwriteCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)
	// appwriteDir is one minor's directory under specs/: "1.9.x", "2.7.x".
	appwriteDir = regexp.MustCompile(`^(\d+)\.(\d+)\.x$`)
)

// appwriteMethods are the operation keys a path item can hold.
var appwriteMethods = []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"}

// Appwrite adapts Appwrite's REST API spec in appwrite/specs (BSD-3-Clause)
// for the openapi plugin. The repo has one directory per server minor,
// specs/<X.Y>.x/, and no tags or releases (checked 2026-11-19), so a tag here
// is a commit SHA: the newest minor's spec at that commit. To replay an older
// minor, a tag can name its directory, "1.8.x@<sha>".
//
// A directory holds one of two layouts:
//
//   - open-api3-<X.Y>.x.json, one document for every platform, each
//     operation listing the SDK platforms it's on in x-appwrite.platforms
//     (console, client, server, manager): from 2.0.x on, the only layout up
//     to 2.2.x;
//   - open-api3-<X.Y>.x-server.json, one document per platform: up to
//     1.9.x, and alongside the single document in 2.3.x and 2.4.x.
//
// Fetch reads the single document when the directory has one, keeping the
// operations on the server platform, and the server document otherwise:
// what an Appwrite server SDK (a repo's backend code) can call. Filtered
// this way, the single document has the operations the server document has
// at the same commit: 1.9.x's and 2.0.x's at 0ccd20ab (2026-09-07, which
// added 2.0.x), 2.4.x's two at 98cc1a25 (2026-10-08). Where they part, the
// single document is the one kept up: 06e6fafa (2026-10-12) took the
// server platform off 13 operations in 2.4.x's single document only ("it
// listed server and client methods no released SDK ships yet").
//
// The zero value reads GitHub. The fields point it at another server, for
// tests.
type Appwrite struct {
	APIBase string // GitHub's REST API; "" is https://api.github.com
	RawBase string // files at a commit; "" is https://raw.githubusercontent.com
}

var _ Adapter = Appwrite{}

func (Appwrite) API() string { return "appwrite" }

func (a Appwrite) api(path string) string {
	return cmp.Or(a.APIBase, "https://api.github.com") + "/repos/appwrite/specs" + path
}

// LatestTag is the newest commit on appwrite/specs' default branch that
// changed specs/.
func (a Appwrite) LatestTag(ctx context.Context) (string, error) {
	body, err := fetch.Get(ctx, a.api("/commits?path=specs&per_page=1"), githubHeaders())
	if err != nil {
		return "", fmt.Errorf("looking up appwrite's latest spec commit: %w", err)
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return "", fmt.Errorf("parsing github commits response: %w", err)
	}
	if len(commits) == 0 || !appwriteCommit.MatchString(commits[0].SHA) {
		return "", fmt.Errorf("github lists no commit that changed appwrite's specs")
	}
	return commits[0].SHA, nil
}

// Fetch downloads the spec at a commit: the newest minor's directory, or the
// one the tag names ("1.8.x@<sha>").
//
// Version is the spec's info.version, the Appwrite server version it
// describes ("1.9.5", "2.7.2"): the version the server SDKs send as
// X-Appwrite-Response-Format.
func (a Appwrite) Fetch(ctx context.Context, tag string) (Spec, error) {
	dir, sha, named := strings.Cut(tag, "@")
	if !named {
		dir, sha = "", tag
	}
	if !appwriteCommit.MatchString(sha) || (named && !appwriteDir.MatchString(dir)) {
		return Spec{}, fmt.Errorf("%q is neither a full appwrite/specs commit sha nor <X.Y>.x@<sha>", tag)
	}
	if dir == "" {
		dirs, err := a.list(ctx, "specs", sha)
		if err != nil {
			return Spec{}, err
		}
		if dir = newestAppwriteDir(dirs); dir == "" {
			return Spec{}, fmt.Errorf("appwrite/specs %s has no specs/<X.Y>.x directory", sha)
		}
	}
	files, err := a.list(ctx, "specs/"+dir, sha)
	if err != nil {
		return Spec{}, err
	}
	single, server := "open-api3-"+dir+".json", "open-api3-"+dir+"-server.json"
	var file string
	switch {
	case slices.Contains(files, single):
		file = single
	case slices.Contains(files, server):
		file = server
	default:
		return Spec{}, fmt.Errorf("appwrite/specs %s: specs/%s has neither %s nor %s", sha, dir, single, server)
	}
	url := cmp.Or(a.RawBase, "https://raw.githubusercontent.com") + "/appwrite/specs/" + sha + "/specs/" + dir + "/" + file
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return Spec{}, fmt.Errorf("fetching appwrite spec %s: %w", tag, err)
	}
	if file == single {
		if body, err = appwriteServerOnly(body); err != nil {
			return Spec{}, fmt.Errorf("appwrite spec %s: %w", tag, err)
		}
	}
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return Spec{}, fmt.Errorf("appwrite spec %s is not valid json: %w", tag, err)
	}
	if doc.Info.Version == "" {
		return Spec{}, fmt.Errorf("appwrite spec %s has no info.version", tag)
	}
	return Spec{Content: body, Version: doc.Info.Version, SourceURL: url}, nil
}

// list is the names in a directory of appwrite/specs at a commit.
func (a Appwrite) list(ctx context.Context, path, sha string) ([]string, error) {
	body, err := fetch.Get(ctx, a.api("/contents/"+path+"?ref="+sha), githubHeaders())
	if err != nil {
		return nil, fmt.Errorf("listing appwrite/specs %s at %s: %w", path, sha, err)
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing github contents of %s: %w", path, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names, nil
}

// newestAppwriteDir is the highest <X.Y>.x by number ("2.10.x" above
// "2.9.x"), or "". specs/ also has "latest", a copy of the newest minor's
// spec under another name, which is left out: the minor's own directory
// says which minor it is.
func newestAppwriteDir(names []string) string {
	best, bestV := "", [2]int{-1, -1}
	for _, n := range names {
		m := appwriteDir.FindStringSubmatch(n)
		if m == nil {
			continue
		}
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		if v := [2]int{major, minor}; cmp.Or(cmp.Compare(v[0], bestV[0]), cmp.Compare(v[1], bestV[1])) > 0 {
			best, bestV = n, v
		}
	}
	return best
}

// appwriteServerOnly keeps the operations of the single document whose
// x-appwrite.platforms has "server", and drops a path left with none. The
// kept values are unchanged (numbers kept as written) and nothing moves, so
// a pointer into what's kept names the same node in the published file.
func appwriteServerOnly(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing the spec: %w", err)
	}
	paths := asMap(doc["paths"])
	for p, v := range paths {
		item := asMap(v)
		left := 0
		for _, m := range appwriteMethods {
			op, ok := item[m]
			if !ok {
				continue
			}
			platforms := asSlice(asMap(asMap(op)["x-appwrite"])["platforms"])
			if slices.Contains(platforms, any("server")) {
				left++
			} else {
				delete(item, m)
			}
		}
		if left == 0 {
			delete(paths, p)
		}
	}
	return json.Marshal(doc)
}
