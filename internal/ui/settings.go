package ui

import (
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
)

const (
	fieldName = iota
	fieldColor
	fieldTimeZone
	fieldCancel
	fieldSave
	fieldCount
)

const (
	settingsDialogWidth = 46
	timeZoneMaxLen      = 64
	formLabelWidth      = 12
	formFieldWidth      = 30
)

type settings struct {
	account Account
	theme   theme
	keys    settingsKeys
	help    help.Model
	name    textinput.Model
	tz      textinput.Model
	color   int
	focus   int
	err     string
	saving  bool
	width   int
	height  int
}

func newSettings(a Account, t theme, w, h int) settings {
	p := a.Profile()

	name := textinput.New()
	name.Prompt = ""
	name.CharLimit = chat.MaxNameLen
	name.SetWidth(formFieldWidth - 5)
	name.SetValue(p.Name)

	tz := textinput.New()
	tz.Prompt = ""
	tz.Placeholder = "UTC"
	tz.CharLimit = timeZoneMaxLen
	tz.SetWidth(formFieldWidth - 5)
	tz.SetValue(p.TimeZone)

	s := settings{
		account: a,
		keys:    newSettingsKeys(),
		help:    newHelp(),
		name:    name,
		tz:      tz,
		color:   max(slices.Index(profile.Colors, p.Color), 0),
	}
	return s.setTheme(t).setSize(w, h).focusField(fieldName)
}

func (s settings) setTheme(t theme) settings {
	s.theme = t
	s.name.SetStyles(t.inputStyles())
	s.tz.SetStyles(t.inputStyles())
	s.help.Styles = t.helpStyles()
	return s
}

func (s settings) setSize(w, h int) settings {
	s.width, s.height = w, h
	s.help.SetWidth(w - 2)
	return s
}

func (s settings) focusField(f int) settings {
	s.focus = (f + fieldCount) % fieldCount
	s.name.Blur()
	s.tz.Blur()
	switch s.focus {
	case fieldName:
		s.name.Focus()
	case fieldTimeZone:
		s.tz.Focus()
	}
	return s
}

func (s settings) update(msg tea.Msg) (settings, tea.Cmd) {
	switch msg := msg.(type) {
	case profileSavedMsg:
		s.saving = false
		if msg.err != nil {
			s.err = msg.err.Error()
			return s, nil
		}
		return s, emit(settingsDoneMsg{saved: true})

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, s.keys.Cancel):
			return s, emit(settingsDoneMsg{})
		case key.Matches(msg, s.keys.Save):
			return s.save()
		case key.Matches(msg, s.keys.Next):
			return s.focusField(s.focus + 1), nil
		case key.Matches(msg, s.keys.Prev):
			return s.focusField(s.focus - 1), nil
		case key.Matches(msg, s.keys.Select):
			return s.selectField()
		case key.Matches(msg, s.keys.Left, s.keys.Right):
			if next, ok := s.step(key.Matches(msg, s.keys.Right)); ok {
				return next, nil
			}
		}
		s.err = ""
	}

	var cmd tea.Cmd
	switch s.focus {
	case fieldName:
		s.name, cmd = s.name.Update(msg)
	case fieldTimeZone:
		s.tz, cmd = s.tz.Update(msg)
	}
	return s, cmd
}

func (s settings) step(forward bool) (settings, bool) {
	switch s.focus {
	case fieldColor:
		n := len(profile.Colors)
		if forward {
			s.color = (s.color + 1) % n
		} else {
			s.color = (s.color - 1 + n) % n
		}
		return s, true
	case fieldCancel, fieldSave:
		if forward {
			return s.focusField(fieldSave), true
		}
		return s.focusField(fieldCancel), true
	}
	return s, false
}

func (s settings) selectField() (settings, tea.Cmd) {
	switch s.focus {
	case fieldCancel:
		return s, emit(settingsDoneMsg{})
	case fieldSave:
		return s.save()
	}
	return s.focusField(s.focus + 1), nil
}

func (s settings) draft() profile.Profile {
	return profile.Profile{
		Name:     strings.TrimSpace(s.name.Value()),
		Color:    profile.Colors[s.color],
		TimeZone: strings.TrimSpace(s.tz.Value()),
	}
}

func (s settings) save() (settings, tea.Cmd) {
	if s.saving {
		return s, nil
	}
	p := s.draft()
	if err := p.Validate(); err != nil {
		s.err = err.Error()
		return s, nil
	}
	s.saving = true
	return s, saveProfile(s.account, p)
}

func (s settings) view() string {
	footer := helpLine(s.help, s.keys, s.width)
	avail := s.height - lipgloss.Height(footer)
	dialog := s.dialog(true)
	if lipgloss.Height(dialog) > avail {
		dialog = s.dialog(false)
	}
	return lipgloss.JoinVertical(lipgloss.Left, s.theme.backdrop(s.width, avail, dialog), footer)
}

func (s settings) dialog(spacious bool) string {
	t := s.theme
	p := s.draft()
	gap := []string{""}
	if !spacious {
		gap = nil
	}

	arrows := t.fg(t.muted)
	if s.focus == fieldColor {
		arrows = t.fg(t.primary)
	}
	swatch := t.fg(t.nameColor(p.Color)).Bold(true).Render(p.Color)
	colorPicker := arrows.Render("<") + lipgloss.PlaceHorizontal(formFieldWidth-6, lipgloss.Center, swatch) + arrows.Render(">")

	note := "saved to your SSH key for next time"
	if !s.account.Persistent() {
		note = "guest session, resets when you leave"
	}

	saveLabel := "Save"
	if s.saving {
		saveLabel = "Saving..."
	}

	inner := settingsDialogWidth - 2
	center := lipgloss.NewStyle().Width(inner).Align(lipgloss.Center)
	row := func(label string, field int, value string) string {
		return center.Render(lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Width(formLabelWidth).Render(t.label(label, s.focus == field)),
			t.field(formFieldWidth, s.focus == field, value),
		))
	}

	var parts []string
	add := func(lines ...string) { parts = append(parts, lines...) }
	add(center.Render(t.fg(t.primary).Bold(true).Render("Profile")), center.Render(t.fg(t.muted).Render(note)))
	add(gap...)
	add(row("Name", fieldName, s.name.View()))
	add(row("Color", fieldColor, colorPicker))
	add(row("Time zone", fieldTimeZone, s.tz.View()))
	add(gap...)
	add(center.Render(lipgloss.NewStyle().Width(formLabelWidth + formFieldWidth).Render(
		lipgloss.NewStyle().Width(formLabelWidth).Render(t.fg(t.grey).Bold(true).Render("Preview")) + "  " + s.preview(p),
	)))
	add(center.Render(t.formError(s.err)))
	add(center.Render(lipgloss.JoinHorizontal(lipgloss.Top,
		t.button("Cancel", s.focus == fieldCancel), "  ",
		t.button(saveLabel, s.focus == fieldSave),
	)))

	return t.dialog(settingsDialogWidth, parts...)
}

func (s settings) preview(p profile.Profile) string {
	t := s.theme
	name := p.Name
	if name == "" {
		name = "?"
	}
	loc, err := profile.LoadLocation(p.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	stamp := " (" + time.Now().In(loc).Format(timeLayout) + ")"
	return t.fg(t.nameColor(p.Color)).Bold(true).Render(name) + t.fg(t.muted).Render(stamp)
}
