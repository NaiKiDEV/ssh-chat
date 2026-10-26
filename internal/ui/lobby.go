package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/charmbracelet/x/ansi"
)

const (
	lobbyDialogWidth = 46
	columnWidth      = 36
)

const (
	lobbyFocusRooms = iota
	lobbyFocusInput
	lobbyFocusButtons
	lobbyFocusCount
)

const (
	buttonProfile = iota
	buttonQuit
	buttonJoin
	buttonCount
)

type lobby struct {
	client *chat.Client
	theme  theme
	keys   lobbyKeys
	help   help.Model
	input  textinput.Model
	me     chat.Member
	rooms  []chat.RoomInfo
	cursor int
	focus  int
	button int
	err    string
	notice string
	width  int
	height int
}

func newLobby(c *chat.Client, t theme) lobby {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "room name"
	in.CharLimit = chat.MaxRoomNameLen + 1
	in.SetWidth(columnWidth - 5)

	l := lobby{client: c, keys: newLobbyKeys(), help: newHelp(), input: in, button: buttonJoin}
	l = l.setTheme(t).refresh()
	if len(l.rooms) == 0 {
		return l.focusOn(lobbyFocusInput)
	}
	return l.focusOn(lobbyFocusRooms)
}

func (l lobby) setTheme(t theme) lobby {
	l.theme = t
	l.input.SetStyles(t.inputStyles())
	l.help.Styles = t.helpStyles()
	return l
}

func (l lobby) setSize(w, h int) lobby {
	l.width, l.height = w, h
	l.help.SetWidth(w - 2)
	return l
}

func (l lobby) refresh() lobby {
	l.me = l.client.Member()
	l.rooms = l.client.Rooms()
	l.cursor = clamp(l.cursor, 0, len(l.rooms)-1)
	return l
}

func (l lobby) flash(text string, isErr bool) lobby {
	l.err, l.notice = "", ""
	if isErr {
		l.err = text
	} else {
		l.notice = text
	}
	return l
}

func (l lobby) focusOn(f int) lobby {
	l.focus = (f + lobbyFocusCount) % lobbyFocusCount
	if l.focus == lobbyFocusRooms && len(l.rooms) == 0 {
		l.focus = lobbyFocusInput
	}
	l.input.Blur()
	if l.focus == lobbyFocusInput {
		l.input.Focus()
	}
	return l
}

func (l lobby) update(msg tea.Msg) (lobby, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		l.input, cmd = l.input.Update(msg)
		return l, cmd
	}

	switch {
	case key.Matches(press, l.keys.Profile):
		return l, emit(openSettingsMsg{})
	case key.Matches(press, l.keys.Next):
		return l.focusOn(l.focus + 1), nil
	case key.Matches(press, l.keys.Prev):
		return l.focusOn(l.focus - 1), nil
	case key.Matches(press, l.keys.Clear):
		l.input.Reset()
		return l.flash("", false), nil
	}

	switch l.focus {
	case lobbyFocusRooms:
		return l.updateRooms(press)
	case lobbyFocusButtons:
		return l.updateButtons(press)
	}

	if key.Matches(press, l.keys.Select) {
		return l.join(strings.TrimSpace(l.input.Value()))
	}
	l = l.flash("", false)
	var cmd tea.Cmd
	l.input, cmd = l.input.Update(press)
	return l, cmd
}

func (l lobby) updateRooms(press tea.KeyPressMsg) (lobby, tea.Cmd) {
	switch {
	case key.Matches(press, l.keys.Up):
		l.cursor = clamp(l.cursor-1, 0, len(l.rooms)-1)
	case key.Matches(press, l.keys.Down):
		l.cursor = clamp(l.cursor+1, 0, len(l.rooms)-1)
	case key.Matches(press, l.keys.Select):
		return l.join(l.rooms[l.cursor].Name)
	case press.Text != "":
		l = l.focusOn(lobbyFocusInput)
		var cmd tea.Cmd
		l.input, cmd = l.input.Update(press)
		return l, cmd
	}
	return l, nil
}

