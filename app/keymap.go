package app

import (
	"github.com/charmbracelet/bubbles/v2/key"
)

type KeyMap struct {
	ScrollChatUp   key.Binding
	ScrollChatDown key.Binding
	Help           key.Binding
	Send           key.Binding
	NewChat        key.Binding
	CloseChat      key.Binding // NYI
	Quit           key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Send, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ScrollChatUp, k.ScrollChatDown, k.NewChat}, // first column
		{k.Help, k.Send, k.Quit},                      // second column
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		ScrollChatUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up chat"),
		),
		ScrollChatDown: key.NewBinding(
			key.WithKeys("pgdn"),
			key.WithHelp("pgdn", "page down chat"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("f1", "toggle full help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Send: key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+enter", "send message"),
		),
		NewChat: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new chat"),
		),
	}
}
