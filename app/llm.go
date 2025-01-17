package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type LLMStreamingResponseMsg struct {
	chunk      string
	isComplete bool
	err        error
}

func submitChat(ctx context.Context, llm *ollama.LLM, chatContext []llms.MessageContent, sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		_, err := llm.GenerateContent(ctx, chatContext, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			sub <- LLMStreamingResponseMsg{chunk: string(chunk)}
			return nil
		}))
		if err != nil {
			sub <- LLMStreamingResponseMsg{err: err}
		} else {
			sub <- LLMStreamingResponseMsg{isComplete: true}
		}
		return nil
	}
}
