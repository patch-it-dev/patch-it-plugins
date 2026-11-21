package changelog

import "testing"

func TestOctokitSDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{OctokitRest, "18.0.0", "17.11.2", "github-octokit-rest-18.0.0-release.json", "github-octokit-rest-17.11.2-release.json", "17.11.2"},
		{OctokitNet, "0.19.0", "0.18.0", "github-octokit-net-0.19.0-release.json", "github-octokit-net-0.18.0-release.json", "0.18.0"},
	})
}
