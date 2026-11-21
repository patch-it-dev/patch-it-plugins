// Package openapitest is what the tests of providers' OpenAPI sources share:
// a fixed clock, recorded version listings, and the node a pointer names in
// a saved spec.
package openapitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"gopkg.in/yaml.v3"
)

// Recorded serves saved API responses (testdata/versions/, recorded
// 2026-11-20 and trimmed: testdata/README.md) by request path and query;
// any other request fails the test.
func Recorded(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		file, ok := files[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			http.NotFound(w, r)
			return
		}
		b, err := os.ReadFile("testdata/versions/" + file)
		if err != nil {
			t.Error(err)
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// IDs lists versions as "id@ref", space-separated.
func IDs(vs []plugin.Version) string {
	var out []string
	for _, v := range vs {
		out = append(out, v.ID+"@"+v.Ref)
	}
	return strings.Join(out, " ")
}

// Env is a plugin.Env on kittest's fixed clock.
func Env(*kittest.Mem) plugin.Env {
	return plugin.Env{Now: func() time.Time { return kittest.Clock }}
}

// JSONNode resolves an RFC 6901 pointer in a JSON file and serializes the node
// the way normalize serializes excerpts.
func JSONNode(t *testing.T, file, pointer string) string {
	t.Helper()
	var node any
	if err := json.Unmarshal(kittest.Read(t, file), &node); err != nil {
		t.Fatal(err)
	}
	for _, tok := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch n := node.(type) {
		case map[string]any:
			node = n[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i >= len(n) {
				return "<no node>"
			}
			node = n[i]
		default:
			return "<no node>"
		}
	}
	b, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Truncated reports whether excerpt is node cut to the contract's limit.
func Truncated(excerpt, node string) bool {
	prefix, ok := strings.CutSuffix(excerpt, "…")
	return ok && strings.HasPrefix(node, prefix)
}

// YAMLNodeJSON resolves an RFC 6901 pointer in a YAML file and serializes the
// node the way normalize serializes excerpts.
func YAMLNodeJSON(t *testing.T, file, pointer string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var node any
	if err := yaml.Unmarshal(data, &node); err != nil {
		t.Fatal(err)
	}
	for _, tok := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch n := node.(type) {
		case map[string]any:
			node = n[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i >= len(n) {
				return "<no node>"
			}
			node = n[i]
		default:
			return "<no node>"
		}
	}
	b, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
