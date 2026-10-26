package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
)

type Account interface {
	Profile() profile.Profile
	Persistent() bool
	Update(profile.Profile) error
}

type screen int

const (
	screenLobby screen = iota
	screenRoom
	screenSettings
)

const (
	minWidth  = 50
	minHeight = 20
	appTitle  = "ssh-chat"
)

type (
	hubUpdateMsg    struct{}
	disconnectedMsg struct{}
	joinedRoomMsg   struct{}
	leftRoomMsg     struct{}
	openSettingsMsg struct{}
	settingsDoneMsg struct{ saved bool }
	renameMsg       struct{ name string }
	profileSavedMsg struct{ err error }
)

type Model struct {
	client   *chat.Client
	account  Account
	theme    theme
	width    int
	height   int
	screen   screen
	back     screen
	lobby    lobby
	room     room
	settings settings
}

func New(client *chat.Client, account Account) Model {
	t := newTheme(true)
	return Model{
		client:  client,
		account: account,
		theme:   t,
		lobby:   newLobby(client, t),
		room:    newRoom(client, t, account.Profile().Location()),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, waitForHub(m.client))
}

func waitForHub(c *chat.Client) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-c.Updates():
			return hubUpdateMsg{}
		case <-c.Done():
			return disconnectedMsg{}
		}
	}
}

func emit(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func saveProfile(a Account, p profile.Profile) tea.Cmd {
	return func() tea.Msg { return profileSavedMsg{err: a.Update(p)} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.lobby = m.lobby.setSize(m.width, m.height)
		m.room = m.room.setSize(m.width, m.height)
		if m.screen == screenSettings {
			m.settings = m.settings.setSize(m.width, m.height)
		}
		return m, nil

	case tea.BackgroundColorMsg:
		m.theme = newTheme(msg.IsDark())
		m.lobby = m.lobby.setTheme(m.theme)
		m.room = m.room.setTheme(m.theme)
		if m.screen == screenSettings {
			m.settings = m.settings.setTheme(m.theme)
		}
		return m, nil

	case hubUpdateMsg:
		m.lobby = m.lobby.refresh()
		m.room = m.room.refresh()
		return m, waitForHub(m.client)

	case disconnectedMsg:
		return m, tea.Quit

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case joinedRoomMsg:
		m.screen = screenRoom
		m.room = m.room.enter()
		return m, nil

	case leftRoomMsg:
		m.screen = screenLobby
		m.lobby = m.lobby.refresh()
		return m, nil

	case openSettingsMsg:
		if m.screen == screenSettings {
			return m, nil
		}
		m.back = m.screen
		m.screen = screenSettings
		m.settings = newSettings(m.account, m.theme, m.width, m.height)
		return m, nil

	case settingsDoneMsg:
		m.screen = m.back
		if msg.saved {
			m = m.profileChanged("Profile saved")
		}
		return m, nil

	case renameMsg:
		p := m.account.Profile()
		p.Name = msg.name
		return m, saveProfile(m.account, p)

	case profileSavedMsg:
		if m.screen == screenSettings {
			break
		}
		if msg.err != nil {
			m.room = m.room.flash(msg.err.Error(), true)
			return m, nil
		}
		return m.profileChanged(fmt.Sprintf("You are now %s", m.account.Profile().Name)), nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenLobby:
		m.lobby, cmd = m.lobby.update(msg)
	case screenRoom:
		m.room, cmd = m.room.update(msg)
	case screenSettings:
		m.settings, cmd = m.settings.update(msg)
	}
	return m, cmd
}

func (m Model) profileChanged(notice string) Model {
	m.room = m.room.setLocation(m.account.Profile().Location()).refresh().flash(notice, false)
	m.lobby = m.lobby.refresh().flash(notice, false)
	return m
}

func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	v.WindowTitle = appTitle
	if m.width == 0 || m.height == 0 {
		return v
	}
	if m.width < minWidth || m.height < minHeight {
		msg := fmt.Sprintf("Terminal too small\nneed %dx%d, have %dx%d", minWidth, minHeight, m.width, m.height)
		v.Content = m.theme.backdrop(m.width, m.height,
			m.theme.fg(m.theme.primary).Bold(true).Align(lipgloss.Center).Render(msg))
		return v
	}

	switch m.screen {
	case screenLobby:
		v.Content = m.lobby.view()
	case screenRoom:
		v.Content = m.room.view()
		v.MouseMode = tea.MouseModeCellMotion
		v.WindowTitle = "#" + m.room.snap.Name + " - " + appTitle
	case screenSettings:
		v.Content = m.settings.view()
	}
	return v
}
