package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/charmbracelet/x/ansi"
)

const (
	sidebarWidth       = 23
	sidebarMinScreen   = 72
	compactHeaderBelow = 24
	inputMinHeight     = 1
	inputMaxHeight     = 5
	contentPadding     = 1
	commandsHelpLine   = "/me <action> | /nick <name> | /join <room> | /leave | /settings | /quit | //text sends /text"
)

const (
	roomFocusInput = iota
	roomFocusNone
	roomFocusCount
)

type cacheKey struct {
	id      uint64
	grouped bool
}

type room struct {
	client   *chat.Client
	theme    theme
	keys     roomKeys
	help     help.Model
	input    textarea.Model
	viewport viewport.Model
	loc      *time.Location
	snap     chat.RoomSnapshot
	me       chat.Member
	cache    map[cacheKey]string
	cacheW   int
	starts   map[uint64]int
	focus    int
	unread   int
	notice   string
	isErr    bool
	width    int
	height   int
}

func newRoom(c *chat.Client, t theme, loc *time.Location) room {
	in := textarea.New()
	in.Prompt = ""
	in.Placeholder = "Type your message..."
	in.ShowLineNumbers = false
	in.CharLimit = chat.MaxMessageLen
	in.DynamicHeight = true
	in.MinHeight = inputMinHeight
	in.MaxHeight = inputMaxHeight
	in.KeyMap.InsertNewline = newRoomKeys().Newline
	in.Focus()

	vp := viewport.New()
	vp.MouseWheelEnabled = true

	r := room{
		client:   c,
		keys:     newRoomKeys(),
		help:     newHelp(),
		input:    in,
		viewport: vp,
		loc:      loc,
		cache:    map[cacheKey]string{},
	}
	return r.setTheme(t)
}

func (r room) setTheme(t theme) room {
	r.theme = t
	r.input.SetStyles(t.textareaStyles())
	r.help.Styles = t.helpStyles()
	r.cache = map[cacheKey]string{}
	return r.render()
}

func (r room) setSize(w, h int) room {
	r.width, r.height = w, h
	r.help.SetWidth(w - 2)
	r.input.SetWidth(max(w-2-4, 10))
	return r.layout()
}

func (r room) setLocation(loc *time.Location) room {
	r.loc = loc
	r.cache = map[cacheKey]string{}
	return r.render()
}

func (r room) flash(text string, isErr bool) room {
	r.notice, r.isErr = text, isErr
	return r
}

func (r room) focusOn(f int) room {
	r.focus = (f + roomFocusCount) % roomFocusCount
	r.input.Blur()
	if r.focus == roomFocusInput {
		r.input.Focus()
	}
	return r
}

func (r room) enter() room {
	r.snap = chat.RoomSnapshot{}
	r.cache = map[cacheKey]string{}
	r.unread = 0
	r.notice = ""
	r.input.Reset()
	r = r.focusOn(roomFocusInput).layout().refresh()
	r.viewport.GotoBottom()
	return r
}

func (r room) refresh() room {
	snap, ok := r.client.Room()
	if !ok {
		return r
	}
	r.me = r.client.Member()

	pos := r.scrollPosition()
	r.unread += r.countNew(snap.Messages)
	r.snap = snap
	r = r.render().restoreScroll(pos)
	if pos.atBottom {
		r.unread = 0
	}
	return r
}

type scrollPos struct {
	atBottom bool
	anchor   uint64
	offset   int
	anchored bool
}

func (r room) scrollPosition() scrollPos {
	pos := scrollPos{atBottom: r.viewport.AtBottom()}
	top := r.viewport.YOffset()
	best := -1
	for id, start := range r.starts {
		if start <= top && start > best {
			best, pos.anchor = start, id
		}
	}
	if best >= 0 {
		pos.offset, pos.anchored = top-best, true
	}
	return pos
}

func (r room) restoreScroll(pos scrollPos) room {
	switch {
	case pos.atBottom:
		r.viewport.GotoBottom()
	case pos.anchored:
		start, ok := r.starts[pos.anchor]
		if !ok {
			start, pos.offset = 0, 0
		}
		r.viewport.SetYOffset(start + pos.offset)
	}
	return r
}

func (r room) countNew(msgs []chat.Message) int {
	var last uint64
	if n := len(r.snap.Messages); n > 0 {
		last = r.snap.Messages[n-1].ID
	}
	count := 0
	for _, m := range msgs {
		if m.ID > last && m.Author.ID != r.me.ID && (m.Kind == chat.KindChat || m.Kind == chat.KindEmote) {
			count++
		}
	}
	return count
}

