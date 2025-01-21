package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/NaiKiDEV/ssh-chat/internal/consts"
	"github.com/NaiKiDEV/ssh-chat/internal/model"
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/NaiKiDEV/ssh-chat/internal/terminal"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	noneId = iota - 1
	chatInputId
)

const (
	onlineUsersContainerWidth   = 20
	onlineUsersContainerPadding = 1
	onlineUsersContainerBorder  = 1
	logoOffset                  = 6
	messageBoxOffset            = 4
)

type ChatState struct {
	chatInput         textarea.Model
	chatInputExpanded bool
	chatViewport      viewport.Model
	contentHeight     int
	activeInputId     int
	messages          []model.Message
	activeUsers       []string
	roomId            string
	userName          string
	clientStyles      *styles.ClientStyles
}

func NewChatState(userName string, ts *terminal.TerminalState, cs *styles.ClientStyles) ChatState {
	chatInput := createAreaInput("Type your message...", 0, ts.Width)
	chatInput.Focus()

	contentOffset := chatInput.Height() + logoOffset + messageBoxOffset
	contentHeight := ts.Height - contentOffset

	chatViewport := viewport.New(ts.Width-onlineUsersContainerPadding*2-onlineUsersContainerWidth-onlineUsersContainerBorder, contentHeight)
	chatViewport.YPosition = logoOffset
	chatViewport.Height = contentHeight
	chatViewport.MouseWheelEnabled = true

	return ChatState{
		userName:          userName,
		chatInput:         chatInput,
		chatViewport:      chatViewport,
		activeInputId:     chatInputId,
		contentHeight:     contentHeight,
		chatInputExpanded: false,
		clientStyles:      cs,
	}
}

func createAreaInput(placeholder string, limit int, initialWidth int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.CharLimit = limit
	ta.ShowLineNumbers = false

	ta.MaxHeight = 2
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.KeyMap.InsertNewline = key.NewBinding()

	ta.Prompt = ""

	ta.SetHeight(2)
	ta.SetWidth(initialWidth)

	return ta
}

func (l ChatState) Update(msg tea.Msg) (ChatState, tea.Cmd) {
	var inCmd tea.Cmd
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if l.activeInputId == noneId {
				l.activeInputId = chatInputId
				l.chatInput.Focus()
			}
		case "enter":
			if l.activeInputId == noneId {
				l.chatInput.Focus()
				l.activeInputId = chatInputId
			}
		case "esc":
			l.chatInput.Blur()
			l.activeInputId = noneId
		default:
			l, inCmd = l.handleInput(msg)
		}
	}

	if l.activeInputId == noneId {
		l.chatViewport, vpCmd = l.chatViewport.Update(msg)
	}

	return l, tea.Batch(inCmd, vpCmd)
}

func (l ChatState) handleInput(msg tea.Msg) (ChatState, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch l.activeInputId {
		case chatInputId:
			l.chatInput, cmd = l.chatInput.Update(msg)
		default:
			l.chatViewport, cmd = l.chatViewport.Update(msg)
		}
		return l, cmd
	}
	return l, cmd
}

func (l ChatState) getMessageView(messages []model.Message, styles *styles.ClientStyles) string {
	if messages == nil {
		return ""
	}

	messageContent := strings.Builder{}
	for _, msg := range messages {
		messageContent.WriteString(renderMessage(msg.Text, msg.Username, msg.Username == l.userName, msg.Timestamp, styles))
		messageContent.WriteRune('\n')
	}

	return messageContent.String()
}

func (l ChatState) HandleMouse(msg tea.MouseMsg) (ChatState, tea.Cmd) {
	l.chatViewport.SetContent(l.getMessageView(l.messages, l.clientStyles))

	var cmd tea.Cmd
	l.chatViewport, cmd = l.chatViewport.Update(msg)

	// if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
	// 	l.activeInputId = noneId
	// 	l.chatInput.Blur()
	// }

	return l, cmd
}

func (l ChatState) Submit() (ChatState, string) {
	value := l.chatInput.Value()
	l.chatInput.SetValue("")
	return l, value
}

func (l ChatState) CanSubmit() bool {
	return l.activeInputId == chatInputId
}

func formatTime(time time.Time) string {
	return fmt.Sprintf("%02d:%02d UTC", time.UTC().Hour(), time.UTC().Minute())
}

