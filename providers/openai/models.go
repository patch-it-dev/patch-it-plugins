package openai

import (
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

// Models is OpenAI's deprecations page (checked 2026-10-13). Its tables cover every
// kind of model; these events are for the text models the Chat Completions
// and Responses APIs take. Fine-tuned bases ("ft-gpt-4") aren't names code
// sends (a fine-tuned model is "ft:gpt-4-0613:org::id"), and the "-completions"
// aliases belong to the legacy Completions endpoint.
var Models = retire.Page{
	API:      "openai/models",
	Provider: "openai",
	Vendor:   "OpenAI",
	URL:      "https://developers.openai.com/api/docs/deprecations.md",
	Keep:     openAIText,
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
