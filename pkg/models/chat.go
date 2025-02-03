package models

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
)

var baseSystemPromptTpl = `You are a helpful assistant with access to a knowledge base, tasked with answering questions from the user.

Use an unbiased and journalistic tone. Do not repeat text. Don't make anything up. If you are not sure about something, just say that you don't know.
`

var baseContextTpl = `{{- if .contexts -}}
Try to answer the question based on the provided search results from the knowledge base. If the search results from the knowledge base are not relevant to the question at hand, ask the user if they would like to fallback to your training data. Don't make anything up.

Anything in the following 'context' XML blocks is retrieved from the knowledge base, not part of the conversation with the user. The bullet points are ordered by relevance, so the first one is the most relevant.

Each item is postfixed with a metadata footer, where possible, so you can use that to cite your sources.  The 'Source' metadata field
represents the location within the knowledge base.  The folder segments are separated by slashes and the folder names and file name are
useful context as well.

<context>
    {{- range $context := .contexts -}}
    - {{$context}}{{end}}
</context>

When answering a question relevant to the above context, site your sources. If you are unsure about the source, say that you don't know.

Whenever you reference the knowledge base or the provided context, you should always refer to it as "your notes".
{{- end -}}
`

type Chat struct {
	title             string // TODO: Future use; summarize the conversation
	completedMessages []llms.MessageContent
	streamingParts    []string
	isStreaming       bool
	err               error
	mu                sync.RWMutex

	systemPromptTpl prompts.PromptTemplate
	contextTpl      prompts.PromptTemplate
}

type ChatOption func(*Chat) error

func NewChat(opts ...ChatOption) (*Chat, error) {
	// init new chat
	c := Chat{
		completedMessages: make([]llms.MessageContent, 0),
		streamingParts:    make([]string, 0),
		systemPromptTpl:   prompts.NewPromptTemplate(baseSystemPromptTpl, nil),
		contextTpl:        prompts.NewPromptTemplate(baseContextTpl, nil),
	}

	// apply opts
	for _, opt := range opts {
		err := opt(&c)
		if err != nil {
			return nil, err
		}
	}

	// render system prompt
	err := c.pushSystemPrompt()

	return &c, err
}

func WithSystemPromptTemplateFile(path string) ChatOption {
	return func(c *Chat) error {
		// If the file doesn't exist, bail
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		// Read the file
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c.systemPromptTpl = prompts.NewPromptTemplate(string(b), nil)
		return nil
	}
}

func WithContextTemplateFile(path string) ChatOption {
	return func(c *Chat) error {
		// If the file doesn't exist, bail
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		}
		// Read the file
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c.contextTpl = prompts.NewPromptTemplate(string(b), nil)
		return nil
	}
}

func (c *Chat) pushSystemPrompt() error {
	// render the system prompt
	p, err := c.systemPromptTpl.Format(nil)
	if err != nil {
		return err
	}
	c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeSystem, p))
	return nil
}

func (c *Chat) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = false
	c.err = nil
	c.completedMessages = make([]llms.MessageContent, 0)
	c.streamingParts = make([]string, 0)
	c.pushSystemPrompt()
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

func (c *Chat) AddContexts(contexts []schema.Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract slice of content from the documents
	content := make([]string, 0, len(contexts))
	for _, doc := range contexts {
		content = append(content, doc.PageContent)
	}

	// Render the context template
	t, err := c.contextTpl.Format(map[string]interface{}{"contexts": content})
	if err != nil {
		return err
	}
	c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeSystem, t))
	return nil
}

func (c *Chat) streamingPartsToContent() llms.MessageContent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return llms.TextParts(llms.ChatMessageTypeAI, strings.Join(c.streamingParts, ""))
}

// Everything beyond this point implements schema.ChatMessageHistory from langchaingo; this allows for
// easy adapting to the langchaingo API.
// AddMessage adds a message to the store.
func (c *Chat) AddMessage(ctx context.Context, message llms.ChatMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completedMessages = append(c.completedMessages, llms.TextParts(message.GetType(), message.GetContent()))
	return nil
}

// AddUserMessage is a convenience method for adding a human message string
// to the store.
func (c *Chat) AddUserMessage(ctx context.Context, message string) error {
	return c.AddMessage(ctx, llms.HumanChatMessage{Content: message})
}

// AddAIMessage is a convenience method for adding an AI message string to
// the store.
func (c *Chat) AddAIMessage(ctx context.Context, message string) error {
	return c.AddMessage(ctx, llms.AIChatMessage{Content: message})
}

// Clear removes all messages from the store.
func (c *Chat) Clear(ctx context.Context) error {
	c.Reset()
	return nil
}

// Messages retrieves all messages from the store
func (c *Chat) Messages(ctx context.Context) ([]llms.ChatMessage, error) {
	var messages []llms.ChatMessage
	for _, m := range c.Log() {

		// Extract the content as a string
		sb := strings.Builder{}
		for _, part := range m.Parts {
			s, ok := part.(fmt.Stringer)
			if !ok {
				continue
			}
			sb.WriteString(s.String())
		}
		switch m.Role {
		case llms.ChatMessageTypeAI:
			messages = append(messages, llms.AIChatMessage{Content: sb.String()})
		case llms.ChatMessageTypeHuman:
			messages = append(messages, llms.HumanChatMessage{Content: sb.String()})
		case llms.ChatMessageTypeSystem:
			messages = append(messages, llms.SystemChatMessage{Content: sb.String()})
		}
	}
	return messages, nil
}

// SetMessages replaces existing messages in the store
func (c *Chat) SetMessages(ctx context.Context, messages []llms.ChatMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isStreaming = false
	c.err = nil
	c.completedMessages = make([]llms.MessageContent, 0)
	c.streamingParts = make([]string, 0)
	for _, m := range messages {
		switch m.GetType() {
		case llms.ChatMessageTypeAI:
			c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeAI, m.GetContent()))
		case llms.ChatMessageTypeHuman:
			c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeHuman, m.GetContent()))
		case llms.ChatMessageTypeSystem:
			c.completedMessages = append(c.completedMessages, llms.TextParts(llms.ChatMessageTypeSystem, m.GetContent()))
		}
	}
	return nil
}
