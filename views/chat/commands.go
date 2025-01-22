package chat

import tea "github.com/charmbracelet/bubbletea"

type MessageSentMsg struct {
	Message string
}

type LeaveChatMsg struct{}

func createMessageSentCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return MessageSentMsg{Message: message}
	}
}

func createLeaveChatCmd() tea.Cmd {
	return func() tea.Msg {
		return LeaveChatMsg{}
	}
}
