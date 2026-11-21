package apisurface

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/kinds/apisurface/pysurface"
)

// The limits on what is read out of an archive: a release is data, and a
// crafted one (a zip bomb, a huge generated file) must not exhaust the
// runner. A file over MaxFileBytes is skipped (logged as a gap in the
// surface, never guessed at); an archive whose source files add up to more
// than MaxSourceBytes is refused. Variables only so a test can lower them.
var (
	MaxFileBytes   int64 = 4 << 20
	MaxSourceBytes int64 = 128 << 20
)

// artifact is the file of a release that's read: a wheel if there is one
// (the files as installed), else the sdist.
type artifact struct {
	URL      string
	Filename string
	Kind     string // "bdist_wheel", "sdist"
}

// pickArtifact chooses the release file to read from PyPI's JSON for one
// release: a pure-Python wheel ("py3-none-any"), else any wheel (its .py
// files are the same), else the sdist (.tar.gz or .zip).
func pickArtifact(ctx context.Context, env plugin.Env, pkg, version string) (artifact, error) {
	u := versions.PyPI + "/pypi/" + url.PathEscape(pkg) + "/" + url.PathEscape(version) + "/json"
	body, err := env.Get(ctx, u)
	if err != nil {
		return artifact{}, fmt.Errorf("listing %s %s's files on PyPI: %w", pkg, version, err)
	}
	var doc struct {
		URLs []struct {
			URL         string `json:"url"`
			Filename    string `json:"filename"`
			PackageType string `json:"packagetype"`
			Yanked      bool   `json:"yanked"`
		} `json:"urls"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return artifact{}, fmt.Errorf("parsing PyPI's %s %s: %w", pkg, version, err)
	}
	var wheels, sdists []artifact
	for _, f := range doc.URLs {
		a := artifact{URL: f.URL, Filename: f.Filename, Kind: f.PackageType}
		switch {
		case f.PackageType == "bdist_wheel" && strings.HasSuffix(f.Filename, ".whl"):
			wheels = append(wheels, a)
		case f.PackageType == "sdist" && (strings.HasSuffix(f.Filename, ".tar.gz") || strings.HasSuffix(f.Filename, ".zip")):
			sdists = append(sdists, a)
		}
	}
	byName := func(a, b artifact) int { return strings.Compare(a.Filename, b.Filename) }
	slices.SortFunc(wheels, byName)
	slices.SortFunc(sdists, byName)
	for _, w := range wheels {
		if strings.Contains(w.Filename, "-none-any.whl") {
			return w, nil
		}
	}
	if len(wheels) > 0 {
		return wheels[0], nil
	}
	if len(sdists) > 0 {
		return sdists[0], nil
	}
	return artifact{}, fmt.Errorf("%s %s has no wheel or sdist on PyPI", pkg, version)
}

// sourceFiles are the .py and .pyi files of an archive, each with its path
// from the package's top directory ("openai/_client.py"): a wheel's paths
// as they are, an sdist's without its "<name>-<version>/" and "src/"
// prefixes. Nothing is written to disk and nothing in it is run.
func sourceFiles(a artifact, body []byte, module string) ([]pysurface.File, []string, error) {
	var files []pysurface.File
	var skipped []string
	var total int64
	add := func(name string, size int64, open func() (io.Reader, error)) error {
		if !strings.HasSuffix(name, ".py") && !strings.HasSuffix(name, ".pyi") {
			return nil
		}
		rel, ok := packagePath(name, module, a.Kind == "sdist")
		if !ok {
			return nil
		}
		if size > MaxFileBytes {
			skipped = append(skipped, rel)
			return nil
		}
		total += size
		if total > MaxSourceBytes {
			return fmt.Errorf("%s: over %d bytes of source", a.Filename, MaxSourceBytes)
		}
		r, err := open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
		if err != nil {
			return err
		}
		if int64(len(b)) > MaxFileBytes {
			skipped = append(skipped, rel)
			return nil
		}
		files = append(files, pysurface.File{Path: rel, Text: string(b)})
		return nil
	}
	switch {
	case strings.HasSuffix(a.Filename, ".whl") || strings.HasSuffix(a.Filename, ".zip"):
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", a.Filename, err)
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			if err := add(f.Name, int64(f.UncompressedSize64), func() (io.Reader, error) { return f.Open() }); err != nil {
				return nil, nil, err
			}
		}
	case strings.HasSuffix(a.Filename, ".tar.gz"):
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", a.Filename, err)
		}
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, nil, fmt.Errorf("reading %s: %w", a.Filename, err)
			}
			if h.Typeflag != tar.TypeReg {
				continue // links and devices are never followed
			}
			if err := add(h.Name, h.Size, func() (io.Reader, error) { return tr, nil }); err != nil {
				return nil, nil, err
			}
		}
	default:
		return nil, nil, fmt.Errorf("%s: not a wheel, .tar.gz or .zip", a.Filename)
	}
	slices.SortFunc(files, func(x, y pysurface.File) int { return strings.Compare(x.Path, y.Path) })
	return files, skipped, nil
}

// packagePath is name's path from the top module's directory, and false for
// a file outside it (tests, docs, setup.py, another top-level package, a
// path that climbs out with "..").
func packagePath(name, module string, sdist bool) (string, bool) {
	name = path.Clean(strings.TrimPrefix(name, "./"))
	if strings.HasPrefix(name, "../") || path.IsAbs(name) {
		return "", false
	}
	parts := strings.Split(name, "/")
	if sdist && len(parts) > 1 {
		parts = parts[1:] // "<name>-<version>/"
		if len(parts) > 1 && parts[0] == "src" {
			parts = parts[1:]
		}
	}
	if len(parts) == 0 || (parts[0] != module && parts[0] != module+".py" && parts[0] != module+".pyi") {
		return "", false
	}
	return strings.Join(parts, "/"), true
}
