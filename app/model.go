package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clocklear/texttrove/pkg/models"

	"github.com/tmc/langchaingo/llms/ollama"
)

type status string

const (
	StatusInitializing status = "Initializing"
	StatusReady        status = "Ready"
	StatusQuerying     status = "Querying"
	StatusRetrieving   status = "Retrieving"
)

type Config struct {
	AppName         string
	ChatInputHeight int
	Keys            KeyMap
	SenderColor     uint
	LLMColor        uint
	ErrorColor      uint
	SpinnerColor    uint
	LLM             *ollama.LLM
}

func DefaultConfig() Config {
	return Config{
		AppName:         "TextTrove",
		ChatInputHeight: 5,
		SenderColor:     5,  // ANSI Magenta
		LLMColor:        4,  // ANSI Blue
		ErrorColor:      1,  // ANSI Red
		SpinnerColor:    69, // ANSI Light Blue
		Keys:            DefaultKeyMap(),
	}
}

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type Model struct {
	ready        bool
	help         help.Model
	llmStream    chan responseMsg
	viewport     viewport.Model
	chats        []models.Chat
	selectedChat uint // future use
	textarea     textarea.Model
	spinner      spinner.Model
	chatRenderer chatRenderer
	status       status

	cfg Config
}

func New(cfg Config) Model {
	// Create textarea for receiving user input
	ta := textarea.New()
	ta.Placeholder = "Type your message"
	ta.Prompt = "┃ "
	ta.SetHeight(cfg.ChatInputHeight)
	ta.CharLimit = 0
	ta.Focus()                                       // Direct keyboard input here
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle() // Remove cursor line styling
	ta.ShowLineNumbers = false                       // Hide line numbers

	// Create a spinner for showing that the app is loading
	spn := spinner.New()
	spn.Style = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.SpinnerColor))
	spn.Spinner = spinner.Points

	return Model{
		cfg:       cfg,
		textarea:  ta,
		help:      help.New(),
		spinner:   spn,
		llmStream: make(chan responseMsg),
		chats:     []models.Chat{models.NewChat()},
		chatRenderer: chatRenderer{
			senderStyle: lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.SenderColor)),
			llmStyle:    lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.LLMColor)),
			errorStyle:  lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.ErrorColor)),
		},
		status: StatusInitializing,
	}
}

func (m Model) activeChat() *models.Chat {
	return &m.chats[m.selectedChat]
}

func (m *Model) setStatus(s status) {
	m.status = s
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		waitForActivity(m.llmStream),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Grab a handle for the selected chat
	chat := m.activeChat()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		helpHeight := lipgloss.Height(m.helpView())
		verticalMarginHeight := headerHeight + footerHeight + m.cfg.ChatInputHeight + helpHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.textarea.SetWidth(msg.Width)
			m.ready = true
			m.setStatus(StatusReady)
		} else {
			m.viewport.Width = msg.Width
			m.textarea.SetWidth(msg.Width)
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.cfg.Keys.Quit):
			// Quit
			return m, tea.Quit
		case key.Matches(msg, m.cfg.Keys.Help):
			// Toggle small/large help
			m.help.ShowAll = !m.help.ShowAll
			// TODO: figure out how to trigger resize event so things get painted in the correct location
		case key.Matches(msg, m.cfg.Keys.Send):
			// Reset (chat) err
			chat.ClearError()
			chat.BeginStreaming()
			m.setStatus(StatusQuerying)
			v := m.textarea.Value()

			if v == "" {
				// Don't send empty messages.
				return m, nil
			}

			// Append the user message to the ongoing chat
			chat.AppendUserMessage(m.textarea.Value())
			m.viewport.SetContent(m.chatRenderer.Render(chat))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			// Send the message to the LLM
			return m, tea.Batch(
				submitChat(context.Background(), m.cfg.LLM, chat.Log(), m.llmStream),
				m.spinner.Tick,
			)
		case key.Matches(msg, m.cfg.Keys.NewChat):
			// Only allow new chats when the current chat is not streaming
			if !chat.IsStreaming() {
				// Reset the chat
				chat.Reset()
				m.setStatus(StatusReady)
			}

		default:
			// Allow the text area to respond to these messages
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		if !chat.IsStreaming() {
			// Only update the spinner if we're streaming
			return m, nil
		}
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)

	case responseMsg:
		// Handle incoming messages
		if msg.err != nil {
			chat.SetError(msg.err)
			m.setStatus(StatusReady)
		} else if msg.isComplete {
			chat.EndStreaming()
			m.setStatus(StatusReady)
		} else {
			// Append the incoming message to the buffer
			chat.StreamChunk(msg.chunk)
			m.setStatus(StatusRetrieving)
		}
		// Refresh the viewport content
		m.viewport.SetContent(m.chatRenderer.Render(chat))
		m.viewport.GotoBottom()
		// Await the next message
		cmds = append(cmds, waitForActivity(m.llmStream))
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
		m.textarea.View(),
		m.helpView(),
	)
}

func (m Model) helpView() string {
	return m.help.View(m.cfg.Keys)
}

func (m Model) headerView() string {
	titleText := m.cfg.AppName
	// chat := m.activeChat()
	// if chat != nil && chat.IsStreaming() {
	// 	titleText += " " + m.spinner.View()
	// }
	title := titleStyle.Render(titleText)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) footerView() string {
	// info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	info := string(m.status)
	// TODO: this needs to correctly contemplate multiple chats
	chat := m.activeChat()
	if chat != nil && chat.IsStreaming() {
		info += " " + m.spinner.View()
	}
	info = infoStyle.Render(info)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
