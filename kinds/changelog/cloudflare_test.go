package changelog

import "testing"

func TestCloudflareSDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{CloudflareTypeScript, "7.1.0", "7.0.0", "cloudflare-typescript-7.1.0-CHANGELOG.md", "", "7.0.0"},
		{CloudflarePython, "5.6.0", "5.5.0", "cloudflare-python-5.6.0-CHANGELOG.md", "", "5.4.0"},
		{CloudflareGo, "7.15.0", "7.14.0", "cloudflare-go-7.15.0-CHANGELOG.md", "", "7.13.1"},
		{CloudflareTerraform, "5.29.0", "5.28.1", "cloudflare-terraform-5.29.0-CHANGELOG.md", "", "5.28.1"},
	})
}
