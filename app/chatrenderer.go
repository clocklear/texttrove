package app

import (
	"fmt"
	"strings"

	"github.com/clocklear/texttrove/pkg/models"

	"github.com/charmbracelet/lipgloss"
	"github.com/tmc/langchaingo/llms"
)

type chatRenderer struct {
	senderStyle lipgloss.Style
	llmStyle    lipgloss.Style
	errorStyle  lipgloss.Style
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
	var buf strings.Builder
	// Start by writing the role of the message
	switch m.Role {
	case llms.ChatMessageTypeHuman:
		buf.WriteString(r.senderStyle.Render("You: "))
	case llms.ChatMessageTypeSystem:
		buf.WriteString(r.llmStyle.Render("System: "))
	case llms.ChatMessageTypeAI:
		buf.WriteString(r.llmStyle.Render("AI: "))
	default:
		buf.WriteString(r.llmStyle.Render("Bot: "))
	}
	// MessageContent has a role and a sequence of parts
	// Each of the parts _might_ implement Stringer.
	// If it does, call it and append the value to the buffer.
	// If it doesn't, append the type name.
	for _, part := range m.Parts {
		switch p := part.(type) {
		case fmt.Stringer:
			_, err := buf.WriteString(p.String())
			if err != nil {
				return "", err
			}
		default:
			_, err := buf.WriteString(fmt.Sprintf("%T", p))
			if err != nil {
				return "", err
			}
		}
		// New line
		buf.WriteString("\n")
	}
	return buf.String(), nil
}
