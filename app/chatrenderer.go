package app

import (
	"fmt"
	"strings"

	"github.com/clocklear/texttrove/pkg/models"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/tmc/langchaingo/llms"
)

type chatRenderer struct {
	senderStyle      lipgloss.Style
	llmStyle         lipgloss.Style
	errorStyle       lipgloss.Style
	markdownRenderer *glamour.TermRenderer
}

func (r *chatRenderer) Render(c *models.Chat) string {
	var buf strings.Builder
	for _, m := range c.Log() {
		s, err := r.renderMessageContent(&m)
		c.SetError(err)
		buf.WriteString(s)
	}

	// If there is an error, render that as well
	err := c.Error()
	if err != nil {
		buf.WriteString(r.errorStyle.Render(err.Error()))
	}
	return buf.String()
}

func (r *chatRenderer) renderMessageContent(m *llms.MessageContent) (string, error) {
	var outputBuf, messageBuf strings.Builder

	// Start by writing the role of the message
	switch m.Role {
	case llms.ChatMessageTypeHuman:
		outputBuf.WriteString(r.senderStyle.Render("You: "))
	case llms.ChatMessageTypeAI:
		outputBuf.WriteString(r.llmStyle.Render("AI: "))
	case llms.ChatMessageTypeSystem:
		// System messages will be reserved for passing the prompt to the system.
		// We don't want to render these.
		return "", nil
	default:
		outputBuf.WriteString(r.llmStyle.Render("Bot: "))
	}
	// MessageContent has a role and a sequence of parts
	// Each of the parts _might_ implement Stringer.
	// If it does, call it and append the value to the buffer.
	// If it doesn't, append the type name.
	for _, part := range m.Parts {
		switch p := part.(type) {
		case fmt.Stringer:
			_, err := messageBuf.WriteString(p.String())
			if err != nil {
				return "", err
			}
		default:
			_, err := messageBuf.WriteString(fmt.Sprintf("%T", p))
			if err != nil {
				return "", err
			}
		}
		// New line
		messageBuf.WriteString("\n")
	}

	// Pass the message through the markdown renderer
	message, err := r.markdownRenderer.Render(messageBuf.String())
	if err != nil {
		return "", err
	}

	// Append the rendered message to the output buffer
	outputBuf.WriteString(message)

	// Render the output buffer
	return outputBuf.String(), nil
}
