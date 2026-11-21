package openai

import (
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/kinds/notices/retire"
)

// Models is OpenAI's deprecations page (checked 2026-11-21). Its tables cover every
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

// A GPT name with its generation number ("gpt-4o", "gpt-5.6-sol", "gpt-6-luna")
// is a text model unless it names another modality (openAINotText), so a new
// generation is kept without a change here.
var gptGeneration = regexp.MustCompile(`^gpt-[0-9]`)

var (
	openAIFamilies = []string{"o1", "o3", "o4", "chatgpt-", "computer-use", "codex-"}
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
	if gptGeneration.MatchString(name) {
		return true
	}
	for _, p := range openAIFamilies {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
