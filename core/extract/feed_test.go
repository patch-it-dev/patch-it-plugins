package extract

import (
	"testing"
)

func TestParseFeedRefusesWhatIsntAFeed(t *testing.T) {
	for name, b := range map[string]string{
		"html":     "<!doctype html><html><body>moved</body></html>",
		"no items": `<rss version="2.0"><channel><title>x</title></channel></rss>`,
		"no entry": `<feed xmlns="http://www.w3.org/2005/Atom"><id>x</id></feed>`,
	} {
		if _, err := ParseFeed([]byte(b)); err == nil {
			t.Errorf("%s: want an error, never zero events read as a quiet week", name)
		}
	}
}

// RSS without content:encoded falls back to description; Atom takes the
// alternate link and the category terms.
func TestParseFeedReadsBothFormats(t *testing.T) {
	r, err := ParseFeed([]byte(`<rss version="2.0"><channel><item><title>A &amp; B</title><link>https://x.test/a</link>
		<category>Retired</category><description>&lt;p&gt;text&lt;/p&gt;</description></item></channel></rss>`))
	if err != nil || r[0].Title != "A & B" || r[0].HTML != "<p>text</p>" || r[0].Categories[0] != "Retired" || r[0].ID != "https://x.test/a" {
		t.Errorf("rss: %+v %v", r, err)
	}
	a, err := ParseFeed([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>i</id><title>T</title>
		<link rel="self" href="https://x.test/self"/><link rel="alternate" href="https://x.test/e"/>
		<category term="Deprecated"/><summary type="html">&lt;b&gt;s&lt;/b&gt;</summary></entry></feed>`))
	if err != nil || a[0].Link != "https://x.test/e" || a[0].HTML != "<b>s</b>" || a[0].Categories[0] != "Deprecated" {
		t.Errorf("atom: %+v %v", a, err)
	}
}
