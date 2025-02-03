package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/clocklear/texttrove/pkg/models"

	"github.com/charmbracelet/bubbles/v2/cursor"
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/tmc/langchaingo/schema"
)

// Ragger describes what we expect to be true of a thing that can RAG documents
type Ragger interface {
	LoadDocuments(ctx context.Context, basePath, filePattern string) error
	Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]any) ([]schema.Document, error)
	Shutdown(ctx context.Context) error
}

type status string

const (
	StatusInitializing status = "Initializing"
	StatusReady        status = "Ready"
	StatusQuerying     status = "Querying"
	StatusRetrieving   status = "Retrieving"
)

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
	ready          bool
	help           help.Model
	dispatchStream chan tea.Msg
	viewport       viewport.Model
	// chats          []*models.Chat
	// selectedChat   uint // future use
	chat         *models.Chat
	textarea     textarea.Model
	spinner      spinner.Model
	chatRenderer chatRenderer
	status       status
	logger       Logger

	cfg Config
}

func New(cfg Config) (Model, error) {
	// Create textarea for receiving user input
	ta := textarea.New()
	ta.Placeholder = "Type your message"
	ta.Prompt = "┃ "
	ta.SetHeight(cfg.ChatInputHeight)
	ta.CharLimit = 0
	ta.Focus()                                         // Direct keyboard input here
	ta.Styles.Focused.CursorLine = lipgloss.NewStyle() // Remove cursor line styling
	ta.ShowLineNumbers = false                         // Hide line numbers

	// Create a logger pane
	l := NewLogger(3, cfg.LoggerHistorySize, lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.LogColor)))

	// Create a spinner for showing that the app is loading
	spn := spinner.New()
	spn.Style = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.SpinnerColor))
	spn.Spinner = spinner.Points

	return Model{
		cfg:            cfg,
		textarea:       ta,
		help:           help.New(),
		spinner:        spn,
		dispatchStream: make(chan tea.Msg),
		// chats:          []*models.Chat{chat},
		chat: cfg.Chat,
		chatRenderer: chatRenderer{
			senderStyle:      lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.SenderColor)),
			llmStyle:         lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.LLMColor)),
			errorStyle:       lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(cfg.ErrorColor)),
			markdownRenderer: cfg.MarkdownRenderer,
			showPrompt:       cfg.ShowPromptInChat,
		},
		status: StatusInitializing,
		logger: l,
	}, nil
}

func (m Model) activeChat() *models.Chat {
	// return m.chats[m.selectedChat]
	return m.chat
}

func (m *Model) setStatus(s status) {
	m.status = s
}

func waitForActivity(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (m Model) Init() (tea.Model, tea.Cmd) {
	return m, tea.Batch(
		textarea.Blink,
		waitForActivity(m.dispatchStream),
	)
}

func (m Model) Log(msg string) {

	m.dispatchStream <- LogMsg(msg)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Grab a handle for the selected chat
	chat := m.activeChat()

	// We want to propagate all non-keyboard events to the viewport.
	propagateEventToViewport := true

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		helpHeight := lipgloss.Height(m.helpView())
		loggerHeight := lipgloss.Height(m.logger.View())
		verticalMarginHeight := headerHeight + footerHeight + m.cfg.ChatInputHeight + helpHeight + loggerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMarginHeight))
			m.viewport.SetYOffset(headerHeight)
			m.textarea.SetWidth(msg.Width)
			m.ready = true
			m.setStatus(StatusReady)
		} else {
			m.viewport.SetWidth(msg.Width)
			m.textarea.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMarginHeight)
		}
		// The log viewport needs to be made aware of this as well
		m.logger, cmd = m.logger.Update(msg)
		cmds = append(cmds, cmd)
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

			if m.cfg.ConversationAgent != nil {
				// TODO: Do I need anything here?
			} else {
				// Try to find supporting information for the user's query
				// and add that to conversation as additional context
				ctxs, err := m.cfg.RAG.Query(context.Background(), v, 5, nil, nil) // TODO: Use 'where'?
				if err != nil {
					// m.Log(err.Error())
					fmt.Println(err.Error())
					chat.SetError(err)
				} else {
					err = chat.AddContexts(ctxs)
					if err != nil {
						// m.Log(err.Error())
						fmt.Println(err.Error())
						chat.SetError(err)
					}
				}
			}
			// Append the user message to the ongoing chat
			chat.AppendUserMessage(m.textarea.Value())
			m.viewport.SetContent(m.chatRenderer.Render(chat))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			if m.cfg.ConversationAgent != nil {
				return m, tea.Batch(
					submitChatAgent(context.Background(), m.cfg.ConversationAgent, "", m.dispatchStream),
					m.spinner.Tick,
				)
			} else {
				// Send the message to the LLM
				return m, tea.Batch(
					submitChat(context.Background(), m.cfg.ConversationLLM, chat.Log(), m.dispatchStream),
					m.spinner.Tick,
				)
			}
		case key.Matches(msg, m.cfg.Keys.NewChat):
			// Only allow new chats when the current chat is not streaming
			if !chat.IsStreaming() {
				// Reset the chat
				chat.Reset()
				m.viewport.SetContent("")
				m.setStatus(StatusReady)
			}

		default:
			// Allow the text area to respond to these messages
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
		// We don't want to propagate the event to the viewport
		propagateEventToViewport = false

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

	case LLMStreamingResponseMsg:
		// Handle incoming messages
		if msg.err != nil {
			chat.SetError(msg.err)
			// m.Log(msg.err.Error())
			fmt.Println(msg.err.Error())
			m.setStatus(StatusReady)
		} else {
			// Append the incoming message to the buffer
			chat.StreamChunk(msg.chunk)
			m.setStatus(StatusRetrieving)
			if msg.isComplete {
				chat.EndStreaming()
				m.setStatus(StatusReady)
			}
		}
		// Refresh the viewport content
		m.viewport.SetContent(m.chatRenderer.Render(chat))
		m.viewport.GotoBottom()
		// Await the next message
		cmds = append(cmds, waitForActivity(m.dispatchStream))

	case LogMsg:
		// Invoke the logger with this message
		m.logger, cmd = m.logger.Update(msg)
		// Await the next message
		cmds = append(cmds, waitForActivity(m.dispatchStream), cmd)
		// Can bail here
		return m, tea.Batch(cmds...)
	}

	// Handle events in the viewport, if we're supposed to
	if propagateEventToViewport {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
		m.textarea.View(),
		m.logger.View(),
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
	line := strings.Repeat("─", max(0, m.viewport.Width()-lipgloss.Width(title)))
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
	line := strings.Repeat("─", max(0, m.viewport.Width()-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
