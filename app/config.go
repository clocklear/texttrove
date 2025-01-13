package app

import (
	"github.com/charmbracelet/glamour"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Config struct {
	AppName          string
	ChatInputHeight  int
	Keys             KeyMap
	SenderColor      uint
	LLMColor         uint
	ErrorColor       uint
	SpinnerColor     uint
	ConversationLLM  *ollama.LLM
	RAG              Ragger
	MarkdownRenderer *glamour.TermRenderer
}

func DefaultConfig() (Config, error) {
	g, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return Config{}, err
	}
	return Config{
		AppName:          "TextTrove",
		ChatInputHeight:  5,
		SenderColor:      5,  // ANSI Magenta
		LLMColor:         4,  // ANSI Blue
		ErrorColor:       1,  // ANSI Red
		SpinnerColor:     69, // ANSI Light Blue
		Keys:             DefaultKeyMap(),
		MarkdownRenderer: g,
	}, nil
}
