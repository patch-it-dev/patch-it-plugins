package retire

import (
	"regexp"
	"strings"
)

// OpenAI's deprecations page (checked 2026-10-13). Its tables cover every
// kind of model; these events are for the text models the Chat Completions
// and Responses APIs take. Fine-tuned bases ("ft-gpt-4") aren't names code
// sends (a fine-tuned model is "ft:gpt-4-0613:org::id"), and the "-completions"
// aliases belong to the legacy Completions endpoint.
var OpenAI = Page{
	API:      "openai/models",
	Provider: "openai",
	Vendor:   "OpenAI",
	URL:      "https://developers.openai.com/api/docs/deprecations.md",
	Keep:     openAIText,
}

// Anthropic's model deprecations page (checked 2026-10-13): every model in
// its deprecation history is a Messages API model.
var Anthropic = Page{
	API:      "anthropic/models",
	Provider: "anthropic",
	Vendor:   "Anthropic",
	URL:      "https://platform.claude.com/docs/en/about-claude/model-deprecations.md",
	Keep:     func(name string) bool { return strings.HasPrefix(name, "claude-") },
}

var (
	openAIFamilies = []string{"gpt-3.5", "gpt-4", "gpt-5", "o1", "o3", "o4", "chatgpt-", "computer-use", "codex-"}
	openAINotText  = []string{"image", "dall-e", "tts", "whisper", "transcribe", "realtime", "audio", "sora", "embedding", "moderation"}
)

func openAIText(name string) bool {
	if strings.HasPrefix(name, "ft-") || strings.HasSuffix(name, "-completions") {
		return false
	}
	for _, s := range openAINotText {
		if strings.Contains(name, s) {
			return false
		}
	}
	for _, p := range openAIFamilies {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// googleModel is a model id as code sends it, nothing around it: a cell
// with a footnote mark ("textembedding-gecko@003 *") isn't read.
var googleModel = regexp.MustCompile(`^[a-z0-9][a-z0-9.@_-]*$`)

// Gemini is the Gemini API's deprecations page (ai.google.dev, served by
// aistudio.google.com as markdown, checked 2026-11-21): one Markdoc table per
// model family, | Model | Release date | Shutdown date | Recommended
// replacement |, current models with "No shutdown date announced", shut-down
// ones under "Deprecated models". Every model there is one code names in a
// request (text, image, video, audio, embedding: all through the same SDK);
// the managed agents' tables name an agent, not a model, and are skipped.
// Google says its dates are the earliest a model may be shut down.
var Gemini = Page{
	API:      "google/gemini-models",
	Provider: "google",
	Vendor:   "Google",
	URL:      "https://aistudio.google.com/docs/deprecations",
	Format:   FormatMarkdoc,
	Keep:     googleModel.MatchString,
	Earliest: true,
}

// Vertex is Google Cloud's Vertex AI model versions page (checked
// 2026-11-21): HTML tables of | Model ID | Release date | Retirement date |
// Replacement model ID | (or Recommended upgrade). A GA model's date is a
// minimum lifetime ("October 13, 2027 or later", "No sooner than May 20,
// 2028"), not a notice, and isn't read as a date; a retired or retiring
// model's is a plain date ("June 30, 2026").
var Vertex = Page{
	API:      "google/vertex-model-retirements",
	Provider: "google",
	Vendor:   "Google",
	URL:      "https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/model-versions",
	Format:   FormatHTML,
	Keep:     googleModel.MatchString,
}
