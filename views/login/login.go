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

	activeElementId int

	userName string
}

type LoginSubmitMsg struct {
	RoomId string
}

func NewLoginState(userName string) LoginState {
	roomTextInput := createTextInput("xxxxxxxxx", 9)
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

func (l LoginState) Update(msg tea.Msg) LoginState {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab {
			return l.focusNextFormElement()
		}

		if l.activeElementId == buttonsId {
			switch msg.String() {
			case "left", "right", "h", "l":
				return l.focusNextButton()
			default:
				return l
			}
		}

		if l.activeElementId == roomInputId {
			if msg.Type == tea.KeyEnter {
				return l.focusNextFormElement()
			}

			l.roomTextInput, _ = l.roomTextInput.Update(msg)
			return l
		}
	}
	return l
}

// Submit returns which button was selected.
// Possible values: LoginConfirmed, LoginQuit, LoginNoop
func (l LoginState) Submit() (string, string) {
	if l.activeButtonId == quitButtonId {
		return LoginQuit, ""
	}
	if l.activeButtonId == loginButtonId {
		return LoginConfirmed, l.roomTextInput.Value()
	}
	return LoginNoop, ""
}

func (l LoginState) CanSubmit() bool {
	return l.activeElementId == buttonsId
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

func createTextInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	ti.Prompt = ""
	ti.Width = 10
	return ti
}

func renderButton(label string, active bool, styles *styles.ClientStyles) string {
	if active {
		return styles.ActiveButton.Render(label)
	}
	return styles.Button.Render(label)
}

func renderTextInput(label string, ti textinput.Model, styles *styles.ClientStyles) string {
	inputFocused := ti.Focused()
	labelColor := styles.GreyColor
	if inputFocused {
		labelColor = styles.PrimaryColor
	}

	container := lipgloss.NewStyle().Width(10).Height(3).Align(lipgloss.Center)
	styledLabel := styles.RegularTxt.Width(10).Foreground(labelColor).Bold(true).Render(label)
	input := lipgloss.JoinVertical(lipgloss.Top, styledLabel, ti.View())
	return container.Render(input)
}

func (l LoginState) Render(terminalState *terminal.TerminalState, styles *styles.ClientStyles) string {
	buttonsAreFocused := l.activeElementId == buttonsId
	quitButton := renderButton("Quit", buttonsAreFocused && l.activeButtonId == quitButtonId, styles)
	okButton := renderButton("Join", buttonsAreFocused && l.activeButtonId == loginButtonId, styles)

	userName := styles.BoldRegularTxt.Foreground(styles.PrimaryColor).Render(l.userName)

	logo := lipgloss.NewStyle().Width(40).Align(lipgloss.Center).MarginBottom(1).Foreground(styles.PrimaryColor).Render(consts.LOGO)
	greeter := lipgloss.NewStyle().Width(40).Padding(0, 0, 1).Align(lipgloss.Center).Render(fmt.Sprintf("Welcome back, %s!", userName))
	form := lipgloss.NewStyle().Padding(1, 0).Render(renderTextInput("Room Id", l.roomTextInput, styles))
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, quitButton, "  ", okButton)

	ui := lipgloss.JoinVertical(lipgloss.Center, logo, greeter, form, buttons)

	dialog := lipgloss.Place(terminalState.Width, terminalState.Height,
		lipgloss.Center, lipgloss.Center,
		styles.DialogBox.Render(ui),
		lipgloss.WithWhitespaceChars("|"),
		lipgloss.WithWhitespaceForeground(styles.PlaceholderTxt.GetForeground()),
	)

	return dialog
}
