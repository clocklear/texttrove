package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// Logger is a bubbletea component that displays log messages.
type Logger struct {
	viewport    viewport.Model
	messages    []string
	historySize uint
	ready       bool
	height      uint
	style       lipgloss.Style
}

type LogMsg string

// NewLogger creates a new Logger component
func NewLogger(height, historySize uint, style lipgloss.Style) Logger {
	return Logger{
		messages:    make([]string, 0),
		historySize: historySize,
		height:      height,
		style:       style,
	}
}

func (m *Logger) log(msg string) {
	// Append the log message to the messages slice
	m.messages = append(m.messages, msg)
	// If the messages slice is larger than maxSize, remove entries from the beginning
	// such that the size is not larger than maxSize
	if len(m.messages) > int(m.historySize) {
		m.messages = m.messages[len(m.messages)-int(m.historySize):]
	}
	// Set the viewport content
	sb := new(strings.Builder)
	for n, msg := range m.messages {
		if n != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(msg)
	}
	m.viewport.SetContent(m.style.Render(sb.String()))
	m.viewport.GotoBottom()
}

func (m Logger) Init() (Logger, tea.Cmd) {
	return m, nil
}

func (m Logger) Update(msg tea.Msg) (Logger, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(int(m.height)))
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
		}
	case LogMsg:
		m.log(string(msg))
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Logger) View() string {
	if !m.ready {
		return ""
	}
	return m.viewport.View()
}
