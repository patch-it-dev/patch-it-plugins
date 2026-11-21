package stripe

import (
	"testing"

	"github.com/patch-it-dev/patch-it-plugins/kinds/changelog/changelogtest"
)

func TestStripeSDKs(t *testing.T) {
	changelogtest.CheckSDKs(t, []changelogtest.SDKCase{
		{File: Python, Release: "4.0.0", Prev: "3.5.0", Saved: "stripe-python-4.0.0-CHANGELOG.md", Dropped: "", Drop: "3.5.0"},
		{File: Node, Release: "10.0.0", Prev: "9.16.0", Saved: "stripe-node-10.0.0-CHANGELOG.md", Dropped: "", Drop: "9.16.0"},
		{File: Go, Release: "73.0.0", Prev: "72.122.0", Saved: "stripe-go-73.0.0-CHANGELOG.md", Dropped: "", Drop: "72.122.0"},
		{File: Java, Release: "21.0.0", Prev: "20.136.0", Saved: "stripe-java-21.0.0-CHANGELOG.md", Dropped: "", Drop: "20.136.0"},
		{File: Ruby, Release: "7.0.0", Prev: "6.5.0", Saved: "stripe-ruby-7.0.0-CHANGELOG.md", Dropped: "", Drop: "6.5.0"},
		{File: Dotnet, Release: "40.0.0", Prev: "39.126.0", Saved: "stripe-dotnet-40.0.0-CHANGELOG.md", Dropped: "", Drop: "39.126.0"},
		{File: PHP, Release: "9.0.0", Prev: "8.12.1", Saved: "stripe-php-9.0.0-CHANGELOG.md", Dropped: "", Drop: "8.12.0"},
	})
}
