package models

import (
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type Chat struct {
	title             string // TODO: Future use; summarize the conversation
	prompt            string // TODO: Future use; customizable prompt for the chat
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

func (c *Chat) streamingPartsToContent() llms.MessageContent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return llms.TextParts(llms.ChatMessageTypeSystem, strings.Join(c.streamingParts, ""))
}