func (r room) showSidebar() bool {
	return r.width >= sidebarMinScreen
}

func (r room) header() string {
	t := r.theme
	if r.height < compactHeaderBelow {
		return t.fg(t.primary).Bold(true).Padding(0, 1, 1).Render("# " + r.snap.Name)
	}
	return t.fg(t.primary).Padding(0, 1, 1).Render(logo)
}

func (r room) layout() room {
	form := 1 + 2 + r.input.Height() + 1
	vpWidth := r.width
	if r.showSidebar() {
		vpWidth -= sidebarWidth
	}

	pos := r.scrollPosition()
	r.viewport.SetWidth(max(vpWidth, 1))
	r.viewport.SetHeight(max(r.height-lipgloss.Height(r.header())-form, 1))
	return r.render().restoreScroll(pos)
}

func (r room) render() room {
	width := r.viewport.Width() - 2*contentPadding
	if width <= 0 {
		return r
	}
	if width != r.cacheW {
		r.cache = map[cacheKey]string{}
		r.cacheW = width
	}

	next := make(map[cacheKey]string, len(r.snap.Messages))
	starts := make(map[uint64]int, len(r.snap.Messages))
	line, prevHeight := 0, 0
	var b strings.Builder
	for i := range r.snap.Messages {
		msg := r.snap.Messages[i]
		var prev *chat.Message
		if i > 0 {
			prev = &r.snap.Messages[i-1]
		}
		grouped := isGrouped(prev, msg)
		k := cacheKey{msg.ID, grouped}
		s, ok := r.cache[k]
		if !ok {
			s = renderMessage(r.theme, msg, grouped, r.me.ID, width, r.loc)
		}
		next[k] = s

		if i > 0 {
			b.WriteByte('\n')
			line += prevHeight
			if needsGap(prev, msg, grouped) {
				b.WriteByte('\n')
				line++
			}
		}
		b.WriteString(s)
		starts[msg.ID] = line
		prevHeight = strings.Count(s, "\n") + 1
	}
	r.cache = next
	r.starts = starts
	r.viewport.SetContent(lipgloss.NewStyle().Padding(0, contentPadding).Render(b.String()))
	return r
}

func (r room) update(msg tea.Msg) (room, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		r.viewport, _ = r.viewport.Update(msg)
		return r.syncUnread(), nil
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			r.viewport.GotoBottom()
			r.unread = 0
		}
		return r, nil
	case tea.KeyPressMsg:
		return r.handleKey(msg)
	}
	if r.focus == roomFocusInput {
		return r.updateInput(msg)
	}
	return r, nil
}

func (r room) handleKey(press tea.KeyPressMsg) (room, tea.Cmd) {
	switch {
	case key.Matches(press, r.keys.Profile):
		return r, emit(openSettingsMsg{})
	case key.Matches(press, r.keys.Blur):
		return r.focusOn(roomFocusNone), nil
	case key.Matches(press, r.keys.PageUp):
		r.viewport.PageUp()
		return r.syncUnread(), nil
	case key.Matches(press, r.keys.PageDown):
		r.viewport.PageDown()
		return r.syncUnread(), nil
	}

	switch r.focus {
	case roomFocusNone:
		if key.Matches(press, r.keys.Select) {
			return r.focusOn(roomFocusInput), nil
		}
		if press.Text != "" {
			return r.focusOn(roomFocusInput).updateInput(press)
		}
		r.viewport, _ = r.viewport.Update(press)
		return r.syncUnread(), nil
	case roomFocusInput:
		if key.Matches(press, r.keys.Select) {
			return r.submit()
		}
		r.notice = ""
		return r.updateInput(press)
	}
	return r, nil
}

func (r room) updateInput(msg tea.Msg) (room, tea.Cmd) {
	height := r.input.Height()
	var cmd tea.Cmd
	r.input, cmd = r.input.Update(msg)
	if r.input.Height() != height {
		r = r.layout()
	}
	return r, cmd
}

func (r room) syncUnread() room {
	if r.viewport.AtBottom() {
		r.unread = 0
	}
	return r
}

