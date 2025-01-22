package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/NaiKiDEV/ssh-chat/internal/model"
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

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
	inputFocused := ta.Focused()
	labelColor := styles.GreyColor
	if inputFocused {
		labelColor = styles.PrimaryColor
	}

	styledLabel := styles.BoldRegularTxt.Foreground(labelColor).Render(label)
	input := lipgloss.JoinVertical(lipgloss.Top, styledLabel+": ", ta.View())
	return input
}

func renderMessageView(loggedInUsername string, messages []model.Message, styles *styles.ClientStyles) string {
	if messages == nil {
		return ""
	}

	messageContent := strings.Builder{}
	for _, msg := range messages {
		messageContent.WriteString(renderMessage(msg.Text, msg.Username, msg.Username == loggedInUsername, msg.Timestamp, styles))
		messageContent.WriteRune('\n')
	}

	return messageContent.String()
}

func renderButton(label string, active bool, styles *styles.ClientStyles) string {
	if active {
		return styles.ActiveButton.Bold(true).UnsetBackground().Foreground(styles.PrimaryColor).Render(label)
	}
	return styles.Button.Bold(true).UnsetBackground().Foreground(styles.MutedColor).Render(label)
}
