package app

import (
	"context"
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/tmc/langchaingo/llms"
)

type LLMStreamingResponseMsg struct {
	chunk         string
	isComplete    bool
	err           error
	functionCalls []LLMFunctionCall
}

type LLMFunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func submitChat(ctx context.Context, llm llms.Model, chatContext []llms.MessageContent, sub chan tea.Msg) tea.Cmd {
	// fmt.Println("RUNNING submitChat")
	return func() tea.Msg {
		resp, err := llm.GenerateContent(ctx, chatContext, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			sub <- LLMStreamingResponseMsg{chunk: string(chunk)}
			return nil
		}))
		// Were there any tool calls in this response?
		functionCalls := make([]LLMFunctionCall, 0)
		for _, v := range resp.Choices {
			// Extract the potential function call
			b, err := extractFunctionCall(v.Content)
			if err != nil || len(b) == 0 {
				continue
			}
			// Attempt to parse the content as JSON (a llms.FunctionCall)
			var tc LLMFunctionCall
			if err := json.Unmarshal(b, &tc); err == nil && tc.Name != "" {
				functionCalls = append(functionCalls, tc)
			}
		}
		if err != nil {
			// fmt.Println("ERROR", err)
			sub <- LLMStreamingResponseMsg{err: err}
		} else {
			sub <- LLMStreamingResponseMsg{isComplete: true, functionCalls: functionCalls}
		}
		return nil
	}
}

// extractFunctionCall extracts a potential function call from a string
func extractFunctionCall(c string) ([]byte, error) {
	// Extract the potential function call
	// It's shaped like this: `{"name": "functionName", "arguments": {"arg1": "value1", "arg2": "value2"}}`
	// Find the first `{` and the last `}` and extract the content in between
	start := -1
	end := -1
	for i, v := range c {
		if v == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return nil, nil
	}
	for i := len(c) - 1; i >= 0; i-- {
		if c[i] == '}' {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, nil
	}
	return []byte(c[start : end+1]), nil
}