func (r room) submit() (room, tea.Cmd) {
	text := r.input.Value()
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return r.focusOn(roomFocusInput), nil
	}

	var cmd tea.Cmd
	var err error
	switch {
	case strings.HasPrefix(trimmed, "//"):
		err = r.client.Send(trimmed[1:])
	case strings.HasPrefix(trimmed, "/"):
		r, cmd, err = r.command(trimmed)
	default:
		err = r.client.Send(text)
	}
	if err != nil {
		return r.flash(err.Error(), true).focusOn(roomFocusInput), nil
	}

	r.input.Reset()
	r = r.focusOn(roomFocusInput).layout()
	r.viewport.GotoBottom()
	r.unread = 0
	return r, cmd
}

func (r room) command(line string) (room, tea.Cmd, error) {
	head := strings.Fields(line)[0]
	name := strings.TrimPrefix(head, "/")
	arg := strings.TrimSpace(strings.TrimPrefix(line, head))

	switch strings.ToLower(name) {
	case "me":
		return r, nil, r.client.Emote(arg)
	case "nick", "name":
		if arg == "" {
			return r, nil, fmt.Errorf("usage: /nick <name>")
		}
		return r, emit(renameMsg{name: arg}), nil
	case "join":
		if _, err := r.client.Join(arg); err != nil {
			return r, nil, err
		}
		return r.enter(), nil, nil
	case "leave", "part":
		r.client.Leave()
		return r, emit(leftRoomMsg{}), nil
	case "settings", "profile":
		return r, emit(openSettingsMsg{}), nil
	case "quit", "exit":
		return r, tea.Quit, nil
	case "help", "?":
		return r.flash(commandsHelpLine, false), nil, nil
	default:
		return r, nil, fmt.Errorf("unknown command /%s, try /help", name)
	}
}

func (r room) view() string {
	header := r.header()
	left := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Width(r.viewport.Width()).Render(header),
		r.viewport.View(),
	)
	if r.showSidebar() {
		left = lipgloss.JoinHorizontal(lipgloss.Top, left, r.sidebar(lipgloss.Height(header)+r.viewport.Height()))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		left,
		r.form(),
		helpLine(r.help, r.keys, r.width),
	)
}

func (r room) form() string {
	t := r.theme
	focused := r.focus == roomFocusInput
	inner := r.width - 2

	label := t.label(r.me.Name+":", focused)
	status := ansi.Truncate(r.status(), max(inner-lipgloss.Width(label)-1, 0), ellipsis)
	gap := max(inner-lipgloss.Width(label)-lipgloss.Width(status), 1)

	return lipgloss.NewStyle().Padding(0, 1).Render(lipgloss.JoinVertical(lipgloss.Left,
		label+strings.Repeat(" ", gap)+status,
		t.field(inner, focused, r.input.View()),
	))
}

func (r room) status() string {
	t := r.theme
	var line string
	switch {
	case r.notice != "" && r.isErr:
		line = t.formError(r.notice)
	case r.notice != "":
		line = t.fg(t.primary).Render(r.notice)
	case r.unread > 0:
		noun := "message"
		if r.unread > 1 {
			noun = "messages"
		}
		line = t.fg(t.primary).Bold(true).Render(fmt.Sprintf("%d new %s below (pgdn)", r.unread, noun))
	case r.focus == roomFocusNone:
		line = t.fg(t.muted).Render("scrolling, enter to type")
	case !r.viewport.AtBottom():
		line = t.fg(t.muted).Render("more below (pgdn)")
	}
	return strings.ReplaceAll(line, "\n", " ")
}

func (r room) sidebar(height int) string {
	t := r.theme
	inner := sidebarWidth - 3

	roomName := ansi.Truncate(r.snap.Name, inner-len("Room: "), ellipsis)
	lines := []string{
		"",
		t.bold().Render("Room: ") + t.fg(t.primary).Underline(true).Render(roomName),
	}
	if !r.snap.Listed {
		lines = append(lines, t.fg(t.muted).Render("unlisted"))
	}
	lines = append(lines,
		"",
		t.bold().Render(fmt.Sprintf("Online Count: %d", len(r.snap.Members))),
		t.bold().Render("Online Users:"),
	)

	slots := max(height-len(lines)-1, 1)
	members := r.snap.Members
	overflow := 0
	if len(members) > slots {
		overflow = len(members) - slots + 1
		members = members[:slots-1]
	}
	for _, m := range members {
		lines = append(lines, renderMember(t, m, m.ID == r.me.ID, inner))
	}
	if overflow > 0 {
		lines = append(lines, t.fg(t.muted).Render(fmt.Sprintf("+%d more", overflow)))
	}

	return lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(height).
		Padding(0, 1, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		Render(strings.Join(lines, "\n"))
}
