package cloudflare

import (
	"context"
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/core/event"
	"github.com/patch-it-dev/patch-it-plugins/core/kit/kittest"
	"github.com/patch-it-dev/patch-it-plugins/core/plugin"
	"github.com/patch-it-dev/patch-it-plugins/kinds/catalog"
)

// The commits of cloudflare/workerd the saved files are at
// (testdata/README.md), as GitHub's commit list names them.
var (
	before20261027 = plugin.Version{ID: "2026-10-27", Ref: "bc8e6bc991fa1b7b59f96ceb1708c5b1e83abfb0"}
	temporal       = plugin.Version{ID: "2026-10-29", Ref: "e9e1914f23cf37164329b24e8441ac2d46b956e0"}
	before20261031 = plugin.Version{ID: "2026-10-31", Ref: "3adbfe0d419880f6138ab5f20f7be59af0dc6048"}
	nodeVersion24  = plugin.Version{ID: "2026-11-03", Ref: "22400fed99d386b3ba2e90126dec7ed55ce9b1ea"}
	before20261106 = plugin.Version{ID: "2026-11-06", Ref: "791430df76df40ec0c7c5720120f334266a28d84"}
	pastDate       = plugin.Version{ID: "2026-11-07", Ref: "d791dc63fa7168829214cc19dd09574f497e5dea"}
)

func compatPages() kittest.Pages {
	pages := kittest.Pages{}
	for _, v := range []plugin.Version{before20261027, temporal, before20261031, nodeVersion24, before20261106, pastDate} {
		pages[WorkersCompatFlags.FileURL(v.Ref)] = "testdata/workerd-compatibility-date-" + v.Ref[:8] + ".capnp"
	}
	return pages
}

func compatPair(t *testing.T, from, to plugin.Version) []event.Event {
	t.Helper()
	return kittest.Versions(t, kittest.Env(t, compatPages()), WorkersCompatFlags, from, to)
}

// The golden test: two commits that each add a flag with a default date:
// temporal (from 2026-12-01) on 2026-10-29, nodejs_process_version_24 (from
// 2026-11-24) on 2026-11-03.
func TestCompatFlagsGolden(t *testing.T) {
	got := append(compatPair(t, before20261027, temporal), compatPair(t, before20261031, nodeVersion24)...)
	if len(got) != 2 || got[0].Subject.Name != "temporal" || got[1].Subject.Name != "nodejs_process_version_24" {
		t.Fatalf("%+v", got)
	}
	for i := range got {
		got[i].ID = 0
	}
	kittest.Golden(t, "testdata/golden/cloudflare-workers-compat-flags.json", got)
}

// The pinned drop test: on 2026-11-07 new_module_registry got the default
// date 2026-11-01, before the commit: no announcement of a date to come, no
// event (the same day's later commits move it to 2026-11-08, then 11-12 and
// 11-17, each an event).
func TestCompatFlagsDropsADateBeforeItsCommit(t *testing.T) {
	if got := compatPair(t, before20261106, pastDate); len(got) != 0 {
		t.Errorf("%d events, want 0: %+v", len(got), got)
	}
	from, _ := WorkersCompatFlags.ReadAt(context.Background(), kittest.Env(t, compatPages()), before20261106)
	to, _ := WorkersCompatFlags.ReadAt(context.Background(), kittest.Env(t, compatPages()), pastDate)
	if len(catalog.Flags(to.Content)) != len(catalog.Flags(from.Content))+1 {
		t.Error("the commit adds no dated flag: the drop proves nothing")
	}
}

func TestCompatFlagsVersionsAreTheFilesCommits(t *testing.T) {
	env := kittest.Env(t, kittest.Pages{
		"https://api.github.com/repos/cloudflare/workerd/commits?path=src%2Fworkerd%2Fio%2Fcompatibility-date.capnp&per_page=100&page=1": "testdata/workerd-commits-compatibility-date-excerpt.json",
	})
	vs, err := WorkersCompatFlags.ListVersions(context.Background(), env)
	if err != nil || len(vs) != 3 || vs[2].Ref != "b0031a6dc6c50ecb3611d91781211d9bb60c7f11" {
		t.Errorf("%v %v", vs, err)
	}
}
