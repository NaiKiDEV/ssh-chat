package ui

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
)

const logo = `┏┓┓          ┓
┃ ┣┓┏┓╋╋┓┏ ┏┏┣┓
┗┛┛┗┗┻┗┗┗┫━┛┛┛┗
         ┛`

var nameColors = map[string][2]string{
	"orange": {"#D9480F", "#FFA94D"},
	"yellow": {"#946C00", "#FFD43B"},
	"lime":   {"#5C940D", "#C0EB75"},
	"green":  {"#2B8A3E", "#69DB7C"},
	"teal":   {"#0C8599", "#66D9E8"},
	"sky":    {"#1971C2", "#A5D8FF"},
	"blue":   {"#364FC7", "#748FFC"},
	"purple": {"#6741D9", "#B197FC"},
}

type theme struct {
	dark    bool
	text    color.Color
	primary color.Color
	grey    color.Color
	muted   color.Color
	err     color.Color
	cream   color.Color
	frame   color.Color
}

func newTheme(dark bool) theme {
	ld := lipgloss.LightDark(dark)
	return theme{
		dark:    dark,
		text:    ld(lipgloss.Color("235"), lipgloss.Color("15")),
		primary: lipgloss.Color("#F25D94"),
		grey:    lipgloss.Color("#888B7E"),
		muted:   lipgloss.Color("240"),
		err:     lipgloss.Color("#FF0000"),
		cream:   lipgloss.Color("#FFF7DB"),
		frame:   ld(lipgloss.Color("#888B7E"), lipgloss.Color("#FFF7DB")),
	}
}

func (t theme) nameColor(name string) color.Color {
	pair, ok := nameColors[name]
	if !ok {
		pair = nameColors[profile.Colors[0]]
	}
	return lipgloss.LightDark(t.dark)(lipgloss.Color(pair[0]), lipgloss.Color(pair[1]))
}

func (t theme) fg(c color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

func (t theme) regular() lipgloss.Style {
	return t.fg(t.text)
}

func (t theme) bold() lipgloss.Style {
	return t.regular().Bold(true)
}

func (t theme) label(text string, focused bool) string {
	c := t.grey
	if focused {
		c = t.primary
	}
	return t.fg(c).Bold(true).Render(text)
}

func (t theme) button(text string, active bool) string {
	bg := t.grey
	if active {
		bg = t.primary
	}
	return lipgloss.NewStyle().Foreground(t.cream).Background(bg).Padding(0, 2).Render(text)
}

func (t theme) textButton(text string, active bool) string {
	c := t.muted
	if active {
		c = t.primary
	}
	return t.fg(c).Bold(true).Padding(0, 2).Render(text)
}

func (t theme) field(width int, focused bool, content string) string {
	c := t.muted
	if focused {
		c = t.primary
	}
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Padding(0, 1).
		Render(content)
}

func (t theme) formError(text string) string {
	if text == "" {
		return ""
	}
	return t.fg(t.err).Render(fmt.Sprintf("[%s]", text))
}

func (t theme) dialog(width int, parts ...string) string {
	column := lipgloss.NewStyle().Width(width - 2).Align(lipgloss.Center)
	lines := make([]string, len(parts))
	for i, p := range parts {
		lines[i] = column.Render(p)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.frame).
		Padding(1, 0).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (t theme) backdrop(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceChars("|"),
		lipgloss.WithWhitespaceStyle(t.fg(t.muted)),
	)
}

func (t theme) helpStyles() help.Styles {
	s := help.DefaultStyles(t.dark)
	s.ShortKey = t.fg(t.grey).Bold(true)
	s.ShortDesc = t.fg(t.muted)
	s.ShortSeparator = t.fg(t.muted)
	return s
}

func (t theme) inputStyles() textinput.Styles {
	s := textinput.DefaultStyles(t.dark)
	s.Focused.Text = t.regular()
	s.Blurred.Text = t.fg(t.grey)
	s.Focused.Placeholder = t.fg(t.muted)
	s.Blurred.Placeholder = t.fg(t.muted)
	s.Cursor.Color = t.primary
	s.Cursor.Blink = false
	return s
}

func (t theme) textareaStyles() textarea.Styles {
	s := textarea.DefaultStyles(t.dark)
	s.Focused.Base = lipgloss.NewStyle()
	s.Blurred.Base = lipgloss.NewStyle()
	s.Focused.CursorLine = lipgloss.NewStyle()
	s.Focused.Text = t.regular()
	s.Blurred.Text = t.fg(t.grey)
	s.Focused.Placeholder = t.fg(t.muted)
	s.Blurred.Placeholder = t.fg(t.muted)
	s.Cursor.Color = t.primary
	s.Cursor.Blink = false
	return s
}
