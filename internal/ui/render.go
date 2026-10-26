package ui

import (
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/charmbracelet/x/ansi"
)

const (
	groupWindow = 5 * time.Minute
	ellipsis    = "..."
	timeLayout  = "15:04 MST"
)

func isGrouped(prev *chat.Message, msg chat.Message) bool {
	return prev != nil &&
		prev.Kind == chat.KindChat &&
		msg.Kind == chat.KindChat &&
		prev.Author.ID == msg.Author.ID &&
		msg.Time.Sub(prev.Time) < groupWindow
}

func isConversation(k chat.Kind) bool {
	return k == chat.KindChat || k == chat.KindEmote
}

func needsGap(prev *chat.Message, msg chat.Message, grouped bool) bool {
	return !grouped && (isConversation(msg.Kind) || isConversation(prev.Kind))
}

func authorStyle(t theme, m chat.Member, self uint64) lipgloss.Style {
	if m.ID == self {
		return t.fg(t.primary).Bold(true)
	}
	return t.fg(t.nameColor(m.Color)).Bold(true)
}

func renderMessage(t theme, msg chat.Message, grouped bool, self uint64, width int, loc *time.Location) string {
	name := authorStyle(t, msg.Author, self).Render(msg.Author.Name)
	stamp := t.fg(t.muted).Render(" (" + msg.Time.In(loc).Format(timeLayout) + ")")

	switch msg.Kind {
	case chat.KindChat:
		body := t.regular().Render(lipgloss.Wrap(msg.Text, width, ""))
		if grouped {
			return body
		}
		if msg.Author.ID == self {
			name += t.fg(t.muted).Render(" (you)")
		}
		return name + stamp + "\n" + body
	case chat.KindEmote:
		action := t.regular().Italic(true).Render(msg.Text)
		return lipgloss.Wrap(t.fg(t.primary).Render("* ")+name+" "+action+stamp, width, "")
	case chat.KindJoin:
		return systemLine(t, msg.Author.Name+" joined the room", stamp, width)
	case chat.KindLeave:
		return systemLine(t, msg.Author.Name+" left the room", stamp, width)
	case chat.KindRename:
		return systemLine(t, msg.Text+" is now known as "+msg.Author.Name, stamp, width)
	default:
		return ""
	}
}

func systemLine(t theme, text, stamp string, width int) string {
	return ansi.Truncate(t.fg(t.grey).Italic(true).Render(text)+stamp, width, ellipsis)
}

func renderMember(t theme, m chat.Member, isSelf bool, width int) string {
	style := t.fg(t.nameColor(m.Color)).Bold(true)
	suffix := ""
	if isSelf {
		style = t.fg(t.primary).Bold(true)
		suffix = t.fg(t.muted).Render(" (you)")
	}
	prefix := ""
	if m.Guest {
		prefix = t.fg(t.muted).Render("~")
	}
	return ansi.Truncate(prefix+style.Render(m.Name)+suffix, width, ellipsis)
}
