package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type responseMsg struct {
	chunk      string
	isComplete bool
	err        error
}

func submitChat(ctx context.Context, llm *ollama.LLM, chatContext []llms.MessageContent, sub chan responseMsg) tea.Cmd {
	return func() tea.Msg {
		_, err := llm.GenerateContent(ctx, chatContext, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			sub <- responseMsg{chunk: string(chunk)}
			return nil
		}))
		if err != nil {
			sub <- responseMsg{err: err}
		} else {
			sub <- responseMsg{isComplete: true}
		}
		return nil
	}
}

func waitForActivity(sub chan responseMsg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}