func clamp(value, min, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func renderMessage(message string, userName string, isOwned bool, timestamp time.Time, styles *styles.ClientStyles) string {
	container := lipgloss.NewStyle().Padding(0, 1, 1)

	labelColor := styles.GreyColor
	if isOwned {
		labelColor = styles.PrimaryColor
	}

	styledLabel := styles.BoldRegularTxt.Foreground(labelColor).Render(userName)
	styledMessage := styles.RegularTxt.Render(message)
	styledTimestamp := styles.RegularTxt.Foreground(styles.MutedColor).Render(fmt.Sprintf(" (%s) ", formatTime(timestamp)))

	messageCard := lipgloss.JoinVertical(lipgloss.Top, styledLabel+styledTimestamp, styledMessage)

	return container.Render(messageCard)
}

func renderAreaInput(label string, ta textarea.Model, styles *styles.ClientStyles) string {
	styledLabel := styles.BoldRegularTxt.Foreground(styles.PrimaryColor).Render(label)
	input := lipgloss.JoinVertical(lipgloss.Top, styledLabel+": ", ta.View())
	return input
}

func (l ChatState) Resize(terminalState *terminal.TerminalState) ChatState {
	l.chatInput.SetWidth(terminalState.Width)

	contentOffset := l.chatInput.Height() + logoOffset + messageBoxOffset
	contentHeight := terminalState.Height - contentOffset

	l.chatViewport.Width = terminalState.Width - onlineUsersContainerWidth - onlineUsersContainerPadding*2 - onlineUsersContainerBorder
	l.chatViewport.Height = contentHeight

	l.contentHeight = contentHeight

	return l
}

func (l ChatState) SetRoom(roomId string, messages []model.Message) ChatState {
	l.messages = messages
	l.roomId = roomId
	return l
}

func (l ChatState) SetChatState(messages []model.Message, activeUsers []string) ChatState {
	l.activeUsers = activeUsers
	l.messages = messages
	return l
}

func (l ChatState) Render(terminalState *terminal.TerminalState, messages []model.Message, activeUsers []string) string {
	styles := l.clientStyles

	// Header
	logo := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Padding(0, 1).
		Render(consts.LOGO_NO_MARGIN)

	var roomText string
	if l.roomId != "" {
		roomLabel := styles.BoldRegularTxt.Render("Room: ")
		roomText = lipgloss.NewStyle().PaddingLeft(1).MarginBottom(1).Render(roomLabel + l.roomId)
	}

	header := lipgloss.JoinVertical(lipgloss.Top, logo, roomText)

	// Online Users Card
	styledActiveUsers := strings.Builder{}
	clampedActiveUsersCount := clamp(len(activeUsers), 0, l.contentHeight-1)
	for _, user := range activeUsers[:clampedActiveUsersCount] {
		labelColor := styles.GreyColor
		if user == l.userName {
			labelColor = styles.PrimaryColor
		}
		onlineUserText := styles.BoldRegularTxt.Foreground(labelColor).Render(user)
		styledActiveUsers.WriteString(onlineUserText + "\n")
	}

	sideBarLabelTextStyle := lipgloss.NewStyle().Width(onlineUsersContainerWidth).Bold(true)
	activeUsersCountText := sideBarLabelTextStyle.Render(fmt.Sprintf("Online Count: %d\n", len(activeUsers)))

	onlineUsersLabelText := ""
	styledActiveUsersString := styledActiveUsers.String()
	if styledActiveUsersString != "" {
		onlineUsersLabelText = sideBarLabelTextStyle.Render("Online Users:\n")
		styledActiveUsersString += "\n"
	}
	onlineUsersContainer := lipgloss.NewStyle().
		Height(l.contentHeight).
		Width(onlineUsersContainerWidth).
		Padding(0, onlineUsersContainerPadding, 0).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		Render(activeUsersCountText + onlineUsersLabelText + styledActiveUsersString)

	l.chatViewport.SetContent(l.getMessageView(messages, styles))

	// Chat Input
	form := lipgloss.NewStyle().
		Width(terminalState.Width).
		Padding(2, 1, 0).
		Height(2).
		Render(renderAreaInput(l.userName, l.chatInput, styles))

	content := lipgloss.JoinHorizontal(lipgloss.Left, l.chatViewport.View(), onlineUsersContainer)

	ui := lipgloss.JoinVertical(lipgloss.Left, header, content, form)

	return ui
}
