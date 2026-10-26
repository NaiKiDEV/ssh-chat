package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func newHelp() help.Model {
	h := help.New()
	h.ShortSeparator = " | "
	h.FullSeparator = "   "
	h.Ellipsis = ellipsis
	return h
}

func helpLine(h help.Model, k help.KeyMap, width int) string {
	return lipgloss.NewStyle().Padding(0, 1).Render(ansi.Truncate(h.View(k), width-2, ellipsis))
}

type lobbyKeys struct {
	Next    key.Binding
	Prev    key.Binding
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Select  key.Binding
	Clear   key.Binding
	Profile key.Binding
	Quit    key.Binding
}

func newLobbyKeys() lobbyKeys {
	return lobbyKeys{
		Next:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch")),
		Prev:    key.NewBinding(key.WithKeys("shift+tab")),
		Up:      key.NewBinding(key.WithKeys("up"), key.WithHelp("up/down", "choose")),
		Down:    key.NewBinding(key.WithKeys("down")),
		Left:    key.NewBinding(key.WithKeys("left", "h")),
		Right:   key.NewBinding(key.WithKeys("right", "l")),
		Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Clear:   key.NewBinding(key.WithKeys("esc")),
		Profile: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "profile")),
		Quit:    key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
}

func (k lobbyKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Up, k.Select, k.Profile, k.Quit}
}

func (k lobbyKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type roomKeys struct {
	Select   key.Binding
	Newline  key.Binding
	Blur     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Profile  key.Binding
	Commands key.Binding
	Quit     key.Binding
}

func newRoomKeys() roomKeys {
	return roomKeys{
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		Newline:  key.NewBinding(key.WithKeys("ctrl+enter", "ctrl+j", "shift+enter", "alt+enter"), key.WithHelp("ctrl+enter", "newline")),
		Blur:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "scroll")),
		PageUp:   key.NewBinding(key.WithKeys("pgup")),
		PageDown: key.NewBinding(key.WithKeys("pgdown")),
		Profile:  key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "profile")),
		Commands: key.NewBinding(key.WithKeys("/help"), key.WithHelp("/help", "commands")),
		Quit:     key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
}

func (k roomKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Newline, k.Blur, k.Profile, k.Commands, k.Quit}
}

func (k roomKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type settingsKeys struct {
	Next   key.Binding
	Prev   key.Binding
	Left   key.Binding
	Right  key.Binding
	Select key.Binding
	Save   key.Binding
	Cancel key.Binding
}

func newSettingsKeys() settingsKeys {
	return settingsKeys{
		Next:   key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("tab", "next")),
		Prev:   key.NewBinding(key.WithKeys("shift+tab", "up")),
		Left:   key.NewBinding(key.WithKeys("left"), key.WithHelp("left/right", "change")),
		Right:  key.NewBinding(key.WithKeys("right")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Save:   key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (k settingsKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Left, k.Select, k.Save, k.Cancel}
}

func (k settingsKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
