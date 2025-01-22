package login

import (
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

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
