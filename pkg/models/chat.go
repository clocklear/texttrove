package models

import (
	"html/template"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// from the knowledge base.  You should prefer using the knowledge in the knowledge base to answer your questions, but you can defer back to your general knowledge if the knowledge base lacks enough context
var systemPromptTpl = template.Must(template.New("system_prompt").Parse(`
You are a helpful assistant with access to a knowledge base, tasked with answering questions from the user.

Answer the question in a very concise manner. Use an unbiased and journalistic tone. Do not repeat text. Don't make anything up. If you are not sure about something, just say that you don't know.
{{- /* Stop here if no context is provided. The rest below is for handling contexts. */ -}}
{{- if . -}}
Answer the question solely based on the provided search results from the knowledge base. If the search results from the knowledge base are not relevant to the question at hand, just say that you don't know. Don't make anything up.

Anything between the following 'context' XML blocks is retrieved from the knowledge base, not part of the conversation with the user. The bullet points are ordered by relevance, so the first one is the most relevant.

<context>
    {{- if . -}}
    {{- range $context := .}}
    - {{.}}{{end}}
    {{- end}}
</context>
{{- end -}}

Don't mention the knowledge base, context or search results in your answer.
`))

type Chat struct {
	title             string // TODO: Future use; summarize the conversation
	completedMessages []llms.MessageContent
	streamingParts    []string
	isStreaming       bool
	err               error
	mu                sync.RWMutex
}

func NewChat() Chat {
	return Chat{
		completedMessages: []llms.MessageContent{},
		streamingParts:    make([]string, 0),
	}
}

func (c *Chat) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = false
	c.err = nil
	c.completedMessages = []llms.MessageContent{}
	c.streamingParts = make([]string, 0)
}

func (c *Chat) Error() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *Chat) SetError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
}

func (c *Chat) ClearError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = nil
}

func (c *Chat) IsStreaming() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isStreaming
}

func (c *Chat) BeginStreaming() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = true
	c.streamingParts = make([]string, 0)
}

func (c *Chat) StreamChunk(chunk string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamingParts = append(c.streamingParts, chunk)
}

func (c *Chat) EndStreaming() {
	cnt := c.streamingPartsToContent()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = false
	c.completedMessages = append(c.completedMessages, cnt)
	c.streamingParts = make([]string, 0)
}

func (c *Chat) AppendUserMessage(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeHuman, msg))
}

func (c *Chat) Log() []llms.MessageContent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	msg := c.completedMessages
	if len(c.streamingParts) > 0 {
		msg = append(msg, c.streamingPartsToContent())
	}
	return msg
}

func (c *Chat) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.completedMessages) == 0
}

func (c *Chat) SetContexts(contexts []schema.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract slice of content from the documents
	content := make([]string, 0, len(contexts))
	for _, doc := range contexts {
		content = append(content, doc.PageContent)
	}

	sb := &strings.Builder{}
	err := systemPromptTpl.Execute(sb, content)
	if err != nil {
		return err
	}
	c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeSystem, sb.String()))
	return nil
}

func (c *Chat) streamingPartsToContent() llms.MessageContent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return llms.TextParts(llms.ChatMessageTypeAI, strings.Join(c.streamingParts, ""))
}
