package app

import (
	"github.com/charmbracelet/glamour"
	"github.com/clocklear/texttrove/pkg/models"
	"github.com/tmc/langchaingo/llms"
)

type Config struct {
	AppName         string
	ChatInputHeight int
	Keys            KeyMap
	LogColor        uint
	SenderColor     uint
	LLMColor        uint
	ToolColor       uint
	ErrorColor      uint
	SpinnerColor    uint

	Chat *models.Chat

	ConversationLLM llms.Model
	RAG             Ragger
	UseManualRAG    bool

	MarkdownRenderer        *glamour.TermRenderer
	ShowPromptInChat        bool
	ShowToolResponsesInChat bool
	LoggerHistorySize       uint

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
		ToolColor:         2,   // ANSI Green
		ErrorColor:        1,   // ANSI Red
		SpinnerColor:      69,  // ANSI Light Blue
		LogColor:          184, // ANSI Yellow-ish
		Keys:              DefaultKeyMap(),
		MarkdownRenderer:  g,
		LoggerHistorySize: 100,
		UseManualRAG:      true,
	}, nil
}
