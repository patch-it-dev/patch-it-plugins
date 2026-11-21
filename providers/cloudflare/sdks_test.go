package cloudflare

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestCloudflareSDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: Typescript, Release: "7.1.0", Prev: "7.0.0", Saved: "cloudflare-typescript-7.1.0-CHANGELOG.md", Dropped: "", Drop: "7.0.0"},
		{File: Python, Release: "5.6.0", Prev: "5.5.0", Saved: "cloudflare-python-5.6.0-CHANGELOG.md", Dropped: "", Drop: "5.4.0"},
		{File: Go, Release: "7.15.0", Prev: "7.14.0", Saved: "cloudflare-go-7.15.0-CHANGELOG.md", Dropped: "", Drop: "7.13.1"},
		{File: Terraform, Release: "5.29.0", Prev: "5.28.1", Saved: "cloudflare-terraform-5.29.0-CHANGELOG.md", Dropped: "", Drop: "5.28.1"},
	})
}
