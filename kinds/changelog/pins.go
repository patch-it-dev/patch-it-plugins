package changelog

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
)

// What a release pins: a sentence in its own section of the changelog that
// names the version of another source it pins, in the words its SDK uses
// (Stripe's, checked 2026-11-20 in all seven SDKs' CHANGELOG.md):
//
//	This release changes the pinned API version to `2026-11-18.endive`.
//	This release **doesn't** change the pinned API version; it still uses `2026-05-27.dahlia`.
//	This release **does not** change the pinned API version. It's still `2026-06-24.dahlia`.
//	This release uses the same pinned API version to `2026-03-25.dahlia` as the last major release.
//
// A release that says nothing pins nothing here, even though it keeps the
// release before's: only what a release states is read. Two different
// versions in one section are none (precision over recall). Older wordings
// ("Major version release for API version 2022-08-01", "Move to API version
// …") don't say "pinned" and aren't read.
var statedPin = []*regexp.Regexp{
	regexp.MustCompile("pinned API version to `(" + apiVersion + ")`"),
	regexp.MustCompile("change the pinned API version[.;] (?:[Ii]t's still|it still uses) `(" + apiVersion + ")`"),
}

// apiVersion is a dated API version, with the release's name when it has
// one: "2026-11-18.endive", "2022-08-01".
const apiVersion = `\d{4}-\d{2}-\d{2}(?:\.[a-z]+)?`

// Pins are the versions each release of a Markdown changelog states it
// pins, by release as its heading names it.
func Pins(file []byte) map[string]string {
	out, conflict := map[string]string{}, map[string]bool{}
	release, fenced := "", false
	for _, l := range strings.Split(strings.ReplaceAll(string(file), "\r\n", "\n"), "\n") {
		// A code block first: a "# comment" in an example isn't a heading.
		// A release's heading closes a block its section left open.
		if m := mdRelease.FindStringSubmatch(l); m != nil && strings.HasPrefix(l, "## ") {
			release, fenced = m[1], false
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "# ") {
			release = ""
			if m := mdRelease.FindStringSubmatch(l); m != nil {
				release = m[1]
			}
			continue
		}
		if release == "" {
			continue
		}
		l = strings.ReplaceAll(l, "**", "")
		for _, re := range statedPin {
			for _, m := range re.FindAllStringSubmatch(l, -1) {
				if have, ok := out[release]; ok && have != m[1] {
					conflict[release] = true
				}
				out[release] = m[1]
			}
		}
	}
	for r := range conflict {
		delete(out, r)
	}
	return out
}

// pinned sets each version's MapsTo to what its section states it pins (f.Pins),
// read from the changelog at the newest version's tag, which has every
// earlier release's section, or at the one before when that tag can't be read
// (a release on the registry before its tag is pushed). A changelog that
// can't be fetched leaves the versions as listed: what they pin is read on a
// later listing.
func (f File) pinned(ctx context.Context, env plugin.Env, vs []plugin.Version) []plugin.Version {
	if f.Pins == "" || f.Notes || len(vs) == 0 {
		return vs
	}
	var page []byte
	var err error
	for i := len(vs) - 1; i >= 0 && i >= len(vs)-2; i-- {
		if page, err = env.Get(ctx, f.At(vs[i].ID).URL()); err == nil {
			break
		}
	}
	if err != nil {
		log.Printf("%s: what its releases pin isn't read this time: %v", f.API, err)
		return vs
	}
	stated := Pins(page)
	for i := range vs {
		if v, ok := stated[vs[i].ID]; ok {
			vs[i].MapsTo = plugin.Pin{API: f.Pins, Version: v}
		}
	}
	return vs
}
