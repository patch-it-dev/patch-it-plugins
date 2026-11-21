package appwrite

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
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/fetch"
	"github.com/patch-it-dev/patch-it-plugins/core/github"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/openapi/adapter"
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
// is a commit SHA: the newest minor's spec at that commit. To read an older
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

var _ adapter.Adapter = Appwrite{}

func (Appwrite) API() string { return "appwrite/api" }

func (a Appwrite) api(path string) string {
	return cmp.Or(a.APIBase, "https://api.github.com") + "/repos/appwrite/specs" + path
}

// LatestTag is the newest commit on appwrite/specs' default branch that
// changed specs/.
func (a Appwrite) LatestTag(ctx context.Context) (string, error) {
	sha, err := github.LatestCommit(ctx, github.Get, a.api("/commits?path=specs&per_page=1"))
	if err != nil {
		return "", fmt.Errorf("looking up appwrite's latest spec commit: %w", err)
	}
	if !appwriteCommit.MatchString(sha) {
		return "", fmt.Errorf("github lists no commit that changed appwrite's specs")
	}
	return sha, nil
}

// Fetch downloads the spec at a commit: the newest minor's directory, or the
// one the tag names ("1.8.x@<sha>").
//
// Version is the commit's date (YYYY-MM-DD), not the spec's info.version
// ("1.9.5"), the server version it describes: a repo pins an SDK
// (node-appwrite 32.0.0, the Python SDK 28.0.0), versioned apart from the
// server, so a server version can't be placed against the pin, and the
// specs commits don't line up with server releases anyway.
//
// The server URL's path (/v1) is moved into the paths: see ServerBasePath.
func (a Appwrite) Fetch(ctx context.Context, tag string) (adapter.Spec, error) {
	dir, sha, named := strings.Cut(tag, "@")
	if !named {
		dir, sha = "", tag
	}
	if !appwriteCommit.MatchString(sha) || (named && !appwriteDir.MatchString(dir)) {
		return adapter.Spec{}, fmt.Errorf("%q is neither a full appwrite/specs commit sha nor <X.Y>.x@<sha>", tag)
	}
	_, date, err := github.Commit(ctx, github.Get, a.api("/commits/"+sha))
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("looking up appwrite/specs commit %s: %w", sha, err)
	}
	if dir == "" {
		dirs, err := a.list(ctx, "specs", sha)
		if err != nil {
			return adapter.Spec{}, err
		}
		if dir = newestAppwriteDir(dirs); dir == "" {
			return adapter.Spec{}, fmt.Errorf("appwrite/specs %s has no specs/<X.Y>.x directory", sha)
		}
	}
	files, err := a.list(ctx, "specs/"+dir, sha)
	if err != nil {
		return adapter.Spec{}, err
	}
	single, server := "open-api3-"+dir+".json", "open-api3-"+dir+"-server.json"
	var file string
	switch {
	case slices.Contains(files, single):
		file = single
	case slices.Contains(files, server):
		file = server
	default:
		return adapter.Spec{}, fmt.Errorf("appwrite/specs %s: specs/%s has neither %s nor %s", sha, dir, single, server)
	}
	url := cmp.Or(a.RawBase, "https://raw.githubusercontent.com") + "/appwrite/specs/" + sha + "/specs/" + dir + "/" + file
	body, err := fetch.Get(ctx, url, nil) // raw.githubusercontent.com needs no auth
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("fetching appwrite spec %s: %w", tag, err)
	}
	if file == single {
		if body, err = appwriteServerOnly(body); err != nil {
			return adapter.Spec{}, fmt.Errorf("appwrite spec %s: %w", tag, err)
		}
	}
	// Every server is "https://cloud.appwrite.io/v1" or a regional
	// "https://{region}.cloud.appwrite.io/v1" ("<REGION>" up to 1.8.x), and
	// the paths don't repeat /v1.
	content, err := adapter.ServerBasePath(body)
	if err != nil {
		return adapter.Spec{}, fmt.Errorf("appwrite spec %s: %w", tag, err)
	}
	return adapter.Spec{Content: content, Version: date.UTC().Format("2006-01-02"), SourceURL: url}, nil
}

// SourcePointer: see BasePathPointer.
func (Appwrite) SourcePointer(content []byte) func(string) string {
	return adapter.BasePathPointer(content)
}

var _ adapter.Reshaper = Appwrite{}

