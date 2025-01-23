package app

import (
	"github.com/charmbracelet/glamour"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Config struct {
	AppName           string
	ChatInputHeight   int
	Keys              KeyMap
	LogColor          uint
	SenderColor       uint
	LLMColor          uint
	ErrorColor        uint
	SpinnerColor      uint
	ConversationLLM   *ollama.LLM
	RAG               Ragger
	MarkdownRenderer  *glamour.TermRenderer
	ShowPromptInChat  bool
	LoggerHistorySize uint

	ChatSystemPromptPath  string
	ChatContextPromptPath string
}

func DefaultConfig() (Config, error) {
	g, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return Config{}, err
	}
	return Config{
		AppName:           "TextTrove",
		ChatInputHeight:   5,
		SenderColor:       5,   // ANSI Magenta
		LLMColor:          4,   // ANSI Blue
		ErrorColor:        1,   // ANSI Red
		SpinnerColor:      69,  // ANSI Light Blue
		LogColor:          184, // ANSI Yellow-ish
		Keys:              DefaultKeyMap(),
		MarkdownRenderer:  g,
		LoggerHistorySize: 100,
	}, nil
}
