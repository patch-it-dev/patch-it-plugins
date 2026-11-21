package changelog

import "testing"

func TestStripeSDKs(t *testing.T) {
	checkSDKs(t, []sdkCase{
		{StripePython, "4.0.0", "3.5.0", "stripe-python-4.0.0-CHANGELOG.md", "", "3.5.0"},
		{StripeNode, "10.0.0", "9.16.0", "stripe-node-10.0.0-CHANGELOG.md", "", "9.16.0"},
		{StripeGo, "73.0.0", "72.122.0", "stripe-go-73.0.0-CHANGELOG.md", "", "72.122.0"},
		{StripeJava, "21.0.0", "20.136.0", "stripe-java-21.0.0-CHANGELOG.md", "", "20.136.0"},
		{StripeRuby, "7.0.0", "6.5.0", "stripe-ruby-7.0.0-CHANGELOG.md", "", "6.5.0"},
		{StripeDotnet, "40.0.0", "39.126.0", "stripe-dotnet-40.0.0-CHANGELOG.md", "", "39.126.0"},
		{StripePHP, "9.0.0", "8.12.1", "stripe-php-9.0.0-CHANGELOG.md", "", "8.12.0"},
	})
}
