package github

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestOctokitSDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: OctokitRest, Release: "18.0.0", Prev: "17.11.2", Saved: "github-octokit-rest-18.0.0-release.json", Dropped: "github-octokit-rest-17.11.2-release.json", Drop: "17.11.2"},
		{File: OctokitNet, Release: "0.5.2", Prev: "0.5.1", Saved: "github-octokit-net-0.5.2-release.json", Dropped: "github-octokit-net-0.5.1-release.json", Drop: "0.5.1"},
	})
}
