package google

import (
	"regexp"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

// googleModel is a model id as code sends it, nothing around it: a cell
// with a footnote mark ("textembedding-gecko@003 *") isn't read.
var googleModel = regexp.MustCompile(`^[a-z0-9][a-z0-9.@_-]*$`)

// GeminiModels is the Gemini API's deprecations page (ai.google.dev, served by
// aistudio.google.com as markdown, checked 2026-11-21): one Markdoc table per
// model family, | Model | Release date | Shutdown date | Recommended
// replacement |, current models with "No shutdown date announced", shut-down
// ones under "Deprecated models". Every model there is one code names in a
// request (text, image, video, audio, embedding: all through the same SDK);
// the managed agents' tables name an agent, not a model, and are skipped.
// Google says its dates are the earliest a model may be shut down.
var GeminiModels = retire.Page{
	API:      "google/gemini-models",
	Provider: "google",
	Vendor:   "Google",
	URL:      "https://aistudio.google.com/docs/deprecations",
	Format:   retire.FormatMarkdoc,
	Keep:     googleModel.MatchString,
	Earliest: true,
}

// VertexModelRetirements is Google Cloud's Vertex AI model versions page (checked
// 2026-11-21): HTML tables of | Model ID | Release date | Retirement date |
// Replacement model ID | (or Recommended upgrade). A GA model's date is a
// minimum lifetime ("October 13, 2027 or later", "No sooner than May 20,
// 2028"), not a notice, and isn't read as a date; a retired or retiring
// model's is a plain date ("June 30, 2026").
var VertexModelRetirements = retire.Page{
	API:      "google/vertex-model-retirements",
	Provider: "google",
	Vendor:   "Google",
	URL:      "https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions",
	Format:   retire.FormatHTML,
	Keep:     googleModel.MatchString,
}
