package kit

// Notices is what a reader of a live page of dated notices (a deprecations
// table, an end-of-life table, a feed: kinds/notices) found on one read
// of it. The kind compares what the live page and its snapshot give with
// reflect.DeepEqual (Cite), so a Notices holds plain values, never a func.
type Notices interface {
	// Count is how many notices were read: what the shape guard compares
	// with the last read.
	Count() int
	// Has says whether entry is among what was read, as the reader names
	// an entry: what a source's anchors (plugin.Anchor) are checked with.
	Has(entry string) bool
}
