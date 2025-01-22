package login

import (
	"fmt"

	"github.com/NaiKiDEV/ssh-chat/internal/consts"
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/NaiKiDEV/ssh-chat/internal/terminal"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Button identifiers
const (
	quitButtonId = iota
	loginButtonId
)

const (
	roomInputId = iota
	buttonsId
)

// Submit states (Simplified to strings as no data is being returned)
const (
	LoginConfirmed    = "LoginConfirmed"
	LoginQuit         = "LoginQuit"
	LoginSubmitFailed = "LoginSubmitFailed"
	LoginNoop         = "LoginNoop"
)

type LoginState struct {
	activeButtonId int
	roomTextInput  textinput.Model
	formError      string

	activeElementId int

	userName string
}

type LoginSubmitMsg struct {
	RoomId string
}

func NewLoginState(userName string) LoginState {
	roomTextInput := createTextInput("", 9)
	roomTextInput.Focus()
	return LoginState{
		activeButtonId:  loginButtonId,
		roomTextInput:   roomTextInput,
		activeElementId: roomInputId,
		userName:        userName,
	}
}

func (l LoginState) Init() tea.Cmd {
	return textinput.Blink
}

func (l LoginState) Reset() LoginState {
	l.roomTextInput.SetValue("")
	l.roomTextInput.Focus()
	l.activeElementId = roomInputId
	l.formError = ""
	return l
}

func (l LoginState) SetFormError(err string) LoginState {
	l.formError = err
	l.activeElementId = roomInputId
	l.roomTextInput.Focus()
	return l
}

func (l LoginState) Update(msg tea.Msg) (LoginState, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab {
			return l.focusNextFormElement(), cmd
		}

		if l.activeElementId == buttonsId {
			switch msg.String() {
			case "left", "right", "h", "l":
				return l.focusNextButton(), cmd
			case "enter":
				switch l.activeButtonId {
				case quitButtonId:
					return l, tea.Quit
				case loginButtonId:
					roomId := l.roomTextInput.Value()
					if roomId == "" {
						l = l.SetFormError("room id empty")
						return l, cmd
					}
					return l, createRoomJoinRequestCmd(roomId)
				}
			default:
				return l, cmd
			}
		}

		if l.activeElementId == roomInputId {
			if msg.Type == tea.KeyEnter {
				return l.focusNextFormElement(), cmd
			}

			l.roomTextInput, _ = l.roomTextInput.Update(msg)
			return l, cmd
		}
	}
	return l, cmd
}

// Quick and dirty as form is simple
func (l LoginState) focusNextFormElement() LoginState {
	switch l.activeElementId {
	case roomInputId:
		l.activeElementId = buttonsId
		l.roomTextInput.Blur()
	case buttonsId:
		l.activeElementId = roomInputId
		l.roomTextInput.Focus()
	}
	return l
}

func (l LoginState) focusNextButton() LoginState {
	switch l.activeButtonId {
	case quitButtonId:
		l.activeButtonId = loginButtonId
	case loginButtonId:
		l.activeButtonId = quitButtonId
	}
	return l
}

func (l LoginState) Render(terminalState *terminal.TerminalState, styles *styles.ClientStyles) string {
	buttonsAreFocused := l.activeElementId == buttonsId
	quitButton := renderButton("Quit", buttonsAreFocused && l.activeButtonId == quitButtonId, styles)
	okButton := renderButton("Join", buttonsAreFocused && l.activeButtonId == loginButtonId, styles)

	userName := styles.BoldRegularTxt.Foreground(styles.PrimaryColor).Render(l.userName)

	formErrorMessage := ""
	if l.formError != "" {
		formErrorMessage = fmt.Sprintf("[%s]", l.formError)
	}
	formError := lipgloss.NewStyle().Width(30).Align(lipgloss.Center).MarginBottom(1).Foreground(styles.ErrorColor).Render(formErrorMessage)

	logo := lipgloss.NewStyle().Width(40).Align(lipgloss.Center).MarginBottom(1).Foreground(styles.PrimaryColor).Render(consts.LOGO)
	greeter := lipgloss.NewStyle().Width(40).Padding(0, 0, 1).Align(lipgloss.Center).Render(fmt.Sprintf("Welcome, %s!", userName))
	form := lipgloss.NewStyle().Padding(1, 0, 0).Render(renderTextInput("Room Id", l.roomTextInput, styles))
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, quitButton, "  ", okButton)

	ui := lipgloss.JoinVertical(lipgloss.Center, logo, greeter, form, formError, buttons)

	dialog := lipgloss.Place(terminalState.Width, terminalState.Height,
		lipgloss.Center, lipgloss.Center,
		styles.DialogBox.Render(ui),
		lipgloss.WithWhitespaceChars("|"),
		lipgloss.WithWhitespaceForeground(styles.PlaceholderTxt.GetForeground()),
	)

	return dialog
}
