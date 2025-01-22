package login

import tea "github.com/charmbracelet/bubbletea"

type RoomJoinRequestedMsg struct {
	RoomId string
}

func createRoomJoinRequestCmd(roomId string) tea.Cmd {
	return func() tea.Msg {
		return RoomJoinRequestedMsg{RoomId: roomId}
	}
}