// list is the names in a directory of appwrite/specs at a commit.
func (a Appwrite) list(ctx context.Context, path, sha string) ([]string, error) {
	body, err := fetch.Get(ctx, a.api("/contents/"+path+"?ref="+sha), github.Headers())
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
// None kept is an error, not an empty spec: it means the platforms moved or
// were renamed, and an empty spec would read as nothing deprecated.
func appwriteServerOnly(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing the spec: %w", err)
	}
	paths := adapter.AsMap(doc["paths"])
	for p, v := range paths {
		item := adapter.AsMap(v)
		left := 0
		for _, m := range appwriteMethods {
			op, ok := item[m]
			if !ok {
				continue
			}
			platforms := adapter.AsSlice(adapter.AsMap(adapter.AsMap(op)["x-appwrite"])["platforms"])
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
	if len(paths) == 0 {
		return nil, fmt.Errorf("no operation has \"server\" in x-appwrite.platforms")
	}
	return json.Marshal(doc)
}

// Versions are the days appwrite/specs' default branch changed specs/,
// oldest first, each at the day's last commit (Daily): the ID is the date
// (Fetch's Version), the Ref the SHA, read at the newest minor's directory.
func (a Appwrite) Versions(ctx context.Context) ([]plugin.Version, error) {
	cs, err := versions.GitHubPages[struct {
		SHA    string `json:"sha"`
		URL    string `json:"html_url"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}](ctx, github.Get, a.api("/commits?path=specs"))
	if err != nil {
		return nil, fmt.Errorf("listing the commits that changed appwrite's specs: %w", err)
	}
	var out []plugin.Version
	for _, c := range cs {
		if !appwriteCommit.MatchString(c.SHA) || c.Commit.Committer.Date.IsZero() {
			continue
		}
		at := c.Commit.Committer.Date.UTC()
		out = append(out, plugin.Version{ID: at.Format(time.DateOnly), Released: at, Ref: c.SHA, URL: c.URL})
	}
	slices.SortStableFunc(out, func(a, b plugin.Version) int { return a.Released.Compare(b.Released) })
	return adapter.Daily(out), nil
}

// appwriteDiff is how Appwrite's spec says what it says:
//
//   - A deprecation is the `deprecated` flag plus x-appwrite.deprecated,
//     {"since": "1.8.0", "replaceWith": "tablesDB.createTransaction"}: the
//     only place the spec says what to use instead (the description doesn't).
//     x-appwrite sorts last in the operation, after parameters and
//     responses, so quoting the whole operation can lose it to the
//     2,000-character cut: the event quotes x-appwrite.deprecated itself.
//     The flag without it is quoted as the flag (never seen so far).
//   - The flag also marks an operation one of whose SDK method names was
//     renamed: POST /account/mfa/challenges is flagged with "replaceWith":
//     "account.createMFAChallenge", which is the same operation's other
//     name in x-appwrite.methods (createMfaChallenge became
//     createMFAChallenge). The route isn't deprecated, so that flag isn't
//     read as a deprecation: see appwriteMethodLevel.
//   - Path parameters are matched by place, as for the other specs.
var appwriteDiff = diff.Options{
	PathTemplates:      true,
	QuoteDeprecated:    "/x-appwrite/deprecated",
	FlagNotDeprecation: appwriteMethodLevel,
}

// appwriteMethodLevel reports whether an operation's `deprecated` flag is
// about its SDK methods, not its route:
//
//   - x-appwrite.deprecated.replaceWith names a method of the same
//     operation's x-appwrite.methods ("<namespace>.<name>"): a rename;
//   - or a method in x-appwrite.methods isn't itself deprecated: the
//     operation still has a live SDK method, so the route stays.
//
// An operation with no methods listed (the 14 of the golden test have
// "methods": []) is read by its flag. In 2.7.x's server operations the two
// rules pick the same 23 of the 99 flagged, all flagged since 1.8.0
// (1.8.x's server document at ac705cd3 has the same 23); each has a live
// method named by replaceWith.
func appwriteMethodLevel(op map[string]any) bool {
	xa, _ := op["x-appwrite"].(map[string]any)
	dep, _ := xa["deprecated"].(map[string]any)
	replaceWith, _ := dep["replaceWith"].(string)
	methods, _ := xa["methods"].([]any)
	for _, m := range methods {
		method, _ := m.(map[string]any)
		namespace, _ := method["namespace"].(string)
		name, _ := method["name"].(string)
		if replaceWith != "" && name != "" && namespace+"."+name == replaceWith {
			return true
		}
		if d, ok := method["deprecated"]; !ok || d == nil || d == false {
			return true
		}
	}
	return false
}

// appwriteKeep keeps operation deprecations only: not even a deprecated
// operation's removal. Read as Fetch reads it (the newest minor's server
// operations at each of the 139 commits on appwrite/specs' main, 2026-02-17
// to 2026-11-17), the spec's history has 14 deprecations, all on
// 2026-06-24 (see the golden test), and 78 operations leaving it (from
// the spec, or from the server platform). Only one was marked deprecated
// first, and that one was a path move: PATCH
// /databases/{databaseId}/collections/{collectionId}/attributes/{key}/relationship
// became .../attributes/relationship/{key} on 2026-03-20. Of the others, 13
// are back now: 2.4.x's single document took the server platform off them
// on 2026-10-12 as not yet in a released SDK, and 2.5.x had them on it again
// the next day. Nothing in the spec says when an operation stops answering,
// so a removal isn't an event.
//
// Property changes and new operations are dropped too, as for vercel.
func appwriteKeep(c diff.Change) bool { return c.Kind == diff.OperationDeprecated }
