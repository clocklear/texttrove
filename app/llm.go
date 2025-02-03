package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
)

type LLMStreamingResponseMsg struct {
	chunk      string
	isComplete bool
	err        error
}

func submitChat(ctx context.Context, llm llms.Model, chatContext []llms.MessageContent, sub chan tea.Msg) tea.Cmd {
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

func submitChatAgent(ctx context.Context, agent agents.Agent, query string, sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		// Delegate the user query to the conversational agent
		executor := agents.NewExecutor(agent)
		answer, err := chains.Run(context.Background(), executor, query)
		if err != nil {
			sub <- LLMStreamingResponseMsg{err: err}
		} else {
			sub <- LLMStreamingResponseMsg{
				chunk:      answer,
				isComplete: true,
			}
		}
		return nil
	}
}
