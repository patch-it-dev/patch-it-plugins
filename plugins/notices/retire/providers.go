package retire

import "strings"

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
