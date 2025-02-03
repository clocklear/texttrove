package app

import (
	"github.com/charmbracelet/glamour"
	"github.com/clocklear/texttrove/pkg/models"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
)

type Config struct {
	AppName         string
	ChatInputHeight int
	Keys            KeyMap
	LogColor        uint
	SenderColor     uint
	LLMColor        uint
	ErrorColor      uint
	SpinnerColor    uint

	Chat *models.Chat

	// These two are used independently when the app is doing it's own RAG
	ConversationLLM llms.Model
	RAG             Ragger

	// This is used when the app is relying on the RAG 'tool'
	ConversationAgent agents.Agent

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
