package agent

import (
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
)

const _conversationTemplate = `
{{.history}}
`

func NewConversation(llm llms.Model, memory schema.Memory) chains.LLMChain {
	return chains.LLMChain{
		Prompt: prompts.NewPromptTemplate(
			_conversationTemplate,
			[]string{"history"},
		),
		LLM:          llm,
		Memory:       memory,
		OutputParser: outputparser.NewSimple(),
		OutputKey:    "text",
	}
}
