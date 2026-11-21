// Package extract is what readers of prose share: an RSS or Atom feed's
// entries, an HTML fragment as plain text or lines, its sections by
// heading, a "Month D, YYYY" date, the code spans in it. A reader of any
// kind uses it; it knows no vendor.
package extract

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// Entry is one item of an RSS 2.0 feed or one entry of an Atom feed, in the
// fields both have.
type Entry struct {
	ID         string   // RSS guid, Atom id
	Title      string   // as text
	Link       string   // the entry's own page
	Published  string   // as the feed writes it
	Categories []string // RSS category, Atom category term
	HTML       string   // RSS content:encoded (else description), Atom content (else summary)
}

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []struct {
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		GUID        string   `xml:"guid"`
		PubDate     string   `xml:"pubDate"`
		Categories  []string `xml:"category"`
		Description string   `xml:"description"`
		Content     string   `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	} `xml:"channel>item"`
}

type atom struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []struct {
		ID        string `xml:"id"`
		Title     string `xml:"title"`
		Updated   string `xml:"updated"`
		Published string `xml:"published"`
		Links     []struct {
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Categories []struct {
			Term string `xml:"term,attr"`
		} `xml:"category"`
		Content string `xml:"content"`
		Summary string `xml:"summary"`
	} `xml:"entry"`
}

// ParseFeed reads an RSS 2.0 or Atom feed, whichever the root element says. A
// feed with no entries is an error: a feed that moved or changed shape must
// not read as a quiet week.
func ParseFeed(b []byte) ([]Entry, error) {
	var root struct{ XMLName xml.Name }
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&root); err != nil {
		return nil, fmt.Errorf("not a feed: %w", err)
	}
	var out []Entry
	switch {
	case root.XMLName.Local == "rss":
		var f rss
		if err := xml.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("reading RSS: %w", err)
		}
		for _, it := range f.Items {
			body := it.Content
			if strings.TrimSpace(body) == "" {
				body = it.Description
			}
			id := it.GUID
			if id == "" {
				id = it.Link
			}
			out = append(out, Entry{ID: id, Title: Text(it.Title), Link: strings.TrimSpace(it.Link),
				Published: it.PubDate, Categories: trimAll(it.Categories), HTML: body})
		}
	case root.XMLName.Local == "feed" && root.XMLName.Space == "http://www.w3.org/2005/Atom":
		var f atom
		if err := xml.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("reading Atom: %w", err)
		}
		for _, e := range f.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = l.Href
					break
				}
			}
			var cats []string
			for _, c := range e.Categories {
				cats = append(cats, c.Term)
			}
			body := e.Content
			if strings.TrimSpace(body) == "" {
				body = e.Summary
			}
			pub := e.Published
			if pub == "" {
				pub = e.Updated
			}
			out = append(out, Entry{ID: e.ID, Title: Text(e.Title), Link: link, Published: pub,
				Categories: trimAll(cats), HTML: body})
		}
	default:
		return nil, fmt.Errorf("not an RSS or Atom feed: the root element is <%s>", root.XMLName.Local)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the feed has no entries: it may have moved or changed shape")
	}
	return out, nil
}

func trimAll(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s = strings.TrimSpace(html.UnescapeString(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

var (
	tag    = regexp.MustCompile(`<[^>]*>`)
	spaces = regexp.MustCompile(`\s+`)
	block  = regexp.MustCompile(`(?i)</?(p|li|ul|ol|h[1-6]|tr|table|thead|tbody|div|br)\b[^>]*>`)
)

// Text is a fragment of HTML as one line of plain text: tags dropped,
// entities decoded, whitespace collapsed.
func Text(fragment string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(html.UnescapeString(tag.ReplaceAllString(fragment, "")), " "))
}

// Lines is a fragment of HTML as text, one line per paragraph, list item,
// heading or table row; table cells are joined with " | ".
func Lines(fragment string) []string {
	s := spaces.ReplaceAllString(fragment, " ") // the source's own line breaks are not lines
	s = regexp.MustCompile(`(?i)</t[dh]>\s*<t[dh][^>]*>`).ReplaceAllString(s, " | ")
	s = block.ReplaceAllString(s, "\n")
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if t := Text(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Section is a part of an entry under one heading.
type Section struct {
	Heading string // as text
	HTML    string // what follows it, up to the next heading of the same level
}

// SplitBy cuts an entry's HTML at every <hN> heading (level n). What comes
// before the first heading is not a section.
func SplitBy(fragment string, n int) []Section {
	re := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d\b[^>]*>(.*?)</h%d>`, n, n))
	idx := re.FindAllStringSubmatchIndex(fragment, -1)
	var out []Section
	for i, m := range idx {
		end := len(fragment)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		out = append(out, Section{Heading: Text(fragment[m[2]:m[3]]), HTML: fragment[m[1]:end]})
	}
	return out
}

var (
	monthDay = regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December) (\d{1,2}), (\d{4})`)
)

// Date is the first "Month D, YYYY" in s, as YYYY-MM-DD, or "".
func Date(s string) string {
	m := monthDay.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	t, err := time.Parse("January 2, 2006", m[0])
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

var codeSpan = regexp.MustCompile(`(?is)<code>(.*?)</code>`)

// Codes are the texts of the <code> spans in fragment.
func Codes(fragment string) []string {
	var out []string
	for _, m := range codeSpan.FindAllStringSubmatch(fragment, -1) {
		out = append(out, Text(m[1]))
	}
	return out
}
