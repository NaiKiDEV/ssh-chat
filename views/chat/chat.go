package chat

import (
	"fmt"
	"strings"

	"github.com/NaiKiDEV/ssh-chat/internal/consts"
	"github.com/NaiKiDEV/ssh-chat/internal/model"
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/NaiKiDEV/ssh-chat/internal/terminal"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	noneId = iota - 1
	chatInputId
	sendButtonId
	leaveButtonId
)

const (
	onlineUsersContainerWidth   = 20
	onlineUsersContainerPadding = 1
	onlineUsersContainerBorder  = 1
	logoOffset                  = 6
	messageBoxOffset            = 4
	sendButtonSize              = 4 + 4
	leaveButtonSize             = 5 + 4
	buttonGap                   = 2
	formGap                     = 3
	containerXPadding           = 1
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
	chatInput := createAreaInput("Type your message...", 0, ts.Width-sendButtonSize-leaveButtonSize-buttonGap-containerXPadding*2-formGap)
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

func (c ChatState) Init() tea.Cmd {
	return textarea.Blink
}

func (c ChatState) Reset() ChatState {
	c.chatInput.SetValue("")
	c.chatInput.Focus()
	c.activeInputId = chatInputId
	return c
}

func (c ChatState) SetRoom(roomId string, messages []model.Message) ChatState {
	c.messages = messages
	c.roomId = roomId
	return c
}

func (c ChatState) SetChatState(messages []model.Message, activeUsers []string) ChatState {
	c.activeUsers = activeUsers
	c.messages = messages
	return c
}

// Very dirty, no abstraction, but it might be fine
func (c ChatState) focusNextFocusableElement(backwards bool) ChatState {
	c.chatInput.Blur()

	switch c.activeInputId {
	case noneId:
		if backwards {
			c.activeInputId = leaveButtonId
		} else {
			c.activeInputId = chatInputId
			c.chatInput.Focus()
		}
	case chatInputId:
		if backwards {
			c.activeInputId = noneId
		} else {
			c.activeInputId = sendButtonId
		}
	case sendButtonId:
		if backwards {
			c.activeInputId = chatInputId
			c.chatInput.Focus()
		} else {
			c.activeInputId = leaveButtonId
		}
	case leaveButtonId:
		if backwards {
			c.activeInputId = sendButtonId
		} else {
			c.activeInputId = noneId
		}
	}

	return c
}

func (c ChatState) Update(msg tea.Msg) (ChatState, tea.Cmd) {
	var inCmd tea.Cmd
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			c = c.focusNextFocusableElement(false)
			return c, nil
		case tea.KeyShiftTab:
			c = c.focusNextFocusableElement(true)
			return c, nil
		case tea.KeyEnter:
			if c.activeInputId == noneId {
				c.activeInputId = chatInputId
				c.chatInput.Focus()
				return c, nil
			}
			if c.activeInputId == sendButtonId {
				if value := c.chatInput.Value(); value != "" {
					c.chatInput.SetValue("")
					c.chatInput.Focus()
					c.activeInputId = chatInputId
					return c, createMessageSentCmd(value)
				}
			}
			if c.activeInputId == leaveButtonId {
				return c, createLeaveChatCmd()
			}

			c.chatInput, inCmd = c.chatInput.Update(msg)
			return c, inCmd
		case tea.KeyEsc:
			c.chatInput.Blur()
			c.activeInputId = noneId
			return c, nil
		default:
			c, inCmd = c.handleInput(msg)
		}

	case tea.MouseMsg:
		var cmd tea.Cmd
		c.chatViewport.SetContent(renderMessageView(c.userName, c.messages, c.clientStyles))
		c.chatViewport, cmd = c.chatViewport.Update(msg)
		return c, cmd

	}

	if c.activeInputId == noneId {
		c.chatViewport, vpCmd = c.chatViewport.Update(msg)
	}

	return c, tea.Batch(inCmd, vpCmd)
}

func (c ChatState) handleInput(msg tea.Msg) (ChatState, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch c.activeInputId {
		case chatInputId:
			c.chatInput, cmd = c.chatInput.Update(msg)
		case noneId:
			c.chatViewport, cmd = c.chatViewport.Update(msg)
		}
		return c, cmd
	}
	return c, cmd
}

func (c ChatState) Resize(terminalState *terminal.TerminalState) ChatState {
	c.chatInput.SetWidth(terminalState.Width)

	contentOffset := c.chatInput.Height() + logoOffset + messageBoxOffset
	contentHeight := terminalState.Height - contentOffset

	c.chatViewport.Width = terminalState.Width - onlineUsersContainerWidth - onlineUsersContainerPadding*2 - onlineUsersContainerBorder
	c.chatViewport.Height = contentHeight

	c.contentHeight = contentHeight

	return c
}

func (c ChatState) Render(terminalState *terminal.TerminalState, messages []model.Message, activeUsers []string) string {
	styles := c.clientStyles

	// Header
	logo := lipgloss.NewStyle().
		Foreground(styles.PrimaryColor).
		Padding(0, 1).
		Render(consts.LOGO_NO_MARGIN)

	var roomText string
	if c.roomId != "" {
		roomLabel := styles.BoldRegularTxt.Render("Room: ")
		roomText = lipgloss.NewStyle().PaddingLeft(1).MarginBottom(1).Render(roomLabel + c.roomId)
	}

	header := lipgloss.JoinVertical(lipgloss.Top, logo, roomText)

	// Online Users Card
	styledActiveUsers := strings.Builder{}
	clampedActiveUsersCount := clamp(len(activeUsers), 0, c.contentHeight-1)
	for _, user := range activeUsers[:clampedActiveUsersCount] {
		labelColor := styles.GreyColor
		if user == c.userName {
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
		Height(c.contentHeight).
		Width(onlineUsersContainerWidth).
		Padding(0, onlineUsersContainerPadding, 0).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		Render(activeUsersCountText + onlineUsersLabelText + styledActiveUsersString)

	c.chatViewport.SetContent(renderMessageView(c.userName, messages, styles))

	// Input Box
	inputBox := lipgloss.NewStyle().
		Width(terminalState.Width - sendButtonSize - leaveButtonSize - buttonGap - containerXPadding*2 - 2).
		Render(renderAreaInput(c.userName, c.chatInput, styles))

	// Button Group
	sendButton := renderButton("send", c.activeInputId == sendButtonId, styles)
	buttonGap := strings.Repeat(" ", buttonGap)
	leaveButton := renderButton("leave", c.activeInputId == leaveButtonId, styles)
	buttonGroup := lipgloss.NewStyle().Padding(1, 0).Render(lipgloss.JoinHorizontal(lipgloss.Left, sendButton, buttonGap, leaveButton))

	formContainer := lipgloss.NewStyle().Padding(2, containerXPadding, 0)
	form := formContainer.Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			inputBox,
			strings.Repeat(" ", formGap),
			buttonGroup,
		))

	content := lipgloss.JoinHorizontal(lipgloss.Left, c.chatViewport.View(), onlineUsersContainer)

	ui := lipgloss.JoinVertical(lipgloss.Left, header, content, form)

	return ui
}
