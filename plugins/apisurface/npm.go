package apisurface

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/core/versions"
	"github.com/patch-it-dev/patch-it-plugins/plugins/apisurface/dtsurface"
)

// readNPM downloads an npm release's tarball (from its version document on
// the registry) and reads its public API from its declaration files. Only
// package.json and the .d.ts, .d.mts and .d.cts files are read out of the
// tarball, in memory: no JavaScript in it is run, and no install script.
func (p Plugin) readNPM(ctx context.Context, env plugin.Env, v plugin.Version) (plugin.Snapshot, error) {
	pkg := p.Package
	u := versions.NPM + "/" + strings.Replace(url.PathEscape(pkg.Name), "%40", "@", 1) + "/" + url.PathEscape(v.ID)
	body, err := env.Get(ctx, u)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("reading %s %s on npm: %w", pkg.Name, v.ID, err)
	}
	var doc struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return plugin.Snapshot{}, fmt.Errorf("parsing npm's %s %s: %w", pkg.Name, v.ID, err)
	}
	if doc.Dist.Tarball == "" {
		return plugin.Snapshot{}, fmt.Errorf("npm's %s %s has no tarball", pkg.Name, v.ID)
	}
	tgz, err := env.Get(ctx, doc.Dist.Tarball)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("fetching %s: %w", doc.Dist.Tarball, err)
	}
	files, skipped, err := npmFiles(tgz)
	if err != nil {
		return plugin.Snapshot{}, fmt.Errorf("%s: %w", doc.Dist.Tarball, err)
	}
	if !slices.ContainsFunc(files, func(f dtsurface.File) bool { return f.Path != "package.json" }) {
		return plugin.Snapshot{}, fmt.Errorf("%s %s has no declaration files: %w", pkg.Name, v.ID, plugin.ErrNothingToRead)
	}
	for _, s := range skipped {
		log.Printf("%s %s: %s skipped: over %d bytes", pkg.Name, v.ID, s, MaxFileBytes)
	}
	surface, classes, bindings := dtsurface.Read(pkg.Name, files)
	content, err := json.Marshal(Release{Artifact: path.Base(doc.Dist.Tarball), Surface: surface, Classes: classes, Bindings: bindings, Skipped: skipped, Positional: true})
	if err != nil {
		return plugin.Snapshot{}, err
	}
	return plugin.Snapshot{Version: v, Content: content, SourceURL: doc.Dist.Tarball}, nil
}

// npmFiles are a tarball's package.json and declaration files, each with
// its path from the package's root (the tarball's top directory, "package/"
// by convention, left out). Links and other non-regular entries are never
// followed; a file over MaxFileBytes is skipped, and more than
// MaxSourceBytes in all is refused.
func npmFiles(tgz []byte) ([]dtsurface.File, []string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, nil, err
	}
	tr := tar.NewReader(gz)
	var files []dtsurface.File
	var skipped []string
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := path.Clean(strings.TrimPrefix(h.Name, "./"))
		_, rel, ok := strings.Cut(name, "/")
		if !ok || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/node_modules/") || strings.HasPrefix(rel, "node_modules/") {
			continue
		}
		if rel != "package.json" && !strings.HasSuffix(rel, ".d.ts") && !strings.HasSuffix(rel, ".d.mts") && !strings.HasSuffix(rel, ".d.cts") {
			continue
		}
		if h.Size > MaxFileBytes {
			skipped = append(skipped, rel)
			continue
		}
		total += h.Size
		if total > MaxSourceBytes {
			return nil, nil, fmt.Errorf("over %d bytes of declarations", MaxSourceBytes)
		}
		b, err := io.ReadAll(io.LimitReader(tr, MaxFileBytes+1))
		if err != nil {
			return nil, nil, err
		}
		files = append(files, dtsurface.File{Path: rel, Text: string(b)})
	}
	slices.SortFunc(files, func(a, b dtsurface.File) int { return strings.Compare(a.Path, b.Path) })
	return files, skipped, nil
}