func (l lobby) updateButtons(press tea.KeyPressMsg) (lobby, tea.Cmd) {
	switch {
	case key.Matches(press, l.keys.Left):
		l.button = (l.button - 1 + buttonCount) % buttonCount
	case key.Matches(press, l.keys.Right):
		l.button = (l.button + 1) % buttonCount
	case key.Matches(press, l.keys.Select):
		switch l.button {
		case buttonProfile:
			return l, emit(openSettingsMsg{})
		case buttonQuit:
			return l, tea.Quit
		case buttonJoin:
			target := strings.TrimSpace(l.input.Value())
			if target == "" && len(l.rooms) > 0 {
				target = l.rooms[l.cursor].Name
			}
			return l.join(target)
		}
	}
	return l, nil
}

func (l lobby) join(target string) (lobby, tea.Cmd) {
	if target == "" {
		return l.flash("room id empty", true).focusOn(lobbyFocusInput), nil
	}
	if _, err := l.client.Join(target); err != nil {
		return l.flash(err.Error(), true).focusOn(lobbyFocusInput), nil
	}
	l.input.Reset()
	return l.flash("", false), emit(joinedRoomMsg{})
}

func (l lobby) view() string {
	footer := helpLine(l.help, l.keys, l.width)
	avail := l.height - lipgloss.Height(footer)

	dialog := l.dialog(true, len(l.rooms))
	if lipgloss.Height(dialog) > avail {
		dialog = l.dialog(false, len(l.rooms))
	}
	if over := lipgloss.Height(dialog) - avail; over > 0 {
		dialog = l.dialog(false, max(len(l.rooms)-over, 1))
	}

	return lipgloss.JoinVertical(lipgloss.Left, l.theme.backdrop(l.width, avail, dialog), footer)
}

func (l lobby) dialog(showLogo bool, maxRows int) string {
	t := l.theme

	var parts []string
	if showLogo {
		parts = append(parts, t.fg(t.primary).MarginBottom(1).Render(logo))
	}
	name := t.fg(t.primary).Bold(true).Render(l.me.Name)
	parts = append(parts, t.regular().Render("Welcome, ")+name+t.regular().Render("!"))
	if l.me.Guest {
		parts = append(parts, t.fg(t.muted).Render("guest session"), t.fg(t.muted).Render("use an SSH key to keep your name"))
	}

	status := t.formError(l.err)
	if status == "" && l.notice != "" {
		status = t.fg(t.primary).Render(l.notice)
	}

	parts = append(parts,
		"",
		t.label("Rooms", l.focus == lobbyFocusRooms),
		l.roomList(maxRows),
		"",
		t.label("Room Id", l.focus == lobbyFocusInput),
		t.field(columnWidth, l.focus == lobbyFocusInput, l.input.View()),
		status,
		l.buttons(),
	)
	return t.dialog(lobbyDialogWidth, parts...)
}

func (l lobby) roomList(maxRows int) string {
	t := l.theme
	if len(l.rooms) == 0 {
		return t.fg(t.muted).Render("no public rooms, create one below")
	}

	first := clamp(l.cursor-maxRows+1, 0, len(l.rooms)-maxRows)
	visible := l.rooms[first:min(first+maxRows, len(l.rooms))]
	rows := make([]string, len(visible))
	for j, r := range visible {
		i := first + j
		style := t.regular()
		marker := "  "
		if i == l.cursor {
			style = t.fg(t.grey).Bold(true)
			marker = t.fg(t.grey).Render("> ")
			if l.focus == lobbyFocusRooms {
				style = t.fg(t.primary).Bold(true)
				marker = t.fg(t.primary).Render("> ")
			}
		}
		count := "empty"
		if r.Online > 0 {
			count = fmt.Sprintf("%d online", r.Online)
		}
		left := marker + style.Render(ansi.Truncate(r.Name, columnWidth-14, ellipsis))
		right := t.fg(t.muted).Render(count)
		gap := max(columnWidth-lipgloss.Width(left)-lipgloss.Width(right), 1)
		rows[j] = left + strings.Repeat(" ", gap) + right
	}
	return strings.Join(rows, "\n")
}

func (l lobby) buttons() string {
	t := l.theme
	active := func(b int) bool { return l.focus == lobbyFocusButtons && l.button == b }
	return lipgloss.JoinHorizontal(lipgloss.Top,
		t.button("Profile", active(buttonProfile)), "  ",
		t.button("Quit", active(buttonQuit)), "  ",
		t.button("Join", active(buttonJoin)),
	)
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}
